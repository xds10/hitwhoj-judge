package service

import (
	"fmt"
	"hitwh-judge/internal/constants"
	"hitwh-judge/internal/model"
	"hitwh-judge/internal/task/compiler"
	"hitwh-judge/internal/task/language"
	"hitwh-judge/internal/task/result"
	"hitwh-judge/internal/task/runner"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

func judgeInteractive(config *model.TaskConfig, task *model.JudgeTask) (*model.JudgeResult, error) {
	startTime := time.Now()

	// 1. 创建临时目录
	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		return nil, err
	}
	defer cleanup()
	task.TempDir = tempDir

	// 2. 写入用户代码
	codeFileName := language.GetCodeFileName(config.Language)
	codePath := filepath.Join(tempDir, codeFileName)
	if err := os.WriteFile(codePath, []byte(task.Code), 0600); err != nil {
		return nil, fmt.Errorf("写入代码文件失败: %w", err)
	}
	var specialCodeFileName, specialCodePath, specialCodeLanguage, specialExePath string
	if config.JudgeType != model.JudgeNormal {
		specialCodeFileName = language.GetCodeFileName(specialCodeLanguage)
		specialCodePath = filepath.Join(tempDir, specialCodeFileName)
		specialCodeLanguage = language.DetectLanguageByExtension(*task.SpecialCodeFileName)
		if err := os.WriteFile(specialCodePath, []byte(*task.SpecialCode), 0600); err != nil {
			return nil, fmt.Errorf("写入特殊评测代码文件失败: %w", err)
		}
	}

	// 3. 编译代码
	exePath := filepath.Join(tempDir, "main")
	compilerInstance := compiler.NewCompiler(constants.Language(config.Language))
	compileErr, err := compilerInstance.Compile(codePath, exePath)
	if err != nil {
		zap.L().Warn("编译失败",
			zap.Int64("task_id", task.TaskID),
			zap.String("compile_err", compileErr),
		)
		return &model.JudgeResult{
			TaskID: task.TaskID,
			Status: model.StatusCE,
			CompileResult: model.CompileResult{
				Success: false,
				Message: compileErr,
			},
			Error:      compileErr,
			SubmitTime: time.Unix(task.CreateTime, 0),
			JudgeTime:  time.Now(),
		}, nil
	}

	if task.Config.JudgeType != model.JudgeNormal {
		specialExePath = filepath.Join(tempDir, "special_main")
		compileErr, err = compileCode(tempDir, specialCodePath, specialExePath, specialCodeLanguage)
		if err != nil {
			zap.L().Warn("编译特殊评测代码失败",
				zap.Int64("task_id", task.TaskID),
				zap.String("compile_err", compileErr),
			)
			return &model.JudgeResult{
				TaskID: task.TaskID,
				Status: model.StatusCE,
				CompileResult: model.CompileResult{
					Success: false,
					Message: compileErr,
				},
				Error:      compileErr,
				SubmitTime: time.Unix(task.CreateTime, 0),
				JudgeTime:  time.Now(),
			}, nil
		}
	}

	// 4. 下载测试用例
	if err := downloadCase(task); err != nil {
		return nil, fmt.Errorf("下载测试用例失败: %w", err)
	}

	// 5. 运行所有测试用例
	var caseResults []model.TestCaseResult
	var maxMemUsed uint64
	var totalTimeUsed time.Duration
	finalStatus := model.StatusAC // 默认AC，遇到错误则更新

	for i, checkPoint := range task.TestCases {
		runParams := model.RunParams{
			TaskID:         task.TaskID,
			TestCaseIndex:  i,
			ExePath:        exePath,
			Input:          checkPoint.Input,
			InputFile:      checkPoint.InputFile,
			Answer:         checkPoint.Output,
			TimeLimit:      int64(config.TimeLimit),
			MemLimit:       int64(config.MemoryLimit),
			Config:         *config,
			SpecialExePath: specialExePath,
		}

		testCaseResult, err := runInteractive(runParams)
		if err != nil {
			// 沙箱运行出错，标记为系统错误
			testCaseResult = &model.TestCaseResult{
				TestCaseIndex: i,
				Status:        model.StatusSE,
				Error:         err.Error(),
				Expected:      checkPoint.Output,
			}
		} else {
			testCaseResult.Expected = checkPoint.Output
		}

		// 6. 对比输出（仅当运行状态为AC时）
		if testCaseResult.Status == model.StatusAC {
			comparator := result.NewComparator(false)
			if comparator.Compare(testCaseResult.Output, checkPoint.Output) {
				testCaseResult.Status = model.StatusAC
			} else {
				testCaseResult.Status = model.StatusWA
				testCaseResult.Error = "输出不匹配"
				// 只在日志中记录详细差异，避免返回过大数据
				zap.L().Debug("输出不匹配",
					zap.Int("case", i),
					zap.String("expected", truncateString(checkPoint.Output, 100)),
					zap.String("actual", truncateString(testCaseResult.Output, 100)),
				)
			}
		}

		// 更新统计信息
		if testCaseResult.MemUsed > maxMemUsed {
			maxMemUsed = testCaseResult.MemUsed
		}
		totalTimeUsed += testCaseResult.TimeUsed

		// 更新最终状态（优先级：SE > CE > RE > TLE > MLE > WA > AC）
		finalStatus = updateFinalStatus(finalStatus, testCaseResult.Status)

		caseResults = append(caseResults, *testCaseResult)

		// 如果不是AC，可以选择是否继续运行后续测试点（可配置）
		// if testCaseResult.Status != model.StatusAC {
		// 	break // 提前终止评测
		// }
	}

	// 7. 构建最终结果
	judgeResult := &model.JudgeResult{
		TaskID:        task.TaskID,
		Status:        finalStatus,
		TotalScore:    calculateScore(caseResults),
		TotalTimeUsed: totalTimeUsed,
		TotalMemUsed:  maxMemUsed,
		CompileResult: model.CompileResult{
			Success: true,
			Message: "编译成功",
		},
		TestResults: caseResults,
		SubmitTime:  time.Unix(task.CreateTime, 0),
		JudgeTime:   time.Now(),
	}

	// 记录评测耗时
	judgeDuration := time.Since(startTime)
	zap.L().Info("评测完成",
		zap.Int64("task_id", task.TaskID),
		zap.String("status", finalStatus),
		zap.Int("total_cases", len(caseResults)),
		zap.Int("ac_cases", countACCases(caseResults)),
		zap.Duration("judge_duration", judgeDuration),
	)

	return judgeResult, nil
}
func judgeNormal(config *model.TaskConfig, task *model.JudgeTask) (*model.JudgeResult, error) {
	startTime := time.Now()

	// 1. 创建临时目录
	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		return nil, err
	}
	defer cleanup()
	task.TempDir = tempDir

	// 2. 写入用户代码
	codeFileName := language.GetCodeFileName(config.Language)
	codePath := filepath.Join(tempDir, codeFileName)
	if err := os.WriteFile(codePath, []byte(task.Code), 0600); err != nil {
		return nil, fmt.Errorf("写入代码文件失败: %w", err)
	}

	// 3. 编译代码
	exePath := filepath.Join(tempDir, "main")
	compilerInstance := compiler.NewCompiler(constants.Language(config.Language))
	compileErr, err := compilerInstance.Compile(codePath, exePath)
	if err != nil {
		zap.L().Warn("编译失败",
			zap.Int64("task_id", task.TaskID),
			zap.String("compile_err", compileErr),
		)
		return &model.JudgeResult{
			TaskID: task.TaskID,
			Status: model.StatusCE,
			CompileResult: model.CompileResult{
				Success: false,
				Message: compileErr,
			},
			Error:      compileErr,
			SubmitTime: time.Unix(task.CreateTime, 0),
			JudgeTime:  time.Now(),
		}, nil
	}

	// 4. 下载测试用例
	if err := downloadCase(task); err != nil {
		return nil, fmt.Errorf("下载测试用例失败: %w", err)
	}

	// 5. 运行所有测试用例
	var caseResults []model.TestCaseResult
	var maxMemUsed uint64
	var totalTimeUsed time.Duration
	finalStatus := model.StatusAC // 默认AC，遇到错误则更新

	for i, checkPoint := range task.TestCases {
		runParams := model.RunParams{
			TaskID:        task.TaskID,
			TestCaseIndex: i,
			ExePath:       exePath,
			Input:         checkPoint.Input,
			InputFile:     checkPoint.InputFile,
			TimeLimit:     int64(config.TimeLimit),
			MemLimit:      int64(config.MemoryLimit),
			Config:        *config,
		}

		testCaseResult, err := runSandboxSafe(runParams)
		if err != nil {
			// 沙箱运行出错，标记为系统错误
			testCaseResult = &model.TestCaseResult{
				TestCaseIndex: i,
				Status:        model.StatusSE,
				Error:         err.Error(),
				Expected:      checkPoint.Output,
			}
		} else {
			testCaseResult.Expected = checkPoint.Output
		}

		// 6. 对比输出（仅当运行状态为AC时）
		if testCaseResult.Status == model.StatusAC {
			comparator := result.NewComparator(false)
			if comparator.Compare(testCaseResult.Output, checkPoint.Output) {
				testCaseResult.Status = model.StatusAC
			} else {
				testCaseResult.Status = model.StatusWA
				testCaseResult.Error = "输出不匹配"
				// 只在日志中记录详细差异，避免返回过大数据
				zap.L().Debug("输出不匹配",
					zap.Int("case", i),
					zap.String("expected", truncateString(checkPoint.Output, 100)),
					zap.String("actual", truncateString(testCaseResult.Output, 100)),
				)
			}
		}

		// 更新统计信息
		if testCaseResult.MemUsed > maxMemUsed {
			maxMemUsed = testCaseResult.MemUsed
		}
		totalTimeUsed += testCaseResult.TimeUsed

		// 更新最终状态（优先级：SE > CE > RE > TLE > MLE > WA > AC）
		finalStatus = updateFinalStatus(finalStatus, testCaseResult.Status)

		caseResults = append(caseResults, *testCaseResult)

		// 如果不是AC，可以选择是否继续运行后续测试点（可配置）
		// if testCaseResult.Status != model.StatusAC {
		// 	break // 提前终止评测
		// }
	}

	// 7. 构建最终结果
	judgeResult := &model.JudgeResult{
		TaskID:        task.TaskID,
		Status:        finalStatus,
		TotalScore:    calculateScore(caseResults),
		TotalTimeUsed: totalTimeUsed,
		TotalMemUsed:  maxMemUsed,
		CompileResult: model.CompileResult{
			Success: true,
			Message: "编译成功",
		},
		TestResults: caseResults,
		SubmitTime:  time.Unix(task.CreateTime, 0),
		JudgeTime:   time.Now(),
	}

	// 记录评测耗时
	judgeDuration := time.Since(startTime)
	zap.L().Info("评测完成",
		zap.Int64("task_id", task.TaskID),
		zap.String("status", finalStatus),
		zap.Int("total_cases", len(caseResults)),
		zap.Int("ac_cases", countACCases(caseResults)),
		zap.Duration("judge_duration", judgeDuration),
	)

	return judgeResult, nil
}

