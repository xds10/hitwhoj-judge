package main

import (
	"context"
	"flag"
	"fmt"
	"hitwh-judge/internal/dao"
	"time"

	"hitwh-judge/internal/conf"
	"hitwh-judge/internal/server"
	"hitwh-judge/pkg/jwt"
	"hitwh-judge/pkg/logging"
	"hitwh-judge/pkg/snowflake"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var confPath = flag.String("conf", "./config/config.yaml", "配置文件路径")

// 查询PostgreSQL所有表的函数
func listPostgresTables(cfg *viper.Viper, logger *zap.Logger) {
	// 从dao获取GORM的Postgres连接
	db := dao.DB
	if db == nil {
		logger.Error("postgres connection is nil") // zap支持直接传字符串，也可用Errorf
		return
	}

	// PostgreSQL查询所有用户表的SQL（public模式，排除系统表）
	sql := `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public' 
		AND tablename NOT LIKE 'pg_%' 
		AND tablename NOT LIKE 'sql_%';
	`

	// ========== 关键修改：GORM执行原生SQL ==========
	// 定义接收结果的结构体（字段名对应查询结果的列名，大小写不敏感）
	type TableName struct {
		Tablename string `gorm:"column:tablename"`
	}
	var tables []TableName

	// GORM执行原生SQL并扫描结果到切片
	if err := db.Raw(sql).Scan(&tables).Error; err != nil {
		// logger.Errorf("query postgres tables failed, err: %v", err) // zap用Errorf格式化错误
		return
	}

	// 处理结果：提取表名
	var tableNames []string
	for _, t := range tables {
		tableNames = append(tableNames, t.Tablename)
	}

	// 打印结果（zap用Infof输出）
	if len(tableNames) == 0 {
		logger.Info("postgres public schema has no tables")
		return
	}
	logger.Info("postgres public schema tables: [%s]", zap.Strings("tables", tableNames))
}

func listMinIOBuckets(logger *zap.Logger) {
	// 从dao获取MinIO客户端（需确保dao中暴露了MinIO客户端实例，变量名根据实际情况调整）
	minioClient := dao.MinIOClient
	if minioClient == nil {
		logger.Error("minio client is nil")
		return
	}
	ctx := context.Background()
	// 获取所有桶列表
	buckets, err := minioClient.ListBuckets(ctx)
	if err != nil {
		logger.Error("list minio buckets failed", zap.Error(err))
		return
	}

	// 处理结果：提取桶的核心信息
	if len(buckets) == 0 {
		logger.Info("minio has no buckets")
		return
	}

	// 定义结构体存储桶信息（便于日志输出）
	type BucketInfo struct {
		Name         string    `json:"name"`          // 桶名
		CreationTime time.Time `json:"creation_time"` // 创建时间
	}
	var bucketInfos []BucketInfo
	var bucketNames []string // 仅桶名列表，便于快速查看

	for _, b := range buckets {
		bucketInfos = append(bucketInfos, BucketInfo{
			Name:         b.Name,
			CreationTime: b.CreationDate,
		})
		bucketNames = append(bucketNames, b.Name)
	}

	// 输出详细信息和简洁列表
	logger.Info("minio buckets list (detailed)", zap.Any("buckets", bucketInfos))
	logger.Info("minio buckets list (names only)", zap.Strings("bucket_names", bucketNames))
}

func main() {
	// 加载配置
	flag.Parse()
	cfg := conf.Load(*confPath)

	// 初始化日志
	logger, err := logging.NewLogger(cfg)
	if err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer logger.Sync()

	//dao.MustInitMySQL(cfg)  // 初始化 MySQL 连接
	//dao.MustInitRedis(cfg)  // 初始化 Redis
	dao.MustInitPostgres(cfg) // 初始化 Postgres 连接
	dao.MustInitMinIO(cfg)    // 初始化 MinIO 连接
	jwt.MustInit(cfg)         // 初始化 jwt
	snowflake.MustInit(cfg)   // 初始化 snowflake

	// 查询PostgreSQL所有表
	// listPostgresTables(cfg, logger)
	listMinIOBuckets(logger)

	// 初始化路由
	r := server.SetupRoutes(cfg)
	// 启动服务
	err = r.Run(fmt.Sprintf(":%d", cfg.GetInt("server.port")))
	if err != nil {
		panic(err)
	}
}
