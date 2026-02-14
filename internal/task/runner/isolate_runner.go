package runner

import (
	"bytes"
	"fmt"
	"hitwh-judge/internal/model"
	file_util "hitwh-judge/internal/util/file"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// IsoRunner Isolate沙箱运行器
type IsoRunner struct {
	IsolatePath string
	boxId       int
	SandboxPath string
}

// DefaultIsolateSandboxConfig 默认isolate沙箱配置
var DefaultIsolateSandboxConfig = model.SandboxConfig{
	Type: "isolate",
	Path: "isolate",
}

func (ir *IsoRunner) InitSandbox() (string, error) {
	currentBoxId := allocateBoxID()
	ir.SetBoxId(currentBoxId)
	// 初始化沙箱
	initCmd := exec.Command(ir.IsolatePath, "--init", "--cg", fmt.Sprintf("--box-id=%d", ir.boxId))
	initOutput, err := initCmd.Output()
	if err != nil {
		return "", fmt.Errorf("初始化沙箱失败: %w", err)
	}
	sandboxPath := strings.TrimSpace(string(initOutput))
	sandboxPath = filepath.Join(sandboxPath, "box")
	return sandboxPath, nil
}

// RunInSandbox 在Isolate沙箱中运行程序
func (ir *IsoRunner) RunInSandbox(runParams model.RunParams) *model.TestCaseResult {
	exePath := runParams.ExePath
	timeLimit := runParams.TimeLimit
	memoryLimit := runParams.MemLimit

	// // 检查isolate是否存在
	// if _, err := exec.LookPath(ir.IsolatePath); err != nil {
	// 	return &model.TestCaseResult{
	// 		TestCaseIndex: runParams.TestCaseIndex,
	// 		Status:        model.StatusSE,
	// 		Error:         fmt.Sprintf("isolate 不存在: %v", err),
	// 	}
	// }

	currentBoxId := allocateBoxID()
	ir.SetBoxId(currentBoxId)

	// 初始化沙箱
	initCmd := exec.Command(ir.IsolatePath, "--init", "--cg", fmt.Sprintf("--box-id=%d", ir.boxId))
	initOutput, err := initCmd.Output()
	if err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("初始化沙箱失败: %v", err),
		}
	}
	sandboxPath := strings.TrimSpace(string(initOutput))
	sandboxPath = filepath.Join(sandboxPath, "box")

	// 确保清理函数
	defer func() {
		cleanupCmd := exec.Command(ir.IsolatePath, "--cleanup", "--cg", fmt.Sprintf("--box-id=%d", ir.boxId))
		cleanupCmd.Run()
		releaseBoxID(ir.boxId)
	}()

	// 获取脚本路径
	scriptSrcPath := "./scripts/runner/normal_judge.sh"
	scriptDstPath := filepath.Join(sandboxPath, "normal_judge.sh")

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
			Error:         fmt.Sprintf("无法写入normal_judge.sh脚本到沙箱目录: %v", err),
		}
	}

	// 复制可执行文件到沙箱目录
	exeFilename := filepath.Base(exePath)
	destExePath := filepath.Join(sandboxPath, exeFilename)
	cpCmd := exec.Command("cp", exePath, destExePath)
	if err := cpCmd.Run(); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("复制可执行文件到沙箱失败: %v", err),
		}
	}

	// 复制输入文件到沙箱目录
	inputFilename := filepath.Base(runParams.InputFile)
	destInputPath := filepath.Join(sandboxPath, inputFilename)
	cpInputCmd := exec.Command("cp", runParams.InputFile, destInputPath)
	if err := cpInputCmd.Run(); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("复制输入文件到沙箱失败: %v", err),
		}
	}

	// 准备运行命令
	args := []string{
		"--run",
		"--cg",
		fmt.Sprintf("--box-id=%d", ir.boxId),
		"--processes", // 允许多个进程
		"-e",          // 设置环境变量
		fmt.Sprintf("--time=%f", float64(timeLimit)),        // 时间限制（秒）
		fmt.Sprintf("--wall-time=%f", float64(timeLimit*2)), // 墙钟时间限制
		fmt.Sprintf("--mem=%d", memoryLimit*1024),           // 内存限制（KB）
		"--meta=meta.txt", // 输出元数据
		"--",
		"/bin/bash",
		"normal_judge.sh",
		exeFilename,
		inputFilename,
	}

	// 创建执行命令
	cmd := exec.Command(ir.IsolatePath, args...)

	// 设置沙箱目录为工作目录
	cmd.Dir = sandboxPath

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 记录开始时间
	startTime := time.Now()

	// 执行命令
	err = cmd.Run()

	// 计算墙钟时间
	realTime := time.Since(startTime)

	// 读取元数据文件
	metaPath := filepath.Join(sandboxPath, "meta.txt")
	metaContent, _ := file_util.ReadFileToString(metaPath)

	// 解析资源使用情况
	var cpuTime time.Duration
	var memUsed int64
	var exitCode int
	var isKilled bool
	var exitSig int

	lines := strings.Split(metaContent, "\n")
	zap.L().Info("Isolate meta content", zap.String("meta", metaContent))
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) >= 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "time":
				if t, err := strconv.ParseFloat(value, 64); err == nil {
					cpuTime = time.Duration(t * float64(time.Second))
				}
			case "cg-mem":
				if m, err := strconv.ParseInt(value, 10, 64); err == nil {
					memUsed = m * 1024 // convert KB to bytes
				}
			case "exitcode":
				if code, err := strconv.Atoi(value); err == nil {
					exitCode = code
				}
			case "exitsig":
				if sig, err := strconv.Atoi(value); err == nil {
					exitSig = sig
				}
			case "killed":
				isKilled = true
			case "cg-oom-killed":
				isKilled = true
				memUsed = memoryLimit * 1024 * 1024 * 2
			}
		}
	}

	output := normalizeString(stdout.String())
	errOutput := stderr.String()

	// 解析状态
	status := model.StatusAC
	var errorMsg string

	if err != nil {
		zap.L().Error("Isolate execution error", zap.Error(err), zap.String("stderr", errOutput))
	}

	// 检查元数据中的状态标志
	if strings.Contains(metaContent, "status:TO") || isKilled {
		status = model.StatusTLE
		errorMsg = "时间超限或被终止"
	} else if strings.Contains(metaContent, "status:SG") || exitSig > 0 {
		status = model.StatusRE
		if exitSig > 0 {
			errorMsg = fmt.Sprintf("程序收到信号 %d 终止", exitSig)
		} else {
			errorMsg = "程序被信号终止"
		}
	} else if strings.Contains(metaContent, "status:XX") {
		status = model.StatusSE
		errorMsg = "沙箱内部错误"
	} else if strings.Contains(metaContent, "status:RE") || exitCode != 0 {
		status = model.StatusRE
		errorMsg = fmt.Sprintf("运行时错误: 退出码 %d", exitCode)
	}

	// 二次检查资源限制
	if status == model.StatusAC {
		if cpuTime > time.Duration(timeLimit)*time.Second {
			status = model.StatusTLE
			errorMsg = fmt.Sprintf("CPU时间超限: %v > %vs", cpuTime, timeLimit)
		}
		if memUsed > memoryLimit*1024*1024 {
			status = model.StatusMLE
			errorMsg = fmt.Sprintf("内存超限: %d bytes > %d MB", memUsed, memoryLimit)
		}
	}

	// 记录运行结果
	zap.L().Info("Isolate execution result",
		zap.Int("box_id", ir.boxId),
		zap.Int("test_case", runParams.TestCaseIndex),
		zap.Duration("cpu_time", cpuTime),
		zap.Duration("real_time", realTime),
		zap.Int64("memory_bytes", memUsed),
		zap.Float64("memory_mb", float64(memUsed)/(1024*1024)),
		zap.String("status", string(status)),
		zap.Int64("time_limit_sec", timeLimit),
		zap.Int64("mem_limit_mb", memoryLimit),
		zap.String("meta_content", metaContent),
	)

	result := &model.TestCaseResult{
		TestCaseIndex: runParams.TestCaseIndex,
		Status:        status,
		TimeUsed:      cpuTime,
		MemUsed:       uint64(memUsed),
		Output:        output,
		Error:         errorMsg,
	}

	return result
}

