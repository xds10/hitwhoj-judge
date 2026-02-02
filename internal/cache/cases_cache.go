package cache

import (
	md5Package "crypto/md5"
	"fmt"
	"hitwh-judge/internal/dao/minio"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EnhancedTestFileCache 增强版测试用例文件缓存
type EnhancedTestFileCache struct {
	cache        map[string]*cachedFile
	mutex        sync.RWMutex
	ttl          time.Duration
	cleanFreq    time.Duration
	cacheDir     string // 本地缓存目录
	maxDiskUsage int64  // 最大磁盘使用量（字节）
	currentUsage int64  // 当前磁盘使用量
}

type cachedFile struct {
	filePath   string    // 缓存文件的路径
	expireTime time.Time // 过期时间
	size       int64     // 文件大小
	accessTime time.Time // 最后访问时间
	MD5Hash    string    // 文件的MD5哈希值
}

var (
	enhancedInstance *EnhancedTestFileCache
	enhancedOnce     sync.Once
)

// GetEnhancedTestFileCache 获取增强版单例缓存实例
func GetEnhancedTestFileCache() *EnhancedTestFileCache {
	enhancedOnce.Do(func() {
		// 创建本地缓存目录
		cacheDir := "~/tmp/judge-cache-enhanced"
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			// 如果临时目录不可用，回退到系统临时目录
			cacheDir = filepath.Join(os.TempDir(), "judge-cache-enhanced")
			os.MkdirAll(cacheDir, 0755)
		}

		enhancedInstance = &EnhancedTestFileCache{
			cache:        make(map[string]*cachedFile),
			ttl:          30 * time.Minute, // 默认缓存30分钟
			cleanFreq:    10 * time.Minute, // 每10分钟清理一次过期数据
			cacheDir:     cacheDir,
			maxDiskUsage: 1024 * 1024 * 1024, // 默认最大2GB
		}
		go enhancedInstance.startCleaner()
	})
	return enhancedInstance
}

// SetMaxDiskUsage 设置最大磁盘使用量
func (c *EnhancedTestFileCache) SetMaxDiskUsage(maxBytes int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.maxDiskUsage = maxBytes
}

// calculateMD5 计算文件的MD5哈希值
func (c *EnhancedTestFileCache) calculateMD5(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := md5Package.Sum(content)
	return fmt.Sprintf("%x", hash), nil
}

// verifyFileIntegrity 验证文件完整性
func (c *EnhancedTestFileCache) verifyFileIntegrity(filePath, expectedMD5 string) bool {
	calculatedMD5, err := c.calculateMD5(filePath)
	if err != nil {
		return false
	}
	return calculatedMD5 == expectedMD5
}

// GetFilePath 获取缓存文件路径
func (c *EnhancedTestFileCache) GetFilePath(bucket, md5 string) (string, bool) {
	key := c.generateKey(bucket, md5)

	// 快速路径：先尝试读取，避免不必要的写锁
	c.mutex.Lock()
	defer c.mutex.Unlock()
	cached, exists := c.cache[key]

	if !exists {
		return "", false
	}

	// 检查是否过期
	if time.Now().After(cached.expireTime) {
		if finalCached, exists := c.cache[key]; exists && time.Now().After(finalCached.expireTime) {
			os.Remove(finalCached.filePath)
			c.currentUsage -= finalCached.size
			delete(c.cache, key)
		}
		return "", false
	}

	// 检查文件是否仍然存在
	if _, err := os.Stat(cached.filePath); os.IsNotExist(err) {
		if existing, exists := c.cache[key]; exists {
			if _, statErr := os.Stat(existing.filePath); os.IsNotExist(statErr) {
				c.currentUsage -= existing.size
				delete(c.cache, key)
			}
		}
		return "", false
	}

	// 验证文件完整性
	if !c.verifyFileIntegrity(cached.filePath, cached.MD5Hash) {
		// 文件损坏，清理缓存
		if existing, exists := c.cache[key]; exists && existing.filePath == cached.filePath {
			os.Remove(existing.filePath)
			c.currentUsage -= existing.size
			delete(c.cache, key)
		}
		return "", false
	}

	if existing, exists := c.cache[key]; exists {
		existing.accessTime = time.Now()
	}

	return cached.filePath, true
}

