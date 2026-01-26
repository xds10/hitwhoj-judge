package result

import "strings"

type Comparator struct {
	strict bool
}

func NewComparator(strict bool) *Comparator {
	return &Comparator{
		strict: strict,
	}
}

// Compare 比较程序输出和标准输出
func (c *Comparator) Compare(programOutput, expectedOutput string) bool {
	if c.strict {
		return normalizeString(programOutput) == normalizeString(expectedOutput)
	}
	// 模糊比较（忽略多余空格和换行）
	progOut := strings.Fields(normalizeString(programOutput))
	expOut := strings.Fields(normalizeString(expectedOutput))
	if len(progOut) != len(expOut) {
		return false
	}
	for i := range progOut {
		if progOut[i] != expOut[i] {
			return false
		}
	}
	return true
}

// normalizeString 清理字符串
func normalizeString(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}
