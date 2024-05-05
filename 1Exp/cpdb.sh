#!/bin/bash

# 源文件路径

source_file="./otherpeer0/blockchain.db"

# 查找并复制到以 "otherpeer" 开头但不以 "otherpeer0" 开头的文件夹
for dir in $(find . -type d -name "otherpeer*" -not -name "otherpeer0*"); do
    cp "$source_file" "$dir"
    echo "复制 $source_file 到 $dir"
done

echo "复制完成。"

