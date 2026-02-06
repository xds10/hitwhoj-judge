package runner

import (
	"encoding/json"
	"fmt"
	"hitwh-judge/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSDUSandboxRunner_BasicExecution 测试基本的程序执行
func TestSDUSandboxRunner_BasicExecution(t *testing.T) {
	// 检查sandbox是否可用
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		// 尝试当前目录
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			t.Skip("sandbox not found, skipping test")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup()
	// 创建一个简单的C程序

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
	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-Wall", "-O2", "-static", "-std=c11")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
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

// TestSDUSandboxRunner_InputOutput 测试输入输出
func TestSDUSandboxRunner_InputOutput(t *testing.T) {
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			t.Skip("sandbox not found, skipping test")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup()
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
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
	}

	// 测试用例
	testCases := []struct {
		input    string
		expected string
	}{
		{"1 2", "3"},
		{"10 20", "30"},
		{"100 200", "300"},
		{"0 0", "0"},
		{"-5 5", "0"},
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
			t.Errorf("Test case %d: Expected status AC, got %s, error: %s", i, result.Status, result.Error)
		}

		if result.Output != tc.expected {
			t.Errorf("Test case %d: Expected output '%s', got '%s'", i, tc.expected, result.Output)
		}

		t.Logf("Test case %d: CPU=%v, Mem=%d bytes", i, result.TimeUsed, result.MemUsed)
	}
}

// TestSDUSandboxRunner_TimeLimit 测试时间限制
func TestSDUSandboxRunner_TimeLimit(t *testing.T) {
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			t.Skip("sandbox not found, skipping test")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup()
	sourceFile := filepath.Join(tempDir, "infinite.c")
	exeFile := filepath.Join(tempDir, "infinite")

	// 创建一个死循环程序
	code := `#include <stdio.h>
int main() {
    volatile int sum = 0;
    while(1) {
        sum += 1;
    }
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
	}

	// 运行测试，设置1秒时间限制
	result := runner.RunInSandbox(model.RunParams{
		TestCaseIndex: 0,
		ExePath:       exeFile,
		Input:         "",
		TimeLimit:     1,
		MemLimit:      64,
	})

	if result.Status != model.StatusTLE {
		t.Errorf("Expected status TLE, got %s, error: %s", result.Status, result.Error)
	}

	t.Logf("CPU Time: %v", result.TimeUsed)
	t.Logf("Status: %s", result.Status)
}

// TestSDUSandboxRunner_MemoryLimit 测试内存限制
func TestSDUSandboxRunner_MemoryLimit(t *testing.T) {
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			t.Skip("sandbox not found, skipping test")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup()
	sourceFile := filepath.Join(tempDir, "memory.c")
	exeFile := filepath.Join(tempDir, "memory")

	// 创建一个分配大量内存的程序
	code := `#include <stdlib.h>
#include <string.h>
int main() {
    // 尝试分配100MB内存
    char *p = malloc(100 * 1024 * 1024);
    // if (p) {
        memset(p, 0, 100 * 1024 * 1024);
    // }
	printf("Memory allocated successfully\n");
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
	}

	// 运行测试，设置32MB内存限制
	result := runner.RunInSandbox(model.RunParams{
		TestCaseIndex: 0,
		ExePath:       exeFile,
		Input:         "",
		TimeLimit:     2,
		MemLimit:      32,
	})

	if result.Status != model.StatusMLE && result.Status != model.StatusRE {
		t.Errorf("Expected status MLE or RE, got %s, error: %s", result.Status, result.Error)
	}

	t.Logf("Memory Used: %d bytes (%.2f MB)", result.MemUsed, float64(result.MemUsed)/(1024*1024))
	t.Logf("Status: %s", result.Status)
}

