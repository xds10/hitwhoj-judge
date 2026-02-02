package cache

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnhancedTestFileCache_BasicOperations(t *testing.T) {
	// 创建测试缓存实例
	cache := &EnhancedTestFileCache{
		cache:        make(map[string]*cachedFile),
		ttl:          5 * time.Second,
		cleanFreq:    10 * time.Second,
		cacheDir:     t.TempDir(),
		maxDiskUsage: 100 * 1024 * 1024, // 100MB
	}

	bucket := "test-bucket"
	md5Hash := fmt.Sprintf("%x", md5.Sum([]byte("hello world test content")))
	content := "hello world test content"

	// 测试Set方法
	err := cache.Set(bucket, md5Hash, content)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 测试GetFilePath方法
	filePath, exists := cache.GetFilePath(bucket, md5Hash)
	if !exists {
		t.Fatal("GetFilePath should return true after Set")
	}

	// 验证文件内容
	actualContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read cached file: %v", err)
	}

	if string(actualContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(actualContent))
	}

	// 验证MD5校验
	if !cache.verifyFileIntegrity(filePath, md5Hash) {
		t.Error("File integrity check failed")
	}

	// 测试GetFileContent方法
	cachedContent, exists := cache.GetFileContent(bucket, md5Hash)
	if !exists {
		t.Fatal("GetFileContent should return true after Set")
	}

	if cachedContent != content {
		t.Errorf("Expected content %q, got %q", content, cachedContent)
	}
}

func TestEnhancedTestFileCache_MD5Validation(t *testing.T) {
	cache := &EnhancedTestFileCache{
		cache:        make(map[string]*cachedFile),
		ttl:          5 * time.Second,
		cleanFreq:    10 * time.Second,
		cacheDir:     t.TempDir(),
		maxDiskUsage: 100 * 1024 * 1024,
	}

	bucket := "test-bucket"
	md5Hash := fmt.Sprintf("%x", md5.Sum([]byte("correct content")))
	wrongMD5 := fmt.Sprintf("%x", md5.Sum([]byte("wrong content")))

	// 尝试使用错误的MD5设置内容
	err := cache.Set(bucket, wrongMD5, "correct content")
	if err == nil {
		t.Fatal("Set should fail with MD5 mismatch")
	}

	// 正确的MD5应该成功
	err = cache.Set(bucket, md5Hash, "correct content")
	if err != nil {
		t.Fatalf("Set with correct MD5 should succeed: %v", err)
	}
}

func TestEnhancedTestFileCache_DiskSpaceManagement(t *testing.T) {
	cache := &EnhancedTestFileCache{
		cache:        make(map[string]*cachedFile),
		ttl:          5 * time.Second,
		cleanFreq:    10 * time.Second,
		cacheDir:     t.TempDir(),
		maxDiskUsage: 100, // 100字节限制
	}

	bucket := "test-bucket"

	// 第一个文件应该能存入
	content1 := "small content 1"
	md5Hash1 := fmt.Sprintf("%x", md5.Sum([]byte(content1)))
	err := cache.Set(bucket, md5Hash1, content1)
	if err != nil {
		t.Fatalf("First set should succeed: %v", err)
	}

	// 第二个文件加上第一个文件超过了限制，应该触发清理
	content2 := "larger content 2 that exceeds the limit when combined"
	md5Hash2 := fmt.Sprintf("%x", md5.Sum([]byte(content2)))
	err = cache.Set(bucket, md5Hash2, content2)
	if err != nil {
		t.Logf("Second set failed as expected due to space limit: %v", err)
	}

	// 检查空间使用情况
	stats := cache.GetCacheStats()
	t.Logf("Cache stats: %+v", stats)
}

