#!/bin/bash

# 设置要运行的进程数



num_processes=19

# 循环运行进程
for ((i = 1; i <= num_processes; i++)); do
  ./blockchain &
done

# 打印一条消息，以指示所有进程已启动
echo "已启动 $num_processes 个'blockchain'进程。"

