#!/bin/bash

#进入当前程序所在目录
cd "$(dirname "$0")"
cd ./8node


rm node8 -f
go build -o node8 *.go

# 设置要运行的进程数
num_processes=15

# 循环运行进程
for ((i = 1; i <= num_processes; i++)); do
  ./node8 &
done

# 打印一条消息，以指示所有进程已启动
echo "已启动 $num_processes 个'node8'进程。"
