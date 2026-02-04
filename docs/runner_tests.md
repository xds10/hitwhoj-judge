# Runner 测试文档

## 📋 测试概述

为 NsJail 和 SDU Sandbox 两个沙箱运行器创建了完整的测试套件，包括：

- ✅ 单元测试
- ✅ 集成测试
- ✅ 对比测试
- ✅ 基准测试
- ✅ 资源监控测试

## 📁 测试文件

```
internal/task/runner/
├── nsjail_runner_test.go      # NsJail 沙箱测试
├── sdu_sandbox_test.go         # SDU Sandbox 测试
└── runner_test.go              # 通用测试和对比测试
```

## 🚀 运行测试

### 运行所有测试

```bash
cd /home/zwp-test/project/hitwhoj-judge
go test ./internal/task/runner/... -v
```

### 运行特定沙箱的测试

```bash
# 只测试 NsJail
go test ./internal/task/runner/... -v -run TestNsJail

# 只测试 SDU Sandbox
go test ./internal/task/runner/... -v -run TestSDU

# 运行对比测试
go test ./internal/task/runner/... -v -run TestBothRunners
```

### 运行基准测试

```bash
# 运行所有基准测试
go test ./internal/task/runner/... -bench=. -benchmem

# 对比两个沙箱的性能
go test ./internal/task/runner/... -bench=BenchmarkBothRunners -benchmem

# 单独测试某个沙箱的性能
go test ./internal/task/runner/... -bench=BenchmarkNsJail -benchmem
go test ./internal/task/runner/... -bench=BenchmarkSDU -benchmem
```

### 运行特定测试

```bash
# 测试基本执行
go test ./internal/task/runner/... -v -run TestNsJailRunner_BasicExecution

# 测试时间限制
go test ./internal/task/runner/... -v -run TimeLimit

# 测试内存限制
go test ./internal/task/runner/... -v -run MemoryLimit

# 测试资源监控
go test ./internal/task/runner/... -v -run ResourceMonitoring
```

## 📊 测试覆盖率

```bash
# 生成覆盖率报告
go test ./internal/task/runner/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 查看覆盖率统计
go test ./internal/task/runner/... -cover
```

## 🧪 测试用例说明

### NsJail 测试 (`nsjail_runner_test.go`)

| 测试函数 | 描述 | 验证内容 |
|---------|------|---------|
| `TestNsJailRunner_BasicExecution` | 基本程序执行 | 输出正确性、状态码 |
| `TestNsJailRunner_InputOutput` | 输入输出测试 | 多组测试用例 |
| `TestNsJailRunner_TimeLimit` | 时间限制测试 | TLE 检测 |
| `TestNsJailRunner_MemoryLimit` | 内存限制测试 | MLE 检测 |
| `TestNsJailRunner_RuntimeError` | 运行时错误 | RE 检测（段错误） |
| `TestNsJailRunner_ResourceMonitoring` | 资源监控 | CPU时间、内存统计 |
| `TestNsJailRunner_NonExistentExecutable` | 错误处理 | 不存在的文件 |
| `TestNsJailRunner_Async` | 异步执行 | 异步运行机制 |
| `TestParseNsJailError` | 错误解析 | 错误信息解析 |
| `TestNormalizeString` | 字符串处理 | 换行符规范化 |
| `BenchmarkNsJailRunner_SimpleProgram` | 性能基准 | 执行效率 |

### SDU Sandbox 测试 (`sdu_sandbox_test.go`)

| 测试函数 | 描述 | 验证内容 |
|---------|------|---------|
| `TestSDUSandboxRunner_BasicExecution` | 基本程序执行 | 输出正确性、状态码 |
| `TestSDUSandboxRunner_InputOutput` | 输入输出测试 | 多组测试用例 |
| `TestSDUSandboxRunner_TimeLimit` | 时间限制测试 | TLE 检测 |
| `TestSDUSandboxRunner_MemoryLimit` | 内存限制测试 | MLE 检测 |
| `TestSDUSandboxRunner_RuntimeError` | 运行时错误 | RE 检测 |
| `TestSDUSandboxRunner_ResourceMonitoring` | 资源监控 | CPU时间、内存统计 |
| `TestSDUSandboxRunner_JSONParsing` | JSON 解析 | 沙箱结果解析 |
| `TestSDUSandboxRunner_ResultMapping` | 结果映射 | 状态码映射 |
| `TestSDUSandboxRunner_Async` | 异步执行 | 异步运行机制 |
| `TestSDUSandboxRunner_MultipleTestCases` | 多测试点 | 批量测试 |
| `BenchmarkSDUSandboxRunner_SimpleProgram` | 性能基准 | 执行效率 |

### 通用测试 (`runner_test.go`)

| 测试函数 | 描述 | 验证内容 |
|---------|------|---------|
| `TestBothRunners_Comparison` | 对比测试 | 两个沙箱结果一致性 |
| `TestBothRunners_ResourceAccuracy` | 资源监控准确性 | 资源统计准确性 |
| `TestRunnerFactory` | 工厂函数测试 | Runner 创建 |
| `TestGetDefaultSandboxConfig` | 配置测试 | 默认配置获取 |
| `BenchmarkBothRunners_Performance` | 性能对比 | 两个沙箱性能对比 |

## 🔧 测试环境要求

### 必需工具

- **GCC/G++**: 用于编译测试程序
  ```bash
  sudo apt install gcc g++
  ```

- **NsJail** (可选，用于 NsJail 测试):
  ```bash
  sudo apt install nsjail
  # 或从源码编译
  ```

- **SDU Sandbox** (可选，用于 SDU Sandbox 测试):
  ```bash
  # 需要从源码编译或下载二进制文件
  # 放在 PATH 中或当前目录
  ```

