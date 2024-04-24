package main

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/willf/bloom"
)

// 定义区块链结构(使用数组模拟区块链)
type BlockChain struct {
	db   *bolt.DB //用于存储数据
	tail []byte   //最后一个区块的哈希值
}

var BlockLen int64 //内存中区块链的长度，不准确，懒汉式，只有发现没有才会去磁盘中读，但是还需要原子性的增加
var MemoryBlockMap = make(map[string]*Block)
var MemoryTxMap = make(map[string]*Transaction)
var Memory1AllBloomMap = make(map[string]*bloom.BloomFilter) //键是block的hash
var Memory2BlockConfirmedListBloomMap = make(map[string]map[uuid.UUID]*bloom.BloomFilter)
var Memory3BlockFianlBloomMap = make(map[string]map[uuid.UUID]map[int]*bloom.BloomFilter)

var mutexBlockMap sync.Mutex //go不允许 多个线程同时写入map
var mutexTxMap sync.Mutex
var mutexbloom1 sync.Mutex
var mutexbloom2 sync.Mutex
var mutexbloom3 sync.Mutex

// 创世语
const genesisInfo = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

func init() {

}

//初始化到内存里面
/*
	初始化区块到内存里面  初始化侧链区块到内存里面
    要依赖next
*/
func InitMemoryMap() {
	startTime := time.Now()
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		Log.Info("GetSideUTXOBySourceId中GetBlockChainInstance失败")
		return
	}
	it := bc.NewIterator()
	if it.currentHash == nil {
		Log.Fatal("尾巴不存在")
	}
	blockcount := 0
	for {

		fmt.Println("正在更新主链区块的第", blockcount, "个")
		block := it.MemoryNextBlock()
		blockcount++
		if blockcount == 9900 {
			endExpTimeSteam = block.TimeStamp
		}
		if blockcount == 9902 {
			startExpTimeSteamp = block.TimeStamp
		}
		//fmt.Printf("%x", block.PrevHash)
		if block.PrevHash == nil {
			EXPSouceID = block.Product.SourceID.String()
			break
		}
	}
	endTime := time.Now()
	duration1 := endTime.Sub(startTime)
	Log.Info("加载入内存共用时间:", duration1)
}

// 创建区块，从无到有：这个函数仅执行一次   里面有创世区块的创建
func CreateBlockChain(address string, productmsg string) (err error) {
	// 1. 区块链不存在，创建   如果创世区块已经存在那么直接返回
	if isFileExist(blockchainDBFile) {
		Log.Debug("区块链文件已经开始存在！")
		err = fmt.Errorf("区块链文件已经开始存在！")
		return
	}
	DBmutex.Lock()
	db, err := bolt.Open(blockchainDBFile, 0600, nil)
	if err != nil {
		fmt.Println("Open失败！")
		err = fmt.Errorf("Open失败！")
		return
	}

	//不要db.Close，后续要使用这个句柄

	// 2. 开始创建  update依赖于一个函数，error是返回的错误，所以这个执行完成后后面有个判断
	//在这个函数里面如果bucket为空则去创建
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))

		//如果bucket为空，说明不存在
		if bucket == nil {
			//创建bucket
			_, err := tx.CreateBucket([]byte(bucketBlock))
			if err != nil {
				return err
			}

			//str := "hello world"
			//bytes := []byte(str)
			//time.Sleep(5 * time.Second)
			//SendtaskChan <- bytes

			//blocks := []*Block{genesisBlock}
			//PackBlcokArrTaskAndToChan(SendGenesisBlock, blocks)
		}
		return nil
	})
	db.Close()
	DBmutex.Unlock()
	//1. 创建挖矿交易
	//coinbase := NewCoinbaseTx(address, genesisInfo)

	fmt.Printf("NewCoinbaseTx address================= %x\n", getPubKeyHashFromAddress(address))
	//2. 拼装txs, txs []*Transaction
	//txs := []*Transaction{coinbase}
	txs := []*Transaction{}
	//3. 创建创世块

	genesisBlock := NewBlock(txs, nil, productmsg, nil)
	Log.Debug("开始序列化")
	//key是区块的哈希值，value是block的字节流

	blockdata := genesisBlock.Serialize()
	Log.Debug("序列化完成，等待更新数据库")
	createOrUpdateKeyValue(genesisBlock.Hash, blockdata)
	//更新最后区块哈希值到数据库
	createOrUpdateKeyValue([]byte(lastBlockHashKey), genesisBlock.Hash)
	Log.Info("传输数据给netpool")
	var blockArray []*Block
	blockArray = append(blockArray, genesisBlock)
	PackBlcokArrTaskAndToChan(SendMinedBlock, blockArray)
	return err //nil
}

