package runner

import (
	"errors"
	"fmt"
	"hitwh-judge/internal/constants"
	"os"
	"strings"

	"go.uber.org/zap"
)

// normalizeString 清理字符串中的换行符
func normalizeString(s string) string {
	// 替换Windows换行符为Unix换行符
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// 去除首尾空白（包括换行、空格、制表符）
	return strings.TrimSpace(s)
}

// createTmpDir 创建临时目录
func createTmpDir() (string, func(), error) {
	// 1. 创建临时目录（权限0700，仅当前用户可访问）
	tempDir, err := os.MkdirTemp("", constants.TempDirPrefix)
	if err != nil {
		errMsg := fmt.Sprintf("创建临时目录失败: %v", err)
		zap.L().Error(errMsg)
		return "", nil, errors.New(errMsg)
	}

	// 2. 修改权限为0777（某些沙箱需要）
	if err := os.Chmod(tempDir, constants.TempDirPerm); err != nil {
		errMsg := fmt.Sprintf("修改临时目录权限失败: %v, 目录路径: %s", err, tempDir)
		zap.L().Error(errMsg)
		// 权限修改失败时，清理已创建的临时目录，避免残留
		_ = os.RemoveAll(tempDir)
		return "", nil, errors.New(errMsg)
	}

	// 3. 定义清理函数（闭包，捕获tempDir变量）
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

// truncateOutput 截断输出（防止输出过大）
func truncateOutput(output string, maxSize int) string {
	if len(output) <= maxSize {
		return output
	}
	return output[:maxSize] + fmt.Sprintf("\n... (输出被截断，总长度: %d)", len(output))
}

// validateRunParams 验证运行参数
func validateRunParams(exePath string, timeLimit, memLimit int64) error {
	// 检查可执行文件是否存在
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("可执行文件不存在: %s", exePath)
	}

	// 检查时间限制
	if timeLimit < int64(constants.MinTimeLimit/1000) || timeLimit > int64(constants.MaxTimeLimit/1000) {
		return fmt.Errorf("时间限制无效: %d (应在 %d-%d 秒之间)",
			timeLimit, constants.MinTimeLimit/1000, constants.MaxTimeLimit/1000)
	}

	// 检查内存限制
	if memLimit < int64(constants.MinMemoryLimit/(1024*1024)) ||
		memLimit > int64(constants.MaxMemoryLimit/(1024*1024)) {
		return fmt.Errorf("内存限制无效: %d (应在 %d-%d MB之间)",
			memLimit, constants.MinMemoryLimit/(1024*1024), constants.MaxMemoryLimit/(1024*1024))
	}

	return nil
}

// sanitizeInput 清理输入数据
func sanitizeInput(input string) string {
	// 规范化换行符
	input = normalizeString(input)

	// 限制输入大小（防止过大输入）
	if len(input) > constants.MaxOutputSize {
		input = input[:constants.MaxOutputSize]
		zap.L().Warn("输入数据过大，已截断", zap.Int("size", len(input)))
	}

	return input
}

// sanitizeError 清理错误信息
func sanitizeError(errMsg string) string {
	// 限制错误信息大小
	if len(errMsg) > constants.MaxErrorSize {
		return errMsg[:constants.MaxErrorSize] + "..."
	}
	return errMsg
}
