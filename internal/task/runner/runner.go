package runner

import (
	"errors"
	"fmt"
	"hitwh-judge/internal/model"
	"os"
	"strings"

	"go.uber.org/zap"
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

func createTmpDir() (string, func(), error) {
	// 1. 创建临时目录（权限0700，仅当前用户可访问）
	tempDir, err := os.MkdirTemp("", "oj-judge-*")
	if err != nil {
		errMsg := fmt.Sprintf("创建临时目录失败: %v", err)
		zap.L().Error(errMsg)
		return "", nil, errors.New(errMsg)
	}
	if err := os.Chmod(tempDir, 0777); err != nil {
		errMsg := fmt.Sprintf("修改临时目录权限失败: %v, 目录路径: %s", err, tempDir)
		zap.L().Error(errMsg)
		// 权限修改失败时，清理已创建的临时目录，避免残留
		_ = os.RemoveAll(tempDir)
		return "", nil, errors.New(errMsg)
	}

	// 2. 定义清理函数（闭包，捕获tempDir变量）
	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			zap.L().Warn("清理临时目录失败", zap.String("dir", tempDir), zap.Error(err))
		} else {
			zap.L().Info("成功清理临时评测目录", zap.String("dir", tempDir))
		}
	}

	zap.L().Info("创建临时评测目录", zap.String("dir", tempDir))
	return tempDir, cleanup, nil
}
