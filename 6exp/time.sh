#!/bin/bash

# 用来自动测试时间的脚本  不载入内存模式

# 定义新的blockchainDBFile值
new_value="/home/kz/1024/blockchain.db"
new_values=("128" "256" "512" "1024" "2048")
for ((i=0; i<${#new_values[@]}; i++)); do
    new_value="/home/kz/${new_values[$i]}/blockchain.db"
     logname="noMemorytime——${new_values[$i]}——%s.log"
#timeExpModel := "1"  tt :=
# 使用sed命令替换文本
sed -i "s#var blockchainDBFile = .*#var blockchainDBFile = \"$new_value\"#" /home/kz/kzworkplace/6exp/bucket.go
sed -i "s#timeExpModel := .*#timeExpModel := \"1\"#" /home/kz/kzworkplace/6exp/cli.go
sed -i "s#tt := .*#tt := \"$logname\"#" /home/kz/kzworkplace/6exp/log.go
  go build -o "blockchain" *.go
  "/home/kz/kzworkplace/6exp/blockchain"
  wait
sed -i "s#tt := .*#tt := \"err_%s.log\"#" /home/kz/kzworkplace/6exp/log.go
sed -i "s#timeExpModel := .*#timeExpModel := \"2\"#" /home/kz/kzworkplace/6exp/cli.go
done
nohup ./time2.sh &
