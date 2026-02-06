package conf

import (
	"fmt"
	"hitwh-judge/internal/constants"

	"github.com/spf13/viper"
)

// ValidateConfig 验证配置文件
func ValidateConfig(cfg *viper.Viper) error {
	// 验证服务器配置
	if err := validateServerConfig(cfg); err != nil {
		return fmt.Errorf("服务器配置错误: %w", err)
	}

	// 验证评测机配置
	if err := validateJudgeConfig(cfg); err != nil {
		return fmt.Errorf("评测机配置错误: %w", err)
	}

	// 验证缓存配置
	if err := validateCacheConfig(cfg); err != nil {
		return fmt.Errorf("缓存配置错误: %w", err)
	}

	return nil
}

// validateServerConfig 验证服务器配置
func validateServerConfig(cfg *viper.Viper) error {
	port := cfg.GetInt("server.port")
	if port <= 0 || port > 65535 {
		return fmt.Errorf("端口号无效: %d (应在1-65535之间)", port)
	}

	mode := cfg.GetString("server.mode")
	if mode != "dev" && mode != "prod" && mode != "test" {
		return fmt.Errorf("运行模式无效: %s (应为dev/prod/test)", mode)
	}

	return nil
}

// validateJudgeConfig 验证评测机配置
func validateJudgeConfig(cfg *viper.Viper) error {
	maxConcurrent := cfg.GetInt("judge.max_concurrent")
	if maxConcurrent < constants.MinConcurrent || maxConcurrent > constants.MaxConcurrent {
		return fmt.Errorf("最大并发数无效: %d (应在%d-%d之间)",
			maxConcurrent, constants.MinConcurrent, constants.MaxConcurrent)
	}

	maxTimeout := cfg.GetInt("judge.max_timeout")
	if maxTimeout <= 0 || maxTimeout > 3600 {
		return fmt.Errorf("最大超时时间无效: %d (应在1-3600秒之间)", maxTimeout)
	}

	maxOutputSize := cfg.GetInt64("judge.max_output_size")
	if maxOutputSize <= 0 || maxOutputSize > 100*1024*1024 {
		return fmt.Errorf("最大输出大小无效: %d (应在1B-100MB之间)", maxOutputSize)
	}

	return nil
}

// validateCacheConfig 验证缓存配置
func validateCacheConfig(cfg *viper.Viper) error {
	ttl := cfg.GetInt("cache.test_case_ttl")
	if ttl <= 0 || ttl > 86400 {
		return fmt.Errorf("缓存TTL无效: %d (应在1-86400秒之间)", ttl)
	}

	maxDiskUsage := cfg.GetInt64("cache.max_disk_usage")
	if maxDiskUsage <= 0 || maxDiskUsage > 100*1024*1024*1024 {
		return fmt.Errorf("最大磁盘使用无效: %d (应在1B-100GB之间)", maxDiskUsage)
	}

	cleanFreq := cfg.GetInt("cache.clean_frequency")
	if cleanFreq <= 0 || cleanFreq > 3600 {
		return fmt.Errorf("清理频率无效: %d (应在1-3600秒之间)", cleanFreq)
	}

	return nil
}

// SetDefaultValues 设置默认配置值
func SetDefaultValues(cfg *viper.Viper) {
	// 服务器默认值
	cfg.SetDefault("server.port", constants.DefaultServerPort)
	cfg.SetDefault("server.mode", "dev")
	cfg.SetDefault("server.name", "hitwhoj-judge")

	// 评测机默认值
	cfg.SetDefault("judge.max_concurrent", constants.DefaultMaxConcurrent)
	cfg.SetDefault("judge.max_timeout", int(constants.MaxJudgeTimeout.Seconds()))
	cfg.SetDefault("judge.temp_dir", "")
	cfg.SetDefault("judge.enable_early_stop", false)
	cfg.SetDefault("judge.max_output_size", constants.MaxOutputSize)
	cfg.SetDefault("judge.enable_compile_cache", false)

	// 缓存默认值
	cfg.SetDefault("cache.test_case_ttl", int(constants.DefaultCacheTTL.Seconds()))
	cfg.SetDefault("cache.max_disk_usage", constants.DefaultMaxDiskUsage)
	cfg.SetDefault("cache.clean_frequency", int(constants.DefaultCleanFrequency.Seconds()))

	// 日志默认值
	cfg.SetDefault("log.level", constants.LogLevelInfo)
	cfg.SetDefault("log.filename", constants.DefaultLogFile)
	cfg.SetDefault("log.max_size", constants.DefaultLogMaxSize)
	cfg.SetDefault("log.max_age", constants.DefaultLogMaxAge)
	cfg.SetDefault("log.max_backups", constants.DefaultLogBackups)

	// Snowflake默认值
	cfg.SetDefault("snowflake.machine_id", 1)
	cfg.SetDefault("snowflake.start_time", "2025-07-01")
}

// GetJudgeConfig 获取评测机配置
func GetJudgeConfig(cfg *viper.Viper) JudgeConfig {
	return JudgeConfig{
		MaxConcurrent:      cfg.GetInt("judge.max_concurrent"),
		MaxTimeout:         cfg.GetDuration("judge.max_timeout"),
		TempDir:            cfg.GetString("judge.temp_dir"),
		EnableEarlyStop:    cfg.GetBool("judge.enable_early_stop"),
		MaxOutputSize:      cfg.GetInt64("judge.max_output_size"),
		EnableCompileCache: cfg.GetBool("judge.enable_compile_cache"),
	}
}

// GetCacheConfig 获取缓存配置
func GetCacheConfig(cfg *viper.Viper) CacheConfig {
	return CacheConfig{
		TestCaseTTL:    cfg.GetDuration("cache.test_case_ttl"),
		MaxDiskUsage:   cfg.GetInt64("cache.max_disk_usage"),
		CleanFrequency: cfg.GetDuration("cache.clean_frequency"),
	}
}
