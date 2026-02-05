package handler

import (
	"hitwh-judge/api"
	"hitwh-judge/internal/cache"
	"hitwh-judge/internal/service"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheckHandler 健康检查接口
func HealthCheckHandler(c *gin.Context) {
	api.ResponseSuccess(c, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "hitwhoj-judge",
	})
}

// MetricsHandler 获取评测统计信息
func MetricsHandler(c *gin.Context) {
	metrics := service.GetGlobalMetrics()
	snapshot := metrics.GetSnapshot()

	api.ResponseSuccess(c, snapshot)
}

// SystemInfoHandler 获取系统信息
func SystemInfoHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := gin.H{
		// Go运行时信息
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"cpu_cores":  runtime.NumCPU(),

		// 内存信息
		"memory": gin.H{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
			"gc_count":       m.NumGC,
		},

		// 评测队列信息
		"judge_stats": service.GetJudgeStats(),

		// 缓存信息
		"cache_stats": cache.GetEnhancedTestFileCache().GetCacheStats(),
	}

	api.ResponseSuccess(c, info)
}

// ReadinessHandler 就绪检查（用于K8s等）
func ReadinessHandler(c *gin.Context) {
	// 检查关键组件是否就绪
	stats := service.GetJudgeStats()

	// 如果评测队列已满，返回未就绪
	if stats["available_slots"].(int) == 0 {
		api.ResponseError(c, api.CodeInternalError)
		return
	}

	api.ResponseSuccess(c, gin.H{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	})
}

// LivenessHandler 存活检查（用于K8s等）
func LivenessHandler(c *gin.Context) {
	api.ResponseSuccess(c, gin.H{
		"status":    "alive",
		"timestamp": time.Now().Unix(),
	})
}