func compileCode(tmpDir string, srcFile string, dstFile string, language string) (string, error) {
	compilerInstance := compiler.NewCompiler(constants.Language(language))
	compileErr, err := compilerInstance.Compile(srcFile, dstFile)
	if err != nil {
		return compileErr, err
	}
	return compileErr, err
}

// runSandboxSafe 安全地运行沙箱，捕获panic
func runSandboxSafe(runParams model.RunParams) (result *model.TestCaseResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("沙箱运行panic: %v", r)
			result = &model.TestCaseResult{
				TestCaseIndex: runParams.TestCaseIndex,
				Status:        model.StatusSE,
				Error:         fmt.Sprintf("系统错误: %v", r),
			}
		}
	}()

	var testCaseResult *model.TestCaseResult

	// 根据评测类型选择不同的沙箱
	switch runParams.Config.JudgeType {
	// case model.JudgeInteractive:
	// 	// 交互题使用nsjail沙箱的特殊方法
	// 	nsJail := runner.GetDefaultSandboxConfig(runner.NsJail)
	// 	nsjailSandBox := runner.NewRunner(runner.NsJail, nsJail.Path)
	// 	// 将普通的Runner转换为NsJailRunner以调用交互方法
	// 	if nsjailRunner, ok := nsjailSandBox.(*runner.NsJailRunner); ok {
	// 		testCaseResult = nsjailRunner.RunInteractiveInSandbox(runParams)
	// 	} else {
	// 		return nil, fmt.Errorf("无法转换为NsJailRunner类型")
	// 	}
	default:
		// 其他类型使用isolate沙箱
		isolate := runner.GetDefaultSandboxConfig(runner.Isolate)
		isolateSandBox := runner.NewRunner(runner.Isolate, isolate.Path)
		testCaseResult = isolateSandBox.RunInSandbox(runParams)

		// isolate := runner.GetDefaultSandboxConfig(runner.NsJail)
		// isolateSandBox := runner.NewRunner(runner.NsJail, isolate.Path)
		// testCaseResult = isolateSandBox.RunInSandbox(runParams)
	}

	if testCaseResult == nil {
		return nil, fmt.Errorf("沙箱返回结果为空")
	}

	if testCaseResult.Error != "" {
		// 有错误信息但不一定是致命错误，返回结果让上层判断
		zap.L().Warn("沙箱运行有错误信息",
			zap.Int("case", runParams.TestCaseIndex),
			zap.String("error", testCaseResult.Error),
			zap.String("status", testCaseResult.Status),
		)
	}

	return testCaseResult, nil
}