### 权限要求

- NsJail: 通常不需要 root 权限
- SDU Sandbox: 可能需要 sudo 权限（取决于配置）

## 📝 测试程序集合

测试中使用的标准程序（在 `runner_test.go` 中定义）：

```go
TestPrograms.HelloWorld      // 简单输出
TestPrograms.AddTwoNumbers   // A+B 问题
TestPrograms.InfiniteLoop    // 死循环（测试 TLE）
TestPrograms.MemoryHog       // 大内存分配（测试 MLE）
TestPrograms.SegFault        // 段错误（测试 RE）
TestPrograms.DivideByZero    // 除零错误
TestPrograms.ArraySum        // 数组求和
TestPrograms.Fibonacci       // 斐波那契数列
```

## 🎯 测试示例

### 示例 1: 运行基本测试

```bash
$ go test ./internal/task/runner/... -v -run TestNsJailRunner_BasicExecution

=== RUN   TestNsJailRunner_BasicExecution
    nsjail_runner_test.go:45: CPU Time: 2.5ms
    nsjail_runner_test.go:46: Memory: 2097152 bytes (2.00 MB)
--- PASS: TestNsJailRunner_BasicExecution (0.15s)
PASS
```

### 示例 2: 对比测试

```bash
$ go test ./internal/task/runner/... -v -run TestBothRunners_Comparison

=== RUN   TestBothRunners_Comparison
=== RUN   TestBothRunners_Comparison/Hello_World
    runner_test.go:150: NsJail - Status: AC, CPU: 2.1ms, Mem: 2.00 MB
    runner_test.go:160: SDU Sandbox - Status: AC, CPU: 2.3ms, Mem: 2.10 MB
    runner_test.go:175: Time difference: 200µs
    runner_test.go:181: Memory difference: 0.10 MB
--- PASS: TestBothRunners_Comparison (0.45s)
    --- PASS: TestBothRunners_Comparison/Hello_World (0.15s)
```

### 示例 3: 性能基准测试

```bash
$ go test ./internal/task/runner/... -bench=BenchmarkBothRunners -benchmem

BenchmarkBothRunners_Performance/NsJail-8         	      50	  23456789 ns/op	    1234 B/op	      12 allocs/op
BenchmarkBothRunners_Performance/SDUSandbox-8     	      45	  25678901 ns/op	    1456 B/op	      14 allocs/op
PASS
```

## 🐛 调试测试

### 查看详细输出

```bash
go test ./internal/task/runner/... -v -run TestName
```

### 只运行失败的测试

```bash
go test ./internal/task/runner/... -v -run TestName -count=1
```

### 设置超时时间

```bash
go test ./internal/task/runner/... -timeout 30s
```

### 并行测试

```bash
# 使用 4 个并行进程
go test ./internal/task/runner/... -parallel 4
```

## 📈 持续集成

### GitHub Actions 示例

```yaml
name: Runner Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.24
      
      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y gcc g++ nsjail
      
      - name: Run tests
        run: go test ./internal/task/runner/... -v -cover
      
      - name: Run benchmarks
        run: go test ./internal/task/runner/... -bench=. -benchmem
```

## 🔍 常见问题

### Q: 测试被跳过 (SKIP)

**A:** 检查是否安装了相应的沙箱工具：

```bash
# 检查 nsjail
which nsjail

# 检查 sandbox
which sandbox
ls -la ./sandbox
```

### Q: 编译失败

**A:** 确保安装了 GCC：

```bash
gcc --version
g++ --version
```

### Q: 权限错误

**A:** 某些沙箱可能需要特殊权限：

```bash
# 给 sandbox 添加执行权限
chmod +x ./sandbox

# 或使用 sudo 运行测试（不推荐）
sudo go test ./internal/task/runner/...
```

### Q: 测试超时

**A:** 增加超时时间：

```bash
go test ./internal/task/runner/... -timeout 5m
```

## 📚 扩展测试

### 添加新的测试用例

1. 在相应的 `*_test.go` 文件中添加测试函数
2. 使用 `TestHelper` 辅助工具编译程序
3. 调用 `RunInSandbox` 执行测试
4. 验证结果

示例：

```go
func TestMyNewFeature(t *testing.T) {
    helper := NewTestHelper(t)
    exeFile := helper.CompileC(TestPrograms.HelloWorld, "test")
    
    runner := &NsJailRunner{NsJailPath: "nsjail"}
    result := runner.RunInSandbox(model.RunParams{
        TestCaseIndex: 0,
        ExePath:       exeFile,
        Input:         "",
        TimeLimit:     1,
        MemLimit:      64,
    })
    
    if result.Status != model.StatusAC {
        t.Errorf("Expected AC, got %s", result.Status)
    }
}
```

## 🎓 最佳实践

1. **使用表驱动测试**: 对于多个相似的测试用例
2. **清理资源**: 使用 `t.TempDir()` 自动清理临时文件
3. **跳过不可用的测试**: 使用 `t.Skip()` 而不是失败
4. **记录详细信息**: 使用 `t.Logf()` 记录调试信息
5. **并行测试**: 对于独立的测试使用 `t.Parallel()`

## 📊 测试报告

生成测试报告：

```bash
# JSON 格式
go test ./internal/task/runner/... -json > test-report.json

# 详细输出
go test ./internal/task/runner/... -v 2>&1 | tee test-report.txt
```

## 🔗 相关文档

- [NsJail 改进文档](./nsjail_improvements.md)
- [Go 测试官方文档](https://golang.org/pkg/testing/)
- [Go 基准测试指南](https://golang.org/pkg/testing/#hdr-Benchmarks)

