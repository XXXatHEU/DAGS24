package main

import (
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/google/uuid"
	"github.com/willf/bloom"
	"github.com/yanyiwu/gojieba"
)

var jieba = gojieba.NewJieba()
var enterflag = 0 //进入区块链网络的标志

func main() {
	defer jieba.Free()
	gob.Register(bloom.BloomFilter{})
	gob.Register(elliptic.P256())
	gob.Register(elliptic.P256())
	gob.Register(elliptic.P256())
	gob.Register(&TXInput{})
	gob.Register(&TXOutput{})
	gob.Register(uuid.UUID{})
	gob.Register(&Transaction{})
	gob.Register(elliptic.P256())
	gob.Register(wallet{})

	// go func() {
	// 	http.HandleFunc("/", hello)
	// 	err := http.ListenAndServe(":8080", nil)
	// 	if err != nil {
	// 		fmt.Println("ListenAndServe Err:", err.Error())
	// 		return
	// 	}
	// }()

	cliInstances := CLI{}
	cliInstances.Run()

	// 创建一个通道切片，每个通道可以传递字符串
	// cli := CLI{}
	// cli.Run()
	//关闭Zkconn连接
	//defer Zkconn.Close()
}
func hello(resp http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(resp, "Hello World, Are You OK?")
}
