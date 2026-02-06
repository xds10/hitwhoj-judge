package language

import (
	"testing"
)

func TestGetCodeFileName(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want string
	}{
		{
			name: "C语言",
			lang: "C",
			want: "main.c",
		},
		{
			name: "C++语言",
			lang: "Cpp",
			want: "main.cpp",
		},
		{
			name: "未知语言默认为C",
			lang: "Unknown",
			want: "main.c",
		},
		{
			name: "空字符串默认为C",
			lang: "",
			want: "main.c",
		},
		{
			name: "小写c",
			lang: "c",
			want: "main.c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCodeFileName(tt.lang)
			if got != tt.want {
				t.Errorf("GetCodeFileName(%q) = %q, want %q", tt.lang, got, tt.want)
			}
		})
	}
}

