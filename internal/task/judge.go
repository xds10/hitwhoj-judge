package task

import (
	"errors"
	"fmt"
	"hitwh-judge/internal/model"
	"hitwh-judge/internal/task/compiler"
	"hitwh-judge/internal/task/result"
	"hitwh-judge/internal/task/runner"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// 清理字符串中的换行符（统一为\n，去除首尾空白）
func normalizeString(s string) string {
	// 替换Windows换行符为Unix换行符
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// 去除首尾空白（包括换行、空格、制表符）
	return strings.TrimSpace(s)
}

// Judge 核心评测函数
// code: 用户代码, input: 输入数据, expectedOutput: 标准输出
// config: 评测配置（可选，不传则用默认配置）
func Judge(code, input, expectedOutput string, config ...model.TaskConfig) (*model.JudgeResult, error) {
	// 使用默认配置（如果未传入）
	judgeConfig := model.DefaultTaskConfig
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
	compiler := compiler.NewCompiler(compiler.LanguageC)
	compileErr, err := compiler.Compile(codePath, exePath)
	if err != nil {
		zap.L().Error("编译代码失败", zap.Error(err), zap.String("compile_err", compileErr))
		return &model.JudgeResult{
			Status: "CE",
			Error:  compileErr,
		}, nil
	}

	// 4. 在沙箱中运行程序
	nsJail := model.DefaultSandboxConfig
	sanbox := runner.NewRunner(runner.NsJail, nsJail.Path)
	programOutput, runErrOutput, runStatus, err := sanbox.RunInSandbox(exePath, input, judgeConfig.TimeLimit, judgeConfig.MemoryLimit)
	if err != nil {
		zap.L().Error("沙箱运行程序失败", zap.Error(err), zap.String("run_err", runErrOutput))
		return &model.JudgeResult{
			Status: runStatus,
			Error:  runErrOutput,
		}, nil
	}

	// 5. 对比输出（仅当运行状态为AC时）
	if runStatus == "AC" {
		comparator := result.NewComparator(false)
		if comparator.Compare(programOutput, expectedOutput) {
			zap.L().Info("评测通过（AC）", zap.String("output", programOutput))
			return &model.JudgeResult{
				Status: "AC",
				Error:  "",
			}, nil
		} else {
			errMsg := fmt.Sprintf("预期输出: %s, 实际输出: %s", normalizeString(expectedOutput), programOutput)
			zap.L().Info("答案错误（WA）", zap.String("error", errMsg))
			return &model.JudgeResult{
				Status: "WA",
				Error:  errMsg,
			}, nil
		}
	}

	// 非AC的运行状态（TLE/MLE/RE）
	zap.L().Info("程序运行异常", zap.String("status", runStatus), zap.String("error", runErrOutput))
	return &model.JudgeResult{
		Status: runStatus,
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
