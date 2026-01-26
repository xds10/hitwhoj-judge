package compiler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CCompiler C语言编译器
type CCompiler struct {
	GCPath string
}

// Compile 编译C代码
func (cc *CCompiler) Compile(codePath, exePath string) (string, error) {
	// 检查gcc是否存在
	if _, err := exec.LookPath(cc.GCPath); err != nil {
		return "", fmt.Errorf("命令不存在: %s, 错误: %w", cc.GCPath, err)
	}

	// 编译命令
	cmd := exec.Command(
		cc.GCPath,
		"-o", exePath,
		codePath,
		"-Wall", "-O2", "-static", "-std=c11", //TODO : o2
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Dir = filepath.Dir(codePath)

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
