package language

import (
	"hitwh-judge/internal/constants"
	"strings"
)

// DetectLanguageByExtension 根据文件扩展名判断编程语言
func DetectLanguageByExtension(filename string) string {
	// 获取文件扩展名
	ext := getFileExtension(filename)

	switch strings.ToLower(ext) {
	case ".c":
		return constants.LanguageC
	case ".cpp", ".cxx", ".cc":
		return constants.LanguageCpp
	case ".java":
		return constants.LanguageJava
	case ".py", ".py3":
		return constants.LanguagePython
	case ".js":
		return constants.LanguageJs
	case ".go":
		return constants.LanguageGo
	case ".rs":
		return constants.LanguageRust
	case ".php":
		return constants.LanguagePHP
	case ".rb":
		return constants.LanguageRuby
	case ".cs":
		return constants.LanguageCSharp
	case ".swift":
		return constants.LanguageSwift
	case ".kt", ".kts":
		return constants.LanguageKotlin
	case ".scala":
		return constants.LanguageScala
	case ".pl", ".pm":
		return constants.LanguagePerl
	case ".lua":
		return constants.LanguageLua
	case ".sh":
		return constants.LanguageShell
	default:
		return constants.LanguageUnknown
	}
}

// getFileExtension 获取文件扩展名
func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0 && !isPathSeparator(filename[i]); i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

// isPathSeparator 检查字符是否为路径分隔符
func isPathSeparator(c byte) bool {
	return c == '/' || c == '\\'
}

func GetCodeFileName(lang string) string {
	switch lang {
	case "C":
		return "main.c"
	case "cpp":
		return "main.cpp"
	case "Python":
		return "main.py"
	case "Java":
		return "Main.java"
	case "Go":
		return "main.go"
	case "JavaScript":
		return "main.js"
	case "Rust":
		return "main.rs"
	default:
		return "main.c"
	}
}
