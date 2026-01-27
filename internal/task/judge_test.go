package task

import (
	"fmt"
	"testing"

	"go.uber.org/zap"
)

func TestJudge(t *testing.T) {
	logger := zap.NewExample()
	defer logger.Sync()

	// 替换全局logger
	zap.ReplaceGlobals(logger)

	// 测试示例
	testCode := `
#include <stdio.h>
int main() {
    int a, b;
    scanf("%d%d", &a, &b);
    printf("%d\n", a + b);
    return 0;
}
`
	testInput := "1 2"
	testExpectedOutput := "3"
	fmt.Printf("测试代码: %s\n", testCode)
	fmt.Printf("测试输入: %s\n", testInput)
	fmt.Printf("期望输出: %s\n", testExpectedOutput)

	// 执行评测
	result, err := Judge(testCode, testInput, testExpectedOutput)
	if err != nil {
		fmt.Printf("评测过程出错: %v\n", err)
		return
	}

	// 输出评测结果
	fmt.Printf("评测状态: %s\n", result.Status)
	if result.Error != "" {
		fmt.Printf("错误信息: %s\n", result.Error)
	}
}