// RunInteractiveInSandbox 在Isolate沙箱中运行交互题
func (ir *IsoRunner) RunInteractiveInSandbox(runParams model.RunParams) *model.TestCaseResult {
	exePath := runParams.ExePath
	timeLimit := runParams.TimeLimit
	memoryLimit := runParams.MemLimit

	specialExePath := runParams.SpecialExePath // 交互题评测程序路径
	answer := runParams.Answer                 // 标准答案

	currentBoxId := allocateBoxID()
	ir.SetBoxId(currentBoxId)

	// 初始化沙箱
	initCmd := exec.Command(ir.IsolatePath, "--init", "--cg", fmt.Sprintf("--box-id=%d", ir.boxId))
	initOutput, err := initCmd.Output()
	if err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("初始化沙箱失败: %v", err),
		}
	}

	sandboxPath := strings.TrimSpace(string(initOutput))
	sandboxPath = filepath.Join(sandboxPath, "box")

	// 确保清理函数
	// defer func() {
	// 	cleanupCmd := exec.Command(ir.IsolatePath, "--cleanup", "--cg", fmt.Sprintf("--box-id=%d", ir.boxId))
	// 	cleanupCmd.Run()
	// 	releaseBoxID(ir.boxId)
	// }()

	// 获取交互题评测脚本路径
	scriptSrcPath := "./scripts/runner/interactive_judge.sh"
	scriptDstPath := filepath.Join(sandboxPath, "interactive_judge.sh")

	scriptContent, err := ioutil.ReadFile(scriptSrcPath)
	if err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("无法读取interactive_run.py脚本: %v", err),
		}
	}

	if err := ioutil.WriteFile(scriptDstPath, scriptContent, 0755); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("无法写入interactive_run.py脚本到沙箱目录: %v", err),
		}
	}

	// 复制选手可执行文件到沙箱目录
	exeFilename := filepath.Base(exePath)
	destExePath := filepath.Join(sandboxPath, exeFilename)
	cpCmd := exec.Command("cp", exePath, destExePath)
	if err := cpCmd.Run(); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("复制选手可执行文件到沙箱失败: %v", err),
		}
	}
	var destSpecialExePath, specialExeFilename string
	// 复制交互评测程序到沙箱目录
	if specialExePath != "" {
		specialExeFilename = filepath.Base(specialExePath)
		destSpecialExePath = filepath.Join(sandboxPath, specialExeFilename)
		cpSpecialCmd := exec.Command("cp", specialExePath, destSpecialExePath)
		if err := cpSpecialCmd.Run(); err != nil {
			return &model.TestCaseResult{
				TestCaseIndex: runParams.TestCaseIndex,
				Status:        model.StatusSE,
				Error:         fmt.Sprintf("复制交互评测程序到沙箱失败: %v", err),
			}
		}
	}

	// 创建输入、输出和答案文件
	outputPath := filepath.Join(sandboxPath, "output.txt")
	answerPath := filepath.Join(sandboxPath, "answer.txt")

	// 复制输入文件到沙箱目录
	inputFilename := filepath.Base(runParams.InputFile)
	destInputPath := filepath.Join(sandboxPath, inputFilename)
	cpInputCmd := exec.Command("cp", runParams.InputFile, destInputPath)
	if err := cpInputCmd.Run(); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("复制输入文件到沙箱失败: %v", err),
		}
	}
	// 创建空输出文件
	if err := ioutil.WriteFile(outputPath, []byte{}, 0666); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("创建输出文件失败: %v", err),
		}
	}

	// 创建答案文件
	if err := ioutil.WriteFile(answerPath, []byte(answer), 0666); err != nil {
		return &model.TestCaseResult{
			TestCaseIndex: runParams.TestCaseIndex,
			Status:        model.StatusSE,
			Error:         fmt.Sprintf("创建答案文件失败: %v", err),
		}
	}

	// 准备运行命令，使用Python脚本运行交互题
	// 根据命令: python3 run.py ./b.out hitwhoj-rebirth_cd286ce977bbf0f10aed62ccc4de1cdf output.txt -- ./a.out
	// 其中 a.out 是选手程序（用户解决方案），b.out 是 special 程序（评测程序）
	args := []string{
		"--run",
		"--cg",
		fmt.Sprintf("--box-id=%d", ir.boxId),
		"--processes", // 允许多个进程
		"-e",          // 设置环境变量
		fmt.Sprintf("--time=%f", float64(timeLimit*4)),      // 时间限制（秒）- 交互题需要更多时间
		fmt.Sprintf("--wall-time=%f", float64(timeLimit*6)), // 墙钟时间限制
		fmt.Sprintf("--mem=%d", memoryLimit*1024*2),         // 内存限制（KB）
		"--meta=meta.txt", // 输出元数据
		"--",
		"/bin/bash",
		"./interactive_judge.sh",
		"./" + specialExeFilename, // 评测程序 (b.out)
		inputFilename,             // 额外参数
		"answer.txt",              // 输出文件
		"--",                      // 分隔符
		"./" + exeFilename,        // 选手程序 (a.out)
	}

	// 创建执行命令
	cmd := exec.Command(ir.IsolatePath, args...)

	// 设置沙箱目录为工作目录
	cmd.Dir = sandboxPath

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 记录开始时间
	startTime := time.Now()

	// 执行命令
	err = cmd.Run()

	// 计算墙钟时间
	realTime := time.Since(startTime)

	// 读取元数据文件
	metaPath := filepath.Join(sandboxPath, "meta.txt")
	metaContent, _ := file_util.ReadFileToString(metaPath)

	// 解析资源使用情况
	var cpuTime time.Duration
	var memUsed int64
	var exitCode int
	var isKilled bool
	var exitSig int

	lines := strings.Split(metaContent, "\n")
	zap.L().Info("Interactive Isolate meta content", zap.String("meta", metaContent))
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) >= 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "time":
				if t, err := strconv.ParseFloat(value, 64); err == nil {
					cpuTime = time.Duration(t * float64(time.Second))
				}
			case "cg-mem":
				if m, err := strconv.ParseInt(value, 10, 64); err == nil {
					memUsed = m * 1024 // convert KB to bytes
				}
			case "exitcode":
				if code, err := strconv.Atoi(value); err == nil {
					exitCode = code
				}
			case "exitsig":
				if sig, err := strconv.Atoi(value); err == nil {
					exitSig = sig
				}
			case "killed":
				isKilled = true
			case "cg-oom-killed":
				isKilled = true
				memUsed = memoryLimit * 1024 * 1024 * 2
			}
		}
	}

	output := normalizeString(stdout.String())
	errOutput := stderr.String()

	// 解析状态
	status := model.StatusAC
	var errorMsg string

	if err != nil {
		zap.L().Error("Interactive isolate execution error", zap.Error(err), zap.String("stderr", errOutput))
	}

	// 检查元数据中的状态标志
	if strings.Contains(metaContent, "status:TO") || isKilled {
		status = model.StatusTLE
		errorMsg = "时间超限或被终止"
	} else if strings.Contains(metaContent, "status:SG") || exitSig > 0 {
		status = model.StatusRE
		if exitSig > 0 {
			errorMsg = fmt.Sprintf("程序收到信号 %d 终止", exitSig)
		} else {
			errorMsg = "程序被信号终止"
		}
	} else if strings.Contains(metaContent, "status:XX") {
		status = model.StatusSE
		errorMsg = "沙箱内部错误"
	} else if strings.Contains(metaContent, "status:RE") || exitCode != 0 {
		status = model.StatusRE
		errorMsg = fmt.Sprintf("运行时错误: 退出码 %d", exitCode)
	}

	// 根据您提供的示例，如果评测程序返回码非0，则是Wrong Answer
	if status == model.StatusAC {
		if strings.Contains(errOutput, "Judge return code:") {
			// 检查是否有judge返回非0码
			if strings.Contains(errOutput, "Judge return code: ") {
				// 提取返回码
				lines := strings.Split(errOutput, "\n")
				for _, line := range lines {
					if strings.Contains(line, "Judge return code: ") {
						// 如果评测程序返回码非0，则是WA
						if !strings.Contains(line, "Judge return code: 0") {
							status = model.StatusWA
							errorMsg = "评测程序返回非0码，视为答案错误"
							break
						}
					}
				}
			}
		}

		// 检查输出中是否包含正确结果的标志
		if strings.Contains(output, "points") && strings.Contains(output, "run") {
			// 如果包含运行和注册成功的信息，但没有错误提示，认为是AC
			if !strings.Contains(output, "error") && !strings.Contains(output, "Error") {
				// 根据您的示例，只有当solution和judge都返回0时才是正确答案
				if strings.Contains(errOutput, "Solution return code: 0") && !strings.Contains(errOutput, "Judge return code: ") {
					// 如果找不到Judge return code，或者找到的是Judge return code: 0
					if !strings.Contains(errOutput, "Judge return code: ") || strings.Contains(errOutput, "Judge return code: 0") {
						status = model.StatusAC
					} else {
						status = model.StatusWA
						errorMsg = "评测程序返回非0码"
					}
				} else if strings.Contains(errOutput, "Solution return code: 0") {
					// 如果solution返回0，但judge返回非0，则是WA
					if strings.Contains(errOutput, "Judge return code: ") && !strings.Contains(errOutput, "Judge return code: 0") {
						status = model.StatusWA
						errorMsg = "评测程序返回非0码"
					}
				} else {
					// 如果solution返回非0，则是RE
					status = model.StatusRE
					errorMsg = "选手程序返回非0码"
				}
			} else {
				status = model.StatusWA
				errorMsg = "输出中包含错误信息"
			}
		} else {
			// 如果输出不符合预期格式，可能有问题
			status = model.StatusRE
			errorMsg = "输出格式不正确"
		}
	}

	// 二次检查资源限制
	if status == model.StatusAC {
		if cpuTime > time.Duration(timeLimit*4)*time.Second {
			status = model.StatusTLE
			errorMsg = fmt.Sprintf("CPU时间超限: %v > %vs", cpuTime, timeLimit*4)
		}
		if memUsed > memoryLimit*1024*1024*2 {
			status = model.StatusMLE
			errorMsg = fmt.Sprintf("内存超限: %d bytes > %d MB", memUsed, memoryLimit*2)
		}
	}

	// 记录运行结果
	zap.L().Info("Interactive isolate execution result",
		zap.Int("box_id", ir.boxId),
		zap.Int("test_case", runParams.TestCaseIndex),
		zap.Duration("cpu_time", cpuTime),
		zap.Duration("real_time", realTime),
		zap.Int64("memory_bytes", memUsed),
		zap.Float64("memory_mb", float64(memUsed)/(1024*1024)),
		zap.String("status", string(status)),
		zap.Int64("time_limit_sec", timeLimit),
		zap.Int64("mem_limit_mb", memoryLimit),
		zap.String("meta_content", metaContent),
		zap.String("output", output),
		zap.String("stderr", errOutput),
	)

	result := &model.TestCaseResult{
		TestCaseIndex: runParams.TestCaseIndex,
		Status:        status,
		TimeUsed:      cpuTime,
		MemUsed:       uint64(memUsed),
		Output:        output,
		Error:         errorMsg,
	}

	return result
}

// SetBoxId 设置沙箱ID
func (ir *IsoRunner) SetBoxId(id int) {
	ir.boxId = id
}

// GetBoxId 获取沙箱ID
func (ir *IsoRunner) GetBoxId() int {
	return ir.boxId
}
