#!/bin/bash

# 设置要查找的变量名和新的值的数组
variable_name1="BlockhasBlnumMin"
variable_name2="BlockhasBlnumMax"
#new_values=("1024" "2048" "512" "256" "128")
new_values=("2048")

# 循环遍历变量名数组，并执行替换操作
for ((i=0; i<${#new_values[@]}; i++)); do
  new_value=${new_values[$i]}
  sed -i "s/$variable_name1 = [0-9]\+/ $variable_name1 = $new_value/" /home/kz/kzworkplace/6exp/miningMain.go
  sed -i "s/$variable_name2 = [0-9]\+/ $variable_name2 = $new_value/" /home/kz/kzworkplace/6exp/miningMain.go
  
  prefolder_path="/home/kz/0/"
  codefolder_path="/home/kz/kzworkplace/6exp/"
  # 构建文件夹路径
  folder_path="/home/kz/$new_value"
  
  # 检查文件夹是否存在，如果不存在则创建
  if [ ! -d "$folder_path" ]; then
    mkdir -p "$folder_path"
    cp $prefolder_path/* "$folder_path"
  fi
  
  # 强行覆盖已存在的 blockchain 文件
  go build -o "$folder_path/blockchain" *.go
  cd $folder_path/
  # 后台运行 blockchain 程序
  nohup "$folder_path/blockchain" & 
 # nohup "$folder_path/blockchain" < /dev/null > nohup.out 2>&1 &

  cd $codefolder_path
done
