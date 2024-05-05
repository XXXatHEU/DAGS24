#!/bin/bash

# 启动第一个 Go 程序并将其放入后台执行
./blockchain &
echo "\n"
echo "enter\n"

# 等待所有程序完成
wait

echo "所有程序执行完成"

