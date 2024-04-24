package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// 协程池
type CoroutinePool struct {
	Tasks chan func()
	wg    sync.WaitGroup
}

func NewPool(maxGoroutines int) *CoroutinePool {
	return &CoroutinePool{
		Tasks: make(chan func()),
		wg:    sync.WaitGroup{},
	}
}

func (p *CoroutinePool) Run(task func()) {
	p.Tasks <- task
}

func (p *CoroutinePool) Start() {
	for i := 0; i < maxGoroutines; i++ {
		go func() {
			for task := range p.Tasks {
				task()
				p.wg.Done()
			}
		}()
	}
}

func (p *CoroutinePool) Wait() {
	p.wg.Wait()
}

// P2P 连接
type Connection struct {
	ID int
}

func handleConn(conn *Connection) {
	// 处理连接...
	fmt.Printf("Handling conn %d\n", conn.ID)
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
}

func main() {
	pool := NewPool(10)
	pool.Start()

	// 分发连接
	for i := 0; i < 50; i++ {
		conn := &Connection{i}
		pool.wg.Add(1)
		pool.Run(func() {
			handleConn(conn)
		})
	}

	pool.Wait()
}
