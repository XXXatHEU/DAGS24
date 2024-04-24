package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

const (
	zkServers = "localhost:2181"
	lockPath  = "/locks"
)

func main() {
	conn, _, err := zk.Connect([]string{zkServers}, time.Second)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 创建分布式锁对象
	lock := NewDistributedLock(conn)

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 请求获取锁
			if err := lock.Lock(); err != nil {
				fmt.Printf("goroutine %d failed to acquire lock: %v\n", id, err)
				return
			}
			defer lock.Unlock()

			// 模拟业务处理
			time.Sleep(time.Second)
			fmt.Printf("goroutine %d acquired lock and completed the task\n", id)
		}(i)
	}

	wg.Wait()
}

type DistributedLock struct {
	conn *zk.Conn
	path string
}

func NewDistributedLock(conn *zk.Conn) *DistributedLock {
	return &DistributedLock{
		conn: conn,
		path: lockPath,
	}
}

func (l *DistributedLock) Lock() error {
	_, err := l.conn.Create(l.path+"/lock-", nil, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		return err
	}

	for {
		children, _, events, err := l.conn.ChildrenW(l.path)
		if err != nil {
			return err
		}

		minChild := minNode(children)
		if minChild == "" || isMinNode(minChild, children) {
			return nil
		}

		_, err = l.awaitEvent(events, zk.EventNodeDeleted)
		if err != nil {
			return err
		}
	}
}

func (l *DistributedLock) Unlock() error {
	return l.conn.Close()
}

func (l *DistributedLock) awaitEvent(ch <-chan zk.Event, eventType zk.EventType) (zk.Event, error) {
	select {
	case event := <-ch:
		if event.Err != nil {
			return zk.Event{}, event.Err
		}
		if event.Type == eventType {
			return event, nil
		}

	case <-time.After(time.Second):
		return zk.Event{}, fmt.Errorf("timeout waiting for event: %v", eventType)
	}

	return zk.Event{}, fmt.Errorf("unexpected event")
}

func isMinNode(node string, nodes []string) bool {
	for _, n := range nodes {
		if n < node {
			return false
		}
	}
	return true
}

func minNode(nodes []string) string {
	min := ""
	for _, node := range nodes {
		if min == "" || node < min {
			min = node
		}
	}
	return min
}
