package model

import "time"

// JudgeStatus 评测状态
type JudgeStatus = string

const (
	StatusPending JudgeStatus = "PENDING" // 待评测
	StatusRunning JudgeStatus = "RUNNING" // 评测中
	StatusCE      JudgeStatus = "CE"      // 编译错误
	StatusAC      JudgeStatus = "AC"      // 答案正确
	StatusWA      JudgeStatus = "WA"      // 答案错误
	StatusTLE     JudgeStatus = "TLE"     // 时间超限
	StatusMLE     JudgeStatus = "MLE"     // 内存超限
	StatusRE      JudgeStatus = "RE"      // 运行时错误
	StatusSE      JudgeStatus = "SE"      // 系统错误
)

// CompileResult 编译结果
type CompileResult struct {
	Success bool   `json:"success"` // 是否编译成功
	Message string `json:"message"` // 编译信息/错误
}

// TestCaseResult 单个测试点结果
type TestCaseResult struct {
	TestCaseIndex int           `json:"test_case_index"` // 测试点索引
	Status        JudgeStatus   `json:"status"`          // 测试点状态
	Score         int           `json:"score"`           // 获得分值
	TimeUsed      time.Duration `json:"time_used"`       // 实际运行时间
	MemUsed       uint64        `json:"mem_used"`        // 实际内存使用
	Output        string        `json:"output"`          // 程序输出
	Expected      string        `json:"expected"`        // 期望输出
	Error         string        `json:"error"`           // 错误信息
}

// JudgeResult 完整评测结果
type JudgeResult struct {
	TaskID        int64            `json:"task_id"`         // 对应任务ID
	Status        JudgeStatus      `json:"status"`          // 最终评测状态
	TotalScore    int              `json:"total_score"`     // 总得分
	TotalTimeUsed time.Duration    `json:"total_time_used"` // 总耗时
	TotalMemUsed  uint64           `json:"total_mem_used"`  // 最大内存使用
	CompileResult CompileResult    `json:"compile_result"`  // 编译结果
	TestResults   []TestCaseResult `json:"test_results"`    // 所有测试点结果
	CodeFileID    int              `json:"code_file_id"`    // 代码文件ID
	SubmitTime    time.Time        `json:"submit_time"`     // 提交时间
	JudgeTime     time.Time        `json:"judge_time"`      // 评测完成时间
	Error         string           `json:"error"`           // 评测错误信息
}
