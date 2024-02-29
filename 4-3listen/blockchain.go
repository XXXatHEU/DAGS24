package main

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
)

// 定义区块链结构(使用数组模拟区块链)
type BlockChain struct {
	db   *bolt.DB //用于存储数据
	tail []byte   //最后一个区块的哈希值
}

// 创世语
const genesisInfo = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
const blockchainDBFile = "blockchain.db"
const bucketBlock = "bucketBlock"           //装block的桶
const lastBlockHashKey = "lastBlockHashKey" //用于访问bolt数据库，得到最后一个区块的哈希值
var DBmutex sync.Mutex

// 创建区块，从无到有：这个函数仅执行一次   里面有创世区块的创建
func CreateBlockChain(address string) error {
	// 1. 区块链不存在，创建   如果创世区块已经存在那么直接返回
	if isFileExist(blockchainDBFile) {
		fmt.Println("区块链文件已经开始存在！")
		return nil
	}

	db, err := bolt.Open(blockchainDBFile, 0600, nil)
	if err != nil {
		return err
	}

	//不要db.Close，后续要使用这个句柄
	defer db.Close()

	// 2. 开始创建  update依赖于一个函数，error是返回的错误，所以这个执行完成后后面有个判断
	//在这个函数里面如果bucket为空则去创建
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))

		//如果bucket为空，说明不存在
		if bucket == nil {
			//创建bucket
			bucket, err := tx.CreateBucket([]byte(bucketBlock))
			if err != nil {
				return err
			}

			//1. 创建挖矿交易
			coinbase := NewCoinbaseTx(address, genesisInfo)
			//2. 拼装txs, txs []*Transaction
			txs := []*Transaction{coinbase}
			//3. 创建创世块
			genesisBlock := NewBlock(txs, nil)

			//key是区块的哈希值，value是block的字节流
			blockdata := genesisBlock.Serialize()
			bucket.Put(genesisBlock.Hash, blockdata) //将block序列化
			//更新最后区块哈希值到数据库
			bucket.Put([]byte(lastBlockHashKey), genesisBlock.Hash)
			Log.Info("传输数据给netpool")

			//str := "hello world"
			//bytes := []byte(str)
			//time.Sleep(5 * time.Second)
			//SendtaskChan <- bytes

			//blocks := []*Block{genesisBlock}
			//PackBlcokArrTaskAndToChan(SendGenesisBlock, blocks)
		}
		return nil
	})
	return err //nil
}

// 获取区块链实例，用于后续操作, 每一次有业务时都会调用
//在自己写业务的时候返回了bc，建议每次调用后，在本函数里面返回
func GetBlockChainInstance() (*BlockChain, error) {

	//获取实例时，要求区块链已经存在，否则需要先创建
	if isFileExist(blockchainDBFile) == false {
		return nil, errors.New("区块链文件不存在，请先创建")
	}

	var lastHash []byte //内存中最后一个区块的哈希值

	//两个功能：
	// 1. 如果区块链不存在，则创建，同时返回blockchain的示例
	db, err := bolt.Open(blockchainDBFile, 0600, nil) //rwx  0110 => 6
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//不要db.Close，后续要使用这个句柄

	// 2. 如果区块链存在，则直接返回blockchain示例
	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))

		//如果bucket为空，说明不存在
		if bucket == nil {
			return errors.New("bucket不应为nil")
		} else {
			//直接读取特定的key，得到最后一个区块的哈希值
			lastHash = bucket.Get([]byte(lastBlockHashKey))
		}

		return nil
	})

	//5. 拼成BlockChain然后返回
	bc := BlockChain{db, lastHash}
	return &bc, nil
}

