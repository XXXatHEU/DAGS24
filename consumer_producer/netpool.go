package main

import (
	"container/list"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type SendPool struct {
	list *list.List
}

var GlobalSendPool = SendPool{
	list: list.New(),
}

type SendTask struct {
	index     int
	TimeStamp int64
	TaskData  []byte
}
type DialconnPool struct {
	maxindex         int
	mutex            sync.Mutex
	notifyDownGoChan map[int]chan int //向子协程发送通知的通道
	notifyUpGoChan   map[int]chan int //子协程向父协程发送消息的通道（可能用不着）这里用来判断连接是否断开了吧！ 需要双方通过这个交互
}

var GlobalDialconnPool = DialconnPool{
	maxindex:         1,
	mutex:            sync.Mutex{},
	notifyDownGoChan: make(map[int]chan int),
	notifyUpGoChan:   make(map[int]chan int),
}

func Ziptask(idx int, TaskData []byte) (task SendTask) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println("获取时区失败：", err)
		return
	}

	now := time.Now().In(loc)
	timestamp := now.Unix()
	task = SendTask{
		index:     idx,
		TimeStamp: timestamp,
		TaskData:  TaskData,
	}
	fmt.Println(task)
	return task
}

//往池子里加task
func AddSendTaskTopool(task SendTask) {
	GlobalSendPool.list.PushBack(task)
}

//后台运行协程
func DelExpiredTasksFromPool() {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println(err)
		return
	}
	now := time.Now().In(loc)
	limit := now.Add(-2 * time.Minute)
	for e := GlobalSendPool.list.Front(); e != nil; {
		task := e.Value
		if time.Unix(task.TimeStamp, 0).Before(limit) {
			next := e.Next()
			GlobalSendPool.list.Remove(e)
			e = next
		} else {
			e = e.Next()
		}
	}
}

//后台接收协程
func AcceptSendTask(taskbyte chan []byte) {
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(10000) + 1
	for {
		taskmsg, ok := <-taskbyte
		if !ok {
			// 关闭通道
			close(taskbyte)
			return
		}
		//1.打包任务
		tsk := Ziptask(index, taskmsg)
		//2.发送到池子里面
		AddSendTaskTopool(tsk)
		//3.通知发送协程，有任务到达
		SendMessageToDialGoPool()
	}
}

func GetSendTaskfromPool(TaskArrived chan int) {
	index := -1
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		<-TaskArrived
		//这里业务逻辑暂时先读出来
		for e := GlobalSendPool.list.Front(); e != nil; {
			task := e.Value
			now := time.Now().In(loc)
			limit := now.Add(-2 * time.Minute)
			//首先判断index 只把大于当前index的进行处理
			if task.index < index {
				e = e.Next()
			} else if time.Unix(task.TimeStamp, 0).Before(limit) {
				e = e.Next()
			} else {
				fmt.Println(task.TaskData)
				e = e.Next()
			}
		}
	}
}

//增加新的协程到连接建立池里面
func AddDialGoPool() (index int /*在pool的下标*/) {
	GlobalDialconnPool.mutex.Lock()
	up := make(chan int)
	Down := make(chan int)
	GlobalDialconnPool.maxindex++
	index = GlobalDialconnPool.maxindex
	GlobalDialconnPool.notifyDownGoChan[index] = Down
	GlobalDialconnPool.notifyUpGoChan[index] = up
	GlobalDialconnPool.mutex.Unlock()
	return index
}
func DelDialGoPool(index int) {
	GlobalDialconnPool.mutex.Lock()
	delete(GlobalDialconnPool.notifyDownGoChan, index)
	delete(GlobalDialconnPool.notifyUpGoChan, index)
	GlobalDialconnPool.mutex.Unlock()
}

//向连接建立协程容器里所有的协程发送一个有新的任务达到需要去处理
func SendMessageToDialGoPool() {
	GlobalDialconnPool.mutex.Lock()
	for index, c := range GlobalDialconnPool.notifyDownGoChan {
		select {
		case c <- 1: //消息发送成功
		default: //消息发送失败，说明已经关闭或者满了，这时候将其删除
			DelDialGoPool(index)
		}
	}
	GlobalDialconnPool.mutex.Unlock()
}

var wg sync.WaitGroup
