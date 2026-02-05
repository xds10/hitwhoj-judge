package service

import (
	"sync"
	"sync/atomic"
	"time"
)

// JudgeMetrics 评测统计指标
type JudgeMetrics struct {
	// 计数器
	TotalSubmissions   int64 // 总提交数
	SuccessSubmissions int64 // 成功评测数
	FailedSubmissions  int64 // 失败评测数

	// 各状态统计
	ACCount  int64 // AC数量
	WACount  int64 // WA数量
	TLECount int64 // TLE数量
	MLECount int64 // MLE数量
	RECount  int64 // RE数量
	CECount  int64 // CE数量
	SECount  int64 // SE数量

	// 性能指标
	TotalJudgeTime int64 // 总评测时间（毫秒）
	MaxJudgeTime   int64 // 最大评测时间（毫秒）
	MinJudgeTime   int64 // 最小评测时间（毫秒）

	// 资源使用
	CurrentActive     int32 // 当前活跃评测数
	MaxConcurrent     int32 // 历史最大并发数
	QueueWaitCount    int64 // 队列等待次数
	QueueTimeoutCount int64 // 队列超时次数

	// 缓存统计
	CacheHits   int64 // 缓存命中次数
	CacheMisses int64 // 缓存未命中次数

	// 时间戳
	StartTime time.Time // 启动时间

	mu sync.RWMutex
}

var globalMetrics = &JudgeMetrics{
	StartTime:    time.Now(),
	MinJudgeTime: int64(^uint64(0) >> 1), // 初始化为最大值
}

// GetGlobalMetrics 获取全局统计实例
func GetGlobalMetrics() *JudgeMetrics {
	return globalMetrics
}

// RecordSubmission 记录提交
func (m *JudgeMetrics) RecordSubmission() {
	atomic.AddInt64(&m.TotalSubmissions, 1)
}

// RecordSuccess 记录成功评测
func (m *JudgeMetrics) RecordSuccess(judgeTime time.Duration, status string) {
	atomic.AddInt64(&m.SuccessSubmissions, 1)

	// 更新状态统计
	switch status {
	case "AC":
		atomic.AddInt64(&m.ACCount, 1)
	case "WA":
		atomic.AddInt64(&m.WACount, 1)
	case "TLE":
		atomic.AddInt64(&m.TLECount, 1)
	case "MLE":
		atomic.AddInt64(&m.MLECount, 1)
	case "RE":
		atomic.AddInt64(&m.RECount, 1)
	case "CE":
		atomic.AddInt64(&m.CECount, 1)
	case "SE":
		atomic.AddInt64(&m.SECount, 1)
	}

	// 更新时间统计
	judgeTimeMs := judgeTime.Milliseconds()
	atomic.AddInt64(&m.TotalJudgeTime, judgeTimeMs)

	// 更新最大时间
	for {
		oldMax := atomic.LoadInt64(&m.MaxJudgeTime)
		if judgeTimeMs <= oldMax {
			break
		}
		if atomic.CompareAndSwapInt64(&m.MaxJudgeTime, oldMax, judgeTimeMs) {
			break
		}
	}

	// 更新最小时间
	for {
		oldMin := atomic.LoadInt64(&m.MinJudgeTime)
		if judgeTimeMs >= oldMin {
			break
		}
		if atomic.CompareAndSwapInt64(&m.MinJudgeTime, oldMin, judgeTimeMs) {
			break
		}
	}
}

// RecordFailure 记录失败评测
func (m *JudgeMetrics) RecordFailure() {
	atomic.AddInt64(&m.FailedSubmissions, 1)
}

// RecordActiveIncrease 记录活跃评测增加
func (m *JudgeMetrics) RecordActiveIncrease() int32 {
	current := atomic.AddInt32(&m.CurrentActive, 1)

	// 更新最大并发数
	for {
		oldMax := atomic.LoadInt32(&m.MaxConcurrent)
		if current <= oldMax {
			break
		}
		if atomic.CompareAndSwapInt32(&m.MaxConcurrent, oldMax, current) {
			break
		}
	}

	return current
}

// RecordActiveDecrease 记录活跃评测减少
func (m *JudgeMetrics) RecordActiveDecrease() {
	atomic.AddInt32(&m.CurrentActive, -1)
}

// RecordQueueWait 记录队列等待
func (m *JudgeMetrics) RecordQueueWait() {
	atomic.AddInt64(&m.QueueWaitCount, 1)
}

