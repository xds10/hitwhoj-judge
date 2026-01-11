package main

import (
	"flag"
	"fmt"
	"hitwh-judge/internal/dao"

	"hitwh-judge/internal/conf"
	"hitwh-judge/internal/server"
	"hitwh-judge/pkg/jwt"
	"hitwh-judge/pkg/logging"
	"hitwh-judge/pkg/snowflake"
)

var confPath = flag.String("conf", "./config/config.yaml", "配置文件路径")

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

	// 初始化路由
	r := server.SetupRoutes(cfg)
	// 启动服务
	err = r.Run(fmt.Sprintf(":%d", cfg.GetInt("server.port")))
	if err != nil {
		panic(err)
	}
}
