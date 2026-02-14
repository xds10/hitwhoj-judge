#!/bin/bash

# ==================== 参数解析 ====================
# 用法：./run.sh <judge命令及参数> -- <solution命令及参数>
# 例如：./run.sh python3 judge.py 0 -- ./solution

if [ "$#" -lt 3 ]; then
    echo "用法错误！正确用法："
    echo "  $0 <judge命令及参数> -- <solution命令及参数>"
    exit 1
fi

# 找到 "--" 的位置
sep_index=-1
for i in "$@"; do
    sep_index=$((sep_index + 1))
    if [ "$i" = "--" ]; then
        break
    fi
done

# 检查 "--" 是否存在且不在开头或结尾
if [ $sep_index -eq -1 ] || [ $sep_index -eq 0 ] || [ $sep_index -eq $# ]; then
    echo "错误：命令行中必须恰好有一个 '--'，且不能位于开头或结尾。"
    exit 1
fi

# 提取 judge 和 solution 的参数数组
judge_args=("${@:1:sep_index}")
sol_args=("${@:sep_index+2}")

# ==================== 创建命名管道 ====================
# 使用进程 ID 保证唯一性
PIPE_JUDGE_TO_SOL="/tmp/fifo_judge_to_sol_$$"
PIPE_SOL_TO_JUDGE="/tmp/fifo_sol_to_judge_$$"
mkfifo "$PIPE_JUDGE_TO_SOL" "$PIPE_SOL_TO_JUDGE"

# 确保退出时清理管道
cleanup() {
    rm -f "$PIPE_JUDGE_TO_SOL" "$PIPE_SOL_TO_JUDGE"
}
trap cleanup EXIT

# ==================== 关键修改：以读写方式打开管道，避免阻塞 ====================
exec 3<> "$PIPE_SOL_TO_JUDGE"   # 对应 solution -> judge 方向
exec 4<> "$PIPE_JUDGE_TO_SOL"   # 对应 judge -> solution 方向

# ==================== 启动进程 ====================
# 启动 judge：标准输入来自 fd3 (solution 的输出)，标准输出到 fd4 (写给 solution)，标准错误加上前缀 "judge: "
"${judge_args[@]}" <&3 >&4 2> >(sed 's/^/judge: /' >&2) &
JUDGE_PID=$!

# 启动 solution：标准输入来自 fd4 (judge 的输出)，标准输出到 fd3 (写给 judge)，标准错误加上前缀 "  sol: "
"${sol_args[@]}" <&4 >&3 2> >(sed 's/^/  sol: /' >&2) &
SOL_PID=$!

# 关闭父进程中打开的文件描述符，避免干扰子进程的引用计数
exec 3>&- 4>&-

# ==================== 等待进程结束 ====================
wait $JUDGE_PID
JUDGE_RC=$?
wait $SOL_PID
SOL_RC=$?

# ==================== 输出结果 ====================
# 打印空行，与 Python 脚本一致
echo
echo "Judge return code: $JUDGE_RC"
if [ $JUDGE_RC -ne 0 ]; then
    echo "Judge error message: (not captured by bash script)"
fi
echo "Solution return code: $SOL_RC"
if [ $SOL_RC -ne 0 ]; then
    echo "Solution error message: (not captured by bash script)"
fi

# 根据返回码给出评测系统解释
if [ $SOL_RC -ne 0 ]; then
    echo "A solution finishing with exit code other than 0 (without exceeding time or memory limits) would be interpreted as a Runtime Error in the system."
elif [ $JUDGE_RC -ne 0 ]; then
    echo "A solution finishing with exit code 0 (without exceeding time or memory limits) and a judge finishing with exit code other than 0 would be interpreted as a Wrong Answer in the system."
else
    echo "A solution and judge both finishing with exit code 0 (without exceeding time or memory limits) would be interpreted as Correct in the system."
fi

exit 0