package service

import (
	"context"
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
	"sync"
	"time"

	"go.uber.org/zap"
)

// 单机评测并发控制
var (
	// 同时最多运行的评测任务数（建议设置为 CPU 核心数）
	judgeSemaphore = make(chan struct{}, 10)
	// 统计当前评测任务数
	activeJudges int
	judgeMutex   sync.Mutex
)

// 评测超时配置
const (
	MaxJudgeTimeout = 5 * time.Minute // 单个评测任务最大超时时间
)

// AddTaskImproved 改进版的添加评测任务
func AddTaskImproved(ctx context.Context, req *v1.TaskReq) (*model.JudgeResult, error) {
	// 1. 参数校验
	if req == nil {
		return nil, fmt.Errorf("req is nil")
	}

	// 校验必要参数
	if req.CodeFile == "" {
		return nil, fmt.Errorf("代码文件不能为空")
	}
	if len(req.CheckPoints) == 0 {
		return nil, fmt.Errorf("测试用例不能为空")
	}
	if req.CPULimit <= 0 || req.CPULimit > 60000 {
		return nil, fmt.Errorf("CPU时间限制无效: %d (应在1-60000ms之间)", req.CPULimit)
	}
	if req.MemLimit <= 0 || req.MemLimit > 1024*1024*1024 {
		return nil, fmt.Errorf("内存限制无效: %d (应在1B-1GB之间)", req.MemLimit)
	}

	config := model.DefaultTaskConfig
	config.TimeLimit = int(req.CPULimit)
	config.MemoryLimit = int(req.MemLimit)
	config.Language = req.CodeLanguage

	if req.IsSpecial != nil && *req.IsSpecial {
		config.JudgeType = model.JudgeSpecial
	} else {
		config.JudgeType = model.JudgeNormal
	}

	taskId, err := snowflake.NextID()
	if err != nil {
		return nil, fmt.Errorf("生成任务ID失败: %w", err)
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

	// 2. 并发控制：获取评测槽位
	select {
	case judgeSemaphore <- struct{}{}:
		defer func() { <-judgeSemaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("评测请求已取消")
	case <-time.After(30 * time.Second):
		GetGlobalMetrics().RecordQueueTimeout()
		return nil, fmt.Errorf("评测队列已满，请稍后重试")
	}

	// 统计活跃评测数
	judgeMutex.Lock()
	activeJudges++
	currentActive := activeJudges
	judgeMutex.Unlock()
	defer func() {
		judgeMutex.Lock()
		activeJudges--
		judgeMutex.Unlock()
	}()

	// 记录统计
	GetGlobalMetrics().RecordSubmission()
	GetGlobalMetrics().RecordActiveIncrease()
	defer GetGlobalMetrics().RecordActiveDecrease()

	zap.L().Info("开始评测任务",
		zap.Int64("task_id", taskId),
		zap.Int("active_judges", currentActive),
		zap.Any("config", config),
	)

	// 3. 带超时的评测执行
	judgeCtx, cancel := context.WithTimeout(ctx, MaxJudgeTimeout)
	defer cancel()

	resultChan := make(chan *model.JudgeResult, 1)
	errChan := make(chan error, 1)

	go func() {
		if config.JudgeType == model.JudgeNormal {
			judgeResult, err := judgeImproved(&config, judgeTask)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- judgeResult
		} else {
			errChan <- fmt.Errorf("暂不支持特殊评测")
		}
	}()

	// 等待评测完成或超时
	select {
	case judgeResult := <-resultChan:
		GetGlobalMetrics().RecordSuccess(judgeResult.TotalTimeUsed, judgeResult.Status)
		zap.L().Info("评测任务完成",
			zap.Int64("task_id", taskId),
			zap.String("status", judgeResult.Status),
			zap.Duration("total_time", judgeResult.TotalTimeUsed),
		)
		return judgeResult, nil
	case err := <-errChan:
		GetGlobalMetrics().RecordFailure()
		zap.L().Error("评测任务失败", zap.Int64("task_id", taskId), zap.Error(err))
		return nil, err
	case <-judgeCtx.Done():
		GetGlobalMetrics().RecordFailure()
		return nil, fmt.Errorf("评测超时（超过%v）", MaxJudgeTimeout)
	}
}

// judgeImproved 改进版的评测逻辑
func judgeImproved(config *model.TaskConfig, task *model.JudgeTask) (*model.JudgeResult, error) {
	startTime := time.Now()

	// 1. 创建临时目录
	tempDir, cleanup, err := createTmpDirImproved()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 2. 写入用户代码
	codeFileName := language.GetCodeFileName(config.Language)
	codePath := filepath.Join(tempDir, codeFileName)
	if err := os.WriteFile(codePath, []byte(task.Code), 0600); err != nil {
		return nil, fmt.Errorf("写入代码文件失败: %w", err)
	}

	// 3. 编译代码
	exePath := filepath.Join(tempDir, "main")
	compilerInstance := compiler.NewCompiler(compiler.Language(config.Language))
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
	if err := downloadCaseImproved(task); err != nil {
		return nil, fmt.Errorf("下载测试用例失败: %w", err)
	}

	// 5. 运行所有测试用例
	var caseResults []model.TestCaseResult
	var maxMemUsed uint64
	var totalTimeUsed time.Duration
	finalStatus := model.StatusAC // 默认AC，遇到错误则更新

	for i, checkPoint := range task.TestCases {
		runParams := model.RunParams{
			TestCaseIndex: i,
			ExePath:       exePath,
			Input:         checkPoint.Input,
			TimeLimit:     int64(config.TimeLimit),
			MemLimit:      int64(config.MemoryLimit),
		}

		testCaseResult, err := runSandboxSafeImproved(runParams)
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
				testCaseResult.Error = fmt.Sprintf("输出不匹配")
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

// runSandboxSafeImproved 安全地运行沙箱，捕获panic
func runSandboxSafeImproved(runParams model.RunParams) (result *model.TestCaseResult, err error) {
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

	// sduSandboxConfig := runner.GetDefaultSandboxConfig(runner.SDUSandbox)
	// sduSandbox := runner.NewRunner(runner.SDUSandbox, sduSandboxConfig.Path)
	// testCaseResult := sduSandbox.RunInSandbox(runParams)
	nsJail := runner.GetDefaultSandboxConfig(runner.NsJail)
	nsjailSandBox := runner.NewRunner(runner.NsJail, nsJail.Path)
	testCaseResult := nsjailSandBox.RunInSandbox(runParams)

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

// updateFinalStatus 更新最终状态（按优先级）
func updateFinalStatus(current, newStatus model.JudgeStatus) model.JudgeStatus {
	priority := map[model.JudgeStatus]int{
		model.StatusSE:  6, // 系统错误优先级最高
		model.StatusCE:  5,
		model.StatusRE:  4,
		model.StatusTLE: 3,
		model.StatusMLE: 2,
		model.StatusWA:  1,
		model.StatusAC:  0, // AC优先级最低
	}

	if priority[newStatus] > priority[current] {
		return newStatus
	}
	return current
}

// calculateScore 计算总分（简单实现：AC的测试点占比）
func calculateScore(results []model.TestCaseResult) int {
	if len(results) == 0 {
		return 0
	}
	acCount := countACCases(results)
	return (acCount * 100) / len(results)
}

// countACCases 统计AC的测试点数量
func countACCases(results []model.TestCaseResult) int {
	count := 0
	for _, r := range results {
		if r.Status == model.StatusAC {
			count++
		}
	}
	return count
}

// truncateString 截断字符串（用于日志）
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// normalizeStringImproved 清理字符串中的换行符（统一为\n，去除首尾空白）
func normalizeStringImproved(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}

// downloadCaseImproved 下载测试用例
func downloadCaseImproved(task *model.JudgeTask) error {
	testCache := cache.GetEnhancedTestFileCache()
	for i := range task.TestCases {
		// 下载输入文件
		inputFilePath, err := testCache.DownloadFileByMD5WithCache(task.FileBucket, task.TestCases[i].InputFile)
		if err != nil {
			return fmt.Errorf("下载输入文件失败 (case %d): %w", i, err)
		}

		inputContent, err := os.ReadFile(inputFilePath)
		if err != nil {
			return fmt.Errorf("读取输入文件失败 (case %d): %w", i, err)
		}
		task.TestCases[i].Input = string(inputContent)

		// 下载输出文件
		outputFilePath, err := testCache.DownloadFileByMD5WithCache(task.FileBucket, task.TestCases[i].OutputFile)
		if err != nil {
			return fmt.Errorf("下载输出文件失败 (case %d): %w", i, err)
		}

		outputContent, err := os.ReadFile(outputFilePath)
		if err != nil {
			return fmt.Errorf("读取输出文件失败 (case %d): %w", i, err)
		}
		task.TestCases[i].Output = string(outputContent)
	}
	return nil
}

// createTmpDirImproved 创建临时目录
func createTmpDirImproved() (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "oj-judge-*")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	if err := os.Chmod(tempDir, 0777); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("修改临时目录权限失败: %w", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			zap.L().Warn("清理临时目录失败", zap.String("dir", tempDir), zap.Error(err))
		} else {
			zap.L().Debug("成功清理临时评测目录", zap.String("dir", tempDir))
		}
	}

	zap.L().Debug("创建临时评测目录", zap.String("dir", tempDir))
	return tempDir, cleanup, nil
}

// GetJudgeStats 获取评测统计信息（用于监控）
func GetJudgeStats() map[string]interface{} {
	judgeMutex.Lock()
	defer judgeMutex.Unlock()

	return map[string]interface{}{
		"active_judges":   activeJudges,
		"max_concurrent":  cap(judgeSemaphore),
		"available_slots": cap(judgeSemaphore) - len(judgeSemaphore),
	}
}
