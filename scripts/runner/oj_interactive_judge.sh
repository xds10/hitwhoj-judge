#!/bin/bash
set -euo pipefail  # 严格模式：遇到错误立即退出，未定义变量报错，管道错误传递

# ===================== 1. 参数解析与检查 =====================
# 参数说明（所有文件已在同一目录下，无需复制）：
# $1: 选手代码文件名（必填，如user.cpp，已在工作目录）
# $2: 评测代码文件名（必填，如interactor.cpp，已在工作目录）
# $3: 输入文件名（必填，如input.txt，已在工作目录）
# $4: 输出文件名（必填，如output.txt，已在工作目录）
# $5: 标准答案文件名（必填，如answer.txt，已在工作目录）
# $6: 工作目录（可选，未指定则用当前目录./）

# 检查参数数量
if [ $# -lt 5 ]; then
    echo "用法错误！正确用法："
    echo "bash $0 <选手代码文件名> <评测代码文件名> <输入文件名> <输出文件名> <标准答案文件名> [工作目录（可选）]"
    echo "示例："
    echo "bash $0 user.cpp interactor.cpp input.txt output.txt answer.txt ./work_dir"
    exit 1
fi

# 提取参数
USER_CODE_FILE="$1"
INTER_CODE_FILE="$2"
INPUT_FILE="$3"
OUTPUT_FILE="$4"
ANSWER_FILE="$5"
WORK_DIR="${6:-.}"  # 可选工作目录，默认当前目录./

# 进入工作目录（所有文件都在此目录下）
echo "=== 进入工作目录 ==="
cd "$WORK_DIR" || { echo "❌ 进入工作目录失败：$WORK_DIR"; exit 1; }
echo "当前工作目录：$(pwd)"

# 检查所有必填文件是否存在（在工作目录下）
echo "=== 检查输入文件 ==="
REQUIRED_FILES=("$USER_CODE_FILE" "$INTER_CODE_FILE" "$INPUT_FILE" "$OUTPUT_FILE" "$ANSWER_FILE")
for FILE in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$FILE" ]; then
        echo "❌ 错误：工作目录下未找到文件 → $FILE"
        exit 1
    fi
    echo "✅ 找到文件：$FILE"
done

# 初始化结果文件（避免testlib写入失败）
touch result.txt

# ===================== 2. 编译裁判程序和选手程序 =====================
echo -e "\n=== 编译代码 ==="

# 编译交互裁判程序（依赖testlib.h）
echo "编译裁判程序..."
g++ "$INTER_CODE_FILE" -o interactor.out -O2 -Wall -ltestlib 2> compile_inter.log
if [ $? -ne 0 ]; then
    echo "❌ 裁判程序编译失败！错误日志："
    cat compile_inter.log
    exit 1
fi
echo "✅ 裁判程序编译成功"

# 编译选手程序
echo "编译选手程序..."
g++ "$USER_CODE_FILE" -o user.out -O2 -Wall 2> compile_user.log
if [ $? -ne 0 ]; then
    echo "❌ 选手程序编译错误（CE）"
    echo "编译错误日志："
    cat compile_user.log
    exit 0  # CE是选手问题，正常退出
fi
echo "✅ 选手程序编译成功"

# ===================== 3. 创建管道并启动交互进程 =====================
echo -e "\n=== 启动交互评测 ==="
# 创建双向通信管道（在工作目录下）
mkfifo pipe_user_to_inter pipe_inter_to_user || { echo "❌ 创建管道失败"; exit 1; }
chmod 666 pipe_user_to_inter pipe_inter_to_user

# 后台启动裁判程序（直接使用工作目录下的文件，无需复制）
# 格式：./interactor.out <Input_File> <Output_File> <Answer_File> <Result_File> -appe
./interactor.out "$INPUT_FILE" pipe_user_to_inter "$ANSWER_FILE" result.txt -appe > pipe_inter_to_user 2> interactor.log &
INTER_PID=$!
echo "裁判程序PID：$INTER_PID"

# 启动选手程序（绑定管道）
./user.out < pipe_inter_to_user > pipe_user_to_inter 2> user.log &
USER_PID=$!
echo "选手程序PID：$USER_PID"

# ===================== 4. 超时监控与进程等待 =====================
# 设定超时时间（单位：秒，可根据题目调整）
TIMEOUT=5
echo -e "\n=== 等待评测完成（超时时间：${TIMEOUT}s） ==="

# 等待进程结束，或超时杀死
wait $USER_PID $INTER_PID > /dev/null 2>&1 &
WAIT_PID=$!
sleep $TIMEOUT

# 检查是否超时
if ps -p $WAIT_PID > /dev/null; then
    echo "❌ 选手程序超时（TLE）"
    # 杀死所有相关进程
    kill -9 $USER_PID $INTER_PID $WAIT_PID > /dev/null 2>&1
    exit 0
fi

# ===================== 5. 解析评测结果 =====================
echo -e "\n=== 解析评测结果 ==="
RESULT_CONTENT=$(cat result.txt 2>/dev/null || echo "")

# 按testlib输出格式解析结果（优先级：PE > WA > AC）
if echo "$RESULT_CONTENT" | grep -qi "^PE"; then
    echo "❌ 格式错误（PE）"
    echo "错误详情：$(echo $RESULT_CONTENT | sed 's/^PE: //i')"
elif echo "$RESULT_CONTENT" | grep -qi "^WA"; then
    echo "❌ 答案错误（WA）"
    echo "错误详情：$(echo $RESULT_CONTENT | sed 's/^WA: //i')"
elif echo "$RESULT_CONTENT" | grep -qi "^AC"; then
    SCORE=$(echo $RESULT_CONTENT | awk '{print $1}')
    MESSAGE=$(echo $RESULT_CONTENT | cut -d' ' -f2-)
    echo "✅ 答案正确（AC）"
    echo "得分：$SCORE"
    echo "详情：$MESSAGE"
else
    # 无结果→运行错误（RE）
    echo "❌ 运行错误（RE）"
    echo "选手程序日志："
    cat user.log
    echo -e "\n裁判程序日志："
    cat interactor.log
fi

# ===================== 6. 清理临时文件（可选） =====================
# 如需保留管道/日志文件调试，注释下面的代码
echo -e "\n=== 清理临时文件 ==="
rm -f pipe_user_to_inter pipe_inter_to_user  # 删除管道
rm -f compile_inter.log compile_user.log     # 删除编译日志
rm -f user.log interactor.log               # 删除运行日志
# rm -f interactor.out user.out result.txt   # 可选：删除编译产物和结果文件

echo -e "\n=== 评测流程完成 ==="
exit 0