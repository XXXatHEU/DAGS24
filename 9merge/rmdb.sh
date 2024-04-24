#!/bin/bash

# 在当前目录下查找所有以otherpeer开头的子目录
for dir in $(ls -d otherpeer*)
do
  # 进入每个子目录
  cd $dir
  
  # 在子目录下查找所有.db文件并删除
  find . -name "*.db" -delete
  
  # 返回上级目录 
  cd ..
done