// RecordQueueTimeout 记录队列超时
func (m *JudgeMetrics) RecordQueueTimeout() {
	atomic.AddInt64(&m.QueueTimeoutCount, 1)
}

// RecordCacheHit 记录缓存命中
func (m *JudgeMetrics) RecordCacheHit() {
	atomic.AddInt64(&m.CacheHits, 1)
}

// RecordCacheMiss 记录缓存未命中
func (m *JudgeMetrics) RecordCacheMiss() {
	atomic.AddInt64(&m.CacheMisses, 1)
}

// GetSnapshot 获取统计快照
func (m *JudgeMetrics) GetSnapshot() map[string]interface{} {
	totalSubmissions := atomic.LoadInt64(&m.TotalSubmissions)
	successSubmissions := atomic.LoadInt64(&m.SuccessSubmissions)
	totalJudgeTime := atomic.LoadInt64(&m.TotalJudgeTime)

	var avgJudgeTime int64
	if successSubmissions > 0 {
		avgJudgeTime = totalJudgeTime / successSubmissions
	}

	cacheHits := atomic.LoadInt64(&m.CacheHits)
	cacheMisses := atomic.LoadInt64(&m.CacheMisses)
	var cacheHitRate float64
	if cacheHits+cacheMisses > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}

	uptime := time.Since(m.StartTime)

	return map[string]interface{}{
		// 基础统计
		"total_submissions":   totalSubmissions,
		"success_submissions": successSubmissions,
		"failed_submissions":  atomic.LoadInt64(&m.FailedSubmissions),

		// 状态统计
		"ac_count":  atomic.LoadInt64(&m.ACCount),
		"wa_count":  atomic.LoadInt64(&m.WACount),
		"tle_count": atomic.LoadInt64(&m.TLECount),
		"mle_count": atomic.LoadInt64(&m.MLECount),
		"re_count":  atomic.LoadInt64(&m.RECount),
		"ce_count":  atomic.LoadInt64(&m.CECount),
		"se_count":  atomic.LoadInt64(&m.SECount),

		// 性能指标
		"avg_judge_time_ms": avgJudgeTime,
		"max_judge_time_ms": atomic.LoadInt64(&m.MaxJudgeTime),
		"min_judge_time_ms": atomic.LoadInt64(&m.MinJudgeTime),

		// 并发统计
		"current_active":      atomic.LoadInt32(&m.CurrentActive),
		"max_concurrent":      atomic.LoadInt32(&m.MaxConcurrent),
		"queue_wait_count":    atomic.LoadInt64(&m.QueueWaitCount),
		"queue_timeout_count": atomic.LoadInt64(&m.QueueTimeoutCount),

		// 缓存统计
		"cache_hits":     cacheHits,
		"cache_misses":   cacheMisses,
		"cache_hit_rate": cacheHitRate,

		// 运行时间
		"uptime_seconds": uptime.Seconds(),
		"start_time":     m.StartTime.Format(time.RFC3339),
	}
}

// Reset 重置统计（谨慎使用）
func (m *JudgeMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt64(&m.TotalSubmissions, 0)
	atomic.StoreInt64(&m.SuccessSubmissions, 0)
	atomic.StoreInt64(&m.FailedSubmissions, 0)
	atomic.StoreInt64(&m.ACCount, 0)
	atomic.StoreInt64(&m.WACount, 0)
	atomic.StoreInt64(&m.TLECount, 0)
	atomic.StoreInt64(&m.MLECount, 0)
	atomic.StoreInt64(&m.RECount, 0)
	atomic.StoreInt64(&m.CECount, 0)
	atomic.StoreInt64(&m.SECount, 0)
	atomic.StoreInt64(&m.TotalJudgeTime, 0)
	atomic.StoreInt64(&m.MaxJudgeTime, 0)
	atomic.StoreInt64(&m.MinJudgeTime, int64(^uint64(0)>>1))
	atomic.StoreInt32(&m.MaxConcurrent, 0)
	atomic.StoreInt64(&m.QueueWaitCount, 0)
	atomic.StoreInt64(&m.QueueTimeoutCount, 0)
	atomic.StoreInt64(&m.CacheHits, 0)
	atomic.StoreInt64(&m.CacheMisses, 0)
	m.StartTime = time.Now()
}
