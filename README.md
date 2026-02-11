


# HITWH-OJ Judge Service

一个轻量级的在线评测系统后端服务，支持多语言代码编译、沙箱运行和测试用例评测。

## 技术栈

- **后端框架**: Go + Gin
- **编译系统**: 支持C/C++等语言
- **沙箱环境**: nsjail
- **存储系统**: MinIO
- **日志系统**: zap
- **配置管理**: yaml
- **认证系统**: JWT

## 项目结构

```
├── api/              # API定义
├── cmd/              # 命令行入口
├── config/           # 配置文件
├── internal/         # 内部实现
│   ├── conf/         # 配置加载
│   ├── dao/          # 数据访问层
│   ├── handler/      # 请求处理
│   ├── middleware/   # 中间件
│   ├── model/        # 数据模型
│   ├── server/       # 服务器配置
│   ├── service/      # 业务逻辑
│   └── task/         # 评测任务核心
│       ├── compiler/ # 编译器
│       ├── result/   # 结果比较
│       └── runner/   # 沙箱运行
├── pkg/              # 公共包
└── example/          # 示例文件
```

## 核心功能

1. **代码编译** - 支持多种编程语言的代码编译
2. **沙箱运行** - 使用nsjail提供安全的代码执行环境
3. **测试评测** - 自动执行测试用例并比较结果
4. **资源限制** - 精确控制CPU、内存、时间等资源使用
5. **结果管理** - 详细记录评测结果和资源消耗

## 安装与配置

### 前置依赖

- Go 1.20+
- GCC/G++ (用于C/C++编译)
- nsjail (用于安全沙箱)
- MinIO (用于文件存储)

### 配置文件

1. 复制环境变量示例文件:
   ```bash
   cp .env.example .env
   ```

2. 编辑配置文件:
   - `.env` - 环境变量配置
   - `config/config.yaml` - 系统配置

## 运行方式

### 编译运行

```bash
# 编译
make build

# 运行
./output/server
```

### 开发模式

```bash
# 直接运行
go run cmd/server/main.go
```

## API接口

### 提交评测任务

```
POST /api/calc/v1/task/add
```

**请求参数**:
```json
{
  "cpu_limit": 1000,
  "mem_limit": 67108864,
  "stack_limit": 8388608,
  "proc_limit": 1,
  "code_file": "#include <stdio.h>\n\nint main() {\n    int a, b;\n    scanf(\"%d %d\", &a, &b);\n    printf(\"%d\\n\", a + b);\n    return 0;\n}",
  "code_language": "c",
  "is_special": false,
  "bucket": "hitwhoj-rebirth",
  "check_points": [
    {
      "input": "f303b7d2f2b87f9e16df05e2bca7c409",
      "output": "6d7fce9fee471194aa8b5b6e47267f03"
    }
  ]
}
```

**响应示例**:
```json
{
  "task_id": 123456789,
  "status": "PENDING"
}
```

## 使用示例

### A+B问题评测

1. 准备C语言代码:
   ```c
   #include <stdio.h>
   
   int main() {
       int a, b;
       scanf("%d %d", &a, &b);
       printf("%d\n", a + b);
       return 0;
   }
   ```

2. 准备测试用例:
   - 输入文件: `1.in` (内容: `1 2`)
   - 输出文件: `1.out` (内容: `3`)

3. 计算文件MD5:
   ```bash
   md5sum 1.in 1.out
   ```

4. 发起API请求（参考API接口部分）

## 评测流程

1. **接收请求** - 验证参数并创建评测任务
2. **代码编译** - 根据语言类型选择编译器
3. **沙箱运行** - 在安全环境中执行编译后的程序
4. **结果比较** - 将程序输出与期望输出比较
5. **生成报告** - 汇总评测结果和资源使用情况

## 开发与贡献

### 代码规范

- 遵循Go语言官方代码规范
- 使用go fmt格式化代码
- 添加必要的注释

### 测试

```bash
# 运行单元测试
make test
```

### 日志管理

日志文件位于 `log/server.log`，包含详细的请求和执行信息。

### TODO

1. 将 runner 命令行里面由.sh 脚本改造一下 (OK)

2. 增加 spj，交互题

3. 改造一下文件管理，从 cache 后下载完全下到一个目录里：（不能用软链） (OK)
   每评测一个可以 copy 出一个代码文件，输入输出，评测文件到一个文件夹

4. 优化配置文件读取，要求沙箱可以直接从配置文件选择

5. nsjail尽量使用墙钟，顺便内存测量不准，需要处理