// 提供一个向区块链中添加区块的方法
func (bc *BlockChain) AddBlock(txs1 []*Transaction) error {
	//有效的交易放到这里，会添加到区块
	txs := []*Transaction{}

	fmt.Println("添加区块前，对交易进行校验...")
	for _, tx := range txs1 {
		if bc.verifyTransaction(tx) {
			fmt.Printf("当前交易校验成功:%x\n", tx.TXID)
			txs = append(txs, tx)
		} else {
			fmt.Printf("当前交易校验失败:%x\n", tx.TXID)
		}
	}

	lashBlockHash := bc.tail //区块链中最后一个区块的哈希值

	//1. 创建区块
	newBlock := NewBlock(txs, lashBlockHash)

	//2. 写入数据库
	err := bc.db.Update(func(tx *bolt.Tx) error {
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

	return err
}

// +++++++++++++++++++i迭代器,遍历区块链 +++++++++++++++++++++

// 定义迭代器
type Iterator struct {
	db          *bolt.DB
	currentHash []byte //不断移动的哈希值，由于访问所有区块
}

// 将Iterator绑定到BlockChain
func (bc *BlockChain) NewIterator() *Iterator {
	it := Iterator{
		db:          bc.db,
		currentHash: bc.tail,
	}

	return &it
}

// 给Iterator绑定一个方法：Next
// 1. 返回当前所指向的区块
// 2. 向左移动（指向前一个区块）
func (it *Iterator) Next() (block *Block) {

	//读取Bucket当前哈希block
	err := it.db.View(func(tx *bolt.Tx) error {
		//读取bucket
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			return errors.New("Iterator Next时bucket不应为nil")
		}

		blockTmpInfo /*block的字节流*/ := bucket.Get(it.currentHash) //一定要注意，是currentHash
		block = Deserialize(blockTmpInfo)
		it.currentHash = block.PrevHash //游标左移

		return nil
	})
	//哈希游标向左移动

	if err != nil {
		fmt.Printf("iterator next err:", err)
		return nil
	}
	return
}

// 定义一个结构，包含output的详情：output本身，位置信息
type UTXOInfo struct {
	//交易id
	Txid []byte

	//索引值
	Index int64

	//output
	TXOutput
}

// 获取指定地址的金额,实现遍历账本的通用函数
// 给定一个地址，返回所有的utxo
func (bc *BlockChain) FindMyUTXO(pubKeyHash []byte) []UTXOInfo {
	//存储所有和目标地址相关的utxo集合
	// var utxos []TXOutput
	var utxoInfos []UTXOInfo

	//定义一个存放已经消耗过的所有的utxos的集合(跟指定地址相关的)
	spentUtxos := make(map[string][]int)

	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()

		//遍历交易
		for _, tx := range block.Transactions {
		LABEL:
			//1. 遍历output，判断这个output的锁定脚本是否为我们的目标地址
			for outputIndex, output := range tx.TXOutputs {
				// LABEL:
				fmt.Println("outputIndex:", outputIndex)

				//这里对比的是哪一些utxo与付款人有关系
				// if output.ScriptPubKeyHash /*某一个被公钥哈希锁定output*/ == pubKeyHash /*张三的哈希*/ {
				if bytes.Equal(output.ScriptPubKeyHash, pubKeyHash) {

					//开始过滤
					//当前交易id
					currentTxid := string(tx.TXID)
					//去spentUtxos中查看
					indexArray := spentUtxos[currentTxid]

					//如果不为零，说明这个交易id在篮子中有数据，一定有某个output被使用了
					if len(indexArray) != 0 {
						for _, spendIndex /*0, 1*/ := range indexArray {
							//接着判断下标
							if outputIndex /*当前的*/ == spendIndex {
								continue LABEL
							}
						}
					}

					//找到属于目标地址的output
					// utxos = append(utxos, output)
					utxoinfo := UTXOInfo{tx.TXID, int64(outputIndex), output}
					utxoInfos = append(utxoInfos, utxoinfo)
				}

			}

			//++++++++++++++++++++++遍历inputs+++++++++++++++++++++
			if tx.isCoinbaseTx() {
				//如果是挖矿交易，则不需要遍历inputs
				fmt.Println("发现挖矿交易，无需遍历inputs")
				continue
			}

			for _, input := range tx.TXInputs {
				// if input.PubKey /*付款人的公钥*/ == pubKeyHash /*张三的公钥哈希*/ {
				if bytes.Equal(getPubKeyHashFromPubKey(input.PubKey), pubKeyHash) {
					//map[key交易id][]int
					//map[string][]int{
					//	0x333: {0, 1}
					//}
					spentKey := string(input.Txid)

					//向篮子中添加已经消耗的output
					//往spentUtxos[spentKey]添加一个int(input.Index)
					spentUtxos[spentKey] = append(spentUtxos[spentKey], int(input.Index))
					// spentUtxos[0x333] =[]int{0}
					// spentUtxos[0x333] =[]int{0, 1}
					// spentUtxos[0x222] =[]int{0}

					//不要使用这种方式，否则spendUtxos不会被赋值
					// indexArray := spentUtxos[spentKey]
					// indexArray = append(indexArray, int(input.Index))
				}
			}

		}

		//退出条件
		if len(block.PrevHash) == 0 {
			break
		}
	}
	// return utxos
	return utxoInfos
}

