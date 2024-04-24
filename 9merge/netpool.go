package main

/*
AddSendTaskTopool 向sendpool池子添加task
Ziptask 打包任务 主要是添加上时间戳
DelExpiredTasksFromPool 删除任务 根据时间戳
AcceptSendTask  从别的地方接收任务并将任务处理后放到池子里面

*/
import (
	"container/list"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var SendPoolRWMutex sync.RWMutex

type SendPool struct {
	Poollist *list.List
}

var GlobalSendPool = SendPool{
	Poollist: list.New(),
}

type SendTask struct {
	index     int
	TimeStamp int64
	TaskData  []byte //SendMessage
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

func Ziptask(idx int, sendTask []byte) (task SendTask) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println("获取时区失败：", err)
		return
	}
	now := time.Now().In(loc)
	timestamp := now.Unix()
	req := SendTask{
		index:     idx,
		TimeStamp: timestamp,
		TaskData:  sendTask,
	}

	Log.Info("Ziptask完成任务的打包")
	return req
}

//往池子里加task
func AddSendTaskTopool(task SendTask) {
	GlobalSendPool.Poollist.PushBack(task)
	Log.Info("AddSendTaskTopool任务已经放到发送池里")
}

//后台运行协程  删除掉里面的信息
func DelExpiredTasksFromPool() {
	Log.Info("DelExpiredTasksFromPool后台程序已经启动")
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println("获取时区失败", err)
		return
	}
	for {
		SendPoolRWMutex.Lock()
		now := time.Now().In(loc)
		limit := now.Add(-20 * time.Second)
		count := 0
		for e := GlobalSendPool.Poollist.Front(); e != nil; {
			temp := e.Value
			if task, ok := temp.(SendTask); ok {
				// todo something with task
				if time.Unix(task.TimeStamp, 0).Before(limit) {
					count = count + 1
					next := e.Next()
					GlobalSendPool.Poollist.Remove(e)
					e = next
				} else {
					e = e.Next()
				}
			}
		}
		Log.Debug("本次清理后台过期任务：", count)
		SendPoolRWMutex.Unlock()
		time.Sleep(100 * time.Second)
	}
}

func AcceptSendTask() {
	Log.Info("AcceptSendTask后台发送广播服务已经启动")
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(10000) + 1
	for {
		taskmsg, ok := <-SendtaskChan
		Log.Info("Netpool收到任务")
		if !ok {
			Log.Error("SendtaskChan管道出现异常，将停止向pool中添加任务")
			return
		}
		index++
		//1.打包任务  这里应该是再次打包
		tsk := Ziptask(index, taskmsg)
		//2.发送到池子里面
		AddSendTaskTopool(tsk)
		//3.通知发送协程，有任务到达
		SendMessageToDialGoPool()
	}
}

var SendtaskChan chan []byte = make(chan []byte) //让SendtaskChan提醒有新的任务生成的管道
var NetPoolStopChan chan int                     //本netpool停止运行管道

//好像没有什么是需要被放到后台运行的 将这个全局chan放到这里
func Netpoolstart() {
	Log.Info("Netpool已经启动")
	//SendtaskChan = make(chan []byte, 200000)
	go DelExpiredTasksFromPool()
	go AcceptSendTask()
	<-NetPoolStopChan
	Log.Error("Netpool已经关闭")
	return
}

//chan表示信号到达，第二个是返回获取到的数据
func GetSendTaskfromPool(TaskArrived chan int, outmsgchan chan []byte) {
	index := -1
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		Log.Error("获取时区出错")
		return
	}

	for {
		<-TaskArrived
		SendPoolRWMutex.RLock()
		Log.Debug("将从发送池中读取数据")
		//这里业务逻辑暂时先读出来
		logget := 0
		loghas := 0
		for e := GlobalSendPool.Poollist.Front(); e != nil; {
			loghas++
			temp := e.Value
			task := temp.(SendTask)
			now := time.Now().In(loc)
			limit := now.Add(-2 * time.Minute)
			//首先判断index 只把大于当前index的进行处理
			if task.index <= index {
				e = e.Next()
			} else if time.Unix(task.TimeStamp, 0).Before(limit) {
				e = e.Next()
			} else {
				logget++
				index = task.index
				outmsgchan <- task.TaskData
				fmt.Println("从发送池里面获得数据,数据索引为：", task.index)
				e = e.Next()
			}
		}
		Log.Debug("遍历了", loghas, "个任务，得到有效任务", logget, "个")
		SendPoolRWMutex.RUnlock()
	}
}

//增加新的协程到连接建立池里面
func AddDialGoPool(up chan int, Down chan int) (index int /*在pool的下标*/) {
	GlobalDialconnPool.mutex.Lock()
	// up := make(chan int)
	// Down := make(chan int)
	GlobalDialconnPool.maxindex++
	index = GlobalDialconnPool.maxindex
	GlobalDialconnPool.notifyDownGoChan[index] = Down
	GlobalDialconnPool.notifyUpGoChan[index] = up
	GlobalDialconnPool.mutex.Unlock()
	Log.Debug("完成将本连接放入连接池内")
	return index
}
func DelDialGoPool(index int) {
	GlobalDialconnPool.mutex.Lock()
	Log.Info("将本连接从连接池中删除")
	delete(GlobalDialconnPool.notifyDownGoChan, index)
	delete(GlobalDialconnPool.notifyUpGoChan, index)
	GlobalDialconnPool.mutex.Unlock()
}

//向连接建立协程容器里所有的协程发送一个有新的任务达到需要去处理
func SendMessageToDialGoPool() {

	GlobalDialconnPool.mutex.Lock()
	count := 0
	for _, c := range GlobalDialconnPool.notifyDownGoChan {
		count++
		select {
		case c <- 1: //消息发送成功
			// default: //消息发送失败，说明已经关闭或者满了，这时候将其删除
			// 	Log.Error("发现有异常连接，已经关闭其通道")
			//DelDialGoPool(index)
		}
	}
	Log.Info("通知了", count, "个连接")
	GlobalDialconnPool.mutex.Unlock()
}
