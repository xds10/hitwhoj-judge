package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// NsJailRunner NsJail沙箱运行器
type NsJailRunner struct {
	NsJailPath string
}

// RunInSandbox 在NsJail沙箱中运行程序
func (nr *NsJailRunner) RunInSandbox(exePath, input string, timeLimit, memoryLimit int) (string, string, string, error) {
	// 检查nsjail是否存在
	if _, err := exec.LookPath(nr.NsJailPath); err != nil {
		return "", "", "", err
	}

	// 获取可执行文件的绝对路径
	absExePath, err := filepath.Abs(exePath)
	if err != nil {
		return "", "", "", fmt.Errorf("获取可执行文件绝对路径失败: %w", err)
	}
	exeDir := filepath.Dir(absExePath)

	// 构建NsJail命令
	cmd := exec.Command(
		nr.NsJailPath,
		"-Mo", "-N",
		"--time_limit", fmt.Sprintf("%d", timeLimit),
		"--rlimit_as", fmt.Sprintf("%d", memoryLimit),
		"--rlimit_nproc", "1",
		"--chroot", exeDir,
		"--hostname", "oj-sandbox",
		"--user", "99999",
		"--group", "99999",
		"--disable_clone_newuser",
		"--",
		filepath.Base(absExePath),
	)

	// 设置输入
	var stdin bytes.Buffer
	if input != "" {
		stdin.WriteString(normalizeString(input))
	}
	cmd.Stdin = &stdin

	// 捕获输出和错误
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err = cmd.Run()
	output := normalizeString(stdout.String())
	errOutput := stderr.String()

	// 解析错误类型
	status := "AC"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			status = parseNsJailError(errOutput, exitErr)
			return output, errOutput, status, fmt.Errorf("沙箱运行失败: %w", err)
		}
		return output, errOutput, "RE", fmt.Errorf("沙箱执行异常: %w", err)
	}

	return output, errOutput, status, nil
}

// parseNsJailError 解析NsJail运行错误
func parseNsJailError(stderr string, exitErr *exec.ExitError) string {
	if exitErr != nil {
		waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok {
			if waitStatus.Signaled() {
				signal := waitStatus.Signal()
				switch signal {
				case syscall.SIGXCPU:
					return "TLE"
				case syscall.SIGKILL:
					if strings.Contains(stderr, "memory limit exceeded") || strings.Contains(stderr, "rlimit_as") {
						return "MLE"
					}
					return "RE"
				case syscall.SIGSEGV, syscall.SIGABRT:
					return "RE"
				default:
					return fmt.Sprintf("RE (signal: %v)", signal)
				}
			}
			if waitStatus.ExitStatus() != 0 {
				return fmt.Sprintf("RE (exit code: %d)", waitStatus.ExitStatus())
			}
		}
	}

	if strings.Contains(stderr, "time limit exceeded") || strings.Contains(stderr, "Timeout") {
		return "TLE"
	}
	if strings.Contains(stderr, "memory limit exceeded") || strings.Contains(stderr, "rlimit_as exceeded") {
		return "MLE"
	}

	return fmt.Sprintf("RE: %s", stderr)
}

// normalizeString 清理字符串中的换行符
func normalizeString(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}