// runSandboxSafe 安全地运行沙箱，捕获panic
func runInteractive(runParams model.RunParams) (result *model.TestCaseResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("沙箱运行panic: %v", r)
			result = &model.TestCaseResult{
				TestCaseIndex: runParams.TestCaseIndex,
				Status:        model.StatusSE,
				Error:         fmt.Sprintf("系统错误: %v", r),
			}
		}
	}()

	var testCaseResult *model.TestCaseResult

	// 根据评测类型选择不同的沙箱
	switch runParams.Config.JudgeType {
	// case model.JudgeInteractive:
	// 	// 交互题使用nsjail沙箱的特殊方法
	// 	nsJail := runner.GetDefaultSandboxConfig(runner.NsJail)
	// 	nsjailSandBox := runner.NewRunner(runner.NsJail, nsJail.Path)
	// 	// 将普通的Runner转换为NsJailRunner以调用交互方法
	// 	if nsjailRunner, ok := nsjailSandBox.(*runner.NsJailRunner); ok {
	// 		testCaseResult = nsjailRunner.RunInteractiveInSandbox(runParams)
	// 	} else {
	// 		return nil, fmt.Errorf("无法转换为NsJailRunner类型")
	// 	}
	default:
		// 其他类型使用isolate沙箱
		isolate := runner.GetDefaultSandboxConfig(runner.Isolate)
		isolateSandBox := runner.NewRunner(runner.Isolate, isolate.Path)
		testCaseResult = isolateSandBox.RunInteractiveInSandbox(runParams)

		// isolate := runner.GetDefaultSandboxConfig(runner.NsJail)
		// isolateSandBox := runner.NewRunner(runner.NsJail, isolate.Path)
		// testCaseResult = isolateSandBox.RunInSandbox(runParams)
	}

	if testCaseResult == nil {
		return nil, fmt.Errorf("沙箱返回结果为空")
	}

	if testCaseResult.Error != "" {
		// 有错误信息但不一定是致命错误，返回结果让上层判断
		zap.L().Warn("沙箱运行有错误信息",
			zap.Int("case", runParams.TestCaseIndex),
			zap.String("error", testCaseResult.Error),
			zap.String("status", testCaseResult.Status),
		)
	}

	return testCaseResult, nil
}
