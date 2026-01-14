package task

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

// JudgeResult 评测结果结构体
type JudgeResult struct {
	Status string // AC/WA/CE/RE/TLE/MLE 等状态
	Output string // 程序输出
	Error  string // 编译/运行错误信息
}

// 清理字符串中的换行符（统一为\n，去除首尾空白）
func normalizeString(s string) string {
	// 替换Windows换行符为Unix换行符
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// 去除首尾空白（包括换行、空格、制表符）
	return strings.TrimSpace(s)
}

// checkCommandExist 检查命令是否存在
func checkCommandExist(cmdPath string) error {
	_, err := exec.LookPath(cmdPath)
	if err != nil {
		return fmt.Errorf("命令不存在: %s, 错误: %w", cmdPath, err)
	}
	return nil
}

// CompileCode 编译用户代码
// dir: 临时目录路径, codePath: 代码文件路径, exePath: 可执行文件路径
// config: 评测配置
func CompileCode(dir, codePath, exePath string, config JudgeConfig) (string, error) {
	// 检查gcc是否存在
	if err := checkCommandExist(config.GCCPath); err != nil {
		return "", err
	}

	// 编译命令：添加-Werror将警告转为错误，-O2优化，-static静态编译（可选，增强隔离性）
	cmd := exec.Command(
		config.GCCPath,
		"-o", exePath,
		codePath,
		"-Wall",    // 显示所有警告
		"-O2",      // 优化编译
		"-static",  // 静态编译（避免沙箱内缺少动态库）
		"-std=c11", // 指定C11标准
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Dir = dir // 设置编译工作目录

	// 执行编译
	if err := cmd.Run(); err != nil {
		compileErr := stderr.String()
		return compileErr, fmt.Errorf("编译失败: %w, 错误详情: %s", err, compileErr)
	}

	// 检查可执行文件是否生成
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return "", fmt.Errorf("编译后可执行文件未生成: %s", exePath)
	}

	return "", nil
}

// parseNsJailError 解析NsJail运行错误，区分TLE/MLE/RE
func parseNsJailError(stderr string, exitErr *exec.ExitError) string {
	// 检查退出状态码（NsJail的退出码规则）
	if exitErr != nil {
		waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
		if ok {
			// 信号终止（比如超时、内存超限、非法指令等）
			if waitStatus.Signaled() {
				signal := waitStatus.Signal()
				switch signal {
				case syscall.SIGXCPU: // CPU超时
					return "TLE"
				case syscall.SIGKILL: // 内存超限/沙箱强制终止
					if strings.Contains(stderr, "memory limit exceeded") || strings.Contains(stderr, "rlimit_as") {
						return "MLE"
					}
					return "RE"
				case syscall.SIGSEGV: // 段错误
					return "RE"
				case syscall.SIGABRT: // 异常终止
					return "RE"
				default:
					return fmt.Sprintf("RE (signal: %v)", signal)
				}
			}
			// 正常退出但非0码
			if waitStatus.ExitStatus() != 0 {
				return fmt.Sprintf("RE (exit code: %d)", waitStatus.ExitStatus())
			}
		}
	}

	// 从stderr中匹配NsJail的超时/内存超限信息
	if strings.Contains(stderr, "time limit exceeded") || strings.Contains(stderr, "Timeout") {
		return "TLE"
	}
	if strings.Contains(stderr, "memory limit exceeded") || strings.Contains(stderr, "rlimit_as exceeded") {
		return "MLE"
	}

	// 其他运行时错误
	return fmt.Sprintf("RE: %s", stderr)
}

