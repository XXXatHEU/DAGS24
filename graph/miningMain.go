package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	InterruptChan    = make(chan int)           //打断chan
	TxPool           map[string]*Transaction    //这里的string是交易的交易id
	UsetxsfromTxPool = make(map[uuid.UUID]bool) //从TxPool正在使用的交易，收到交易后判断是否在这里面，如果在的话需要发送停止挖矿命令
	TxPoolmutex      sync.Mutex
	//TxPacketPool      map[string]*Transaction //为了防止经常打断 只有暂存池tx超过一定数量才会加入TxPool
	//TxPacketPoolmutex sync.Mutex
	//TxPoolcond         *sync.Cond
	MiningControlStop  = make(chan int) //这个是关闭挖矿控制
	MiningTimeStart    = make(chan int) //如果开始了miningControl 这个会周期性的发送让其继续挖矿
	Miningexist        bool
	Miningexistmutex   sync.Mutex
	MiningControlExist bool
	tempSTOP           bool                 //暂时性的停止  在尝试打断时将会判断这个是否为true，如果为true，那么就不再打断 说明刚刚打断了
	LiveMining         = make(map[int]bool) //为了避免使用切片会加锁，这里还是用map吧
	Meave              = 0
	ProductPool        []ProductType
	ProductPoolMu      sync.Mutex
)

const (
	BlockhasBlnumMin = 2   //定义一个区块里面最小的交易数量
	BlockhasBlnumMax = 200 //最大交易数量
	TxPacketPoolMin  = 0   //只有超过一定数量才会通知

)

