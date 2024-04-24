package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type CoroutinePool struct {
	MaxConcurrent int
	Available    chan struct{} // 可用资源通道，控制最大并发数
	Connections  chan net.Conn // 建立的连接通道
	WaitGroup    sync.WaitGroup
}

func NewCoroutinePool(maxConcurrent int) *CoroutinePool {
	pool := &CoroutinePool{
		MaxConcurrent: maxConcurrent,
		Available:    make(chan struct{}, maxConcurrent),
		Connections:  make(chan net.Conn),
	}

	// 填充可用通道
	for i := 0; i < maxConcurrent; i++ {
		pool.Available <- struct{}{}
	}

	return pool
}

func (pool *CoroutinePool) Start() {
	for conn := range pool.Connections {
		<-pool.Available // 从池中获取一个资源
		pool.WaitGroup.Add(1)

		go func(c net.Conn) {
			defer func() {
				c.Close()              // 关闭连接
				pool.Available <- struct{}{} // 将资源释放回池中
				pool.WaitGroup.Done()
			}()

			// 处理连接
			fmt.Printf("处理连接：%s\n", c.RemoteAddr())
			time.Sleep(time.Second * 2) // 模拟处理过程
		}(conn)
	}
}

func (pool *CoroutinePool) AddConnection(conn net.Conn) {
	pool.Connections <- conn
}

func main() {
	maxConcurrent := 3
	pool := NewCoroutinePool(maxConcurrent)
	go pool.Start()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("监听失败：", err)
		return
	}
	defer listener.Close()

	fmt.Println("开始监听：", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("连接失败：", err)
			continue
		}
		fmt.Printf("建立连接：%s\n", conn.RemoteAddr())
		pool.AddConnection(conn)
	}
}
