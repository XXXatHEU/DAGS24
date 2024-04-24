#!/bin/bash

# 定义进程路径 
proc_path="/goworkplace/graph/blockchain"

# 获取进程ID
pid=$(pgrep -f "$proc_path") 

# 输出进程ID
echo "Process ID: $pid"

# 进行strace追踪 
strace -p $pid -ff