//手动创建区块
func addBlockChain1(address string, productmsg string) error {
	// 1. 区块链不存在，创建   如果创世区块已经存在那么直接返回
	if !isFileExist(blockchainDBFile) {
		fmt.Println("区块链文件不存在！")
		CreateBlockChain(address, productmsg)
		return nil
	}
	Log.Debug("通过验证")
	//1. 创建挖矿交易  最终没有用这个 在NewBlock中形成一个新的交易
	//coinbase := NewCoinbaseTx(address, genesisInfo)

	fmt.Printf("NewCoinbaseTx address================= %x\n", getPubKeyHashFromAddress(address))
	//2. 拼装txs, txs []*Transaction
	//txs := []*Transaction{coinbase} //创建挖矿交易  最终没有用这个 在NewBlock中形成一个新的交易
	txs := []*Transaction{} //创建挖矿交易  最终没有用这个 在NewBlock中形成一个新的交易
	//3. 创建创世块
	prehashinfo := getfinalBlock()
	//block := Deserialize(prehashinfo)
	genesisBlock := NewBlock(txs, prehashinfo, productmsg, nil)
	Log.Debug("开始序列化")
	//key是区块的哈希值，value是block的字节流
	blockdata := genesisBlock.Serialize()
	err1 := createOrUpdateKeyValue(genesisBlock.Hash, blockdata)
	Log.Debug("errr", err1)
	//更新最后区块哈希值到数据库
	createOrUpdateKeyValue([]byte(lastBlockHashKey), genesisBlock.Hash)
	var blockArray []*Block
	blockArray = append(blockArray, genesisBlock)
	PackBlcokArrTaskAndToChan(SendMinedBlock, blockArray)
	Log.Info("传输数据给netpool")
	return nil
}

//修改后的，原来在GetBlockChainInstanceBEIFEN
func GetBlockChainInstance() (*BlockChain, error) {
	//获取实例时，要求区块链已经存在，否则需要先创建
	if isFileExist(blockchainDBFile) == false {
		Log.Fatal("区块链文件不存在，请先创建或进入网络同步")
		return nil, errors.New("区块链文件不存在，请先创建")
	}
	lastHash := getfinalBlock()

	//5. 拼成BlockChain然后返回
	bc := BlockChain{nil, lastHash}
	return &bc, nil
}

