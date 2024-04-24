package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	InterruptChan      = make(chan map[int]int)
	TxPool             map[int]int
	TxPoolmutex        sync.Mutex
	TxPoolcond         *sync.Cond
	miningControlStop  = make(chan int) //这个是关闭挖矿控制
	miningTimeStart    = make(chan int) //如果开始了miningControl 这个会周期性的发送让其继续挖矿
	miningexist        bool
	miningexistmutex   sync.Mutex
	miningControlExist bool
	LiveMining         = make(map[int]bool) //为了避免使用切片会加锁，这里还是用map吧
	leave              = 0
)

func init() {
	TxPoolcond = sync.NewCond(&TxPoolmutex)
	TxPool = make(map[int]int)
	miningexist = false
	miningControlExist = false

}
func AmIAlive(index int) (exist bool) {
	if !LiveMining[index] {
		return false
	} else {
		return true
	}
}

func getTxFromTxPool(liveindex int) (dd map[int]int, err error) {
	TxPoolmutex.Lock()
	var mp = make(map[int]int)
	if !AmIAlive(liveindex) {
		TxPoolmutex.Unlock()
		return nil, fmt.Errorf("我已经死了")
	}
	//一般是拿大于两个，如果不足了，
	for len(TxPool) < 5 {
		if !AmIAlive(liveindex) {
			TxPoolmutex.Unlock()
			return nil, fmt.Errorf("我已经死了")
		}
		fmt.Println("交易池交易过少，阻塞等待...")
		TxPoolcond.Wait() // 等待信号通知，并在等待期间释放互斥锁
		if !AmIAlive(liveindex) {
			TxPoolmutex.Unlock()
			return nil, fmt.Errorf("我已经死了")
		}
	}
	flag := 0
	for key, value := range TxPool {
		if flag == 4 {
			break
		}
		mp[key] = value
		flag++
	}
	TxPoolmutex.Unlock()
	return mp, nil
}

func miningReal(miningDone chan string, liveindex int) {
	dd, err := getTxFromTxPool(liveindex)
	if err != nil {
		fmt.Println("监测到我已经死亡，即将退出")
		return
	}
	fmt.Println("进入挖矿过程，获取交易，挖矿中的交易为")
	for key, _ := range dd {
		fmt.Print(" ", key)
	}
	fmt.Println()
	fmt.Println("挖矿中......")
	time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	if !AmIAlive(liveindex) {
		fmt.Println("我已经死了")
		return
	}
	fmt.Println("矿工已经完成挖矿，向上返回挖矿结果")
	miningDone <- "挖矿成功"
}

//挖矿包工头
func minningPMC(stop chan int, miningDone chan string, liveindex int) {
	//真实挖矿人给包工头发完成消息的chan
	leave++
	miningexistmutex.Lock()
	miningexist = true
	miningexistmutex.Unlock()
	var miningRealDone = make(chan string)
	//包工头打印下交易

	go miningReal(miningRealDone, liveindex)
	for {
		select {
		//一旦收到打断信号，那么就停止 default
		case <-stop:
			LiveMining[liveindex] = false
			fmt.Println("收到stop信号，minningPMC跑路了")
			miningexistmutex.Lock()
			miningexist = false
			miningexistmutex.Unlock()
			leave = leave - 1
			fmt.Println("挖矿剩余数量：", leave)
			return
			//如果子协程没有停止，那么继续挖,就是费点cpu而已
			//挖矿成功后即使发回消息也不会干扰上面大人的操作
		case temp := <-miningRealDone:
			fmt.Println("包工头收到完成挖矿，将向上要钱")
			miningDone <- temp
			miningexistmutex.Lock()
			miningexist = false
			miningexistmutex.Unlock()
			leave = leave - 1
			fmt.Println("挖矿剩余数量：", leave)
			return
		}
	}
}

//会不断的发送启动命令
// func startMining() {
// 	ticker := time.NewTicker(5 * time.Second)
// 	miningDone := make(chan struct{})
// 	go miningControl(miningDone)
// 	for {
// 		select {
// 		case <-ticker.C:
// 			fmt.Println("定时器触发")
// 			miningTimeStart <- 1
// 		}
// 	}
// }

