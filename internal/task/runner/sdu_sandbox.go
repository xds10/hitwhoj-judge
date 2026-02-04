package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hitwh-judge/internal/model"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SDUSandboxRunner 自定义沙箱运行器
type SDUSandboxRunner struct {
	SandboxPath string
}

var DefaultSDUSandboxConfig = model.SandboxConfig{
	Type: "sdu_sandbox",
	Path: "sandbox",
	CompilerMap: map[model.LanguageType]string{
		model.LanguageC:    "gcc",
		model.LanguageCPP:  "g++",
		model.LanguageJava: "javac",
		model.LanguagePy:   "python3",
	},
}

// SandboxResult 沙箱运行结果
type SandboxResult struct {
	CpuTime  int   `json:"cpu_time"`
	RealTime int   `json:"real_time"`
	Memory   int64 `json:"memory"`
	Signal   int   `json:"signal"`
	ExitCode int   `json:"exit_code"`
	Error    int   `json:"error"`
	Result   int   `json:"result"`
}

// 运行结果映射
var resultMapping = map[int]string{
	0: "AC",  // Success
	1: "TLE", // Time Limit Exceeded
	2: "TLE", // Time Limit Exceeded
	3: "MLE", // Memory Limit Exceeded
	4: "RE",  // Runtime Error
	5: "SE",  // System Error
	6: "OLE", // Output Limit Exceeded
}

// RunInSandbox 在自定义沙箱中运行程序
func (csr *SDUSandboxRunner) RunInSandbox(runParams model.RunParams) *model.TestCaseResult {
	// 提取参数
	exePath := runParams.ExePath
	input := runParams.Input
	timeLimit := runParams.TimeLimit
	memoryLimit := runParams.MemLimit

	// 检查sandbox是否存在
	if _, err := exec.LookPath(csr.SandboxPath); err != nil {
		// 尝试直接执行文件
		if _, err := os.Stat(csr.SandboxPath); os.IsNotExist(err) {
			return &model.TestCaseResult{
				TestCaseIndex: runParams.TestCaseIndex,
				Status:        model.StatusSE,
				Error:         fmt.Sprintf("sandbox 不存在: %s", csr.SandboxPath),
			}
		}
	}

	// 创建临时文件用于输入和输出
	// tempDir, err := ioutil.TempDir("", "sandbox_*")
	// if err != nil {
	// 	return "", "", "", fmt.Errorf("创建临时目录失败: %w", err)
	// }
	// defer os.RemoveAll(tempDir)
	tempDir := filepath.Join(os.Getenv("HOME"), "tmp/sandbox_*")
	// 创建临时目录
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("创建临时目录失败: %v", err),
		}
	}
	defer os.RemoveAll(tempDir)
	zap.L().Info("Sandbox temp dir", zap.String("tempDir", tempDir))

	inputPath := filepath.Join(tempDir, "input.txt")
	outputPath := filepath.Join(tempDir, "output.txt")

	// 写入输入数据
	if input != "" {
		if err := ioutil.WriteFile(inputPath, []byte(normalizeString(input)), 0644); err != nil {
			return &model.TestCaseResult{
				TestCaseIndex: runParams.TestCaseIndex,
				Status:        model.StatusSE,
				Error:         fmt.Sprintf("写入输入文件失败: %v", err),
			}
		}
	} else {
		// 创建空输入文件
		if err := ioutil.WriteFile(inputPath, []byte(""), 0644); err != nil {
			return &model.TestCaseResult{
				TestCaseIndex: runParams.TestCaseIndex,
				Status:        model.StatusSE,
				Error:         fmt.Sprintf("创建输入文件失败: %v", err),
			}
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

	// 构建沙箱命令参数
	cmd := exec.Command(
		"sudo",
		csr.SandboxPath,
		"--exe_path="+absExePath,
		"--input_path="+inputPath,
		"--output_path="+outputPath,
		"--seccomp_rules=general",
		fmt.Sprintf("--max_memory=%d", memoryLimit*1024*1024),  // 转换为字节
		fmt.Sprintf("--max_real_time=%d", int(timeLimit)*1200), // 转换为毫秒
	)

	// 捕获标准输出和错误输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("沙箱运行失败: %v, 错误输出: %s", err, stderr.String()),
		}
	}
	outputBytes := normalizeString(stdout.String())

	// 解析沙箱返回的JSON结果
	var result SandboxResult
	jsonStr := string(outputBytes)
	zap.L().Info("Sandbox command output", zap.String("output", jsonStr))

	// 从输出中提取JSON部分
	lines := strings.Split(jsonStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			jsonStr = line
			break
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("解析沙箱结果失败: %v, 输出: %s", err, jsonStr),
		}
	}
	zap.L().Info("Sandbox result", zap.Any("result", result))

	// 读取输出文件内容
	var output string
	if _, err := os.Stat(outputPath); err == nil {
		outputBytes, err := ioutil.ReadFile(outputPath)
		if err != nil {
			return &model.TestCaseResult{
				TestCaseIndex: runParams.TestCaseIndex,
				Status:        model.StatusSE,
				Error:         fmt.Sprintf("读取输出文件失败: %v", err),
			}
		}
		output = normalizeString(string(outputBytes))
	}

	// 获取错误输出
	errOutput := stderr.String()

	// 根据沙箱返回的结果码确定状态
	status, exists := resultMapping[result.Result]
	if !exists {
		status = fmt.Sprintf("RE (%d)", result.Result)
	}

	// 记录运行信息
	zap.L().Info("Sandbox execution result",
		zap.Int("cpu_time", result.CpuTime),
		zap.Int("real_time", result.RealTime),
		zap.Int64("memory", result.Memory),
		zap.Int("signal", result.Signal),
		zap.Int("exit_code", result.ExitCode),
		zap.Int("error", result.Error),
		zap.Int("result", result.Result),
		zap.String("status", status),
	)
	zap.L().Info("Sandbox execution result",
		zap.String("output", output),
		zap.String("err_output", errOutput),
		zap.String("status", status),
	)
	if result.Result == 0 {
		if result.Memory > memoryLimit*1024*1024 {
			status = model.StatusMLE
		}
		if result.CpuTime > int(timeLimit)*1000 {
			status = model.StatusTLE
		}
	}
	testCaseResult := &model.TestCaseResult{
		TestCaseIndex: runParams.TestCaseIndex,
		Status:        model.JudgeStatus(status),
		TimeUsed:      time.Duration(result.CpuTime) * time.Millisecond,
		MemUsed:       uint64(result.Memory),
		Output:        output,
		Error:         errOutput,
	}
	return testCaseResult
}

