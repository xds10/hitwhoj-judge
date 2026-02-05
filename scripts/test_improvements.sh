#!/bin/bash

# 评测系统改进对比测试脚本

echo "=========================================="
echo "评测系统改进对比测试"
echo "=========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 服务器地址
SERVER="http://localhost:53333"

# 测试函数
test_endpoint() {
    local endpoint=$1
    local name=$2
    
    echo -n "测试 $name ... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER$endpoint")
    
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓ 成功${NC} (HTTP $response)"
    else
        echo -e "${RED}✗ 失败${NC} (HTTP $response)"
    fi
}

echo "1. 检查服务健康状态"
echo "-------------------------------------------"
test_endpoint "/health" "健康检查"
test_endpoint "/ping" "Ping测试"
echo ""

echo "2. 检查监控端点"
echo "-------------------------------------------"
test_endpoint "/metrics" "评测统计"
test_endpoint "/system" "系统信息"
test_endpoint "/readiness" "就绪检查"
test_endpoint "/liveness" "存活检查"
echo ""

echo "3. 获取详细统计信息"
echo "-------------------------------------------"
echo "评测统计："
curl -s "$SERVER/metrics" | python3 -m json.tool 2>/dev/null || echo "无法解析JSON"
echo ""

echo "系统信息："
curl -s "$SERVER/system" | python3 -m json.tool 2>/dev/null || echo "无法解析JSON"
echo ""

echo "4. 测试评测任务提交"
echo "-------------------------------------------"

# 创建测试请求
cat > /tmp/test_judge_request.json <<EOF
{
  "cpu_limit": 1000,
  "mem_limit": 67108864,
  "stack_limit": 8388608,
  "proc_limit": 1,
  "code_file": "#include <stdio.h>\n\nint main() {\n    int a, b;\n    scanf(\"%d %d\", &a, &b);\n    printf(\"%d\\\\n\", a + b);\n    return 0;\n}",
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
EOF

echo "提交评测任务..."
response=$(curl -s -X POST "$SERVER/api/v1/task/add" \
  -H "Content-Type: application/json" \
  -d @/tmp/test_judge_request.json)

echo "评测结果："
echo "$response" | python3 -m json.tool 2>/dev/null || echo "$response"
echo ""

echo "5. 再次检查统计信息（查看变化）"
echo "-------------------------------------------"
curl -s "$SERVER/metrics" | python3 -m json.tool 2>/dev/null || echo "无法解析JSON"
echo ""

echo "=========================================="
echo "测试完成"
echo "=========================================="
echo ""
echo "提示："
echo "- 如果看到 ✓ 表示测试通过"
echo "- 如果看到 ✗ 表示测试失败"
echo "- 查看 /metrics 端点了解详细统计"
echo "- 查看 /system 端点了解系统状态"