// 挖矿控制协程
func miningControl(miningDone chan struct{}) {
	// 创建通道来发送挖矿计算结果
	//result := make(chan string)
	//挖矿协程的标识
	rand.Seed(time.Now().UnixNano())
	liveindex := rand.Intn(100000)
	miningControlExist = true
	stop := make(chan int)
	minintstop := make(chan string)
	//验证交易的有效性
	//交易验证成功后生成一个新的区块然后交给其进行计算
	go minningPMC(stop, minintstop, liveindex)
	LiveMining[liveindex] = true
	for {
		select {
		//一旦收到打断信号，那么就停止 default
		case <-InterruptChan:
			fmt.Println("挖矿线程收到中断信号，停止挖矿")
			miningexistmutex.Lock()
			if !miningexist {
				miningexistmutex.Unlock()
				continue
			}
			miningexistmutex.Unlock()
			stop <- 1

			// 处理中断信号，可以保存当前的计算状态或进度
			// ...
			fmt.Println("挖矿线程恢复，并开始新的挖矿计算")
			stop = make(chan int)
			go minningPMC(stop, minintstop, liveindex)

		case block := <-minintstop:
			//挖矿成功
			fmt.Println("负责人已经知道了", block)
			//返回挖矿结果 <-
			//继续新建挖矿 go
			fmt.Println("挖矿线程恢复，并开始新的挖矿计算")
			stop = make(chan int)
			go minningPMC(stop, minintstop, liveindex)
		case <-miningTimeStart:
			//继续挖矿，防止被干掉
			fmt.Println("从心跳处激活，挖矿线程恢复，并开始新的挖矿计算")
			stop = make(chan int)
			go minningPMC(stop, minintstop, liveindex)
		case <-miningControlStop:
			fmt.Print("停止挖矿控制协程")
			miningControlExist = false
			return
		}
	}
}

func updateTxPool(arry map[int]int) {
	for key, _ := range arry {
		delete(TxPool, key)
		fmt.Println("删除 :", key)
	}
}
func Listen() {
	for {
		//模拟监听
		time.Sleep(2 * time.Second)
		//收到广播
		var mp = make(map[int]int)
		rand.Seed(time.Now().UnixNano())
		randomInt := rand.Intn(5 + 1)
		fmt.Println("监听到有事件发生")
		fmt.Println("随机整数:", randomInt)
		mp[randomInt] = randomInt
		//拿到区块
		//验证区块是否合法
		// 获取交易池mutex   尝试清空通道后通知停止挖矿  更新交易池 释放交易池锁
		fmt.Println("监听组尝试上锁")
		TxPoolmutex.Lock()
		existflag := false
		if miningControlExist {
			fmt.Println("监听组发现挖矿控制协程存在，发送停止挖矿命令")
			InterruptChan <- mp
			existflag = true
		}
		updateTxPool(mp)
		fmt.Println("完成交易池更新过程")
		fmt.Println("现在交易池中交易的数量:", len(TxPool))

		if len(TxPool) > 5 {
			TxPoolcond.Broadcast()
			fmt.Println("广播有新的交易生成")
		}
		TxPoolmutex.Unlock()
		fmt.Println("lisnten放弃交易池的锁")
		if miningControlExist {
			fmt.Println("监听组发现挖矿控制协程存在，发送停止挖矿命令")
			InterruptChan <- mp
		}
		//如果人家本来在挖矿，那你这里就要给人家重新启动啊
		if existflag {
			miningTimeStart <- 1
		}
		//继续监听
	}

}
func main() {
	TxPool[1] = 1
	TxPool[2] = 2
	TxPool[3] = 3
	TxPool[4] = 4
	TxPool[5] = 5
	TxPool[6] = 6
	TxPool[7] = 7
	// 挖矿线程

	miningDone := make(chan struct{})
	go miningControl(miningDone)
	//go startMining()
	go Listen()
	time.Sleep(50000 * time.Second)
}