// GetFileContent 获取缓存文件内容
func (c *EnhancedTestFileCache) GetFileContent(bucket, md5 string) (string, bool) {
	filePath, exists := c.GetFilePath(bucket, md5)
	if !exists {
		return "", false
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		// 如果读取失败，从缓存中移除该条目
		c.mutex.Lock()
		if cached, exists := c.cache[c.generateKey(bucket, md5)]; exists && cached.filePath == filePath {
			os.Remove(cached.filePath)
			c.currentUsage -= cached.size
			delete(c.cache, c.generateKey(bucket, md5))
		}
		c.mutex.Unlock()
		return "", false
	}

	return string(content), true
}

// checkAndFreeSpace 检查磁盘空间并在必要时释放空间
func (c *EnhancedTestFileCache) checkAndFreeSpace(newFileSize int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 检查是否会超过最大使用量
	if c.currentUsage+newFileSize > c.maxDiskUsage {
		// 按访问时间排序，删除最久未使用的文件
		files := make([]*cachedFile, 0, len(c.cache))
		for _, file := range c.cache {
			files = append(files, file)
		}

		sort.Slice(files, func(i, j int) bool {
			return files[i].accessTime.Before(files[j].accessTime)
		})

		// 删除最久未使用的文件直到有足够空间
		for _, file := range files {
			if c.currentUsage+newFileSize <= c.maxDiskUsage {
				break
			}

			os.Remove(file.filePath)
			c.currentUsage -= file.size
			delete(c.cache, c.generateKeyFromFilePath(file.filePath))
		}
	}

	// 检查是否仍有足够空间
	if c.currentUsage+newFileSize > c.maxDiskUsage {
		return fmt.Errorf("not enough disk space available")
	}

	return nil
}

// generateKeyFromFilePath 从文件路径生成缓存键
func (c *EnhancedTestFileCache) generateKeyFromFilePath(filePath string) string {
	// 解析文件路径获取bucket和md5信息
	// 这里假设文件名为 bucket_md5 的格式
	baseName := filepath.Base(filePath)
	parts := filepath.SplitList(baseName)
	if len(parts) >= 2 {
		return fmt.Sprintf("%s:%s", parts[0], parts[1])
	}
	return ""
}

// Set 添加文件到缓存
func (c *EnhancedTestFileCache) Set(bucket, md5Hash, content string) error {
	key := c.generateKey(bucket, md5Hash)

	// 计算新文件的MD5
	newMD5 := fmt.Sprintf("%x", md5Package.Sum([]byte(content)))
	if newMD5 != md5Hash {
		return fmt.Errorf("MD5 hash mismatch: expected %s, got %s", md5Hash, newMD5)
	}

	// 创建缓存文件路径
	cacheFileName := fmt.Sprintf("%s_%s", bucket, md5Hash)
	cacheFilePath := filepath.Join(c.cacheDir, cacheFileName)

	// 检查并释放空间
	newFileSize := int64(len(content))
	if err := c.checkAndFreeSpace(newFileSize); err != nil {
		return err
	}

	// 写入文件
	if err := os.WriteFile(cacheFilePath, []byte(content), 0644); err != nil {
		return err
	}

	// 获取文件信息
	fileInfo, err := os.Stat(cacheFilePath)
	if err != nil {
		return err
	}

	cached := &cachedFile{
		filePath:   cacheFilePath,
		expireTime: time.Now().Add(c.ttl),
		size:       fileInfo.Size(),
		accessTime: time.Now(),
		MD5Hash:    md5Hash,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 如果已有缓存，先删除旧文件
	if oldCached, exists := c.cache[key]; exists {
		os.Remove(oldCached.filePath)
		c.currentUsage -= oldCached.size
	}

	c.cache[key] = cached
	c.currentUsage += cached.size

	return nil
}

// generateKey 生成缓存键
func (c *EnhancedTestFileCache) generateKey(bucket, md5 string) string {
	return fmt.Sprintf("%s:%s", bucket, md5)
}

// startCleaner 启动清理协程
func (c *EnhancedTestFileCache) startCleaner() {
	ticker := time.NewTicker(c.cleanFreq)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanExpired()
	}
}

