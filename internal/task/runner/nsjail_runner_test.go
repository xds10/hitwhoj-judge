package runner

import (
	"hitwh-judge/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestNsJailRunner_BasicExecution 测试基本的程序执行
func TestNsJailRunner_BasicExecution(t *testing.T) {
	// 检查nsjail是否可用
	if _, err := exec.LookPath("nsjail"); err != nil {
		t.Skip("nsjail not found, skipping test")
	}

	runner := &NsJailRunner{
		NsJailPath: "nsjail",
	}

	// 创建一个简单的C程序
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "hello.c")
	exeFile := filepath.Join(tempDir, "hello")

	code := `#include <stdio.h>
int main() {
    printf("Hello, World!\n");
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// 编译程序
	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	// 运行测试
	result := runner.RunInSandbox(model.RunParams{
		TestCaseIndex: 0,
		ExePath:       exeFile,
		Input:         "",
		TimeLimit:     1,
		MemLimit:      64,
	})

	if result.Status != model.StatusAC {
		t.Errorf("Expected status AC, got %s, error: %s", result.Status, result.Error)
	}

	if result.Output != "Hello, World!" {
		t.Errorf("Expected output 'Hello, World!', got '%s'", result.Output)
	}

	t.Logf("CPU Time: %v", result.TimeUsed)
	t.Logf("Memory: %d bytes (%.2f MB)", result.MemUsed, float64(result.MemUsed)/(1024*1024))
}

// TestNsJailRunner_InputOutput 测试输入输出
func TestNsJailRunner_InputOutput(t *testing.T) {
	if _, err := exec.LookPath("nsjail"); err != nil {
		t.Skip("nsjail not found, skipping test")
	}

	runner := &NsJailRunner{
		NsJailPath: "nsjail",
	}

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "add.c")
	exeFile := filepath.Join(tempDir, "add")

	code := `#include <stdio.h>
int main() {
    int a, b;
    scanf("%d %d", &a, &b);
    printf("%d\n", a + b);
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// 测试用例
	testCases := []struct {
		input    string
		expected string
	}{
		{"1 2", "3"},
		{"10 20", "30"},
		{"100 200", "300"},
	}

	for i, tc := range testCases {
		result := runner.RunInSandbox(model.RunParams{
			TestCaseIndex: i,
			ExePath:       exeFile,
			Input:         tc.input,
			TimeLimit:     1,
			MemLimit:      64,
		})

		if result.Status != model.StatusAC {
			t.Errorf("Test case %d: Expected status AC, got %s", i, result.Status)
		}

		if result.Output != tc.expected {
			t.Errorf("Test case %d: Expected output '%s', got '%s'", i, tc.expected, result.Output)
		}
	}
}

// TestNsJailRunner_ResourceMonitoring 测试资源监控
func TestNsJailRunner_ResourceMonitoring(t *testing.T) {
	if _, err := exec.LookPath("nsjail"); err != nil {
		t.Skip("nsjail not found, skipping test")
	}

	runner := &NsJailRunner{
		NsJailPath: "nsjail",
	}

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "compute.c")
	exeFile := filepath.Join(tempDir, "compute")

	// 创建一个消耗一定资源的程序
	code := `#include <stdio.h>
#include <stdlib.h>
int main() {
    int sum = 0;
    for (int i = 0; i < 100000000; i++) {
        sum += i;
    }
    printf("%d\n", sum);
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	result := runner.RunInSandbox(model.RunParams{
		TestCaseIndex: 0,
		ExePath:       exeFile,
		Input:         "",
		TimeLimit:     2,
		MemLimit:      64,
	})

	if result.Status != model.StatusAC {
		t.Errorf("Expected status AC, got %s", result.Status)
	}

	// 验证资源监控数据
	if result.TimeUsed == 0 {
		t.Error("CPU time should be greater than 0")
	}

	if result.MemUsed == 0 {
		t.Error("Memory usage should be greater than 0")
	}

	t.Logf("CPU Time: %v", result.TimeUsed)
	t.Logf("Memory: %d bytes (%.2f MB)", result.MemUsed, float64(result.MemUsed)/(1024*1024))
	t.Logf("Output: %s", result.Output)
}

// TestNsJailRunner_Async 测试异步执行
func TestNsJailRunner_Async(t *testing.T) {
	if _, err := exec.LookPath("nsjail"); err != nil {
		t.Skip("nsjail not found, skipping test")
	}

	runner := &NsJailRunner{
		NsJailPath: "nsjail",
	}

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "sleep.c")
	exeFile := filepath.Join(tempDir, "sleep")

	code := `#include <stdio.h>
#include <unistd.h>
int main() {
    printf("Start\n");
    fflush(stdout);
    sleep(1);
    printf("End\n");
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// 异步运行
	pid, resultChan, err := runner.RunInSandboxAsync(exeFile, "")
	if err != nil {
		t.Fatalf("Failed to start async execution: %v", err)
	}

	t.Logf("Started process with PID: %d", pid)

	// 等待结果
	select {
	case result := <-resultChan:
		t.Logf("Output: %s", result.Output)
		t.Logf("Status: %s", result.Status)
		if result.Status != string(model.StatusAC) {
			t.Errorf("Expected AC status, got %s", result.Status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for async result")
	}
}

// TestNormalizeString 测试字符串规范化
func TestNormalizeString(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello\r\n", "hello"},
		{"  hello  ", "hello"},
		{"hello\nworld\n", "hello\nworld"},
		{"\r\n\r\n", ""},
	}

	for _, tc := range testCases {
		result := normalizeString(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeString(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
