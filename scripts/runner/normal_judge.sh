#!/bin/bash

# 参数1：可执行文件的路径
# 参数2：输入文件的路径
EXECUTABLE="$1"
INPUT_FILE="$2"

# 直接执行，假设都在当前目录下
./$EXECUTABLE < $INPUT_FILE

# 脚本正常退出
exit 0