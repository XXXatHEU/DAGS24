package main

import (
	"context"
	"errors"
	"fmt"

	"sync"
	"time"

	"github.com/boltdb/bolt"
)

var (
	InterruptChan     = make(chan int)
	TxPool            map[string]*Transaction //这里的string是交易的交易id
	TxPoolmutex       sync.Mutex
	TxPacketPool      map[string]*Transaction //为了防止经常打断 只有暂存池tx超过一定数量才会加入TxPool
	TxPacketPoolmutex sync.Mutex
	//TxPoolcond         *sync.Cond
	MiningControlStop  = make(chan int) //这个是关闭挖矿控制
	MiningTimeStart    = make(chan int) //如果开始了miningControl 这个会周期性的发送让其继续挖矿
	Miningexist        bool
	Miningexistmutex   sync.Mutex
	MiningControlExist bool
	LiveMining         = make(map[int]bool) //为了避免使用切片会加锁，这里还是用map吧
	Meave              = 0
)

const (
	BlockhasBlnumMin = 5   //定义一个区块里面最小的交易数量
	BlockhasBlnumMax = 200 //最大交易数量
	TxPacketPoolMin  = 5   //只有超过一定数量才会通知
)

func init() {
	//TxPoolcond = sync.NewCond(&TxPoolmutex)
	TxPool = make(map[string]*Transaction)
	TxPacketPool = make(map[string]*Transaction)
	Miningexist = false
	MiningControlExist = false

}
func TxPoolmutexLock() {
	TxPoolmutex.Lock()
	Log.Debug("加锁=======================================")
}
func TxPoolmutexUnlock() {
	TxPoolmutex.Unlock()
	Log.Debug("解锁==========================================")
}
func AmIAlive(index int) (exist bool) {
	if !LiveMining[index] {
		return false
	} else {
		return true
	}
}

func tryLock(mu *sync.Mutex, timeout time.Duration) bool {
	Log.Debug("尝试获取锁")
	//新的一个ctx
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan int)

	go func() {
		//mu.Lock()
		TxPoolmutexLock()
		select {
		case <-ctx.Done(): // 上下文取消，放弃
			//mu.Unlock()
			TxPoolmutexUnlock()
			fmt.Println("tryLock上下文取消，放弃加锁")
		case ch <- 1:
			//fmt.Println("tryLock尝试上锁成功")
			return

		}
	}()
	for {
		select {
		case <-ch:
			return true
			// Log.Info("加锁成功")
			// select {
			// case <-ctx.Done():
			// 	// 超时时间到达，执行相应的处理逻辑
			// 	fmt.Println("Timeout exceeded")
			// default:
			// 	fmt.Printf("返回锁\n")
			// 	return true // 成功获得锁
			// }

		case <-ctx.Done():
			Log.Info("尝试加锁失败")
			return false // 加锁超时，放弃
		}
	}

}
func getTxFromTxPool(ctx context.Context) (dd map[string]*Transaction) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("getTxFromTxPool发现已经取消，Operation cancelled")
			return nil
		default:
			//自定义了一个tryLock 如果一秒能够加锁成功
			if tryLock(&TxPoolmutex, 100*time.Millisecond) {
				//defer TxPoolmutex.Unlock()
				if len(TxPool) < BlockhasBlnumMin {
					TxPoolmutexUnlock()
					fmt.Println("交易池中数据过少:", len(TxPool))
					time.Sleep(2 * time.Second)
					continue
				} else {

					Log.Warn(len(TxPool), "=======================")
					Log.Info("===========================开始打包")
					//获得到了一把锁
					var mp = make(map[string]*Transaction)
					var usedTxto = make(map[string]int)
					flag := 0
					bc, err := GetBlockChainInstance()
					defer bc.db.Close()
					if err != nil {
						Log.Fatal("getTxFromTxPool中执行GetBlockChainInstance出错 :", err)
						return
					}

					for _, value := range TxPool {
						if flag == BlockhasBlnumMax {
							Log.Info("已经达到打包的最大数量，退出！")
							break
						}

						//这里验证已形成区块的内容是否有这个交易由来
						bc.verifyTxconverter(value, mp, &flag, &usedTxto)
					}
					TxPoolmutexUnlock()
					fmt.Println("getTxFromTxPool完成交易的获取，即将释放锁")
					Log.Info("最终打包的交易数量为：", len(mp))
					return mp

				}
			}
		}
	}

}