func init() {
	//TxPoolcond = sync.NewCond(&TxPoolmutex)
	TxPool = make(map[string]*Transaction)
	//TxPacketPool = make(map[string]*Transaction)
	Miningexist = false
	MiningControlExist = false
	tempSTOP = false
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

//判断某个uuid是否正在使用
func IsTransactionUUIDInUse(tx *Transaction) bool {
	uuid := tx.TXOutputs[0].SourceID
	_, ok := UsetxsfromTxPool[uuid]
	return ok
}

//向上封装  判断区块中的交易是否正在使用
func IsBlockUUIDInUse(block *Block) bool {
	txs := block.Transactions
	for _, tx := range txs {
		use := IsTransactionUUIDInUse(tx)
		if use {
			return true
		}
	}
	return false
}

//从交易池中删除，此函数会判断是否需要打断挖矿协程
//删除挖矿成功后的交易,
//第二个参数如果是false的话将不进行打断操作，如果是true的话将会有打断操作
//第三个参数是不管是否相关，不相关也会强制打断挖矿   本函数弃用，虽然处理更完善
func RemoveTxpooltxAndtrystop(block *Block, isInterrupt bool, forceInterrupt bool) {
	TxPoolmutexLock()
	Log.Info("更新交易池")
	//正在使用且并在挖矿
	existflag := false
	//如果正在使用而且正在挖矿那么就直接打断  非0将有打断操作
	Interrupt := false
	if IsBlockUUIDInUse(block) && isInterrupt {
		if MiningControlExist {
			Interrupt = true
		}
	}
	//需要打断挖矿的情况
	if (Interrupt || forceInterrupt) && MiningControlExist {
		if tempSTOP == false {
			fmt.Println("监听组发现挖矿控制协程存在，发送停止挖矿命令")

			//如果正在使用需要打断其挖矿协程  发送1,并且修改existflag
			// InterruptChan <- 1
			// existflag = true
			// tempSTOP = true //提示不要再打断了
		}

	}

	Log.Info("删除挖矿成功后的交易，删除的tx.TXID列表为:")
	for _, tx := range block.Transactions {
		fmt.Printf("%x  ", tx.TXID)
		delete(TxPool, string(tx.TXID))
	}
	fmt.Printf("\n现在交易池交易数量为:%d\n", len(TxPool))
	//如果给人家打断了需要重新启动挖矿
	if existflag {
		// tempSTOP = false
		// fmt.Printf("发送继续挖矿命令\n")
		// MiningTimeStart <- 1
	}
	TxPoolmutexUnlock()
}

//尝试唤醒
func activeMing() bool {
	//唤醒
	if MiningControlExist {
		MiningTimeStart <- 1 //发送启动挖矿命令
		if InterruptChan != nil {
			Log.Fatal("致命性错误，发生逻辑错误，未打断情况下竟然尝试唤醒")
		}
		InterruptChan = make(chan int)
		return true
	} else {
		Log.Warn("挖矿协程不存在")
		return false
	}

}

//尝试打断，如果打断了那么返回true
func trystopMing() bool {

	if MiningControlExist && InterruptChan != nil {
		fmt.Println("监听组发现挖矿控制协程存在，发送停止挖矿命令")

		//如果正在使用需要打断其挖矿协程  发送1,并且修改existflag
		InterruptChan <- 1
		InterruptChan = nil // 临时关闭
		return true
	}
	return false
}

//7月21日添加的删除pool交易
func RemoveTxpooltx(block *Block) {
	TxPoolmutexLock()
	Log.Info("更新交易池")
	Log.Info("删除挖矿成功后的交易，删除的tx.TXID列表为:")
	for _, tx := range block.Transactions {
		fmt.Printf("%x  ", tx.TXID)
		delete(TxPool, string(tx.TXID))
	}
	fmt.Printf("\n现在交易池交易数量为:%d\n", len(TxPool))
	TxPoolmutexUnlock()
}

//这个主要是ReceiveTxArr收到广播的交易后并且验证成功了就会放到这里面
func AddToTransactionPool(tx *Transaction) {
	TxPoolmutexLock()
	Log.Debug("向TxPacketPool中添加数据")
	//添加不需要有共斥关系
	key := string(tx.TXID)
	TxPool[key] = tx
	TxPoolmutexUnlock()
}

//尝试上锁给txpool
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
func getTxFromTxPool(ctx context.Context) (dd map[string]*Transaction, verifytxsmap map[uuid.UUID][]*Transaction) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("getTxFromTxPool发现已经取消，Operation cancelled")
			return nil, nil
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
					//清空原来的内容
					UsetxsfromTxPool = make(map[uuid.UUID]bool)

					Log.Warn(len(TxPool), "=======================")
					Log.Info("===========================开始打包")
					//获得到了一把锁
					var mp = make(map[string]*Transaction)
					var usedTxto = make(map[string]int)
					verifytxsmap = make(map[uuid.UUID][]*Transaction)
					flag := 0
					bc, err := GetBlockChainInstance()
					defer bc.closeDB()
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
						bc.verifyTxconverter(value, mp, &flag, &usedTxto, &verifytxsmap)

					}
					//设置uuid正在使用
					for key := range verifytxsmap {
						UsetxsfromTxPool[key] = true
					}
					TxPoolmutexUnlock()

					fmt.Println("getTxFromTxPool完成交易的获取，即将释放锁")
					Log.Info("最终打包的交易数量为：", len(mp))
					Log.Info("打包的简易信息为：")
					Log.Warn(len(verifytxsmap))
					for uuid, _ := range verifytxsmap {
						fmt.Printf("================uuid:%s", uuid.String())
					}
					for _, value := range mp {
						printtxfromto(value)
					}
					return mp, verifytxsmap

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
	verifytxsmaptemp := ctx.Value("verifytxsmap")
	verifytxsmap, ok := verifytxsmaptemp.(map[uuid.UUID][]*Transaction)
	if !ok {
		Log.Fatal("获取miningRealOver的chan出错 verifytxsmap")
	}
	Log.Warn("尝试得到bc")
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Fatal("MiningReal中 print err:", err)
		return
	}
	//打包区块  注意必须先转换成string，再转换成[]byte，不知道为什么直接传入[]byte后续会有问题
	cc := string(bc.tail)
	blocktemp, pow := MiningNewBlock(txarry, []byte(cc), verifytxsmap)
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
				//fmt.Println("来自realmining真实矿工的打印：")
				//fmt.Printf("=========block.Hash %x\n", block.Hash)
				//fmt.Printf("=========block.Nonce %x\n", block.Nonce)
				//fmt.Printf("=========block.PrevHash %x\n", block.PrevHash)
				Log.Debug("真实矿工向上返回区块")
				MiningRealOver <- block
				return
			}
		}
	}

}

