package model

// NsJailConfig NsJail 配置
type NsJailConfig struct {
	TimeLimit  int    // 时间限制（秒）
	MemLimit   int    // 内存限制（MB）
	ChrootPath string // chroot 路径（建议为空或 /）
	UserID     int    // 运行用户ID（非root，如 99999）
	ExecPath   string // 待评测可执行文件路径
	InputData  string // 输入数据
}
