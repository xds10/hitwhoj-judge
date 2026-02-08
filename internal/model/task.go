package model

// TestCase 单个测试用例
type TestCase struct {
	InputFile  string `json:"input_file"`  // 输入数据文件路径
	OutputFile string `json:"output_file"` // 期望输出文件路径
	Input      string `json:"input"`       // 输入数据
	Output     string `json:"output"`      // 期望输出
	Score      int    `json:"score"`       // 测试点分值
}

// JudgeTask 完整评测任务
type JudgeTask struct {
	TaskID int64 `json:"task_id"` // 任务唯一标识
	// UserID      int        `json:"user_id"`      // 用户ID
	// ProblemID   int        `json:"problem_id"`   // 题目ID
	// ContestID   *int       `json:"contest_id"`   // 比赛ID（可选）
	TempDir     string     `json:"temp_dir"`     // 临时目录
	Code        string     `json:"code"`         // 用户代码
	Config      TaskConfig `json:"config"`       // 评测配置
	TestCases   []TestCase `json:"test_cases"`   // 测试用例列表
	FileBucket  string     `json:"file_bucket"`  // 文件存储桶名称
	SpecialCode *string    `json:"special_code"` // 特殊评测代码（可选）
	CreateTime  int64      `json:"create_time"`  // 任务创建时间戳
}

type RunParams struct {
	TestCaseIndex int    `json:"test_case_index"` // 测试用例索引
	ExePath       string `json:"exe_path"`        // 可执行文件路径
	Input         string `json:"input"`           // 输入数据
	InputFile     string `json:"input_file"`      // 输入数据文件路径
	TimeLimit     int64  `json:"time_limit"`      // 时间限制（秒）
	MemLimit      int64  `json:"mem_limit"`       // 内存限制（字节）
	StackLimit    int64  `json:"stack_limit"`     // 栈限制（字节）
}
