package file_util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile 完整的文件复制函数
// 参数:
//
//	src: 源文件路径
//	dst: 目标文件路径
//	options: 可选参数
//	  - preservePerm: 是否保留权限 (默认true)
//	  - preserveTime: 是否保留修改时间 (默认true)
//	  - bufferSize: 缓冲区大小 (默认32KB)
//	  - overwrite: 是否覆盖已存在的文件 (默认true)
func CopyFile(src, dst string, options ...CopyOption) error {
	// 默认配置
	config := &copyConfig{
		preservePerm: true,
		preserveTime: true,
		bufferSize:   32 * 1024,
		overwrite:    true,
	}

	// 应用选项
	for _, opt := range options {
		opt(config)
	}

	// 验证源文件
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("源文件错误: %v", err)
	}

	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("%s 不是常规文件", src)
	}

	// 检查目标文件
	if _, err := os.Stat(dst); err == nil {
		if !config.overwrite {
			return fmt.Errorf("目标文件已存在: %s", dst)
		}
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	var dstFilePerm os.FileMode = 0666
	if config.preservePerm {
		dstFilePerm = srcInfo.Mode()
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, dstFilePerm)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer func() {
		dstFile.Close()
		if err != nil {
			os.Remove(dst) // 如果出错，清理目标文件
		}
	}()

	// 复制内容
	if config.bufferSize > 0 {
		buf := make([]byte, config.bufferSize)
		_, err = io.CopyBuffer(dstFile, srcFile, buf)
	} else {
		_, err = io.Copy(dstFile, srcFile)
	}

	if err != nil {
		return fmt.Errorf("复制内容失败: %v", err)
	}

	// 同步到磁盘
	if err = dstFile.Sync(); err != nil {
		return fmt.Errorf("同步到磁盘失败: %v", err)
	}

	// 保留修改时间
	if config.preserveTime {
		if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
			return fmt.Errorf("设置文件时间失败: %v", err)
		}
	}

	return nil
}

// 配置结构
type copyConfig struct {
	preservePerm bool
	preserveTime bool
	bufferSize   int
	overwrite    bool
}

// CopyOption 配置选项
type CopyOption func(*copyConfig)

func WithPreservePerm(preserve bool) CopyOption {
	return func(c *copyConfig) {
		c.preservePerm = preserve
	}
}

func WithPreserveTime(preserve bool) CopyOption {
	return func(c *copyConfig) {
		c.preserveTime = preserve
	}
}

func WithBufferSize(size int) CopyOption {
	return func(c *copyConfig) {
		c.bufferSize = size
	}
}

func WithOverwrite(overwrite bool) CopyOption {
	return func(c *copyConfig) {
		c.overwrite = overwrite
	}
}