func TestEnhancedTestFileCache_Expired(t *testing.T) {
	// 创建测试缓存实例，TTL很短用于测试
	cache := &EnhancedTestFileCache{
		cache:        make(map[string]*cachedFile),
		ttl:          100 * time.Millisecond, // 很短的TTL
		cleanFreq:    10 * time.Second,
		cacheDir:     t.TempDir(),
		maxDiskUsage: 100 * 1024 * 1024,
	}

	bucket := "test-bucket"
	content := "hello world test content"
	md5Hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))

	// 设置缓存
	err := cache.Set(bucket, md5Hash, content)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 验证缓存存在
	_, exists := cache.GetFilePath(bucket, md5Hash)
	if !exists {
		t.Fatal("GetFilePath should return true before expiration")
	}

	// 等待缓存过期
	time.Sleep(200 * time.Millisecond)

	// 验证缓存已过期
	_, exists = cache.GetFilePath(bucket, md5Hash)
	if exists {
		t.Fatal("GetFilePath should return false after expiration")
	}

	// 验证文件已被删除
	_, exists = cache.GetFileContent(bucket, md5Hash)
	if exists {
		t.Fatal("GetFileContent should return false after expiration")
	}
}

func TestEnhancedTestFileCache_FileIntegrity(t *testing.T) {
	cache := &EnhancedTestFileCache{
		cache:        make(map[string]*cachedFile),
		ttl:          5 * time.Second,
		cleanFreq:    10 * time.Second,
		cacheDir:     t.TempDir(),
		maxDiskUsage: 100 * 1024 * 1024,
	}

	bucket := "test-bucket"
	content := "original content"
	md5Hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))

	// 设置缓存
	err := cache.Set(bucket, md5Hash, content)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 获取文件路径
	filePath, exists := cache.GetFilePath(bucket, md5Hash)
	if !exists {
		t.Fatal("GetFilePath should return true after Set")
	}

	// 验证完整性
	if !cache.verifyFileIntegrity(filePath, md5Hash) {
		t.Error("File integrity check should pass for original content")
	}

	// 手动修改文件内容
	modifiedContent := "modified content"
	err = os.WriteFile(filePath, []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// 验证完整性检查失败
	if cache.verifyFileIntegrity(filePath, md5Hash) {
		t.Error("File integrity check should fail for modified content")
	}

	// 验证GetFilePath会检测到损坏并删除缓存
	_, exists = cache.GetFilePath(bucket, md5Hash)
	if exists {
		t.Error("GetFilePath should return false after detecting corrupted file")
	}
}

func TestEnhancedTestFileCache_GetCacheStats(t *testing.T) {
	cache := &EnhancedTestFileCache{
		cache:        make(map[string]*cachedFile),
		ttl:          5 * time.Second,
		cleanFreq:    10 * time.Second,
		cacheDir:     t.TempDir(),
		maxDiskUsage: 100 * 1024 * 1024,
	}

	// 初始状态
	stats := cache.GetCacheStats()
	if stats["cache_size"].(int) != 0 {
		t.Errorf("Initial cache size should be 0, got %d", stats["cache_size"])
	}

	// 添加一些内容
	content := "test content for stats"
	md5Hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	err := cache.Set("test-bucket", md5Hash, content)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// 检查统计信息
	stats = cache.GetCacheStats()
	if stats["cache_size"].(int) != 1 {
		t.Errorf("Cache size should be 1, got %d", stats["cache_size"])
	}
	if stats["cache_dir"].(string) != cache.cacheDir {
		t.Errorf("Cache dir mismatch")
	}
	if stats["max_usage"].(int64) != cache.maxDiskUsage {
		t.Errorf("Max usage mismatch")
	}
}

func TestEnhancedTestFileCache_CalculateMD5(t *testing.T) {
	cache := &EnhancedTestFileCache{
		cache:        make(map[string]*cachedFile),
		ttl:          5 * time.Second,
		cleanFreq:    10 * time.Second,
		cacheDir:     t.TempDir(),
		maxDiskUsage: 100 * 1024 * 1024,
	}

	// 创建一个临时文件
	content := "test content for md5 calculation"
	tempFile := filepath.Join(t.TempDir(), "temp.txt")
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// 计算MD5
	calculatedMD5, err := cache.calculateMD5(tempFile)
	if err != nil {
		t.Fatalf("calculateMD5 failed: %v", err)
	}

	// 验证MD5是否正确
	expectedMD5 := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	if calculatedMD5 != expectedMD5 {
		t.Errorf("Expected MD5 %s, got %s", expectedMD5, calculatedMD5)
	}
}