// 输入：地址，金额
// 返回：map[string][]int, float64
func (bc *BlockChain) findNeedUTXO(pubKeyHash /*付款人的公钥哈希*/ []byte, targetSourceId uuid.UUID) (map[string][]int64, bool) {

	//两个返回值
	var retMap = make(map[string][]int64)
	// var retValue float64

	//1. 遍历账本，根据公钥哈希找到所有utxo
	utxoInfos := bc.FindMyUTXO(pubKeyHash)
	//{0x222, 0, output1}, {0x333, 0, output2}, {0x333, 1, output3}

	//2. 遍历utxo，统计当前总额，与amount比较
	for _, utxoinfo := range utxoInfos {
		// //统计当前utxo总额
		// retValue += utxoinfo.Value

		// //统计将要消耗的utxo
		// key := string(utxoinfo.Txid)
		// retMap[key] = append(retMap[key], utxoinfo.Index)

		// // > 如果大于等于amount直接返回
		// if retValue >= amount {
		// 	break
		// }
		// // > 反之继续遍历
		if utxoinfo.SourceID == targetSourceId {
			Log.Info("找到对应的SourceID")
			key := string(utxoinfo.Txid)
			retMap[key] = append(retMap[key], utxoinfo.Index)
			return retMap, true
		}
	}

	return retMap, false
}

// 签名函数
func (bc *BlockChain) signTransaction(tx *Transaction, priKey *ecdsa.PrivateKey) bool {
	fmt.Println("signTransaction开始签名交易...")
	//根据传递进来tx得到所有需要的前交易即可prevTxs
	prevTxs := make(map[string]*Transaction)
	// map[0x222] = tx1
	// map[0x333] = tx2

	//遍历账本，找到所有需要的交易集合
	for _, input := range tx.TXInputs {
		prevTx /*这个input引用的交易*/ := bc.findTransaction(input.Txid)
		if prevTx == nil {
			fmt.Println("没有找到有效引用的交易")
			return false
		}

		fmt.Println("找到了引用的交易")
		//容易出现的错误：tx.TXID
		// prevTxs[string(tx.TXID)] = prevTx <<<<====这个是错的
		prevTxs[string(input.Txid)] = prevTx
	}

	return tx.sign(priKey, prevTxs)
}

