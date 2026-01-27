package service

import (
	"context"
	"errors"
	"fmt"
	v1 "hitwh-judge/api/calc/v1"
	"hitwh-judge/internal/model"
	"hitwh-judge/internal/task/compiler"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func AddTask(ctx context.Context, req *v1.TaskReq) error {
	// 1. 参数校验
	if req == nil {
		return fmt.Errorf("req is nil")
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
	if config.JudgeType == model.JudgeNormal {
		judge(&config, req.CodeFile)
	}

	return nil
}
func judge(config *model.TaskConfig, code string) (*model.JudgeResult, error) {
	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 2. 写入用户代码到临时文件（C语言main.c）
	codePath := filepath.Join(tempDir, "main.code")
	if err := os.WriteFile(codePath, []byte(code), 0600); err != nil { // 权限0600，仅当前用户可读写
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

	return nil, nil

}

func createTmpDir() (string, func(), error) {
	// 1. 创建临时目录（权限0700，仅当前用户可访问）
	tempDir, err := os.MkdirTemp("", "oj-judge-*")
	if err != nil {
		errMsg := fmt.Sprintf("创建临时目录失败: %v", err)
		zap.L().Error(errMsg)
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
