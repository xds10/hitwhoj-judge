package service

import (
	"hitwh-judge/internal/model"
	"testing"
)

func TestUpdateFinalStatus(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		newStatus  string
		wantStatus string
	}{
		{
			name:       "AC -> WA",
			current:    model.StatusAC,
			newStatus:  model.StatusWA,
			wantStatus: model.StatusWA,
		},
		{
			name:       "WA -> AC (保持WA)",
			current:    model.StatusWA,
			newStatus:  model.StatusAC,
			wantStatus: model.StatusWA,
		},
		{
			name:       "WA -> TLE",
			current:    model.StatusWA,
			newStatus:  model.StatusTLE,
			wantStatus: model.StatusTLE,
		},
		{
			name:       "MLE -> TLE (TLE优先级更高)",
			current:    model.StatusMLE,
			newStatus:  model.StatusTLE,
			wantStatus: model.StatusTLE,
		},
		{
			name:       "TLE -> MLE (保持TLE)",
			current:    model.StatusTLE,
			newStatus:  model.StatusMLE,
			wantStatus: model.StatusTLE,
		},
		{
			name:       "MLE -> RE",
			current:    model.StatusMLE,
			newStatus:  model.StatusRE,
			wantStatus: model.StatusRE,
		},
		{
			name:       "RE -> CE",
			current:    model.StatusRE,
			newStatus:  model.StatusCE,
			wantStatus: model.StatusCE,
		},
		{
			name:       "CE -> SE",
			current:    model.StatusCE,
			newStatus:  model.StatusSE,
			wantStatus: model.StatusSE,
		},
		{
			name:       "SE -> AC (保持SE)",
			current:    model.StatusSE,
			newStatus:  model.StatusAC,
			wantStatus: model.StatusSE,
		},
		{
			name:       "AC -> AC",
			current:    model.StatusAC,
			newStatus:  model.StatusAC,
			wantStatus: model.StatusAC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateFinalStatus(tt.current, tt.newStatus)
			if got != tt.wantStatus {
				t.Errorf("updateFinalStatus(%q, %q) = %q, want %q",
					tt.current, tt.newStatus, got, tt.wantStatus)
			}
		})
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name    string
		results []model.TestCaseResult
		want    int
	}{
		{
			name: "全部AC",
			results: []model.TestCaseResult{
				{Status: model.StatusAC},
				{Status: model.StatusAC},
				{Status: model.StatusAC},
			},
			want: 100,
		},
		{
			name: "部分AC",
			results: []model.TestCaseResult{
				{Status: model.StatusAC},
				{Status: model.StatusWA},
				{Status: model.StatusAC},
				{Status: model.StatusTLE},
			},
			want: 50,
		},
		{
			name: "全部WA",
			results: []model.TestCaseResult{
				{Status: model.StatusWA},
				{Status: model.StatusWA},
			},
			want: 0,
		},
		{
			name:    "空结果",
			results: []model.TestCaseResult{},
			want:    0,
		},
		{
			name: "1个AC，3个WA",
			results: []model.TestCaseResult{
				{Status: model.StatusAC},
				{Status: model.StatusWA},
				{Status: model.StatusWA},
				{Status: model.StatusWA},
			},
			want: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateScore(tt.results)
			if got != tt.want {
				t.Errorf("calculateScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountACCases(t *testing.T) {
	tests := []struct {
		name    string
		results []model.TestCaseResult
		want    int
	}{
		{
			name: "全部AC",
			results: []model.TestCaseResult{
				{Status: model.StatusAC},
				{Status: model.StatusAC},
				{Status: model.StatusAC},
			},
			want: 3,
		},
		{
			name: "部分AC",
			results: []model.TestCaseResult{
				{Status: model.StatusAC},
				{Status: model.StatusWA},
				{Status: model.StatusAC},
			},
			want: 2,
		},
		{
			name: "无AC",
			results: []model.TestCaseResult{
				{Status: model.StatusWA},
				{Status: model.StatusTLE},
			},
			want: 0,
		},
		{
			name:    "空结果",
			results: []model.TestCaseResult{},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countACCases(tt.results)
			if got != tt.want {
				t.Errorf("countACCases() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "短字符串",
			input:  "Hello",
			maxLen: 10,
			want:   "Hello",
		},
		{
			name:   "刚好等于长度",
			input:  "Hello",
			maxLen: 5,
			want:   "Hello",
		},
		{
			name:   "需要截断",
			input:  "Hello World",
			maxLen: 5,
			want:   "Hello...",
		},
		{
			name:   "空字符串",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// 基准测试
func BenchmarkUpdateFinalStatus(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updateFinalStatus(model.StatusAC, model.StatusWA)
	}
}

func BenchmarkCalculateScore(b *testing.B) {
	results := []model.TestCaseResult{
		{Status: model.StatusAC},
		{Status: model.StatusWA},
		{Status: model.StatusAC},
		{Status: model.StatusTLE},
		{Status: model.StatusAC},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateScore(results)
	}
}