// cleanExpired 清理过期的缓存项
func (c *EnhancedTestFileCache) cleanExpired() {
	now := time.Now()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key, cached := range c.cache {
		if now.After(cached.expireTime) {
			// 删除过期的缓存文件
			os.Remove(cached.filePath)
			c.currentUsage -= cached.size
			delete(c.cache, key)
		}
	}
}

// removeCacheEntry 移除特定的缓存条目
func (c *EnhancedTestFileCache) removeCacheEntry(bucket, md5 string) {
	key := c.generateKey(bucket, md5)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if cached, exists := c.cache[key]; exists {
		os.Remove(cached.filePath)
		c.currentUsage -= cached.size
		delete(c.cache, key)
	}
}

// DownloadFileByMD5WithCache 使用缓存下载文件（返回文件路径）
func (c *EnhancedTestFileCache) DownloadFileByMD5WithCache(bucket, md5 string) (string, error) {
	// 先尝试从缓存获取文件路径
	cachedFilePath, found := c.GetFilePath(bucket, md5)

	zap.L().Info("DownloadFileByMD5WithCache", zap.String("md5", md5), zap.Bool("found", found))
	if found {
		return cachedFilePath, nil
	}
	// 缓存未命中，从MinIO下载
	data, err := minio.DownloadFileByMD5AsString(bucket, md5)
	if err != nil {
		return "", err
	}
	// md5 = fmt.Sprintf("%x", md5Package.Sum([]byte(data)))
	// 将下载的数据存入缓存（以文件形式）
	if err := c.Set(bucket, md5, data); err != nil {
		return "", err
	}

	// 返回新缓存的文件路径
	filePath, ok := c.GetFilePath(bucket, md5)
	if !ok {
		return "", fmt.Errorf("file path not found in cache")
	}
	return filePath, nil
}

// DownloadFileByMD5WithCacheContent 使用缓存下载文件（返回内容）
func (c *EnhancedTestFileCache) DownloadFileByMD5WithCacheContent(bucket, md5 string) (string, error) {
	// 先尝试从缓存获取内容
	if cachedContent, found := c.GetFileContent(bucket, md5); found {
		return cachedContent, nil
	}

	// 缓存未命中，从MinIO下载
	data, err := minio.DownloadFileByMD5AsString(bucket, md5)
	if err != nil {
		return "", err
	}
	// md5 = fmt.Sprintf("%x", md5Package.Sum([]byte(data)))

	// 将下载的数据存入缓存
	if err := c.Set(bucket, md5, data); err != nil {
		return "", err
	}

	return data, nil
}

// Clear 清空所有缓存（可用于调试或特殊场景）
func (c *EnhancedTestFileCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 删除所有缓存文件
	for _, cached := range c.cache {
		os.Remove(cached.filePath)
	}

	c.cache = make(map[string]*cachedFile)
	c.currentUsage = 0
}

// GetCacheStats 获取缓存统计信息
func (c *EnhancedTestFileCache) GetCacheStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"cache_size":    len(c.cache),
		"current_usage": c.currentUsage,
		"max_usage":     c.maxDiskUsage,
		"cache_dir":     c.cacheDir,
		"ttl":           c.ttl,
		"clean_freq":    c.cleanFreq,
		"usage_percent": float64(c.currentUsage) / float64(c.maxDiskUsage) * 100,
	}
}