func MiningReal(ctx context.Context) {
	//程序保证了可以在两秒左右返回  不会有饥饿现象 不会有僵尸协程三秒后还活着的情况
	Log.Info("MiningReal真实矿工开始工作")
	miningRealOvertemp := ctx.Value("MiningRealOver")
	MiningRealOver, ok := miningRealOvertemp.(chan Block)
	if !ok {
		Log.Fatal("获取MiningRealOver的chan出错")
	}
	txarrytemp := ctx.Value("tx")
	txarry, ok := txarrytemp.([]*Transaction)
	if !ok {
		Log.Fatal("获取miningRealOver的chan出错")
	}
	bc, err := GetBlockChainInstance()
	defer bc.db.Close()
	if err != nil {
		Log.Fatal("MiningReal中 print err:", err)
		return
	}
	//打包区块  注意必须先转换成string，再转换成[]byte，不知道为什么直接传入[]byte后续会有问题
	cc := string(bc.tail)
	blocktemp, pow := MiningNewBlock(txarry, []byte(cc))
	var block Block
	block = *blocktemp
	fmt.Printf("===============打包的prehash值为:%x\n", block.PrevHash)
	Log.Info("真实矿工完成区块的打包")
	var nonce uint64
	var hash [32]byte
	var succes bool
	nonce, hash, succes = pow.MiningRun(nonce, hash)
	for {
		select {
		case <-ctx.Done():
			Log.Info("miningReal真实矿工收到取消信号")
			return
		default:
			nonce, hash, succes = pow.MiningRun(nonce, hash)
			if succes {
				Log.Info("挖矿成功！！")
				block.Hash = hash[:]
				block.Nonce = nonce
				fmt.Println("来自realmining真实矿工的打印：")
				//fmt.Printf("=========block.Hash %x\n", block.Hash)
				//fmt.Printf("=========block.Nonce %x\n", block.Nonce)
				fmt.Printf("=========block.PrevHash %x\n", block.PrevHash)
				Log.Info("真实矿工向上返回区块")
				MiningRealOver <- block
				return
			}
		}
	}

}

//挖矿包工头
func MinningPMC(ctx context.Context) {
	Log.Info("MinningPMC挖矿包工头开始运行")
	tx := getTxFromTxPool(ctx)
	//将map转换为切片
	transactionSlice := make([]*Transaction, 0, len(tx))
	for _, transaction := range tx {
		transactionSlice = append(transactionSlice, transaction)
	}

	MiningRealOver := make(chan Block)
	ctx = context.WithValue(ctx, "MiningRealOver", MiningRealOver)
	ctx = context.WithValue(ctx, "tx", transactionSlice)

	minningPMCOvertemp := ctx.Value("MinningPMCOver")
	MinningPMCOver, ok := minningPMCOvertemp.(chan Block)
	if !ok {
		Log.Fatal("获取MinningPMCOver的chan出错")
	}
	go MiningReal(ctx)
	for {
		select {
		//一旦收到打断信号，那么就停止 default
		case <-ctx.Done():
			Log.Info("minningPMC包工头收到取消信号")
			return
		case temp := <-MiningRealOver:
			Log.Info("包工头收到完成挖矿，向上传递消息")
			// fmt.Printf("=========temp.Hash %x\n", temp.Hash)
			// fmt.Printf("=========temp.Nonce %x\n", temp.Nonce)
			fmt.Printf("=========temp.PrevHash %x\n", temp.PrevHash)
			MinningPMCOver <- temp
			return
		}
	}
}

