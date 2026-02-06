package result

import (
	"testing"
)

func TestComparator_Compare_Strict(t *testing.T) {
	comparator := NewComparator(true)

	tests := []struct {
		name           string
		programOutput  string
		expectedOutput string
		want           bool
	}{
		{
			name:           "完全相同",
			programOutput:  "Hello World",
			expectedOutput: "Hello World",
			want:           true,
		},
		{
			name:           "忽略首尾空白",
			programOutput:  "  Hello World  \n",
			expectedOutput: "Hello World",
			want:           true,
		},
		{
			name:           "Windows换行符",
			programOutput:  "Hello\r\nWorld",
			expectedOutput: "Hello\nWorld",
			want:           true,
		},
		{
			name:           "内容不同",
			programOutput:  "Hello World",
			expectedOutput: "Hello Universe",
			want:           false,
		},
		{
			name:           "多余空格（严格模式）",
			programOutput:  "Hello  World",
			expectedOutput: "Hello World",
			want:           false,
		},
		{
			name:           "空字符串",
			programOutput:  "",
			expectedOutput: "",
			want:           true,
		},
		{
			name:           "多行输出",
			programOutput:  "1\n2\n3",
			expectedOutput: "1\n2\n3",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparator.Compare(tt.programOutput, tt.expectedOutput)
			if got != tt.want {
				t.Errorf("Compare() = %v, want %v\nProgram: %q\nExpected: %q",
					got, tt.want, tt.programOutput, tt.expectedOutput)
			}
		})
	}
}

func TestComparator_Compare_Fuzzy(t *testing.T) {
	comparator := NewComparator(false)

	tests := []struct {
		name           string
		programOutput  string
		expectedOutput string
		want           bool
	}{
		{
			name:           "完全相同",
			programOutput:  "Hello World",
			expectedOutput: "Hello World",
			want:           true,
		},
		{
			name:           "多余空格（模糊模式）",
			programOutput:  "Hello  World",
			expectedOutput: "Hello World",
			want:           true,
		},
		{
			name:           "多余换行",
			programOutput:  "Hello\n\nWorld",
			expectedOutput: "Hello World",
			want:           true,
		},
		{
			name:           "数字序列",
			programOutput:  "1 2 3 4 5",
			expectedOutput: "1\n2\n3\n4\n5",
			want:           true,
		},
		{
			name:           "内容不同",
			programOutput:  "1 2 3",
			expectedOutput: "1 2 4",
			want:           false,
		},
		{
			name:           "数量不同",
			programOutput:  "1 2 3",
			expectedOutput: "1 2",
			want:           false,
		},
		{
			name:           "A+B问题",
			programOutput:  "3\n",
			expectedOutput: "3",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparator.Compare(tt.programOutput, tt.expectedOutput)
			if got != tt.want {
				t.Errorf("Compare() = %v, want %v\nProgram: %q\nExpected: %q",
					got, tt.want, tt.programOutput, tt.expectedOutput)
			}
		})
	}
}

func TestNormalizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Windows换行符",
			input: "Hello\r\nWorld\r\n",
			want:  "Hello\nWorld",
		},
		{
			name:  "首尾空白",
			input: "  Hello World  \n\n",
			want:  "Hello World",
		},
		{
			name:  "制表符",
			input: "\tHello\t",
			want:  "Hello",
		},
		{
			name:  "空字符串",
			input: "",
			want:  "",
		},
		{
			name:  "只有空白",
			input: "   \n\n\t  ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeString(tt.input)
			if got != tt.want {
				t.Errorf("normalizeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// 基准测试
func BenchmarkComparator_Compare_Strict(b *testing.B) {
	comparator := NewComparator(true)
	programOutput := "Hello World\nThis is a test\n1 2 3 4 5"
	expectedOutput := "Hello World\nThis is a test\n1 2 3 4 5"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comparator.Compare(programOutput, expectedOutput)
	}
}

func BenchmarkComparator_Compare_Fuzzy(b *testing.B) {
	comparator := NewComparator(false)
	programOutput := "Hello World\nThis is a test\n1 2 3 4 5"
	expectedOutput := "Hello World\nThis is a test\n1 2 3 4 5"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comparator.Compare(programOutput, expectedOutput)
	}
}

