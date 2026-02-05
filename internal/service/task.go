package service

import (
	"context"
	"errors"
	"fmt"
	v1 "hitwh-judge/api/calc/v1"
	"hitwh-judge/internal/cache"
	"hitwh-judge/internal/model"
	"hitwh-judge/internal/task/compiler"
	"hitwh-judge/internal/task/language"
	"hitwh-judge/internal/task/result"
	"hitwh-judge/internal/task/runner"
	"hitwh-judge/pkg/snowflake"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

func AddTask(ctx context.Context, req *v1.TaskReq) (*model.JudgeResult, error) {
	// 1. 参数校验
	if req == nil {
		return nil, fmt.Errorf("req is nil")
	}
	config := model.DefaultTaskConfig

	config.TimeLimit = int(req.CPULimit)
	config.MemoryLimit = int(req.MemLimit)
	config.Language = req.CodeLanguage
	zap.L().Info("评测参数", zap.Any("config", config))
	if req.IsSpecial != nil && *req.IsSpecial {
		config.JudgeType = model.JudgeSpecial
	} else {
		config.JudgeType = model.JudgeNormal
	}

	taskId, err := snowflake.NextID()
	if err != nil {
		return nil, err
	}
	judgeTask := &model.JudgeTask{
		TaskID:      taskId,
		Config:      config,
		Code:        req.CodeFile,
		FileBucket:  req.Bucket,
		SpecialCode: &req.SpecialCodeFile,
		CreateTime:  time.Now().Unix(),
	}
	for _, checkPoint := range req.CheckPoints {
		judgeTask.TestCases = append(judgeTask.TestCases, model.TestCase{
			InputFile:  checkPoint.InputFile,
			OutputFile: checkPoint.OutputFile,
		})
	}
	var judgeResult *model.JudgeResult
	if config.JudgeType == model.JudgeNormal {
		judgeResult, err = judge(&config, judgeTask)
		if err != nil {
			return nil, err
		}
		zap.L().Info("judgeResult", zap.Any("judgeResult", judgeResult))
	}
	return judgeResult, nil
}
func judge(config *model.TaskConfig, task *model.JudgeTask) (*model.JudgeResult, error) {
	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 2. 写入用户代码到临时文件（C语言main.c）
	codeFileName := language.GetCodeFileName(config.Language)
	codePath := filepath.Join(tempDir, codeFileName)
	if err := os.WriteFile(codePath, []byte(task.Code), 0600); err != nil { // 权限0600，仅当前用户可读写
		errMsg := fmt.Sprintf("写入代码文件失败: %v", err)
		zap.L().Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// 3. 编译代码
	exePath := filepath.Join(tempDir, "main")
	compiler := compiler.NewCompiler(compiler.Language(config.Language))
	compileErr, err := compiler.Compile(codePath, exePath)
	if err != nil {
		zap.L().Error("编译代码失败", zap.Error(err), zap.String("compile_err", compileErr))
		return &model.JudgeResult{
			Status: "CE",
			Error:  compileErr,
		}, nil
	}
	err = downloadCase(task)
	if err != nil {
		return nil, err
	}
	zap.L().Info("评测任务", zap.Any("task", task))
	// 4. 在沙箱中运行程序
	var caseResults []model.TestCaseResult
	for i, checkPoint := range task.TestCases {
		runParams := model.RunParams{
			TestCaseIndex: int(i),
			ExePath:       exePath,
			Input:         checkPoint.Input,
			TimeLimit:     int64(config.TimeLimit),
			MemLimit:      int64(config.MemoryLimit),
		}
		testCaseResult, err := runSanBox(runParams)
		testCaseResult.Expected = checkPoint.Output

		if err != nil {
			zap.L().Error("沙箱运行程序失败", zap.Error(err), zap.String("run_err", testCaseResult.Error))
			return &model.JudgeResult{
				Status: testCaseResult.Status,
				Error:  testCaseResult.Error,
			}, nil
		}
		zap.L().Info("runRes", zap.Any("runRes", testCaseResult))
		// 5. 对比输出（仅当运行状态为AC时）
		if testCaseResult.Status == model.StatusAC {
			expectedOutput := checkPoint.Output
			comparator := result.NewComparator(false)

			result := comparator.Compare(testCaseResult.Output, expectedOutput)
			if result {
				testCaseResult.Status = model.StatusAC
				caseResults = append(caseResults, *testCaseResult)
			} else {
				testCaseResult.Status = model.StatusWA
				testCaseResult.Error = fmt.Sprintf("预期输出: %s, 实际输出: %s", normalizeString(expectedOutput), testCaseResult.Output)
				caseResults = append(caseResults, *testCaseResult)
			}

		} else {
			caseResults = append(caseResults, *testCaseResult)
		}

	}
	judgeResult := &model.JudgeResult{
		TaskID:        task.TaskID,
		Status:        "AC",
		TotalScore:    0,
		TotalTimeUsed: 0,
		TotalMemUsed:  0,
		CompileResult: model.CompileResult{
			Success: true,
			Message: "",
		},
		TestResults: caseResults,
		CodeFileID:  0,
		SubmitTime:  time.Now(),
		JudgeTime:   time.Now(),
		Error:       "",
	}
	return judgeResult, nil

}

// 清理字符串中的换行符（统一为\n，去除首尾空白）
func normalizeString(s string) string {
	// 替换Windows换行符为Unix换行符
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// 去除首尾空白（包括换行、空格、制表符）
	return strings.TrimSpace(s)
}

func runSanBox(runParams model.RunParams) (*model.TestCaseResult, error) {

	// config := model.TaskConfig{
	// 	TimeLimit:   timeLimit,
	// 	MemoryLimit: memoryLimit,
	// }
	// sanbox := runner.NewRunner(runner.NsJail, nsJail.Path)
	// judgeController := NewJudgeController(sanbox)
	// runResult, err := judgeController.RunWithResourceLimit(exePath, input, config)
	// if err != nil {
	// 	return runResult.Status, runResult.Error, runResult.Output, err
	// }

	// nsJail := runner.GetDefaultSandboxConfig(runner.NsJail)
	// nsjailSandBox := runner.NewRunner(runner.NsJail, nsJail.Path)
	// testCaseResult := nsjailSandBox.RunInSandbox(runParams)
	// if testCaseResult != nil && testCaseResult.Error != "" {
	// 	return testCaseResult, errors.New(testCaseResult.Error)
	// }
	sduSandboxConfig := runner.GetDefaultSandboxConfig(runner.SDUSandbox)
	sduSandbox := runner.NewRunner(runner.SDUSandbox, sduSandboxConfig.Path)
	testCaseResult := sduSandbox.RunInSandbox(runParams)
	// 检查沙箱是否返回错误
	if testCaseResult != nil && testCaseResult.Error != "" {
		return testCaseResult, errors.New(testCaseResult.Error)
	}
	return testCaseResult, nil
}
func downloadCase(task *model.JudgeTask) (err error) {
	testCache := cache.GetEnhancedTestFileCache()
	for i := range task.TestCases {
		// 下载输入文件
		inputFilePath, err := testCache.DownloadFileByMD5WithCache(task.FileBucket, task.TestCases[i].InputFile)
		if err != nil {
			return err
		}
		// 读取输入文件内容
		inputContent, err := os.ReadFile(inputFilePath)
		if err != nil {
			return err
		}
		task.TestCases[i].Input = string(inputContent)

		// 下载输出文件
		outputFilePath, err := testCache.DownloadFileByMD5WithCache(task.FileBucket, task.TestCases[i].OutputFile)
		if err != nil {
			return err
		}
		// 读取输出文件内容
		outputContent, err := os.ReadFile(outputFilePath)
		if err != nil {
			return err
		}
		task.TestCases[i].Output = string(outputContent)
	}
	return nil
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