// RunInSandboxAsync 异步运行沙箱程序，返回进程PID和控制通道
func (csr *SDUSandboxRunner) RunInSandboxAsync(exePath, input string) (int, <-chan RunResult, error) {
	// 检查sandbox是否存在
	if _, err := exec.LookPath(csr.SandboxPath); err != nil {
		// 尝试直接执行文件
		if _, err := os.Stat(csr.SandboxPath); os.IsNotExist(err) {
			return 0, nil, fmt.Errorf("sandbox 不存在: %s", csr.SandboxPath)
		}
	}

	// 创建临时文件用于输入和输出
	tempDir, err := ioutil.TempDir("", "sandbox_async_*")
	if err != nil {
		return 0, nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	inputPath := filepath.Join(tempDir, "input.txt")
	outputPath := filepath.Join(tempDir, "output.txt")

	// 写入输入数据
	if input != "" {
		if err := ioutil.WriteFile(inputPath, []byte(normalizeString(input)), 0644); err != nil {
			os.RemoveAll(tempDir)
			return 0, nil, fmt.Errorf("写入输入文件失败: %w", err)
		}
	} else {
		// 创建空输入文件
		if err := ioutil.WriteFile(inputPath, []byte(""), 0644); err != nil {
			os.RemoveAll(tempDir)
			return 0, nil, fmt.Errorf("创建输入文件失败: %w", err)
		}
	}

	// 获取可执行文件的绝对路径
	absExePath, err := filepath.Abs(exePath)
	if err != nil {
		os.RemoveAll(tempDir)
		return 0, nil, fmt.Errorf("获取可执行文件绝对路径失败: %w", err)
	}

	// 构建沙箱命令参数
	cmd := exec.Command(
		csr.SandboxPath,
		"--exe_path="+absExePath,
		"--input_path="+inputPath,
		"--output_path="+outputPath,
		"--seccomp_rules=general",
		"--max_memory=335544320", // 默认内存限制
	)

	// 捕获错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 启动命令（非阻塞）
	if err := cmd.Start(); err != nil {
		os.RemoveAll(tempDir)
		return 0, nil, fmt.Errorf("启动沙箱进程失败: %w", err)
	}

	// 创建结果通道
	resultChan := make(chan RunResult, 1)

	// 启动goroutine等待命令执行完成
	go func() {
		defer close(resultChan)
		defer os.RemoveAll(tempDir)

		// 等待命令执行完成
		err := cmd.Wait()

		var output string
		if _, statErr := os.Stat(outputPath); statErr == nil {
			outputBytes, readErr := ioutil.ReadFile(outputPath)
			if readErr == nil {
				output = normalizeString(string(outputBytes))
			}
		}

		errOutput := stderr.String()

		// 如果有错误，尝试解析可能的结果
		status := "AC"
		if err != nil {
			status = "RE"
		}

		// 发送结果
		resultChan <- RunResult{
			Output:    output,
			ErrOutput: errOutput,
			Status:    status,
			Err:       err,
		}
	}()

	// 返回进程PID和结果通道
	return cmd.Process.Pid, resultChan, nil
}
