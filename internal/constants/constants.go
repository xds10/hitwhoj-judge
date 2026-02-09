package constants

import "time"

// 评测相关常量
const (
	// 默认资源限制
	DefaultTimeLimit   = 1000              // 默认时间限制（毫秒）
	DefaultMemoryLimit = 256 * 1024 * 1024 // 默认内存限制（256MB）
	DefaultStackLimit  = 8 * 1024 * 1024   // 默认栈限制（8MB）
	DefaultProcLimit   = 1                 // 默认进程数限制

	// 资源限制范围
	MinTimeLimit   = 100                // 最小时间限制（毫秒）
	MaxTimeLimit   = 60000              // 最大时间限制（60秒）
	MinMemoryLimit = 16 * 1024 * 1024   // 最小内存限制（16MB）
	MaxMemoryLimit = 1024 * 1024 * 1024 // 最大内存限制（1GB）

	// 评测超时配置
	MaxJudgeTimeout     = 5 * time.Minute  // 单个评测任务最大超时时间
	MaxCompileTimeout   = 30 * time.Second // 编译超时时间
	MaxQueueWaitTimeout = 30 * time.Second // 队列等待超时时间

	// 并发控制
	DefaultMaxConcurrent = 2  // 默认最大并发评测数
	MinConcurrent        = 1  // 最小并发数
	MaxConcurrent        = 16 // 最大并发数

	// 输出限制
	MaxOutputSize = 10 * 1024 * 1024 // 最大输出大小（10MB）
	MaxErrorSize  = 1024             // 最大错误信息大小（1KB）

	// 临时文件
	TempDirPrefix = "oj-judge-" // 临时目录前缀
	TempDirPerm   = 0777        // 临时目录权限
	CodeFilePerm  = 0600        // 代码文件权限
)

// 缓存相关常量
const (
	// 缓存配置
	DefaultCacheTTL       = 30 * time.Minute       // 默认缓存过期时间
	DefaultCleanFrequency = 10 * time.Minute       // 默认清理频率
	DefaultMaxDiskUsage   = 2 * 1024 * 1024 * 1024 // 默认最大磁盘使用（2GB）

	// 缓存目录
	CacheDirName = "judge-cache-enhanced"
	CacheDirPerm = 0755
)

// 沙箱相关常量
const (
	// NsJail 配置
	NsJailDefaultPath = "nsjail"
	NsJailDefaultUID  = 99999
	NsJailDefaultGID  = 99999

	// SDU Sandbox 配置
	SDUSandboxDefaultPath = "sandbox"
	SDUSandboxSeccompRule = "general"

	NormalJudgePath = "./script/normal_judge.sh"

	BoxIDPoolSize = 500
)

// 文件名常量
const (
	// 代码文件名
	CCodeFileName    = "main.c"
	CppCodeFileName  = "main.cpp"
	JavaCodeFileName = "Main.java"
	PyCodeFileName   = "main.py"
	GoCodeFileName   = "main.go"

	// 可执行文件名
	DefaultExeName = "main"

	// 输入输出文件名
	InputFileName  = "input.txt"
	OutputFileName = "output.txt"
)

// 编译器相关常量
const (
	// 编译器路径
	GCCPath    = "gcc"
	GPPPath    = "g++"
	JavacPath  = "javac"
	PythonPath = "python3"
	GoPath     = "go"

	// 编译选项
	GCCDefaultFlags = "-O2 -Wall -std=c11"
	GPPDefaultFlags = "-O2 -Wall -std=c++17"
)

// 日志相关常量
const (
	// 日志级别
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"

	// 日志文件
	DefaultLogFile    = "log/server.log"
	DefaultLogMaxSize = 200 // MB
	DefaultLogMaxAge  = 30  // days
	DefaultLogBackups = 7
)

// HTTP 相关常量
const (
	// 默认端口
	DefaultServerPort = 53333

	// 超时配置
	DefaultReadTimeout  = 30 * time.Second
	DefaultWriteTimeout = 30 * time.Second
	DefaultIdleTimeout  = 60 * time.Second
)

// 监控相关常量
const (
	// 指标收集间隔
	MetricsCollectInterval = 10 * time.Second

	// 健康检查
	HealthCheckInterval = 5 * time.Second
)
