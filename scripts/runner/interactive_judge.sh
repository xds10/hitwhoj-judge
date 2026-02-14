#!/bin/bash
set -euo pipefail  # 严格模式：遇到错误立即退出，未定义变量报错，管道错误传递

# ===================== 通用函数定义 =====================
# 函数：检查文件是否存在且为可执行文件，不存在则报错退出（带提示）
check_executable_exists() {
    local file_path="$1"
    local file_desc="$2"  # 文件描述（用于日志提示）
    
    if [ ! -f "$file_path" ]; then
        echo "❌ $file_desc 文件不存在：$file_path"
        exit 1
    fi
    
    if [ ! -x "$file_path" ]; then
        echo "⚠️ $file_desc 文件无执行权限，尝试添加执行权限..."
        chmod +x "$file_path" || { echo "❌ 为$file_desc文件添加执行权限失败：$file_path"; exit 1; }
        echo "✅ 已为$file_desc文件添加执行权限：$file_path"
    fi
    
    echo "✅ 找到可执行的$file_desc文件：$file_path"
}

# 函数：检查文件是否存在，不存在则创建空文件（带提示）
create_empty_if_not_exists() {
    local file_path="$1"
    local file_desc="$2"  # 文件描述（用于日志提示）
    if [ ! -f "$file_path" ]; then
        echo "⚠️  $file_desc 文件不存在（$file_path），自动创建空文件！"
        touch "$file_path" || { echo "❌ 创建空 $file_desc 文件失败：$file_path"; exit 1; }
        echo "✅ 已创建空 $file_desc 文件：$file_path"
    else
        echo "✅ 找到 $file_desc 文件：$file_path"
    fi
}

# ===================== 1. 参数解析与检查 =====================
# 参数说明（所有文件均在当前目录下）：
# $1: 选手可执行文件名（必填，如user.out）
# $2: 评测可执行文件名（必填，如interactor.out）
# $3: 输入文件名（必填，如input.txt）
# $4: 输出文件名（可选，如output.txt，不传则不传递该参数给交互器）
# $5: 标准答案文件名（可选，如answer.txt，不传则不传递该参数给交互器）

