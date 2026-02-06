#!/bin/bash

# 快速测试脚本 - 运行所有测试并显示摘要

echo "🧪 运行单元测试..."
echo ""

# 运行测试并捕获输出
TEST_OUTPUT=$(go test ./internal/task/result/ ./internal/task/language/ ./internal/service/ -v 2>&1)
TEST_EXIT_CODE=$?

# 统计测试结果
TOTAL_TESTS=$(echo "$TEST_OUTPUT" | grep -c "^=== RUN")
PASSED_TESTS=$(echo "$TEST_OUTPUT" | grep -c "^--- PASS:")
FAILED_TESTS=$(echo "$TEST_OUTPUT" | grep -c "^--- FAIL:")

# 显示结果
echo "=========================================="
echo "📊 测试结果摘要"
echo "=========================================="
echo "总测试数: $TOTAL_TESTS"
echo "通过: $PASSED_TESTS ✅"
echo "失败: $FAILED_TESTS ❌"
echo ""

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "🎉 所有测试通过！"
    echo ""
    echo "运行详细测试: make test"
    echo "生成覆盖率报告: make test-coverage"
    exit 0
else
    echo "⚠️  有测试失败"
    echo ""
    echo "$TEST_OUTPUT"
    exit 1
fi

