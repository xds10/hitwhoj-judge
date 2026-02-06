package compiler

import (
	"fmt"
	"hitwh-judge/internal/constants"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

// CppCompiler C++编译器
type CppCompiler struct {
	GPPPath string
	Flags   string
}

// Compile 编译C++代码
func (c *CppCompiler) Compile(codePath, exePath string) (string, error) {
	// 默认编译选项
	flags := c.Flags
	if flags == "" {
		flags = constants.GPPDefaultFlags
	}

	// 构建编译命令
	args := []string{codePath, "-o", exePath}
	// 添加编译选项
	if flags != "" {
		flagList := strings.Fields(flags)
		args = append(flagList, args...)
	}

	cmd := exec.Command(c.GPPPath, args...)

	// 设置编译超时
	timer := time.AfterFunc(constants.MaxCompileTimeout, func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	// 执行编译
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(output)
		if errMsg == "" {
			errMsg = err.Error()
		}
		zap.L().Warn("C++编译失败",
			zap.String("code_path", codePath),
			zap.String("error", errMsg),
		)
		return errMsg, fmt.Errorf("编译失败")
	}

	zap.L().Info("C++编译成功",
		zap.String("code_path", codePath),
		zap.String("exe_path", exePath),
	)
	return "", nil
}
