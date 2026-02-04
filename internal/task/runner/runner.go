package runner

import (
	"hitwh-judge/internal/model"
	"strings"
)

type SandboxType int

const (
	NsJail SandboxType = iota
	SDUSandbox
)

// Runner 沙箱运行器接口
type Runner interface {
	RunInSandbox(runParams model.RunParams) *model.TestCaseResult
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
	case SDUSandbox:
		return &SDUSandboxRunner{
			SandboxPath: sandboxPath,
		}
	default:
		return nil
	}
}

func GetDefaultSandboxConfig(sandboxType SandboxType) model.SandboxConfig {
	switch sandboxType {
	case NsJail:
		return DefaultNsJailSandboxConfig
	case SDUSandbox:
		return DefaultSDUSandboxConfig
	default:
		return model.SandboxConfig{}
	}
}

// normalizeString 清理字符串中的换行符
func normalizeString(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}
