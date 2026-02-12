package compiler

import (
	"fmt"
	"hitwh-judge/internal/constants"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// JavaCompiler Java编译器
type JavaCompiler struct {
	JavacPath string
}

// Compile 编译Java代码
func (j *JavaCompiler) Compile(codePath, exePath string) (string, error) {
	// Java编译后生成.class文件，不是可执行文件
	// exePath参数在这里表示输出目录
	outputDir := filepath.Dir(exePath)

	cmd := exec.Command(j.JavacPath, "-d", outputDir, codePath)

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
		zap.L().Warn("Java编译失败",
			zap.String("code_path", codePath),
			zap.String("error", errMsg),
		)
		return errMsg, fmt.Errorf("编译失败")
	}

	zap.L().Info("Java编译成功",
		zap.String("code_path", codePath),
		zap.String("output_dir", outputDir),
	)
	return "", nil
}

// PythonCompiler Python解释器（无需编译）
type PythonCompiler struct {
	PythonPath string
}

// Compile Python不需要编译，只检查语法
func (p *PythonCompiler) Compile(codePath, exePath string) (string, error) {
	// 使用 python -m py_compile 检查语法
	cmd := exec.Command(p.PythonPath, "-m", "py_compile", codePath)

	// 设置超时
	timer := time.AfterFunc(constants.MaxCompileTimeout, func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	// 执行语法检查
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(output)
		if errMsg == "" {
			errMsg = err.Error()
		}
		zap.L().Warn("Python语法检查失败",
			zap.String("code_path", codePath),
			zap.String("error", errMsg),
		)
		return errMsg, fmt.Errorf("语法检查失败")
	}

	zap.L().Info("Python语法检查成功",
		zap.String("code_path", codePath),
	)
	return "", nil
}

// GoCompiler Go编译器
type GoCompiler struct {
	GoPath string
}

// Compile 编译Go代码
func (g *GoCompiler) Compile(codePath, exePath string) (string, error) {
	cmd := exec.Command(g.GoPath, "build", "-o", exePath, codePath)

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
		zap.L().Warn("Go编译失败",
			zap.String("code_path", codePath),
			zap.String("error", errMsg),
		)
		return errMsg, fmt.Errorf("编译失败")
	}

	zap.L().Info("Go编译成功",
		zap.String("code_path", codePath),
		zap.String("exe_path", exePath),
	)
	return "", nil
}

// GetCompilerFlags 获取编译器标志
func GetCompilerFlags(lang constants.Language) string {
	switch lang {
	case constants.LanguageC:
		return constants.GCCDefaultFlags
	case constants.LanguageCpp:
		return constants.GPPDefaultFlags
	default:
		return ""
	}
}

// ValidateLanguage 验证语言是否支持
func ValidateLanguage(lang string) bool {
	supportedLanguages := []string{
		string(constants.LanguageC),
		string(constants.LanguageCpp),
		string(constants.LanguageJava),
		string(constants.LanguagePython),
		string(constants.LanguageGo),
	}

	lang = strings.ToLower(lang)
	for _, supported := range supportedLanguages {
		if lang == supported {
			return true
		}
	}
	return false
}
