package model

import "time"

// JudgeResult 评测结果
type JudgeResult struct {
	Status   string // AC/WA/TLE/MLE/RE 等
	Output   string // 程序输出
	Expected string // 期望输出
	TimeUsed time.Duration
	MemUsed  uint64
	Error    string // 错误信息（如运行时错误）
}
