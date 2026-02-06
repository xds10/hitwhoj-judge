#!/bin/bash

# 单元测试运行脚本
# 用于运行项目的所有单元测试并生成报告

set -e

echo "=========================================="
echo "🧪 运行单元测试"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试计数
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 运行测试函数
run_test() {
    local package=$1
    local name=$2
    
    echo "📦 测试: $name"
    echo "   包: $package"
    
    if go test -v "$package" 2>&1 | tee /tmp/test_output.txt; then
        echo -e "${GREEN}✅ 通过${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ 失败${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo ""
}

# 1. 测试结果比较器
run_test "./internal/task/result/" "结果比较器 (Comparator)"

# 2. 测试语言检测
run_test "./internal/task/language/" "语言检测 (Language Detector)"

# 3. 测试统计指标
run_test "./internal/service/" "统计指标 (Metrics)"

# 4. 测试改进版评测逻辑
echo "📦 测试: 改进版评测逻辑"
echo "   包: ./internal/service/"
if go test -v ./internal/service/ -run "Test(UpdateFinalStatus|CalculateScore|CountACCases|TruncateString)" 2>&1; then
    echo -e "${GREEN}✅ 通过${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ 失败${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo ""

# 总结
echo "=========================================="
echo "📊 测试总结"
echo "=========================================="
echo "总测试数: $TOTAL_TESTS"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}⚠️  有测试失败，请检查${NC}"
    exit 1
fi

