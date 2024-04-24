package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestGuangbo(t *testing.T) {
	arr = make([]int, 0)
	mutex = sync.Mutex{}
	cond = sync.NewCond(&mutex)

	go producer()
	consumer()
}
func TestZi(t *testing.T) {
	stop := make(chan bool)

	fmt.Println("启动子协程")
	go childRoutine(stop)

	time.Sleep(5 * time.Second)

	fmt.Println("发送停止信号给子协程")
	stop <- true

	fmt.Println("主协程继续执行")

	time.Sleep(2 * time.Second)
}
func TestZi2(t *testing.T) {

}

var (
	arr   []int
	mutex sync.Mutex
	cond  *sync.Cond
)

func producer() {
	for i := 1; i <= 5; i++ {
		mutex.Lock()
		arr = append(arr, i) // 往数组中添加元素
		arr[0] = 10          // 删除元素
		fmt.Println("生产者：产生新元素", i)
		if len(arr) == 3 {
			cond.Signal() // 发送信号通知等待的消费者
		}
		mutex.Unlock()
	}
}

func consumer() {
	mutex.Lock()
	for len(arr) < 3 {
		fmt.Println("消费者：等待元素...")
		cond.Wait() // 等待信号通知，并在等待期间释放互斥锁
	}
	for i := 0; i < 3; i++ {
		fmt.Println("消费者：消费元素", arr[i])
	}
	arr = arr[3:] // 移除消费的元素
	mutex.Unlock()
}

func childRoutine(stop chan bool) {
	for {
		select {
		case <-stop:
			fmt.Println("子协程收到停止信号，终止执行")
			return
		default:
			fmt.Println("子协程执行中...")
			time.Sleep(20 * time.Second)
		}
	}
}
