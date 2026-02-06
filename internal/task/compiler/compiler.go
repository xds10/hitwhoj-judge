package compiler

import (
	"hitwh-judge/internal/conf"
	"hitwh-judge/internal/constants"
)

type Language string

const (
	LanguageC      Language = "c"
	LanguageCpp    Language = "cpp"
	LanguageJava   Language = "java"
	LanguagePython Language = "python"
	LanguageGo     Language = "go"
	LanguageJs     Language = "js"
	LanguageHtml   Language = "html"
	LanguageCss    Language = "css"
	LanguageSql    Language = "sql"
)

// Compiler 编译器接口
type Compiler interface {
	Compile(codePath, exePath string) (string, error)
}

// NewCompiler 创建编译器实例
func NewCompiler(lang Language) Compiler {
	switch lang {
	case LanguageC:
		gccPath := conf.DefaultOptions.GCPath
		if gccPath == "" {
			gccPath = constants.GCCPath
		}
		return &CCompiler{
			GCPath: gccPath,
		}
	case LanguageCpp:
		return &CppCompiler{
			GPPPath: constants.GPPPath,
			Flags:   constants.GPPDefaultFlags,
		}
	case LanguageJava:
		return &JavaCompiler{
			JavacPath: constants.JavacPath,
		}
	case LanguagePython:
		return &PythonCompiler{
			PythonPath: constants.PythonPath,
		}
	case LanguageGo:
		return &GoCompiler{
			GoPath: constants.GoPath,
		}
	default:
		return nil
	}
}
