package main

import (
	"fmt"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

func main() {
	// 创建zk连接地址
	hosts := []string{"127.0.0.1:2181"}
	// 连接zk
	conn, _, err := zk.Connect(hosts, time.Second*5)
	defer conn.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	println(conn.Server())
}
