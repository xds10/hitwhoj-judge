package model

// LanguageType 编程语言类型
type LanguageType = string

const (
	LanguageC    LanguageType = "c"
	LanguageCPP  LanguageType = "cpp"
	LanguageJava LanguageType = "java"
	LanguagePy   LanguageType = "python"
)

// JudgeType 评测类型
type JudgeType = string

const (
	JudgeNormal      JudgeType = "normal"      // 普通评测
	JudgeSpecial     JudgeType = "special"     // 特殊评测
	JudgeIO          JudgeType = "io"          // IO比对评测
	JudgeInteractive JudgeType = "interactive" // 交互题评测
)

// TaskConfig 评测任务配置
type TaskConfig struct {
	TimeLimit   int          `json:"time_limit"`    // 时间限制（秒）
	MemoryLimit int          `json:"memory_limit"`  // 内存限制（MB）
	StackLimit  int          `json:"stack_limit"`   // 栈内存限制（MB，可选）
	Language    LanguageType `json:"language"`      // 编程语言
	JudgeType   JudgeType    `json:"judge_type"`    // 评测类型
	IsO2Enabled bool         `json:"is_o2_enabled"` // 是否启用O2优化
}

// DefaultTaskConfig 默认评测配置
var DefaultTaskConfig = TaskConfig{
	TimeLimit:   1,
	MemoryLimit: 64,
	StackLimit:  8,
	Language:    LanguageC,
	JudgeType:   JudgeIO,
	IsO2Enabled: false,
}

// SandboxConfig 沙箱配置
type SandboxConfig struct {
	Type        string                  `json:"type"`         // 沙箱类型（nsjail/docker等）
	Path        string                  `json:"path"`         // 沙箱可执行文件路径
	CompilerMap map[LanguageType]string `json:"compiler_map"` // 编译器路径映射
}