//verifyTransaction函数使用时的转换器  第一个参数是暂存器  第二个参数是待打包区块中的所有交易
func (bc *BlockChain) verifyTxconverter(tx *Transaction, result map[string]*Transaction, count *int, usedTxto *map[string]int) {
	//如果通过了验证，那么要这个交易引用的前面交易就不能用了

	txid := string(tx.TXID)
	_, ok := result[txid]
	if ok { //这笔交易已经处理了
		return
	}
	//到这里说明没有处理
	//交易引用的是交易池中的交易

	_, ok2 := TxPool[string(tx.TXInputs[0].Txid)] //判断交易池中有这笔交易的输入吗
	if ok2 {                                      //输入在交易池中
		transactions := []*Transaction{tx}
		inputTxid := tx.TXInputs[0].Txid
		flag := 0
		for {

			_, ok3 := TxPool[string(inputTxid)] //判断是否有这个输入
			//如果在交易池中，继续添加
			if ok3 {
				transactions = append(transactions, TxPool[string(inputTxid)])
				inputTxid = TxPool[string(inputTxid)].TXInputs[0].Txid
			} else if inputTxid == nil { //如果是溯源起点  直接退出
				flag = 1
				break
			} else { //需要验证是否在区块中
				tempTx := transactions[len(transactions)-1]
				if bc.verifyTransaction(tempTx) && bc.verifyHaveSourceid(tempTx, usedTxto) {
					flag = 1
					break
				} else { //没有在区块中
					flag = -1
					break
				}
			}
		}
		if flag == 1 { //有效 将数组中的交易都加入到待打包区块的交易s中
			for i := len(transactions) - 1; i >= 0; i-- {
				tx := transactions[i]
				result[string(tx.TXID)] = tx
				Log.Info("选取交易：")
				fmt.Printf("%x\n", string(tx.TXID))
				*count++
				if *count == BlockhasBlnumMax {

					return
				}
			}
		} else { //都是无效区块 将区块都删除
			for _, transaction := range transactions {
				fmt.Printf("从交易池中删除交易：%x\n", string(transaction.TXID))
				delete(TxPool, string(transaction.TXID))
			}

		}

	} else { //引用的是区块链上的交易
		//验证通过
		if bc.verifyTransaction(tx) && bc.verifyHaveSourceid(tx, usedTxto) {
			result[string(tx.TXID)] = tx
			Log.Info("选取交易：")
			fmt.Printf("%x\n", string(tx.TXID))
			*count++
			return
		} else { //没有验证通过就删除
			delete(TxPool, string(tx.TXID))
			fmt.Printf("从交易池中删除交易：%x\n", string(tx.TXID))
			return
		}
	}

}

//验证是否
func (bc *BlockChain) verifyHaveSourceid(tx *Transaction, usedTxto *map[string]int) bool {
	//验证是否已经花费了
	txid := string(tx.TXInputs[0].Txid)
	_, ok := (*usedTxto)[txid]
	if ok {
		Log.Info("发现双花交易，返回并删除！")
		return false
	}
	//验证引用的交易是否合法
	pubKeyHash := getPubKeyHashFromPubKey(tx.TXInputs[0].PubKey)
	targetSourceId := tx.TXOutputs[0].SourceID
	_, havetheid := bc.findNeedUTXO(pubKeyHash, targetSourceId)
	// map[0x222] = []int{0}
	// map[0x333] = []int{0,1}
	if !havetheid {
		fmt.Println("当前SourceID终点不为当前节点所有！")
		return false
	}
	(*usedTxto)[txid] = 1
	return true
}

// 校验单笔交易
func (bc *BlockChain) verifyTransaction(tx *Transaction) bool {
	fmt.Println("verifyTransaction开始...")

	if tx.isCoinbaseTx() {
		fmt.Println("发现挖矿交易，无需校验!")
		return true
	}
	//根据传递进来tx得到所有需要的前交易即可prevTxs
	prevTxs := make(map[string]*Transaction)

	//遍历账本，找到所有需要的交易集合
	for _, input := range tx.TXInputs {
		prevTx /*这个input引用的交易*/ := bc.findTransaction(input.Txid)
		if prevTx == nil {
			fmt.Println("没有找到有效引用的交易")
			return false
		}

		fmt.Println("找到了引用的交易")
		//容易出现的错误：tx.TXID
		// prevTxs[string(tx.TXID)] = prevTx <<<<====这个是错的
		prevTxs[string(input.Txid)] = prevTx
	}

	return tx.verify(prevTxs)
}

func (bc *BlockChain) findTransaction(txid []byte) *Transaction {
	//遍历区块，遍历账本，比较txid与交易id，如果相同，返回交易，反之返回nil
	it := bc.NewIterator()

	for {
		block := it.Next()

		for _, tx := range block.Transactions {
			//如果当前对比的交易的id与我们查找的交易的id相同，那么就找到了目标交易
			if bytes.Equal(tx.TXID, txid) {
				return tx
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
	return nil
}
