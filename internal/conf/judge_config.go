package conf

import (
	"time"

	"github.com/spf13/viper"
)

// JudgeConfig 评测配置
type JudgeConfig struct {
	MaxConcurrent      int           // 最大并发评测数
	MaxTimeout         time.Duration // 单个评测最大超时时间
	TempDir            string        // 临时目录
	EnableEarlyStop    bool          // 遇到错误是否提前终止
	MaxOutputSize      int64         // 最大输出大小
	EnableCompileCache bool          // 是否启用编译缓存
}

// CacheConfig 缓存配置
type CacheConfig struct {
	TestCaseTTL    time.Duration // 测试用例缓存时间
	MaxDiskUsage   int64         // 最大磁盘使用
	CleanFrequency time.Duration // 清理频率
}

// LoadJudgeConfig 从配置文件加载评测配置
func LoadJudgeConfig(cfg *viper.Viper) *JudgeConfig {
	return &JudgeConfig{
		MaxConcurrent:      cfg.GetInt("judge.max_concurrent"),
		MaxTimeout:         time.Duration(cfg.GetInt("judge.max_timeout")) * time.Second,
		TempDir:            cfg.GetString("judge.temp_dir"),
		EnableEarlyStop:    cfg.GetBool("judge.enable_early_stop"),
		MaxOutputSize:      cfg.GetInt64("judge.max_output_size"),
		EnableCompileCache: cfg.GetBool("judge.enable_compile_cache"),
	}
}

// LoadCacheConfig 从配置文件加载缓存配置
func LoadCacheConfig(cfg *viper.Viper) *CacheConfig {
	return &CacheConfig{
		TestCaseTTL:    time.Duration(cfg.GetInt("cache.test_case_ttl")) * time.Second,
		MaxDiskUsage:   cfg.GetInt64("cache.max_disk_usage"),
		CleanFrequency: time.Duration(cfg.GetInt("cache.clean_frequency")) * time.Second,
	}
}

// GetDefaultJudgeConfig 获取默认评测配置
func GetDefaultJudgeConfig() *JudgeConfig {
	return &JudgeConfig{
		MaxConcurrent:      2,
		MaxTimeout:         5 * time.Minute,
		TempDir:            "",
		EnableEarlyStop:    false,
		MaxOutputSize:      10 * 1024 * 1024, // 10MB
		EnableCompileCache: false,
	}
}

// GetDefaultCacheConfig 获取默认缓存配置
func GetDefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		TestCaseTTL:    30 * time.Minute,
		MaxDiskUsage:   2 * 1024 * 1024 * 1024, // 2GB
		CleanFrequency: 10 * time.Minute,
	}
}
