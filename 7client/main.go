package main

import (
	"fmt"
	"net"
	"os"
)

func init() {
}

func main() {
	ClinetOrder()
}

func ClinetOrder() {

	serverAddr := "127.0.0.1:10000"
	ReadconnChan := make(chan string)
	readDone := make(chan struct{}) //连接断开

	// 连接服务器
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}
	fmt.Println("节点连接成功")
	defer conn.Close()

	//由子协程去负责接收数据，并将数据给osinput
	go func() {
		for {
			// 从服务器接收响应
			buffer := make([]byte, 1024)
			_, err = conn.Read(buffer)
			fmt.Println("从服务器获得响应")
			if err != nil {
				fmt.Println("Error receiving response:", err)
				close(readDone)
				return
			}
			receivedData := string(buffer)
			fmt.Println("Server response:", receivedData)
			ReadconnChan <- receivedData
		}
	}()

	for {
		select {
		case msg := <-ReadconnChan:
			fmt.Println("Server response:", msg)
		case <-readDone:
			//连接已经断开
			return
		}
	}
}
