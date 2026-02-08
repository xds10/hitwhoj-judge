package runner

import (
	"bytes"
	"fmt"
	"hitwh-judge/internal/model"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// NsJailRunner NsJail沙箱运行器
type NsJailRunner struct {
	NsJailPath string
}

// DefaultNsJailSandboxConfig 默认沙箱配置
var DefaultNsJailSandboxConfig = model.SandboxConfig{
	Type: "nsjail",
	Path: "nsjail",
	CompilerMap: map[model.LanguageType]string{
		model.LanguageC:    "gcc",
		model.LanguageCPP:  "g++",
		model.LanguageJava: "javac",
		model.LanguagePy:   "python3",
	},
}

// ResourceUsage 资源使用统计
type ResourceUsage struct {
	CpuTime  time.Duration // CPU时间（用户态+内核态）
	RealTime time.Duration // 墙钟时间
	Memory   int64         // 内存使用（字节）
}

// RunInSandbox 在NsJail沙箱中运行程序，返回详细的资源使用信息
func (nr *NsJailRunner) RunInSandbox(runParams model.RunParams) *model.TestCaseResult {
	exePath := runParams.ExePath
	// input := runParams.Input
	timeLimit := runParams.TimeLimit
	memoryLimit := runParams.MemLimit

	// 检查nsjail是否存在
	if _, err := exec.LookPath(nr.NsJailPath); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("nsjail 不存在: %v", err),
		}
	}

	// 获取可执行文件的绝对路径
	absExePath, err := filepath.Abs(exePath)
	if err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("获取可执行文件绝对路径失败: %v", err),
		}
	}
	fmt.Println("absExePath", absExePath)
	exeDir := filepath.Dir(absExePath)

	scriptSrcPath := "./scripts/runner/normal_judge.sh"
	scriptDstPath := filepath.Join(exeDir, "normal_judge.sh")

	scriptContent, err := ioutil.ReadFile(scriptSrcPath)
	if err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("无法读取normal_judge.sh脚本: %v", err),
		}
	}

	if err := ioutil.WriteFile(scriptDstPath, scriptContent, 0755); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("无法写入normal_judge.sh脚本到执行目录: %v", err),
		}
	}
	// 构建NsJail命令
	// 添加资源限制参数
	cmd := exec.Command(
		"sudo",
		nr.NsJailPath,
		"-Mo",                  // 一次性模式
		"-N",                   // 禁用网络
		"--rlimit_nproc", "32", // 进程数限制
		"--rlimit_as", fmt.Sprintf("%d", memoryLimit*1024*1024), // 内存限制（字节）
		"--rlimit_cpu", fmt.Sprintf("%d", timeLimit+1), // CPU时间限制（秒）
		"--time_limit", fmt.Sprintf("%d", timeLimit*2), // 墙钟时间限制（秒）
		"--chroot", exeDir, // chroot到可执行文件目录
		"--user", "99999", // 使用非特权用户
		"--group", "99999", // 使用非特权组
		"--disable_clone_newuser", // 禁用user namespace
		"--bindmount_ro", "/bin",  // 挂载/bin目录
		"--bindmount_ro", "/lib", // 挂载/lib目录
		"--bindmount_ro", "/lib64", // 挂载/lib64目录
		"--",
		"/bin/bash",
		"normal_judge.sh",
		filepath.Base(absExePath),
		filepath.Base(runParams.InputFile),
	)

	inputFileReader, err := ioutil.ReadFile(runParams.InputFile)
	if err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("读取临时输入文件失败: %v", err),
		}
	}

	var stdin bytes.Buffer
	stdin.Write(inputFileReader)
	cmd.Stdin = &stdin

	// 设置输入
	// var stdin bytes.Buffer
	// if input != "" {
	// 	stdin.WriteString(normalizeString(input))
	// }
	// cmd.Stdin = &stdin

	// 捕获输出和错误
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 记录开始时间（墙钟时间）
	startTime := time.Now()

	// 执行命令
	err = cmd.Run()

	// 计算墙钟时间
	realTime := time.Since(startTime)

	// 获取资源使用情况
	var rusage syscall.Rusage
	var cpuTime time.Duration
	var memUsed int64

	if cmd.ProcessState != nil {
		// 获取进程的资源使用统计
		sysUsage := cmd.ProcessState.SysUsage()
		if sysUsage != nil {
			if usage, ok := sysUsage.(*syscall.Rusage); ok {
				rusage = *usage
				// CPU时间 = 用户态时间 + 内核态时间
				cpuTime = time.Duration(rusage.Utime.Sec)*time.Second + time.Duration(rusage.Utime.Usec)*time.Microsecond +
					time.Duration(rusage.Stime.Sec)*time.Second + time.Duration(rusage.Stime.Usec)*time.Microsecond
				// 最大常驻集大小（RSS），单位是KB，需要转换为字节
				memUsed = rusage.Maxrss * 1024
			}
		}
	}

	output := normalizeString(stdout.String())
	errOutput := stderr.String()

	// 解析错误类型和状态
	status := model.StatusAC
	var errorMsg string

	if err != nil {
		zap.L().Error("NsJail execution error", zap.Error(err), zap.String("stderr", errOutput))
		if exitErr, ok := err.(*exec.ExitError); ok {
			status, errorMsg = parseNsJailError(errOutput, exitErr, cpuTime, timeLimit, memUsed, memoryLimit)
		} else {
			status = model.StatusRE
			errorMsg = fmt.Sprintf("沙箱执行异常: %v", err)
		}
	}

	// 二次检查：即使没有错误，也要检查是否超限
	if status == model.StatusAC {
		// 检查CPU时间是否超限
		if cpuTime > time.Duration(timeLimit)*time.Second {
			status = model.StatusTLE
			errorMsg = fmt.Sprintf("CPU时间超限: %v > %vs", cpuTime, timeLimit)
		}
		// 检查内存是否超限
		if memUsed > memoryLimit*1024*1024 {
			status = model.StatusMLE
			errorMsg = fmt.Sprintf("内存超限: %d bytes > %d MB", memUsed, memoryLimit)
		}
	}

	// 记录详细的运行信息
	zap.L().Info("NsJail execution result",
		zap.Int("test_case", runParams.TestCaseIndex),
		zap.Duration("cpu_time", cpuTime),
		zap.Duration("real_time", realTime),
		zap.Int64("memory_bytes", memUsed),
		zap.Float64("memory_mb", float64(memUsed)/(1024*1024)),
		zap.String("status", string(status)),
		zap.Int64("time_limit_sec", timeLimit),
		zap.Int64("mem_limit_mb", memoryLimit),
	)

	testCaseResult := &model.TestCaseResult{
		TestCaseIndex: runParams.TestCaseIndex,
		Status:        status,
		TimeUsed:      cpuTime,
		MemUsed:       uint64(memUsed),
		Output:        output,
		Error:         errorMsg,
	}

	return testCaseResult
}

