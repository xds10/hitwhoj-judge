.PHONY: build test test-verbose test-coverage clean run help

# 默认目标
.DEFAULT_GOAL := help

# 编译
build:
	@echo "🔨 编译项目..."
	@go build -o output/server cmd/server/main.go
	@echo "✅ 编译完成: output/server"

# 运行
run: build
	@echo "🚀 启动服务..."
	@./output/server

# 运行所有测试
test:
	@echo "🧪 运行单元测试..."
	@go test -v ./...

# 运行测试（详细输出）
test-verbose:
	@echo "🧪 运行单元测试（详细模式）..."
	@go test -v -race ./...

# 生成测试覆盖率报告
test-coverage:
	@echo "📊 生成测试覆盖率报告..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

# 运行特定包的测试
test-service:
	@echo "🧪 测试 service 包..."
	@go test -v ./internal/service/...

test-result:
	@echo "🧪 测试 result 包..."
	@go test -v ./internal/task/result/...

test-language:
	@echo "🧪 测试 language 包..."
	@go test -v ./internal/task/language/...

# 基准测试
bench:
	@echo "⚡ 运行基准测试..."
	@go test -bench=. -benchmem ./...

# 代码检查
lint:
	@echo "🔍 运行代码检查..."
	@golangci-lint run ./...

# 格式化代码
fmt:
	@echo "✨ 格式化代码..."
	@go fmt ./...
	@goimports -w .

# 清理
clean:
	@echo "🧹 清理构建文件..."
	@rm -rf output/
	@rm -f coverage.out coverage.html
	@echo "✅ 清理完成"

# 安装依赖
deps:
	@echo "📦 安装依赖..."
	@go mod download
	@go mod tidy
	@echo "✅ 依赖安装完成"

# 帮助信息
help:
	@echo "📖 可用命令:"
	@echo "  make build          - 编译项目"
	@echo "  make run            - 编译并运行服务"
	@echo "  make test           - 运行所有单元测试"
	@echo "  make test-verbose   - 运行测试（详细输出 + 竞态检测）"
	@echo "  make test-coverage  - 生成测试覆盖率报告"
	@echo "  make test-service   - 测试 service 包"
	@echo "  make test-result    - 测试 result 包"
	@echo "  make test-language  - 测试 language 包"
	@echo "  make bench          - 运行基准测试"
	@echo "  make lint           - 运行代码检查"
	@echo "  make fmt            - 格式化代码"
	@echo "  make clean          - 清理构建文件"
	@echo "  make deps           - 安装依赖"
	@echo "  make help           - 显示此帮助信息"