// 挖矿控制协程
func MiningControl() {

	//挖矿协程存活的标识
	MiningControlExist = true

	ctx := context.Background()            // 创建根上下文
	ctx, cancel := context.WithCancel(ctx) // 在根上下文的基础上创建可取消的上下文，并返回取消函数
	MinningPMCOver := make(chan Block)     //包工头完成后将区块转给这个通道
	ctx = context.WithValue(ctx, "MinningPMCOver", MinningPMCOver)

	defer cancel()

	//运行包工头程序
	go MinningPMC(ctx)
	for {
		select {
		//一旦收到打断信号，那么就停止 default
		case <-InterruptChan:
			Log.Warn("挖矿控制收到中断信号，停止挖矿")
			cancel() //幂等函数

		case block := <-MinningPMCOver:
			//挖矿成功
			// realBlocktemp := ctx.Value("realBlock")
			// realBlock, ok := realBlocktemp.(Block)
			// if !ok {
			// 	Log.Fatal("MinningPMC获取realBlock出错")
			// }
			//挖矿控制收到挖取的区块，下面将调用广播协程
			SaveBlockAndBroadcast(block)
			//从交易池中删除挖矿成功后的交易
			RemoveMinedTransactions(block)

			// fmt.Printf("=========block.Hash %x\n", block.Hash)
			// fmt.Printf("=========block.Nonce %x\n", block.Nonce)
			fmt.Printf("===============完成挖矿prehash值为:%x\n", block.PrevHash)
			//返回挖矿结果 <-
			//fmt.Printf("挖矿结果：%x\n", block)
			//继续新建挖矿 go

			cancel() //幂等函数
			Log.Info("挖矿线程恢复，并开始新的挖矿计算")
			ctx = context.Background()
			ctx, cancel = context.WithCancel(ctx)
			MinningPMCOver = make(chan Block)
			Log.Info("新建MinningPMCOver")
			ctx = context.WithValue(ctx, "MinningPMCOver", MinningPMCOver)
			defer cancel()
			go MinningPMC(ctx)
		case <-MiningTimeStart:
			//继续挖矿，防止被干掉
			Log.Info("从心跳处激活，挖矿线程恢复，并开始新的挖矿计算")
			cancel() //幂等函数

			ctx = context.Background()
			ctx, cancel = context.WithCancel(ctx)
			MinningPMCOver := make(chan Block)
			ctx = context.WithValue(ctx, "MinningPMCOver", MinningPMCOver)
			go MinningPMC(ctx)
		case <-MiningControlStop:
			Log.Info("停止挖矿控制协程")
			MiningControlExist = false
			cancel()
			return
		}
	}
}
func SaveBlockAndBroadcast(newBlock Block) {
	bc, err := GetBlockChainInstance()

	if err != nil {
		fmt.Println("send err:", err)
		return
	}

	defer bc.db.Close()

	//2. 写入数据库
	err2 := bc.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			return errors.New("AddBlock时Bucket不应为空")
		}

		//key是新区块的哈希值， value是这个区块的字节流
		bucket.Put(newBlock.Hash, newBlock.Serialize())
		bucket.Put([]byte(lastBlockHashKey), newBlock.Hash)

		//更新bc的tail，这样后续的AddBlock才会基于我们newBlock追加
		bc.tail = newBlock.Hash
		return nil
	})
	if err2 != nil {
		Log.Info("更新新的区块成功")
	}
	var blockArray []*Block
	blockArray = append(blockArray, &newBlock)
	PackBlcokArrTaskAndToChan(SendMinedBlock, blockArray)
}

//删除挖矿成功后的交易
func RemoveMinedTransactions(block Block) {
	TxPoolmutexLock()
	Log.Info("删除挖矿成功后的交易，tx.TXID列表为:")
	for _, tx := range block.Transactions {
		delete(TxPool, string(tx.TXID))
		fmt.Printf("%x  ", tx.TXID)
	}
	fmt.Printf("\n现在交易池交易数量为:%d\n", len(TxPool))
	TxPoolmutexUnlock()
}

func DelTxPool(arry map[string]*Transaction) {
	for key, _ := range arry {
		delete(TxPool, key)
		fmt.Println("删除 :", key)
	}
}
func AddTxPool(arry map[string]*Transaction) {
	for key, value := range arry {
		TxPool[key] = value
		fmt.Println("添加 :", key)
	}
}

/*
func Listen() {
	for {
		//模拟监听
		time.Sleep(2 * time.Second)
		//收到广播
		var mp = make(map[string]Transaction)
		rand.Seed(time.Now().UnixNano())
		randomInt := rand.Intn(8)
		fmt.Println("监听到有事件发生")
		fmt.Println("随机整数:", randomInt)
		mp[string(randomInt)] = nil
		//拿到区块
		//验证区块是否合法
		// 获取交易池mutex   尝试清空通道后通知停止挖矿  更新交易池 释放交易池锁
		fmt.Println("监听组尝试上锁")
		TxPoolmutex.Lock()
		existflag := false
		if MiningControlExist {
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
func main_test() {
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
}*/

//辅助函数 打印交易池中的交易数量
func PrintTxPool() {
	TxPoolmutex.Lock()
	Log.Info("TxPool中交易数量为:", len(TxPool))
	TxPoolmutex.Unlock()
}