// TestSDUSandboxRunner_RuntimeError 测试运行时错误
func TestSDUSandboxRunner_RuntimeError(t *testing.T) {
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			t.Skip("sandbox not found, skipping test")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup()
	sourceFile := filepath.Join(tempDir, "segfault.c")
	exeFile := filepath.Join(tempDir, "segfault")

	// 创建一个会产生段错误的程序
	code := `#include <stdio.h>
int main() {
    int *p = NULL;
    *p = 42;  // 段错误
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
	}

	result := runner.RunInSandbox(model.RunParams{
		TestCaseIndex: 0,
		ExePath:       exeFile,
		Input:         "",
		TimeLimit:     1,
		MemLimit:      64,
	})

	if result.Status != model.StatusRE {
		t.Errorf("Expected status RE, got %s", result.Status)
	}

	t.Logf("Error: %s", result.Error)
}

// TestSDUSandboxRunner_ResourceMonitoring 测试资源监控
func TestSDUSandboxRunner_ResourceMonitoring(t *testing.T) {
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			t.Skip("sandbox not found, skipping test")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup()
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

	cmd := exec.Command("gcc", "-O2", sourceFile, "-o", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
	}

	result := runner.RunInSandbox(model.RunParams{
		TestCaseIndex: 0,
		ExePath:       exeFile,
		Input:         "",
		TimeLimit:     2,
		MemLimit:      64,
	})

	if result.Status != model.StatusAC {
		t.Errorf("Expected status AC, got %s, error: %s", result.Status, result.Error)
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

// TestSDUSandboxRunner_JSONParsing 测试JSON解析
func TestSDUSandboxRunner_JSONParsing(t *testing.T) {
	testCases := []struct {
		name     string
		jsonStr  string
		expected SandboxResult
		wantErr  bool
	}{
		{
			name:    "Valid JSON - Success",
			jsonStr: `{"cpu_time":100,"real_time":150,"memory":1024000,"signal":0,"exit_code":0,"error":0,"result":0}`,
			expected: SandboxResult{
				CpuTime:  100,
				RealTime: 150,
				Memory:   1024000,
				Signal:   0,
				ExitCode: 0,
				Error:    0,
				Result:   0,
			},
			wantErr: false,
		},
		{
			name:    "Valid JSON - TLE",
			jsonStr: `{"cpu_time":1000,"real_time":1200,"memory":512000,"signal":0,"exit_code":0,"error":0,"result":1}`,
			expected: SandboxResult{
				CpuTime:  1000,
				RealTime: 1200,
				Memory:   512000,
				Signal:   0,
				ExitCode: 0,
				Error:    0,
				Result:   1,
			},
			wantErr: false,
		},
		{
			name:    "Valid JSON - MLE",
			jsonStr: `{"cpu_time":50,"real_time":60,"memory":67108864,"signal":0,"exit_code":0,"error":0,"result":3}`,
			expected: SandboxResult{
				CpuTime:  50,
				RealTime: 60,
				Memory:   67108864,
				Signal:   0,
				ExitCode: 0,
				Error:    0,
				Result:   3,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result SandboxResult
			err := json.Unmarshal([]byte(tc.jsonStr), &result)

			if (err != nil) != tc.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr {
				if result.CpuTime != tc.expected.CpuTime {
					t.Errorf("CpuTime = %d, expected %d", result.CpuTime, tc.expected.CpuTime)
				}
				if result.RealTime != tc.expected.RealTime {
					t.Errorf("RealTime = %d, expected %d", result.RealTime, tc.expected.RealTime)
				}
				if result.Memory != tc.expected.Memory {
					t.Errorf("Memory = %d, expected %d", result.Memory, tc.expected.Memory)
				}
				if result.Result != tc.expected.Result {
					t.Errorf("Result = %d, expected %d", result.Result, tc.expected.Result)
				}
			}
		})
	}
}

// TestSDUSandboxRunner_ResultMapping 测试结果映射
func TestSDUSandboxRunner_ResultMapping(t *testing.T) {
	testCases := []struct {
		resultCode int
		expected   string
	}{
		{0, "AC"},
		{1, "TLE"},
		{2, "TLE"},
		{3, "MLE"},
		{4, "RE"},
		{5, "SE"},
		{6, "OLE"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Result_%d", tc.resultCode), func(t *testing.T) {
			status, exists := resultMapping[tc.resultCode]
			if !exists {
				t.Errorf("Result code %d not found in mapping", tc.resultCode)
				return
			}
			if status != tc.expected {
				t.Errorf("Result code %d mapped to %s, expected %s", tc.resultCode, status, tc.expected)
			}
		})
	}
}

// BenchmarkSDUSandboxRunner_SimpleProgram 基准测试
func BenchmarkSDUSandboxRunner_SimpleProgram(b *testing.B) {
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			b.Skip("sandbox not found, skipping benchmark")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir := b.TempDir()
	sourceFile := filepath.Join(tempDir, "hello.c")
	exeFile := filepath.Join(tempDir, "hello")

	code := `#include <stdio.h>
int main() {
    printf("Hello\n");
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		b.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		b.Fatalf("Failed to compile: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.RunInSandbox(model.RunParams{
			TestCaseIndex: i,
			ExePath:       exeFile,
			Input:         "",
			TimeLimit:     1,
			MemLimit:      64,
		})
	}
}

// TestSDUSandboxRunner_MultipleTestCases 测试多个测试用例
func TestSDUSandboxRunner_MultipleTestCases(t *testing.T) {
	sandboxPath := "sandbox"
	if _, err := exec.LookPath(sandboxPath); err != nil {
		if _, err := os.Stat(sandboxPath); os.IsNotExist(err) {
			t.Skip("sandbox not found, skipping test")
		}
	}

	runner := &SDUSandboxRunner{
		SandboxPath: sandboxPath,
	}

	tempDir, cleanup, err := createTmpDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup()
	sourceFile := filepath.Join(tempDir, "multiply.c")
	exeFile := filepath.Join(tempDir, "multiply")

	code := `#include <stdio.h>
int main() {
    int a, b;
    scanf("%d %d", &a, &b);
    printf("%d\n", a * b);
    return 0;
}`

	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	cmd := exec.Command("gcc", sourceFile, "-o", exeFile, "-static")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
	}
	cmd = exec.Command("chmod", "+x", exeFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set executable permission: %v", err)
	}

	testCases := []struct {
		input    string
		expected string
	}{
		{"2 3", "6"},
		{"5 7", "35"},
		{"10 10", "100"},
		{"0 100", "0"},
		{"-3 4", "-12"},
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
			t.Errorf("Test case %d failed: status=%s, error=%s", i, result.Status, result.Error)
			continue
		}

		if result.Output != tc.expected {
			t.Errorf("Test case %d: expected output '%s', got '%s'", i, tc.expected, result.Output)
		}
	}
}
