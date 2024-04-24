package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main02() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入本地监听的端口号：")
	localPort, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	localPort = strings.TrimSpace(localPort)

	fmt.Print("请输入目标IP地址和端口号（例如：127.0.0.1:8000）：")
	remoteAddr, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	remoteAddr = strings.TrimSpace(remoteAddr)

	// 启动接收协程
	go receiveUDP(localPort)

	for {
		fmt.Print("请输入要发送的消息：")
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// 发送消息
		sendUDP(remoteAddr, message)
	}
}

// 接收UDP数据包
func receiveUDP(port string) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("收到来自 %s 的消息：%s\n", remoteAddr.String(), string(buffer[:n]))
	}
}

// 发送UDP数据包
func sendUDP(remoteAddrStr string, message string) {
	remoteAddr, err := net.ResolveUDPAddr("udp", remoteAddrStr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		log.Fatal(err)
	}
}
