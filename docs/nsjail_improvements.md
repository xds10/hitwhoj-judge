# NsJail 沙箱资源监控改进

## 📋 改进概述

将 NsJail 沙箱实现升级为与 SDU Sandbox 相同的资源监控能力，能够准确获取：
- ✅ **CPU 时间**（用户态 + 内核态）
- ✅ **墙钟时间**（Real Time）
- ✅ **内存占用**（最大 RSS）

## 🔄 主要变化

### 1. 资源统计获取

**使用 `syscall.Rusage` 获取进程资源使用情况：**

```go
if cmd.ProcessState != nil {
    sysUsage := cmd.ProcessState.SysUsage()
    if usage, ok := sysUsage.(*syscall.Rusage); ok {
        // CPU时间 = 用户态 + 内核态
        cpuTime = time.Duration(usage.Utime.Sec)*time.Second + 
                  time.Duration(usage.Utime.Usec)*time.Microsecond +
                  time.Duration(usage.Stime.Sec)*time.Second + 
                  time.Duration(usage.Stime.Usec)*time.Microsecond
        
        // 内存使用（RSS，单位KB转字节）
        memUsed = usage.Maxrss * 1024
    }
}
```

### 2. 墙钟时间测量

```go
startTime := time.Now()
err = cmd.Run()
realTime := time.Since(startTime)
```

### 3. 增强的资源限制参数

```go
cmd := exec.Command(
    nr.NsJailPath,
    "--rlimit_as", fmt.Sprintf("%d", memoryLimit*1024*1024),  // 内存限制
    "--rlimit_cpu", fmt.Sprintf("%d", timeLimit+1),           // CPU时间限制
    "--time_limit", fmt.Sprintf("%d", timeLimit*2),           // 墙钟时间限制
    // ... 其他参数
)
```

### 4. 双重超限检查

即使程序正常退出，也会检查资源是否超限：

```go
if status == model.StatusAC {
    // 检查CPU时间
    if cpuTime > time.Duration(timeLimit)*time.Second {
        status = model.StatusTLE
    }
    // 检查内存
    if memUsed > memoryLimit*1024*1024 {
        status = model.StatusMLE
    }
}
```

### 5. 详细的日志记录

```go
zap.L().Info("NsJail execution result",
    zap.Int("test_case", runParams.TestCaseIndex),
    zap.Duration("cpu_time", cpuTime),
    zap.Duration("real_time", realTime),
    zap.Int64("memory_bytes", memUsed),
    zap.Float64("memory_mb", float64(memUsed)/(1024*1024)),
    zap.String("status", string(status)),
)
```

## 📊 与 SDU Sandbox 对比

| 功能 | SDU Sandbox | NsJail (改进后) |
|------|-------------|-----------------|
| CPU 时间 | ✅ `result.CpuTime` | ✅ `rusage.Utime + Stime` |
| 墙钟时间 | ✅ `result.RealTime` | ✅ `time.Since(startTime)` |
| 内存占用 | ✅ `result.Memory` | ✅ `rusage.Maxrss` |
| 返回格式 | JSON | Go 结构体 |
| 需要 sudo | ✅ | ❌ |

## 🎯 使用示例

```go
runner := &NsJailRunner{
    NsJailPath: "nsjail",
}

result := runner.RunInSandbox(model.RunParams{
    TestCaseIndex: 0,
    ExePath:       "/path/to/executable",
    Input:         "1 2\n",
    TimeLimit:     1,    // 1秒
    MemLimit:      64,   // 64MB
})

// 获取资源使用情况
fmt.Printf("CPU时间: %v\n", result.TimeUsed)
fmt.Printf("内存: %d bytes (%.2f MB)\n", 
    result.MemUsed, 
    float64(result.MemUsed)/(1024*1024))
fmt.Printf("状态: %s\n", result.Status)
```

## ⚠️ 注意事项

### 1. Rusage 的局限性

- `Maxrss` 在 Linux 上单位是 **KB**，在 macOS 上是 **字节**
- 只能获取直接子进程的资源使用，不包括孙进程
- 内存统计是 RSS（常驻集大小），不是虚拟内存

### 2. 时间精度

- CPU 时间精度：微秒级
- 墙钟时间精度：纳秒级（Go time.Now()）
- nsjail 的时间限制精度：秒级

### 3. 兼容性

- 需要 Linux 系统（nsjail 仅支持 Linux）
- 需要安装 nsjail：`apt install nsjail` 或从源码编译

## 🚀 启用 NsJail

在代码中使用：

```go
// 创建 NsJail runner
runner := runner.NewRunner(runner.NsJail, "nsjail")

// 或使用默认配置
config := runner.GetDefaultSandboxConfig(runner.NsJail)
```

## 📈 性能对比

| 指标 | SDU Sandbox | NsJail |
|------|-------------|--------|
| 启动开销 | 中等（需要 sudo） | 较低 |
| 隔离级别 | 高 | 高 |
| 资源监控 | 精确 | 精确 |
| 配置复杂度 | 简单 | 中等 |

## 🔍 调试技巧

查看详细日志：

```bash
# 查看评测日志
tail -f log/server.log | grep "NsJail execution"
```

手动测试 nsjail：

```bash
nsjail -Mo -N \
  --rlimit_as 67108864 \
  --rlimit_cpu 2 \
  --time_limit 4 \
  --chroot /path/to/dir \
  --user 99999 \
  --group 99999 \
  --disable_clone_newuser \
  -- ./program
```

## 📝 总结

改进后的 NsJail 实现现在具备了与 SDU Sandbox 相同的资源监控能力，能够：

1. ✅ 准确获取 CPU 时间（用户态 + 内核态）
2. ✅ 准确获取墙钟时间
3. ✅ 准确获取内存占用（RSS）
4. ✅ 双重检查防止超限程序被误判为 AC
5. ✅ 详细的日志记录便于调试

这使得 NsJail 成为一个可靠的沙箱选择，特别适合不需要 sudo 权限的场景。

