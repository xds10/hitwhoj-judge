package service

import (
	"hitwh-judge/internal/model"
	"hitwh-judge/internal/task/runner"
)

// JudgeController 评测控制器，负责协调runner和资源限制
type JudgeController struct {
	runner runner.Runner
}

// NewJudgeController 创建新的评测控制器
func NewJudgeController(runner runner.Runner) *JudgeController {
	return &JudgeController{
		runner: runner,
	}
}

// RunWithResourceLimit 运行带有资源限制的评测
func (jc *JudgeController) RunWithResourceLimit(exePath, input string, config model.TaskConfig) (*model.RunResult, error) {
	return nil, nil
}

type ResourceUsage struct {
	CPUUsec int64
	MemPeak int64
}
