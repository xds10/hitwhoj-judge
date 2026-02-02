package runner

import "hitwh-judge/internal/model"

type SandboxType int

const (
	NsJail SandboxType = iota
)

// Runner 沙箱运行器接口
type Runner interface {
	RunInSandbox(runParams model.RunParams) (string, string, string, error)
	// RunInSandboxAsync 异步运行沙箱程序，返回进程PID和控制通道
	RunInSandboxAsync(exePath, input string) (int, <-chan RunResult, error)
}

// RunResult 异步运行结果
type RunResult struct {
	Output    string
	ErrOutput string
	Status    string
	Err       error
}

func NewRunner(sandboxType SandboxType, sandboxPath string) Runner {
	switch sandboxType {
	case NsJail:
		return &NsJailRunner{
			NsJailPath: sandboxPath,
		}
	default:
		return nil
	}
}