// RunInSandbox 在NsJail沙箱中运行可执行文件
// exePath: 可执行文件路径, input: 输入数据, config: 评测配置
func RunInSandbox(exePath, input string, config JudgeConfig) (string, string, string, error) {
	// 检查nsjail是否存在
	if err := checkCommandExist(config.NsJailPath); err != nil {
		return "", "", "", err
	}

	// 获取可执行文件的绝对路径
	absExePath, err := filepath.Abs(exePath)
	if err != nil {
		return "", "", "", fmt.Errorf("获取可执行文件绝对路径失败: %w", err)
	}
	// 获取临时目录（可执行文件所在目录）
	exeDir := filepath.Dir(absExePath)

	// 构建NsJail命令（使用正确的参数）
	// 核心参数说明：
	// -Mo: ONCE模式，执行一次后退出
	// -N: 禁用网络命名空间（增强隔离）
	// --time_limit: 时间限制（秒）
	// --rlimit_as: 地址空间限制（MB）
	// --rlimit_nproc: 进程数限制（1）
	// --chroot: 沙箱根目录（使用临时目录，增强隔离）
	// --is_root_rw: 允许chroot根目录可写
	// --user/--group: 非特权用户/组
	cmd := exec.Command(
		config.NsJailPath,
		"-Mo", // ONCE模式（单次执行）
		"-N",  // 禁用网络命名空间
		"--time_limit", fmt.Sprintf("%d", config.TimeLimit),
		"--rlimit_as", fmt.Sprintf("%d", config.MemoryLimit), // 内存限制（MB）
		"--rlimit_nproc", "1", // 进程数限制
		"--chroot", exeDir, // 沙箱根目录（临时目录）
		"--hostname", "oj-sandbox", // 沙箱主机名
		"--user", "99999", // 非特权用户ID
		"--group", "99999", // 非特权组ID
		"--disable_clone_newuser", // 禁用新用户命名空间（根据环境调整）
		"--",
		filepath.Base(absExePath), // 沙箱内的可执行文件路径（相对chroot）
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

// CompareOutput 对比程序输出和标准输出（归一化后）
func CompareOutput(programOutput, expectedOutput string) bool {
	return normalizeString(programOutput) == normalizeString(expectedOutput)
}

// Judge 核心评测函数
// code: 用户代码, input: 输入数据, expectedOutput: 标准输出
// config: 评测配置（可选，不传则用默认配置）
func Judge(code, input, expectedOutput string, config ...JudgeConfig) (*JudgeResult, error) {
	// 使用默认配置（如果未传入）
	judgeConfig := DefaultJudgeConfig
	if len(config) > 0 {
		judgeConfig = config[0]
	}
	fmt.Printf("评测配置: %+v\n", judgeConfig)

	// 1. 创建临时目录（权限0700，仅当前用户可访问）
	tempDir, err := os.MkdirTemp("", "oj-judge-*")
	if err != nil {
		errMsg := fmt.Sprintf("创建临时目录失败: %v", err)
		zap.L().Error(errMsg)
		return nil, errors.New(errMsg)
	}
	// 延迟清理临时目录（即使发生错误也清理）
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			zap.L().Warn("清理临时目录失败", zap.String("dir", tempDir), zap.Error(err))
		}
	}()
	zap.L().Info("创建临时评测目录", zap.String("dir", tempDir))

	// 2. 写入用户代码到临时文件（C语言main.c）
	codePath := filepath.Join(tempDir, "main.c")
	if err := os.WriteFile(codePath, []byte(code), 0600); err != nil { // 权限0600，仅当前用户可读写
		errMsg := fmt.Sprintf("写入代码文件失败: %v", err)
		zap.L().Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// 3. 编译代码
	exePath := filepath.Join(tempDir, "main")
	compileErr, err := CompileCode(tempDir, codePath, exePath, judgeConfig)
	if err != nil {
		zap.L().Error("编译代码失败", zap.Error(err), zap.String("compile_err", compileErr))
		return &JudgeResult{
			Status: "CE",
			Output: "",
			Error:  compileErr,
		}, nil
	}

	// 4. 在沙箱中运行程序
	programOutput, runErrOutput, runStatus, err := RunInSandbox(exePath, input, judgeConfig)
	if err != nil {
		zap.L().Error("沙箱运行程序失败", zap.Error(err), zap.String("run_err", runErrOutput))
		return &JudgeResult{
			Status: runStatus,
			Output: programOutput,
			Error:  runErrOutput,
		}, nil
	}

	// 5. 对比输出（仅当运行状态为AC时）
	if runStatus == "AC" {
		if CompareOutput(programOutput, expectedOutput) {
			zap.L().Info("评测通过（AC）", zap.String("output", programOutput))
			return &JudgeResult{
				Status: "AC",
				Output: programOutput,
				Error:  "",
			}, nil
		} else {
			errMsg := fmt.Sprintf("预期输出: %s, 实际输出: %s", normalizeString(expectedOutput), programOutput)
			zap.L().Info("答案错误（WA）", zap.String("error", errMsg))
			return &JudgeResult{
				Status: "WA",
				Output: programOutput,
				Error:  errMsg,
			}, nil
		}
	}

	// 非AC的运行状态（TLE/MLE/RE）
	zap.L().Info("程序运行异常", zap.String("status", runStatus), zap.String("error", runErrOutput))
	return &JudgeResult{
		Status: runStatus,
		Output: programOutput,
		Error:  runErrOutput,
	}, nil
}

// 辅助函数：获取当前系统的换行符（用于调试）
func getSystemLineSeparator() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
