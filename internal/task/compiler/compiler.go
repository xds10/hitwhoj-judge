package compiler

import (
	"hitwh-judge/internal/conf"
	"hitwh-judge/internal/constants"
)

// Compiler 编译器接口
type Compiler interface {
	Compile(codePath, exePath string) (string, error)
}

// NewCompiler 创建编译器实例
func NewCompiler(lang constants.Language) Compiler {
	switch lang {
	case constants.LanguageC:
		gccPath := conf.DefaultOptions.GCPath
		if gccPath == "" {
			gccPath = constants.GCCPath
		}
		return &CCompiler{
			GCPath: gccPath,
		}
	case constants.LanguageCpp:
		return &CppCompiler{
			GPPPath: constants.GPPPath,
			Flags:   constants.GPPDefaultFlags,
		}
	case constants.LanguageJava:
		return &JavaCompiler{
			JavacPath: constants.JavacPath,
		}
	case constants.LanguagePython:
		return &PythonCompiler{
			PythonPath: constants.PythonPath,
		}
	case constants.LanguageGo:
		return &GoCompiler{
			GoPath: constants.GoPath,
		}
	default:
		return nil
	}
}