//挖矿包工头
func MinningPMC(ctx context.Context) {
	Log.Info("MinningPMC挖矿包工头开始运行")
	tx, verifytxsmap := getTxFromTxPool(ctx)
	//将map转换为切片
	transactionSlice := make([]*Transaction, 0, len(tx))
	for _, transaction := range tx {
		transactionSlice = append(transactionSlice, transaction)
	}

	MiningRealOver := make(chan Block)
	ctx = context.WithValue(ctx, "MiningRealOver", MiningRealOver)
	ctx = context.WithValue(ctx, "tx", transactionSlice)
	ctx = context.WithValue(ctx, "verifytxsmap", verifytxsmap)

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
			Log.Debug("包工头收到完成挖矿，向上传递消息")
			// fmt.Printf("=========temp.Hash %x\n", temp.Hash)
			// fmt.Printf("=========temp.Nonce %x\n", temp.Nonce)
			//fmt.Printf("=========temp.PrevHash %x\n", temp.PrevHash)
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
			// blockhash, err3 := getValueByKey(block.PrevHash)
			// if err3 != nil {
			// 	Log.Fatal("MiningControl的getValueByKey出错")
			// }

			if bytes.Equal(getfinalBlock(), block.PrevHash) && InterruptChan != nil {
				Log.Info("======================区块状态正常")
				Log.Info("尝试上锁")
				DBfilemutex.Lock()
				//保存侧链上的区块
				saveConfirmedSideBlock(block)
				//挖矿控制收到挖取的区块，下面将调用广播协程
				//AppendBlocksToChain()
				SaveBlockAndBroadcast(block)
				//从交易池中删除挖矿成功后的交易
				RemoveTxpooltx(&block)
				//清除正在使用的uuid
				UsetxsfromTxPool = make(map[uuid.UUID]bool)
				printBlock(block)
				// fmt.Printf("=========block.Hash %x\n", block.Hash)
				// fmt.Printf("=========block.Nonce %x\n", block.Nonce)
				fmt.Printf("===============完成挖矿prehash值为:%x\n", block.PrevHash)
				//返回挖矿结果 <-
				//fmt.Printf("挖矿结果：%x\n", block)
				//继续新建挖矿 go
				DBfilemutex.Unlock()
			} else if InterruptChan == nil {
				Log.Warn("已经被打断，丢弃挖掉的区块")
			} else {

				fmt.Printf("buck中的 的前hash %x\n", getfinalBlock())
				fmt.Printf("挖矿成功的区块前一个hash %x\n", block.PrevHash)
				Log.Warn("区块状态发生改变，丢弃挖矿成功的区块")
			}

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
	//SaveMainBlockToBucket(newBlock)
	var blockArr []Block
	blockArr = append(blockArr, newBlock)
	AppendBlocksToChain(blockArr)
	var blockArray []*Block
	blockArray = append(blockArray, &newBlock)
	PackBlcokArrTaskAndToChan(SendMinedBlock, blockArray)
}
func saveConfirmedSideBlock(block Block) {

	//TxPoolmutexLock()
	for _, tx := range block.Transactions {
		Log.Info("存入侧链区块:")
		if tx.TxHash == nil {
			Log.Debug("tx.TxHash为nil")
			continue
		}
		fmt.Printf("tx.TxHash：%x\n", tx.TxHash)
		saveSideBlockToBucket(tx)
	}
	Log.Info("存入侧链区块完成")
	//TxPoolmutexUnlock()
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

//辅助函数 打印交易池中的交易数量
func PrintTxPool() {
	TxPoolmutex.Lock()
	Log.Info("TxPool中交易数量为:", len(TxPool))
	for _, value := range TxPool {
		printtx(value)
	}
	TxPoolmutex.Unlock()
}
