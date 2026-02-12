package runner

import (
	"hitwh-judge/internal/model"
)

type SandboxType int

const (
	NsJail SandboxType = iota
	SDUSandbox
	Isolate
)

// Runner 沙箱运行器接口
type Runner interface {
	InitSandbox() (string, error)
	RunInSandbox(runParams model.RunParams) *model.TestCaseResult
}

// RunResult 异步运行结果
type RunResult struct {
	Output    string
	ErrOutput string
	Status    string
	Err       error
}

// NewRunner 创建沙箱运行器实例
func NewRunner(sandboxType SandboxType, sandboxPath string) Runner {
	switch sandboxType {
	case Isolate:
		return &IsoRunner{
			IsolatePath: sandboxPath,
			boxId:       0,
		}
	default:
		return nil
	}
}

// GetDefaultSandboxConfig 获取默认沙箱配置
func GetDefaultSandboxConfig(sandboxType SandboxType) model.SandboxConfig {
	switch sandboxType {
	case NsJail:
		return DefaultNsJailSandboxConfig
	case SDUSandbox:
		return DefaultSDUSandboxConfig
	case Isolate:
		return DefaultIsolateSandboxConfig
	default:
		return model.SandboxConfig{}
	}
}
