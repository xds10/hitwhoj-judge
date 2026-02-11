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
}

// DefaultIsolateSandboxConfig 默认isolate沙箱配置
var DefaultIsolateSandboxConfig = model.SandboxConfig{
	Type: "isolate",
	Path: "isolate",
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

// SetBoxId 设置沙箱ID
func (ir *IsoRunner) SetBoxId(id int) {
	ir.boxId = id
}

// GetBoxId 获取沙箱ID
func (ir *IsoRunner) GetBoxId() int {
	return ir.boxId
}
