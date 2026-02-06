# 测试文档

本目录包含项目的测试相关文档和资源。

## 测试结构

项目采用Go标准的测试结构，测试文件与源代码文件放在同一目录下：

```
internal/
├── service/
│   ├── metrics.go
│   ├── metrics_test.go          # metrics的单元测试
│   ├── task_improved.go
│   └── task_improved_test.go    # task_improved的单元测试
├── task/
│   ├── result/
│   │   ├── comparator.go
│   │   └── comparator_test.go   # comparator的单元测试
│   └── language/
│       ├── detector.go
│       └── detector_test.go     # detector的单元测试
```

## 运行测试

### 运行所有测试

```bash
make test
```

或

```bash
go test ./...
```

### 运行特定包的测试

```bash
# 测试 service 包
make test-service

# 测试 result 包
make test-result

# 测试 language 包
make test-language
```

### 运行详细测试（包含竞态检测）

```bash
make test-verbose
```

### 生成测试覆盖率报告

```bash
make test-coverage
```

这会生成 `coverage.html` 文件，可以在浏览器中打开查看详细的覆盖率报告。

### 运行基准测试

```bash
make bench
```

## 测试覆盖的功能

### 1. 结果比较器测试 (`comparator_test.go`)

- ✅ 严格模式比较
- ✅ 模糊模式比较（忽略多余空格）
- ✅ 换行符处理（Windows/Unix）
- ✅ 首尾空白处理
- ✅ 多行输出比较
- ✅ 性能基准测试

### 2. 统计指标测试 (`metrics_test.go`)

- ✅ 提交记录
- ✅ 成功/失败统计
- ✅ 各状态计数（AC/WA/TLE/MLE/RE/CE/SE）
- ✅ 时间统计（平均/最大/最小）
- ✅ 并发控制统计
- ✅ 缓存命中率统计
- ✅ 并发安全性测试
- ✅ 性能基准测试

### 3. 语言检测测试 (`detector_test.go`)

- ✅ C语言文件名
- ✅ C++语言文件名
- ✅ 未知语言处理

### 4. 改进版评测逻辑测试 (`task_improved_test.go`)

- ✅ 状态优先级更新
- ✅ 得分计算
- ✅ 参数校验
- ✅ 边界条件测试

## 测试最佳实践

### 1. 测试命名

测试函数命名遵循 `Test<FunctionName>_<Scenario>` 格式：

```go
func TestComparator_Compare_Strict(t *testing.T) { ... }
func TestJudgeMetrics_RecordSubmission(t *testing.T) { ... }
```

### 2. 表驱动测试

使用表驱动测试提高测试覆盖率：

```go
tests := []struct {
    name string
    input string
    want string
}{
    {name: "case1", input: "test", want: "expected"},
    {name: "case2", input: "test2", want: "expected2"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // 测试逻辑
    })
}
```

### 3. 并发测试

对于并发安全的代码，添加并发测试：

```go
func TestJudgeMetrics_Concurrent(t *testing.T) {
    // 启动多个goroutine并发访问
    for i := 0; i < 10; i++ {
        go func() {
            // 并发操作
        }()
    }
}
```

### 4. 基准测试

为性能关键的代码添加基准测试：

```go
func BenchmarkComparator_Compare(b *testing.B) {
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // 被测试的代码
    }
}
```

## 测试覆盖率目标

- 核心业务逻辑：**80%+**
- 工具函数：**90%+**
- 整体覆盖率：**70%+**

## 持续集成

项目配置了GitHub Actions自动运行测试：

- 每次push到main/develop分支时运行
- 每次创建Pull Request时运行
- 自动生成覆盖率报告并上传到Codecov

查看配置文件：`.github/workflows/test.yml`

## 添加新测试

当添加新功能时，请同时添加相应的测试：

1. 在同一目录下创建 `*_test.go` 文件
2. 编写测试用例
3. 运行 `make test` 确保测试通过
4. 运行 `make test-coverage` 检查覆盖率

## 常见问题

### Q: 如何跳过某个测试？

```go
func TestSomething(t *testing.T) {
    t.Skip("跳过此测试的原因")
}
```

### Q: 如何只运行特定的测试？

```bash
go test -run TestComparator_Compare_Strict ./internal/task/result/
```

### Q: 如何查看详细的测试输出？

```bash
go test -v ./...
```

### Q: 如何检测竞态条件？

```bash
go test -race ./...
```

## 参考资料

- [Go Testing 官方文档](https://golang.org/pkg/testing/)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Test Comments](https://github.com/golang/go/wiki/TestComments)

