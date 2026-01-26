package runner

type SandboxType int

const (
	NsJail SandboxType = iota
)

// Runner 沙箱运行器接口
type Runner interface {
	RunInSandbox(exePath, input string, timeLimit, memoryLimit int) (string, string, string, error)
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
