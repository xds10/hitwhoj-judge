package compiler

import "hitwh-judge/internal/conf"

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

func NewCompiler(lang Language) Compiler {
	switch lang {
	case LanguageC:
		return &CCompiler{
			GCPath: conf.DefaultOptions.GCPath,
		}
	default:
		return nil
	}
}
