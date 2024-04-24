package main

import (
	"context"
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

const (
	BlockhasBlnumMin = 3   //定义一个区块里面最小的交易数量
	BlockhasBlnumMax = 200 //最大交易数量
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

func tryLock(mu *sync.Mutex, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan struct{}, 1)

	go func() {
		mu.Lock()
		select {
		case ch <- struct{}{}:
			//fmt.Println("tryLock尝试上锁成功")
		case <-ctx.Done(): // 上下文取消，放弃
			mu.Unlock()
			fmt.Println("tryLock上下文取消，放弃加锁")
		}
	}()

	select {
	case <-ch:
		return true // 成功获得锁
	case <-ctx.Done():
		return false // 加锁超时，放弃
	}
}
func getTxFromTxPool(ctx context.Context) (dd map[int]int) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("getTxFromTxPool发现已经取消，Operation cancelled")
			return nil
		default:
			//自定义了一个tryLock 如果一秒能够加锁成功
			if tryLock(&TxPoolmutex, 1*time.Second) {
				//defer TxPoolmutex.Unlock()
				if len(TxPool) < BlockhasBlnumMin {
					//fmt.Println("getTxFromTxPool完成锁的释放")
					TxPoolmutex.Unlock()
					continue
				} else {
					var mp = make(map[int]int)
					flag := 0
					for key, value := range TxPool {
						if flag == BlockhasBlnumMax {
							break
						}
						mp[key] = value
						flag++
					}
					fmt.Println("完成锁的释放")
					TxPoolmutex.Unlock()
					return mp
				}
			}
		}
	}

}

func miningReal(ctx context.Context) {
	//程序保证了可以在两秒左右返回  不会有饥饿现象 不会有僵尸协程三秒后还活着的情况

	miningRealOvertemp := ctx.Value("miningRealOver")
	miningRealOver, ok := miningRealOvertemp.(chan string)
	if !ok {
		fmt.Println("获取miningRealOver的chan出错")
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("miningReal收到信号被取消   Operation cancelled")
			return
		case <-time.After(4 * time.Second):
			//应该改成default
			fmt.Println("miningReal完成挖矿任务")
			miningRealOver <- "完成任务"
			return
		}
	}

	// dd, err := getTxFromTxPool(liveindex)
	// if err != nil {
	// 	fmt.Println("监测到我已经死亡，即将退出")
	// 	return
	// }
	// fmt.Println("进入挖矿过程，获取交易，挖矿中的交易为")
	// for key, _ := range dd {
	// 	fmt.Print(" ", key)
	// }
	// fmt.Println()
	// fmt.Println("挖矿中......")
	// time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	// if !AmIAlive(liveindex) {
	// 	fmt.Println("我已经死了")
	// 	return
	// }
	// fmt.Println("矿工已经完成挖矿，向上返回挖矿结果")
	// miningDone <- "挖矿成功"
}

//挖矿包工头
func minningPMC(ctx context.Context) {
	tx := getTxFromTxPool(ctx)
	miningRealOver := make(chan string, 1)
	ctx = context.WithValue(ctx, "miningRealOver", miningRealOver)
	ctx = context.WithValue(ctx, "tx", tx)
	minningPMCOvertemp := ctx.Value("minningPMCOver")
	minningPMCOver, ok := minningPMCOvertemp.(chan string)
	if !ok {
		fmt.Println("获取miningRealOver的chan出错")
	}
	go miningReal(ctx)
	for {
		select {
		//一旦收到打断信号，那么就停止 default
		case <-ctx.Done():
			fmt.Println("minningPMC收到信号被取消  Operation cancelled")
			return
		case temp := <-miningRealOver:
			fmt.Println("包工头收到完成挖矿，向上传递消息")
			minningPMCOver <- temp
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

// func newminingContext() (ctx context.Context) {
// 	ctx = context.Background()
// 	ctx = context.WithValue(ctx, "key", "value")
// 	ctx, _ = context.WithCancel(ctx)
// 	minningPMCOver = make(chan string, 1)
// 	ctx = context.WithValue(ctx, "minningPMCOver", minningPMCOver)
// }

// 挖矿控制协程
func miningControl(miningDone chan struct{}) {
	// 创建通道来发送挖矿计算结果
	//result := make(chan string)
	//挖矿协程的标识
	miningControlExist = true

	ctx := context.Background()            // 创建根上下文
	ctx, cancel := context.WithCancel(ctx) // 在根上下文的基础上创建可取消的上下文，并返回取消函数
	minningPMCOver := make(chan string, 1) //包工头完成后将区块转给这个通道
	ctx = context.WithValue(ctx, "minningPMCOver", minningPMCOver)
	defer cancel()

	//验证交易的有效性
	//交易验证成功后生成一个新的区块然后交给其进行计算
	go minningPMC(ctx)
	for {
		select {
		//一旦收到打断信号，那么就停止 default
		case <-InterruptChan:
			fmt.Println("挖矿控制收到中断信号，停止挖矿")
			cancel() //幂等函数

		case block := <-minningPMCOver:
			//挖矿成功
			fmt.Println("挖矿控制已经知道了，下面将调用广播协程", block)
			//返回挖矿结果 <-
			//继续新建挖矿 go
			fmt.Println("挖矿线程恢复，并开始新的挖矿计算")

			ctx = context.Background()
			ctx, cancel = context.WithCancel(ctx)
			minningPMCOver = make(chan string, 1)
			ctx = context.WithValue(ctx, "minningPMCOver", minningPMCOver)
			defer cancel()
			go minningPMC(ctx)
		case <-miningTimeStart:
			//继续挖矿，防止被干掉
			fmt.Println("从心跳处激活，挖矿线程恢复，并开始新的挖矿计算")
			cancel() //幂等函数

			ctx = context.Background()
			ctx, cancel = context.WithCancel(ctx)
			minningPMCOver := make(chan string, 1)
			ctx = context.WithValue(ctx, "minningPMCOver", minningPMCOver)
			defer cancel()
			go minningPMC(ctx)
		case <-miningControlStop:
			fmt.Print("停止挖矿控制协程")
			cancel()
			return
		}
	}
}

func DelTxPool(arry map[int]int) {
	for key, _ := range arry {
		delete(TxPool, key)
		fmt.Println("删除 :", key)
	}
}
func AddTxPool(arry map[int]int) {
	for key, value := range arry {
		TxPool[key] = value
		fmt.Println("添加 :", key)
	}
}
func Listen() {
	for {
		//模拟监听
		time.Sleep(2 * time.Second)
		//收到广播
		var mp = make(map[int]int)
		rand.Seed(time.Now().UnixNano())
		randomInt := rand.Intn(8)
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
		DelTxPool(mp)
		randomInt = rand.Intn(2)
		if randomInt == 0 {
			randomInt = rand.Intn(7)
			TxPool[randomInt] = randomInt
			fmt.Println("添加数据:", randomInt)
			randomInt = rand.Intn(7)
			TxPool[randomInt] = randomInt
			fmt.Println("添加数据:", randomInt)
		}
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