# 检查参数数量（至少需要3个必填参数）
if [ $# -lt 3 ]; then
    echo "用法错误！正确用法："
    echo "bash $0 <选手可执行文件名> <评测可执行文件名> <输入文件名> [输出文件名（可选）] [标准答案文件名（可选）]"
    echo "示例1（仅必填参数）："
    echo "bash $0 user.out interactor.out input.txt"
    echo "示例2（含可选参数）："
    echo "bash $0 user.out interactor.out input.txt output.txt answer.txt"
    exit 1
fi

# 提取参数（必填参数直接赋值，可选参数赋默认空值）
USER_EXEC_FILE="$1"
INTER_EXEC_FILE="$2"
INPUT_FILE="$3"
OUTPUT_FILE="${4:-}"  # 可选输出文件，默认空
ANSWER_FILE="${5:-}"  # 可选标准答案文件，默认空

# ===================== 2. 检查核心文件 =====================
echo -e "\n=== 检查核心文件 ==="
# 检查必填的可执行文件（选手程序、裁判程序）
check_executable_exists "$USER_EXEC_FILE" "选手程序"
check_executable_exists "$INTER_EXEC_FILE" "裁判程序"

# 检查必填的输入文件
create_empty_if_not_exists "$INPUT_FILE" "输入"

# 检查可选的输出文件（仅当传入参数时检查/创建）
if [ -n "$OUTPUT_FILE" ]; then
    create_empty_if_not_exists "$OUTPUT_FILE" "输出"
else
    echo "ℹ️  未指定输出文件，交互器将不使用该参数"
fi

# 检查可选的标准答案文件（仅当传入参数时检查/创建）
if [ -n "$ANSWER_FILE" ]; then
    create_empty_if_not_exists "$ANSWER_FILE" "标准答案"
else
    echo "ℹ️  未指定标准答案文件，交互器将不使用该参数"
fi

# 初始化结果文件和日志文件（确保存在，避免后续读写失败）
echo -e "\n=== 初始化评测辅助文件 ==="
touch result.txt user.log interactor.log
echo "✅ 已初始化结果文件和日志文件（result.txt/user.log/interactor.log）"

# ===================== 3. 创建管道并启动交互进程 =====================
echo -e "\n=== 启动交互评测 ==="
# 创建双向通信管道（容错：先删除旧管道再创建，避免冲突）
rm -f pipe_user_to_inter pipe_inter_to_user
mkfifo pipe_user_to_inter pipe_inter_to_user || { echo "❌ 创建管道失败"; exit 1; }
chmod 666 pipe_user_to_inter pipe_inter_to_user

# 拼接交互器启动参数（适配可选参数）
INTER_ARGS=("$INPUT_FILE")
# 添加输出文件参数（如果指定），否则使用管道作为输出载体

INTER_ARGS+=("pipe_user_to_inter")  # 无输出文件时，管道作为交互输出
if [ -n "$OUTPUT_FILE" ]; then
    INTER_ARGS+=("$OUTPUT_FILE")
fi
# 添加标准答案文件参数（如果指定）
if [ -n "$ANSWER_FILE" ]; then
    INTER_ARGS+=("$ANSWER_FILE")
fi

# 后台启动裁判程序（动态拼接参数）
./"$INTER_EXEC_FILE" "${INTER_ARGS[@]}" > pipe_inter_to_user 2> interactor.log &
INTER_PID=$!
echo "裁判程序PID：$INTER_PID"
echo "裁判程序启动参数：${INTER_ARGS[*]}"

# 启动选手程序（绑定管道）
./"$USER_EXEC_FILE" < pipe_inter_to_user > pipe_user_to_inter 2> user.log &
USER_PID=$!
echo "选手程序PID：$USER_PID"

# ===================== 4. 超时监控与进程等待（容错优化） =====================
# 设定超时时间（单位：秒，可根据题目调整）
TIMEOUT=5
echo -e "\n=== 等待评测完成（超时时间：${TIMEOUT}s） ==="

# 等待进程结束，或超时杀死（容错：先检查进程是否存在）
wait $USER_PID $INTER_PID > /dev/null 2>&1 &
WAIT_PID=$!
sleep $TIMEOUT

# 检查是否超时（容错：先判断进程是否存在）
if ps -p $WAIT_PID > /dev/null; then
    echo "❌ 选手程序超时（TLE）"
    # 杀死所有相关进程（容错：仅杀死存在的进程，避免报错）
    [ -n "$USER_PID" ] && ps -p $USER_PID > /dev/null && kill -9 $USER_PID > /dev/null 2>&1
    [ -n "$INTER_PID" ] && ps -p $INTER_PID > /dev/null && kill -9 $INTER_PID > /dev/null 2>&1
    [ -n "$WAIT_PID" ] && ps -p $WAIT_PID > /dev/null && kill -9 $WAIT_PID > /dev/null 2>&1
    # 清理管道后退出
    rm -f pipe_user_to_inter pipe_inter_to_user
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
    cat user.log 2>/dev/null || echo "（无选手日志）"
    echo -e "\n裁判程序日志："
    cat interactor.log 2>/dev/null || echo "（无裁判日志）"
fi

# ===================== 6. 清理临时文件（容错优化） =====================
echo -e "\n=== 清理临时文件 ==="
# 删除管道（容错：忽略不存在的文件）
rm -f pipe_user_to_inter pipe_inter_to_user
# 删除日志文件（容错：忽略不存在的文件）
rm -f user.log interactor.log
# 可选：删除结果文件（注释默认关闭，如需开启请取消注释）
# rm -f result.txt

echo -e "\n=== 评测流程完成 ==="
exit 0