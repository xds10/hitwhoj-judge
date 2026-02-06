package service

import (
	"testing"
	"time"
)

func TestJudgeMetrics_RecordSubmission(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 记录提交
	metrics.RecordSubmission()
	metrics.RecordSubmission()
	metrics.RecordSubmission()

	if metrics.TotalSubmissions != 3 {
		t.Errorf("TotalSubmissions = %d, want 3", metrics.TotalSubmissions)
	}
}

func TestJudgeMetrics_RecordSuccess(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 记录不同状态的成功评测
	metrics.RecordSuccess(100*time.Millisecond, "AC")
	metrics.RecordSuccess(200*time.Millisecond, "WA")
	metrics.RecordSuccess(150*time.Millisecond, "TLE")

	if metrics.SuccessSubmissions != 3 {
		t.Errorf("SuccessSubmissions = %d, want 3", metrics.SuccessSubmissions)
	}

	if metrics.ACCount != 1 {
		t.Errorf("ACCount = %d, want 1", metrics.ACCount)
	}

	if metrics.WACount != 1 {
		t.Errorf("WACount = %d, want 1", metrics.WACount)
	}

	if metrics.TLECount != 1 {
		t.Errorf("TLECount = %d, want 1", metrics.TLECount)
	}

	// 检查时间统计
	if metrics.MaxJudgeTime != 200 {
		t.Errorf("MaxJudgeTime = %d, want 200", metrics.MaxJudgeTime)
	}

	if metrics.MinJudgeTime != 100 {
		t.Errorf("MinJudgeTime = %d, want 100", metrics.MinJudgeTime)
	}

	if metrics.TotalJudgeTime != 450 {
		t.Errorf("TotalJudgeTime = %d, want 450", metrics.TotalJudgeTime)
	}
}

func TestJudgeMetrics_RecordFailure(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	metrics.RecordFailure()
	metrics.RecordFailure()

	if metrics.FailedSubmissions != 2 {
		t.Errorf("FailedSubmissions = %d, want 2", metrics.FailedSubmissions)
	}
}

func TestJudgeMetrics_RecordActive(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 增加活跃数
	current := metrics.RecordActiveIncrease()
	if current != 1 {
		t.Errorf("CurrentActive = %d, want 1", current)
	}

	current = metrics.RecordActiveIncrease()
	if current != 2 {
		t.Errorf("CurrentActive = %d, want 2", current)
	}

	if metrics.MaxConcurrent != 2 {
		t.Errorf("MaxConcurrent = %d, want 2", metrics.MaxConcurrent)
	}

	// 减少活跃数
	metrics.RecordActiveDecrease()
	if metrics.CurrentActive != 1 {
		t.Errorf("CurrentActive = %d, want 1", metrics.CurrentActive)
	}

	metrics.RecordActiveDecrease()
	if metrics.CurrentActive != 0 {
		t.Errorf("CurrentActive = %d, want 0", metrics.CurrentActive)
	}
}

func TestJudgeMetrics_RecordCache(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 记录缓存命中
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()

	// 记录缓存未命中
	metrics.RecordCacheMiss()

	if metrics.CacheHits != 3 {
		t.Errorf("CacheHits = %d, want 3", metrics.CacheHits)
	}

	if metrics.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1", metrics.CacheMisses)
	}
}

func TestJudgeMetrics_GetSnapshot(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 记录一些数据
	metrics.RecordSubmission()
	metrics.RecordSuccess(100*time.Millisecond, "AC")
	metrics.RecordSuccess(200*time.Millisecond, "WA")
	metrics.RecordFailure()
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheMiss()

	snapshot := metrics.GetSnapshot()

	// 验证快照数据
	if snapshot["total_submissions"].(int64) != 1 {
		t.Errorf("total_submissions = %v, want 1", snapshot["total_submissions"])
	}

	if snapshot["success_submissions"].(int64) != 2 {
		t.Errorf("success_submissions = %v, want 2", snapshot["success_submissions"])
	}

	if snapshot["failed_submissions"].(int64) != 1 {
		t.Errorf("failed_submissions = %v, want 1", snapshot["failed_submissions"])
	}

	if snapshot["ac_count"].(int64) != 1 {
		t.Errorf("ac_count = %v, want 1", snapshot["ac_count"])
	}

	if snapshot["wa_count"].(int64) != 1 {
		t.Errorf("wa_count = %v, want 1", snapshot["wa_count"])
	}

	// 平均时间应该是 (100 + 200) / 2 = 150
	if snapshot["avg_judge_time_ms"].(int64) != 150 {
		t.Errorf("avg_judge_time_ms = %v, want 150", snapshot["avg_judge_time_ms"])
	}

	// 缓存命中率应该是 2 / 3 * 100 = 66.67
	cacheHitRate := snapshot["cache_hit_rate"].(float64)
	if cacheHitRate < 66.6 || cacheHitRate > 66.7 {
		t.Errorf("cache_hit_rate = %v, want ~66.67", cacheHitRate)
	}
}

func TestJudgeMetrics_Reset(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 记录一些数据
	metrics.RecordSubmission()
	metrics.RecordSuccess(100*time.Millisecond, "AC")
	metrics.RecordFailure()

	// 重置
	metrics.Reset()

	// 验证所有计数器都被重置
	if metrics.TotalSubmissions != 0 {
		t.Errorf("TotalSubmissions = %d, want 0", metrics.TotalSubmissions)
	}

	if metrics.SuccessSubmissions != 0 {
		t.Errorf("SuccessSubmissions = %d, want 0", metrics.SuccessSubmissions)
	}

	if metrics.FailedSubmissions != 0 {
		t.Errorf("FailedSubmissions = %d, want 0", metrics.FailedSubmissions)
	}

	if metrics.ACCount != 0 {
		t.Errorf("ACCount = %d, want 0", metrics.ACCount)
	}
}

// 并发测试
func TestJudgeMetrics_Concurrent(t *testing.T) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 并发记录提交
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				metrics.RecordSubmission()
				metrics.RecordSuccess(100*time.Millisecond, "AC")
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证计数正确
	if metrics.TotalSubmissions != 1000 {
		t.Errorf("TotalSubmissions = %d, want 1000", metrics.TotalSubmissions)
	}

	if metrics.SuccessSubmissions != 1000 {
		t.Errorf("SuccessSubmissions = %d, want 1000", metrics.SuccessSubmissions)
	}

	if metrics.ACCount != 1000 {
		t.Errorf("ACCount = %d, want 1000", metrics.ACCount)
	}
}

// 基准测试
func BenchmarkJudgeMetrics_RecordSubmission(b *testing.B) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordSubmission()
	}
}

func BenchmarkJudgeMetrics_RecordSuccess(b *testing.B) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordSuccess(100*time.Millisecond, "AC")
	}
}

func BenchmarkJudgeMetrics_GetSnapshot(b *testing.B) {
	metrics := &JudgeMetrics{
		StartTime:    time.Now(),
		MinJudgeTime: int64(^uint64(0) >> 1),
	}

	// 预先记录一些数据
	for i := 0; i < 100; i++ {
		metrics.RecordSuccess(100*time.Millisecond, "AC")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.GetSnapshot()
	}
}

