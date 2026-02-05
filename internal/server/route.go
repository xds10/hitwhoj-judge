package server

import (
	"hitwh-judge/internal/handler"
	"hitwh-judge/internal/handler/calc"
	"hitwh-judge/pkg/logging"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func SetupRoutes(cfg *viper.Viper) *gin.Engine {
	r := gin.New()
	r.Use(logging.GinLogger(), logging.GinRecovery(true)) // 日志中间件，记录请求日志
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowHeaders = append(corsCfg.AllowHeaders, "Authorization")
	corsCfg.AllowAllOrigins = true
	r.Use(cors.New(corsCfg)) // CORS 跨域中间件，简单粗暴，直接放行所有跨域请求

	// 健康检查和监控端点（不需要认证）
	r.GET("/health", handler.HealthCheckHandler)
	r.GET("/metrics", handler.MetricsHandler)
	r.GET("/system", handler.SystemInfoHandler)
	r.GET("/readiness", handler.ReadinessHandler)
	r.GET("/liveness", handler.LivenessHandler)

	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/add", calc.AddHandler())
		apiV1.POST("/task/add", handler.AddTaskHandler)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"msg": "404",
		})
	})
	return r
}
