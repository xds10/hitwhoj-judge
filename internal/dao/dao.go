package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB          *gorm.DB      // 全局数据库连接
	RedisClient *redis.Client // 全局 Redis 连接
	MinIOClient *minio.Client // 全局 MinIO 客户端连接

)

// MustInitMySQL 初始化 MySQL 连接
func MustInitMySQL(cfg *viper.Viper) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.GetString("mysql.user"),
		cfg.GetString("mysql.password"),
		cfg.GetString("mysql.host"),
		cfg.GetString("mysql.port"),
		cfg.GetString("mysql.dbname"),
	)
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect db fail: %w", err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("connect db fail: %w", err))
	}
	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.GetInt("mysql.max_idle_conns"))
	sqlDB.SetMaxOpenConns(cfg.GetInt("mysql.max_open_conns"))
	sqlDB.SetConnMaxLifetime(cfg.GetDuration("mysql.max_lifetime"))
	// query.SetDefault(db)
}

// MustInitPostgres 完善后的 Postgres 初始化函数
func MustInitPostgres(cfg *viper.Viper) {
	// 1. 拼接 Postgres DSN（补充 TimeZone 配置，避免时区问题）
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		cfg.GetString("postgres.host"),
		cfg.GetInt("postgres.port"),
		cfg.GetString("postgres.user"),
		cfg.GetString("postgres.password"),
		cfg.GetString("postgres.dbname"),
	)

	// 2. 打开 Postgres 连接，配置 GORM 日志（方便调试）
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 开启 SQL 日志，生产环境可改为 Silent
	})
	if err != nil {
		panic(fmt.Errorf("postgres connect fail: %w", err))
	}

	// 3. 获取底层 sql.DB 对象，配置连接池（和 MySQL 逻辑对齐）
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("get postgres sql.DB fail: %w", err))
	}

	// 4. 配置连接池参数（从 viper 读取配置，需在配置文件中定义）
	// 配置项示例（yaml/toml 等）：
	// postgres:
	//   max_idle_conns: 10
	//   max_open_conns: 100
	//   max_lifetime: 30m
	sqlDB.SetMaxIdleConns(cfg.GetInt("postgres.max_idle_conns"))       // 最大空闲连接数
	sqlDB.SetMaxOpenConns(cfg.GetInt("postgres.max_open_conns"))       // 最大打开连接数
	sqlDB.SetConnMaxLifetime(cfg.GetDuration("postgres.max_lifetime")) // 连接最大存活时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)                         // 补充：连接最大空闲时间（可选）

	// 5. 校验连接有效性（避免连接成功但不可用的情况）
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		panic(fmt.Errorf("postgres ping fail: %w", err))
	}

	DB = db

	fmt.Println("postgres connect success!") // 可选：打印成功日志，方便排查
}

// MustInitRedis 初始化 Redis 连接
func MustInitRedis(conf *viper.Viper) {
	addr := fmt.Sprintf("%s:%d", conf.GetString("redis.host"), conf.GetInt("redis.port"))
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: conf.GetString("redis.password"),
		DB:       conf.GetInt("redis.db"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Errorf("init redis failed, err:%w", err))
	}
	RedisClient = rdb
}

// MustInitMinIO 初始化 MinIO 客户端连接
// 失败则 panic，和其他初始化函数保持一致的错误处理风格
func MustInitMinIO(cfg *viper.Viper) {
	// 1. 从配置文件读取 MinIO 连接信息
	endpoint := cfg.GetString("minio.endpoint")    // MinIO 地址（如：127.0.0.1:9000）
	accessKey := cfg.GetString("minio.access_key") // 访问密钥（用户名）
	secretKey := cfg.GetString("minio.secret_key") // 秘密密钥（密码）
	useSSL := cfg.GetBool("minio.use_ssl")         // 是否启用 HTTPS
	region := cfg.GetString("minio.region")        // 区域（如：us-east-1，默认空即可）

	// 2. 创建 MinIO 客户端
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		panic(fmt.Errorf("create minio client fail: %w", err))
	}

	// 3. 校验连接有效性（调用 ListBuckets 测试）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.ListBuckets(ctx)
	if err != nil {
		panic(fmt.Errorf("minio connect fail: %w", err))
	}

	// 4. 赋值给全局变量
	MinIOClient = client
	fmt.Println("minio connect success!")
}
