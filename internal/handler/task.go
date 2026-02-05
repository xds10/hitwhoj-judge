package handler

import (
	"hitwh-judge/api"
	v1 "hitwh-judge/api/calc/v1"
	"hitwh-judge/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AddTaskHandler 评测任务处理函数（使用改进版API）
func AddTaskHandler(c *gin.Context) {
	var req *v1.TaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("add-task bind json failed", zap.Error(err))
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}
	zap.L().Info("add-task", zap.Any("req", req))

	// 使用改进版API：
	// ✅ 修复状态计算错误（不再硬编码为AC）
	// ✅ 正确累加时间和统计内存
	// ✅ 添加并发控制（避免资源竞争）
	// ✅ 添加超时保护（避免请求阻塞）
	// ✅ 完善错误处理
	// ✅ 详细的日志记录
	judgeResult, err := service.AddTaskImproved(c, req)
	if err != nil {
		zap.L().Error("add-task failed", zap.Error(err))
		api.ResponseError(c, api.CodeInternalError)
		return
	}
	api.ResponseSuccess(c, judgeResult)
}