// 获取区块链实例，用于后续操作, 每一次有业务时都会调用================这里是原来的 现在修改乐
//在自己写业务的时候返回了bc，建议每次调用后，在本函数里面返回
func GetBlockChainInstanceBEIFEN() (*BlockChain, error) {

	//获取实例时，要求区块链已经存在，否则需要先创建
	if isFileExist(blockchainDBFile) == false {
		return nil, errors.New("区块链文件不存在，请先创建")
	}

	var lastHash []byte //内存中最后一个区块的哈希值
	options := &bolt.Options{
		Timeout:  time.Second * 5, // 设置超时时间为5秒
		ReadOnly: true,            // 以只读模式打开数据库
	}
	//两个功能：
	// 1. 如果区块链不存在，则创建，同时返回blockchain的示例

	db, err := bolt.Open(blockchainDBFile, 0600, options) //rwx  0110 => 6
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

// 提供一个向区块链中添加区块的方法   这个调用的时候不能挖矿  因为调用addBlockToBucket会死锁
func (bc *BlockChain) AddBlock(txs1 []*Transaction, productmsg string) error {
	//有效的交易放到这里，会添加到区块
	txs := []*Transaction{}

	Log.Debug("添加区块前，对交易进行校验...")
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
	newBlock := NewBlock(txs, lashBlockHash, productmsg, nil)
	var err error
	bc.tail, err = addBlockToBucket(newBlock)
	if err != nil {
		Log.Fatal("AddBlock失败：", err)
	}
	/*//2. 写入数据库
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
	})*/

	return err
}

// +++++++++++++++++++i迭代器,遍历区块链 +++++++++++++++++++++

// 定义迭代器
type Iterator struct {
	db          *bolt.DB
	currentHash []byte //不断移动的哈希值，由于访问所有区块
}

func (bc *BlockChain) closeDB() {
	if bc.db != nil {
		err := bc.db.Close()
		if err != nil {
			log.Println("Error closing database:", err)
		}
	}
	bc.db = nil // Set db to nil to avoid further usage
}

// 将Iterator绑定到BlockChain
func (bc *BlockChain) NewIterator() *Iterator {
	it := Iterator{
		db:          bc.db,
		currentHash: bc.tail,
	}

	return &it
}

// 给Iterator绑定一个方法：Next  修改后的版本
// 1. 返回当前所指向的区块
// 2. 向左移动（指向前一个区块）
func (it *Iterator) Next() (block *Block) {
	blockbyte, err2 := getValueByKey(it.currentHash)
	if err2 != nil {
		fmt.Println("iterator next err:", err2)
		return nil
	}
	block = Deserialize(blockbyte)
	it.currentHash = block.PrevHash //游标左移
	return block
}

//如果内存中存在那么就直接返回，否则就去磁盘里面拿
func GetBlockFromMemory(currentHash []byte) (block *Block) {
	mutexBlockMap.Lock()
	getBlock, exists := MemoryBlockMap[string(currentHash)]
	mutexBlockMap.Unlock()
	if exists {
		Log.Debug("内存中存在直接返回")
		return getBlock
	}
	fmt.Println("内存中不存在，进行磁盘io")
	block = getBlockByhash(currentHash)
	return block
}

//如果内存中存在那么就直接返回，否则就去磁盘里面拿
func GetTxFromMemory(currentHash []byte) (tx *Transaction) {
	mutexTxMap.Lock()
	tx, exists := MemoryTxMap[string(currentHash)]
	mutexTxMap.Unlock()
	if exists {
		Log.Debug("内存中存在直接返回")
		return tx
	}
	Log.Info("进入磁盘")
	tx = getTxByhash(currentHash)
	return tx
}

//最慢的方式
func getBlockByhash(curBlockcur []byte) *Block {
	//curBlockcur := getfinalBlock()
	curBlockhash, err := getValueByKey(curBlockcur)
	if curBlockhash != nil {
		curBlock := Deserialize(curBlockhash)
		return curBlock
	}
	if err != nil {
		Log.Warn(err)
		return nil

	}
	if curBlockhash == nil {
		return nil
	}
	return nil
}

//最慢的方式
func getTxByhash(curTXcur []byte) *Transaction {

	curTXcurhash, err := getValueByKey(curTXcur)
	if err != nil {
		//Log.Warn(err)
		return nil
	}
	if curTXcurhash == nil {
		return nil
	}
	curTx := Deserialize_tx(curTXcurhash)
	return curTx
}

func (it *Iterator) MemoryNextBlock() (block *Block) {
	/*
		如果在map中存在那么就直接返回，如果不存在那么获取将其加入map并返回
	*/
	//mutexBlockMap.Lock()
	getBlock, exists := MemoryBlockMap[string(it.currentHash)]
	//mutexBlockMap.Unlock()
	if exists {
		//Log.Info("内存中存在直接返回")
		it.currentHash = getBlock.PrevHash
		return getBlock
	}
	//不存在
	fmt.Println("内存中不存在，将进行磁盘io")
	block = getBlockByhash(it.currentHash)
	if block == nil {
		Log.Fatal("block为空,键为，", it.currentHash)
	}
	it.currentHash = block.PrevHash //游标左移
	Log.Debug("本区块磁盘io完成")

	//更新Memory
	UpdateMemory(block)
	atomic.StoreInt64(&BlockLen, 1) //增加长度
	return block
}
func UpdateMemory(block *Block) {

	//一、更新主链区块
	mutexBlockMap.Lock()
	MemoryBlockMap[string(block.Hash)] = block
	mutexBlockMap.Unlock()

	//二、更新布隆过滤器
	buf := bytes.NewReader(block.AllBlooms)
	var filter1 bloom.BloomFilter
	filter1.ReadFrom(buf)
	mutexbloom1.Lock()
	Memory1AllBloomMap[string(block.Hash)] = &filter1
	mutexbloom1.Unlock()

	for uuid2, ConfirmedBlockList := range block.ConfirmedLists {
		buf2 := bytes.NewReader(ConfirmedBlockList.Bloom)
		var filter2 bloom.BloomFilter
		filter2.ReadFrom(buf2)
		mutexbloom2.Lock()
		if Memory2BlockConfirmedListBloomMap[string(block.Hash)] == nil {
			Memory2BlockConfirmedListBloomMap[string(block.Hash)] = make(map[uuid.UUID]*bloom.BloomFilter)
		}
		Memory2BlockConfirmedListBloomMap[string(block.Hash)][uuid2] = &filter2
		mutexbloom2.Unlock()
		for i, ConfirmedBlock := range ConfirmedBlockList.BlockArray {
			buf3 := bytes.NewReader(ConfirmedBlock.Bloom)
			var filter3 bloom.BloomFilter
			filter3.ReadFrom(buf3)
			mutexbloom3.Lock()
			if Memory3BlockFianlBloomMap[string(block.Hash)] == nil {
				Memory3BlockFianlBloomMap[string(block.Hash)] = make(map[uuid.UUID]map[int]*bloom.BloomFilter)
			}
			if Memory3BlockFianlBloomMap[string(block.Hash)][uuid2] == nil {
				Memory3BlockFianlBloomMap[string(block.Hash)][uuid2] = make(map[int]*bloom.BloomFilter)
			}
			Memory3BlockFianlBloomMap[string(block.Hash)][uuid2][i] = &filter3
			mutexbloom3.Unlock()
		}
	}

	//Log.Info("更新主链区块")
	//三、更新侧链区块

	/*  这里为了节省时间，不再采用下面的方法
	//1.拿到侧链对应的区块数据
	hashpre := make(map[string][]byte)
	for _, confirmedBlockList := range block.ConfirmedLists {
		blockArray := confirmedBlockList.BlockArray
		for _, confirmedBlockType := range blockArray {
			hashpre[string(confirmedBlockType.Hash)] = confirmedBlockType.Hash
		}
	}
	hashget, _ := getValueByString2(hashpre)
	//2.反序列化并加入到内存中
	for _, sideblockinfo := range hashget {
		sideblock := Deserialize_tx(sideblockinfo)
		mutexTxMap.Lock()
		if sideblock != nil {
			MemoryTxMap[string(sideblock.TxHash)] = sideblock
		}
		mutexTxMap.Unlock()
	}*/

	//第二种更新侧链区块的方法
	txcount := 0
	for _, tx := range block.Transactions {
		txcount++
		mutexTxMap.Lock()
		MemoryTxMap[string(tx.TxHash)] = tx
		mutexTxMap.Unlock()
	}
	Log.Debug("更新侧链区块个数为:", txcount)

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

//得到主链区块的长度
func getMainBlockLength() (len int, err error) {
	if isFileExist(blockchainDBFile) == false {
		len = 0
		return
	}
	//如果最后一个区块都不存在那么就直接返回0
	_, errget := getValueByKey([]byte(lastBlockHashKey))
	if errget != nil {
		len = 0
		return
	}

	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Warn("getMainBlockLength长度出错")
		err = fmt.Errorf("getMainBlockLength长度出错")
		return
	}
	count := 0
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()
		count = count + 1
		//退出条件
		if block == nil || block.PrevHash == nil {
			break
		}
	}
	len = count
	return
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
				//fmt.Println("outputIndex:", outputIndex)

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
				Log.Debug("发现挖矿交易，无需遍历inputs")
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

/*
	1.先拿到最小交易数量的区块，放到map中
	2.验证区块
	  遍历区块，找到交易输入在交易池中的交易
	     验证是否在本map里面，如果在map里面，从map中删除这个交易，（  不做   然后再尝试添加元素，如果添加元素时没有了，那么就报数量不够错误，并退出）
		                    如果在交易池里，先删除这个交易，然后一直到这个引用的交易的最顶端，将最顶端往map中加，将其他元素放到一个这个特定的顶端标识的容器中

	  到这里可能比最小容量小，但不会比最大容量大
      遍历区块，获得这个交易是否合法
	    如果合法，就将这个元素放到容器中，然后判断是否有这个交易标识的容器，将容器中的内容从顶端到下面加入，一直加到满为止（其他已经验证通过的就直接放弃，为了简单）
		如果不合法，那么就删除这个元素。
	3.再次遍历区块
	   用特殊标记表示再次进入，如果引用引用的事这里面的，那么直接加入，否则再次加入判断

*/

// 签名函数
func (bc *BlockChain) signTransaction(tx *Transaction, priKey *ecdsa.PrivateKey) bool {
	Log.Debug("signTransaction开始签名交易...")
	pretxBlockinfo, err := getValueByKey(tx.PoI)
	if err != nil {
		Log.Warn("signTransaction的getValueByKey出错", err)
	}
	pretxBlock := Deserialize_tx(pretxBlockinfo)

	preScriptPubKeyHash := pretxBlock.TXOutputs[0].ScriptPubKeyHash
	return tx.signByPreScript(priKey, preScriptPubKeyHash)
}

//verifyTransaction函数使用时的转换器  第一个参数是暂存器  第二个参数是待打包区块中的所有交易  第三个参数是已经使用的usedtxto，输入只能使用一次
func (bc *BlockChain) verifyTxconverter(tx *Transaction, result map[string]*Transaction, count *int, usedTxto *map[string]int, verifytxsmap *map[uuid.UUID][]*Transaction) {
	//如果通过了验证，那么要这个交易引用的前面交易就不能用了
	Log.Debug("开始验证一个新的区块======")
	//每笔交易都有唯一的id，判断是否已经在区块里了
	txid := string(tx.TXID) //这个是交易区块的txid
	_, ok := result[txid]
	if ok { //这笔交易已经处理了
		Log.Debug("该交易已经被包含在了区块里面了")
		return
	}
	//到这里说明没有处理
	//交易引用的是交易池中的交易
	//下面是引用的区块
	_, ok2 := TxPool[string(tx.TXInputs[0].Txid)] //判断交易池中有这笔交易的输入吗
	if ok2 {                                      //输入在交易池中
		transactions := []*Transaction{tx}
		inputTxid := tx.TXInputs[0].Txid
		flag := 0
		for {
			_, ok3 := TxPool[string(inputTxid)] //判断是否有这个输入
			//如果在交易池中，继续添加  需要找到最早的交易
			if ok3 {
				Log.Debug("发现引用的是交易池中的交易")
				transactions = append(transactions, TxPool[string(inputTxid)])
				inputTxid = TxPool[string(inputTxid)].TXInputs[0].Txid
			} else if inputTxid == nil { //如果是溯源起点  直接退出（这个已经修改了，溯源起点也有txid，因此不会进入这个位置）
				Log.Debug("发现引用的是溯源起点")
				flag = 1
				break
			} else { //需要验证是否在区块中
				Log.Debug("检查是否在区块中")
				tempTx := transactions[len(transactions)-1]
				//第一个是验证是否有这个财产，第二个验证是否双花
				if bc.verifyTransaction(tempTx) && bc.verifyHaveSourceid(tempTx, usedTxto) {
					Log.Debug("区块验证通过，有效！！")
					//if bc.verifyTransaction(tempTx) {
					flag = 1
					break
				} else { //没有在区块中
					Log.Debug("区块验证没有通过！！")
					flag = -1
					break
				}
			}
		}
		if flag == 1 { //有效 将数组中的交易都加入到待打包区块的交易s中
			//transactions 越往后的元素，这笔交易就越早  所以for循环从后向前形成时间上的顺序
			for i := len(transactions) - 1; i >= 0; i-- {
				txx := transactions[i]
				result[string(txx.TXID)] = txx
				Log.Info("选取交易：")
				fmt.Printf("%x\n", string(txx.TXID))
				*count++

				arry := (*verifytxsmap)[txx.TXOutputs[0].SourceID]
				arry = append(arry, txx)
				(*verifytxsmap)[txx.TXOutputs[0].SourceID] = arry
				if *count == BlockhasBlnumMax {

					return
				}
			}
		} else { //都是无效区块 将区块都删除
			Log.Debug("由于没有验证通过将删除交易")
			for _, transaction := range transactions {
				fmt.Printf("从交易池中删除交易：%x\n", string(transaction.TXID))
				Log.Debug("详细信息：")
				printtxfromto(transaction)
				delete(TxPool, string(transaction.TXID))
			}

		}

	} else { //引用的是区块链上的交易   大部分都是这种情况
		Log.Debug("引用的是区块链上的交易")
		//验证通过
		hasMoney := bc.verifyTransaction(tx)              //是否有这笔钱  会遍历区块链
		hasSpended := bc.verifyHaveSourceid(tx, usedTxto) //是否已经花费  验证输入是否已经花费，没有花费就置为花费，花费就返回
		if hasMoney && hasSpended {
			//if bc.verifyTransaction(tx) {
			result[string(tx.TXID)] = tx
			Log.Debug("此交易验证成功，将加入区块中\n")
			//fmt.Printf("交易txid：%x，交易引用id：%x\n", tx.TXID, tx.TXInputs[0].Txid)
			*count++

			arry := (*verifytxsmap)[tx.TXOutputs[0].SourceID]
			arry = append(arry, tx)
			(*verifytxsmap)[tx.TXOutputs[0].SourceID] = arry
			if *count == BlockhasBlnumMax {
				return
			}
			return
		} else if hasMoney { //没有验证通过就删除
			Log.Info("交易已经使用了，判定为无效交易")
			delete(TxPool, string(tx.TXID))
			fmt.Printf("无效交易，将从交易池中删除交易：%x\n", string(tx.TXID))
			Log.Debug("详细信息：")
			printtxfromto(tx)
			return
		} else { //直接没有这笔交易的权限
			delete(TxPool, string(tx.TXID))
			fmt.Printf("无效交易，将从交易池中删除交易：%x\n", string(tx.TXID))
			Log.Debug("详细信息：")
			printtxfromto(tx)
			return
		}
	}

}

//验证输入是否已经花费，没有花费就置为花费
func (bc *BlockChain) verifyHaveSourceid(tx *Transaction, usedTxto *map[string]int) bool {
	//验证是否已经花费了
	Log.Debug("检查这笔资产是否已经花费了，避免双花交易")
	txid := string(tx.TXInputs[0].Txid)
	_, ok := (*usedTxto)[txid]
	if ok {
		Log.Warn("发现双花交易，返回并删除！")
		//fmt.Printf("双花交易引用：%x\n", txid)
		return false
	}
	(*usedTxto)[txid] = 1
	return true
}

////验证单笔交易并verify  这个函数主要是看这笔交易的输入方公钥是否拥有这笔财产
func (bc *BlockChain) verifyTransaction(tx *Transaction) bool {
	Log.Debug("验证是否拥有这笔资产...")

	//根据传递进来tx得到所有需要的前交易即可prevTxs
	//prevTxs := make(map[string]*Transaction)
	id := tx.TXOutputs[0].SourceID
	address := getPubKeyHashFromPubKey(tx.TXInputs[0].PubKey)
	exit, returnerr := hasAmount2Memory(id, address)
	if returnerr != nil {
		Log.Info("验证失败")
		return false
	}
	if exit {
		Log.Debug("验证通过，拥有这笔资产")
		return true
		//return tx.verify(prevTxs)
	}
	Log.Debug("验证失败")
	return false
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

//没哟使用
func SaveMainBlockToBucket(newBlock Block) {
	bc, err := GetBlockChainInstance()

	if err != nil {
		Log.Debug("send err:", err)
		return
	}

	defer bc.closeDB()

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
		Log.Info("加入新的区块到bucket 成功")
	} else {
		log.Fatal("加入新的区块到失败！！")
	}
}
