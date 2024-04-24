package main

import (
	"bytes"
	"fmt"

	"github.com/willf/bloom"
	"github.com/yanyiwu/gojieba"
)

func main() {
	// 创建布隆过滤器
	m := uint(1000000) // 位数组大小
	k := uint(5)       // 哈希函数数量
	filter := bloom.New(m, k)
	// 创建分词器实例
	var jieba = gojieba.NewJieba()
	defer jieba.Free()

	// 要分词的文本
	text := "hello tom,我爱自然语言处理,我给tom一笔24元的巨资"

	// 对文本进行分词
	words := jieba.Cut(text, true)

	// 打印分词结果
	fmt.Println(words)
	for index, str := range words {
		filter.Add([]byte(str))
		fmt.Printf("Index: %d, Value: %s\n", index, str)
	}
	// 添加元素
	filter.Add([]byte("apple"))
	filter.Add([]byte("banana"))
	filter.Add([]byte("orange"))
	filter.Add([]byte("你好"))

	// 验证元素是否存在
	fmt.Println(filter.Test([]byte("apple")))      // 输出: true
	fmt.Println("你好 :", filter.Test([]byte("你好"))) // 输出: true
	fmt.Println(filter.Test([]byte("banana")))     // 输出: true
	fmt.Println(filter.Test([]byte("orange")))     // 输出: true
	fmt.Println(filter.Test([]byte("grape")))      // 输出: false
	fmt.Println(filter.Test([]byte("watermelon"))) // 输出: false

	// 序列化和反序列化
	// 假设有一个文件名为 "filter.bin" 的文件

	// 将布隆过滤器保存到文件
	// file, _ := os.Create("filter.bin")
	// filter.WriteTo(file)
	// file.Close()

	//将过滤器写入
	var buf bytes.Buffer
	bytesWritten, err := filter.WriteTo(&buf)
	if err != nil {
		fmt.Println("出错")
	}
	fmt.Printf("内容：%x\n", bytesWritten)

	// 从文件加载布隆过滤器
	var filter2 bloom.BloomFilter
	filter2.ReadFrom(&buf)

	// 验证从文件加载的布隆过滤器是否仍然有效
	fmt.Println(filter2.Test([]byte("apple")))         // 输出: true
	fmt.Println(filter2.Test([]byte("banana")))        // 输出: true
	fmt.Println(filter2.Test([]byte("orange")))        // 输出: true
	fmt.Println(filter2.Test([]byte("grape")))         // 输出: false
	fmt.Println(filter2.Test([]byte("watermelon")))    // 输出: false
	fmt.Println("我爱", filter.Test([]byte("我爱")))       // 输出: true
	fmt.Println("巨资:", filter.Test([]byte("巨资")))      // 输出: true
	fmt.Println("tom:", filter.Test([]byte("tom")))    // 输出: true
	fmt.Println("处理 :", filter.Test([]byte("处理")))     // 输出: true
	fmt.Println("自然语言 :", filter.Test([]byte("自然语言"))) // 输出: true
}
