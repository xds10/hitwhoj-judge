package model

// NsJailConfig NsJail 配置
type NsJailConfig struct {
	TimeLimit   int    // 时间限制（秒）
	MemoryLimit int    // 内存限制（MB）
	NsJailPath  string // nsjail可执行文件路径
	GCCPath     string // gcc可执行文件路径
}

var DefaultJudgeConfig = NsJailConfig{
	TimeLimit:   1,
	MemoryLimit: 64,
	NsJailPath:  "nsjail",
	GCCPath:     "gcc",
}