// parseNsJailError 解析NsJail运行错误
func parseNsJailError(stderr string, exitErr *exec.ExitError, cpuTime time.Duration, timeLimit int64, memUsed int64, memLimit int64) (model.JudgeStatus, string) {
	if exitErr != nil {
		waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok {
			if waitStatus.Signaled() {
				signal := waitStatus.Signal()
				switch signal {
				case syscall.SIGXCPU:
					return model.StatusTLE, "CPU时间超限信号 (SIGXCPU)"
				case syscall.SIGKILL:
					// SIGKILL可能是内存超限或时间超限
					if strings.Contains(stderr, "memory limit exceeded") || strings.Contains(stderr, "rlimit_as") {
						return model.StatusMLE, "内存超限 (SIGKILL)"
					}
					if strings.Contains(stderr, "time limit exceeded") || strings.Contains(stderr, "Timeout") {
						return model.StatusTLE, "时间超限 (SIGKILL)"
					}
					// 如果有资源使用数据，进一步判断
					if cpuTime > time.Duration(timeLimit)*time.Second {
						return model.StatusTLE, fmt.Sprintf("时间超限 (SIGKILL): %v > %vs", cpuTime, timeLimit)
					}
					if memUsed > memLimit*1024*1024 {
						return model.StatusMLE, fmt.Sprintf("内存超限 (SIGKILL): %d bytes > %d MB", memUsed, memLimit)
					}
					return model.StatusRE, "进程被终止 (SIGKILL)"
				case syscall.SIGSEGV:
					return model.StatusRE, "段错误 (SIGSEGV)"
				case syscall.SIGABRT:
					return model.StatusRE, "程序异常终止 (SIGABRT)"
				case syscall.SIGFPE:
					return model.StatusRE, "浮点异常 (SIGFPE)"
				default:
					return model.StatusRE, fmt.Sprintf("运行时错误 (signal: %v)", signal)
				}
			}
			if waitStatus.ExitStatus() != 0 {
				return model.StatusRE, fmt.Sprintf("非零退出码: %d", waitStatus.ExitStatus())
			}
		}
	}

	// 检查stderr中的错误信息
	if strings.Contains(stderr, "time limit exceeded") || strings.Contains(stderr, "Timeout") {
		return model.StatusTLE, "时间超限"
	}
	if strings.Contains(stderr, "memory limit exceeded") || strings.Contains(stderr, "rlimit_as exceeded") {
		return model.StatusMLE, "内存超限"
	}

	return model.StatusRE, fmt.Sprintf("运行时错误: %s", stderr)
}
