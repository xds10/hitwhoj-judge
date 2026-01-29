package handler

import (
	"hitwh-judge/api"
	v1 "hitwh-judge/api/calc/v1"
	"hitwh-judge/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AddTaskHandler(c *gin.Context) {
	var req *v1.TaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("add-task bind json failed", zap.Error(err))
		api.ResponseError(c, api.CodeInvalidParam)
		return
	}
	zap.L().Info("add-task", zap.Any("req", req))
	judgeResult, err := service.AddTask(c, req)
	if err != nil {
		zap.L().Error("add-task failed", zap.Error(err))
		api.ResponseError(c, api.CodeInternalError)
		return
	}
	api.ResponseSuccess(c, judgeResult)
}
