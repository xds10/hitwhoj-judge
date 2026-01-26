package conf

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Options struct {
	NsJailPath string
	GCPath     string
	GoPath     string
	JavaPath   string
}

var DefaultOptions = &Options{
	NsJailPath: "nsjail",

	GCPath:   "gcc",
	GoPath:   "go",
	JavaPath: "java",
}

// Load 加载配置文件（新增：支持.env + ${}占位符解析）
// confPath: 配置文件路径（如 config.yaml/config.toml）
func Load(confPath string) *viper.Viper {
	conf := viper.New()

	// ========== 新增1：加载.env文件到系统环境变量 ==========
	if err := godotenv.Load(); err != nil {
		fmt.Println("警告：未找到.env文件，将使用系统环境变量/配置文件默认值")
	}

	// ========== 新增2：读取配置文件原始内容并替换占位符 ==========
	// 读取原始配置文件内容
	confContent, err := os.ReadFile(confPath)
	if err != nil {
		panic(fmt.Sprintf("读取配置文件失败: %v", err))
	}
	// 替换${VAR:-默认值}占位符为环境变量值
	processedContent := replaceEnvPlaceholders(confContent)

	// ========== 原有逻辑改造：解析处理后的配置内容 ==========
	// 从处理后的内容读取配置（替代原有的conf.SetConfigFile + ReadInConfig）
	conf.SetConfigType(getConfigType(confPath)) // 自动识别配置文件类型（yaml/toml/json等）
	if err := conf.ReadConfig(bytes.NewBuffer(processedContent)); err != nil {
		panic(fmt.Sprintf("解析配置文件失败: %v", err))
	}

	// ========== 可选：将配置值绑定到DefaultOptions（保持原有使用习惯） ==========
	bindOptions(conf)

	return conf
}

// replaceEnvPlaceholders 替换配置内容中的${VAR:-默认值}占位符
func replaceEnvPlaceholders(content []byte) []byte {
	// 正则匹配 ${变量名:-默认值} 或 ${变量名}
	re := regexp.MustCompile(`\$\{([^}:-]+)(:-([^}]+))?}`)
	return re.ReplaceAllFunc(content, func(match []byte) []byte {
		groups := re.FindSubmatch(match)
		varName := string(groups[1]) // 环境变量名
		defaultVal := ""             // 默认值
		if len(groups) >= 4 && len(groups[3]) > 0 {
			defaultVal = string(groups[3])
		}

		// 优先读环境变量，无则用默认值
		val := os.Getenv(varName)
		if val == "" {
			val = defaultVal
		}
		return []byte(val)
	})
}

// getConfigType 从配置文件路径识别类型（支持yaml/toml/json/ini等）
func getConfigType(confPath string) string {
	ext := ""
	// 截取文件后缀
	for i := len(confPath) - 1; i >= 0; i-- {
		if confPath[i] == '.' {
			ext = confPath[i+1:]
			break
		}
	}
	// 兼容常见后缀
	switch ext {
	case "yml":
		return "yaml"
	case "toml", "json", "ini", "properties":
		return ext
	default:
		return "yaml" // 默认按yaml解析
	}
}

// bindOptions 将配置值绑定到DefaultOptions（保持原有使用习惯）
func bindOptions(v *viper.Viper) {
	// 读取配置值（环境变量 > 配置文件 > 默认值）
	if v.GetString("ns_jail_path") != "" {
		DefaultOptions.NsJailPath = v.GetString("ns_jail_path")
	}
	if v.GetString("gc_path") != "" {
		DefaultOptions.GCPath = v.GetString("gc_path")
	}
	if v.GetString("go_path") != "" {
		DefaultOptions.GoPath = v.GetString("go_path")
	}
	if v.GetString("java_path") != "" {
		DefaultOptions.JavaPath = v.GetString("java_path")
	}
}
