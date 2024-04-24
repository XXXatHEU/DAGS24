package main

import (
	"bytes"
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/willf/bloom"
)

const MaxExpBlockLen = 10000 //最大测量距离
var maxExpTimeStamp uint64   //最大时间跳跃距离
var endExpTimeSteam uint64
var startExpTimeSteamp uint64

/*

区块头和区块主体是分开的，每个区块头只包含了元数据，而区块主体包含了一系列的交易记录



Getsocuid对于MH会判断布隆过滤器里是否存在，但由于存在序列化和反序列的过程，因此甚至比直接遍历会花费更长的时间
BT会序列化交易，将区块主体全部反序列后才会
而BT需要每次都要对数据进行处理，将数据进行切片来查看是否有相同的数据，所以相比正常的溯源 时间会长一点

TimeRange 结果正常



haveid 对于MH只能挨个区块遍历 所以和BT一样====================
对于图区块链的遍历添加了额外的的反序列化过程，本身go语言的在序列化和反序列化这一块是传入一个类型，他会将数据转换成这种类型（并没有）
那么1024个交易就会造成一些额外的负担


*/

/*
1.socuid
   Graph 应该 远远小于


2.haveid时间
	看能够有大的改善吗

3.queryword
    看能够有大的改善吗



*/

func etime2() {
	numbers := []int{128, 256, 512, 1024, 2048}
	//numbers := []int{256}

	for _, number := range numbers {

		blockchainDBFile = "/goworkplace/6exp/" + strconv.Itoa(number) + "/blockchain.db"
		Log.Info(blockchainDBFile)
		etime2()
	}
}

func etime() {
	//SetExpSouceid()
	// go func() {
	// 	Log.Info(http.ListenAndServe("localhost:6060", nil))
	// }()

	Log.Info("时间测量")

	//Log.Info("GraphQueryWithKeywordAttributes函数运行时间：", GraphQueryWithKeywordAttributes("爱奇艺"))
	//Log.Info("BTQueryWithKeywordAttributes函数运行时间：", BTQueryWithKeywordAttributes("爱奇艺"))
	// Log.Info("MHQueryWithKeywordAttributes函数运行时间：", MHQueryWithKeywordAttributes("爱奇艺"))

	// Log.Info("BTGetBlocksInTimeRange函数运行时间：", BTGetBlocksInTimeRange("1696447660", "1696457660"))
	// Log.Info("GraphGetBlocksInTimeRange函数运行时间：", GraphGetBlocksInTimeRange("1696447660", "1696457660"))
	// Log.Info("MHGetBlocksInTimeRange函数运行时间：", MHGetBlocksInTimeRange("1696447660", "1696457660"))

	// Log.Info("BTGetsocuid函数运行时间：", BTGetsocuid("8cabcfe4-1721-4dd5-adf6-dafb07a56f1f"))
	//Log.Info("GraphGetsocuid函数运行时间：", GraphGetsocuid("8cabcfe4-1721-4dd5-adf6-dafb07a56f1f"))
	//Log.Info("MHGetsocuid函数运行时间：", MHGetsocuid("8cabcfe4-1721-4dd5-adf6-dafb07a56f1f"))

	//137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR
	//mywallet.Address
	// Log.Info("Graphhaveid2函数运行时间：", Graphhaveid2("137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR"))
	// Log.Info("BThaveid函数运行时间：", BThaveid("137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR"))
	// // //Log.Info("Graphhaveid函数运行时间：", Graphhaveid("137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR"))
	// Log.Info("MHhaveid函数运行时间：", MHhaveid("137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR"))
	//TimeList()
	TimeList()
	TimeList()
	TimeList()
	Log.Info("时间运行完毕")
	Log.Fatal("==============================================")

}

//将会在最后结果中列出所有时间
func TimeList() {
	fmt.Println("==============测试查询某个关键词=======")
	duration2 := BTQueryWithKeywordAttributes("爱奇艺")
	fmt.Println("BTQueryWithKeywordAttributes函数运行时间：", duration2)
	duration21 := MHQueryWithKeywordAttributes("爱奇艺")
	duration1 := GraphQueryWithKeywordAttributes("爱奇艺")
	fmt.Println("GraphQueryWithKeywordAttributes函数运行时间：", duration1)

	fmt.Println("BTQueryWithKeywordAttributes函数运行时间：", duration2)
	fmt.Println("MHQueryWithKeywordAttributes函数运行时间：", duration21)
	fmt.Println("GraphQueryWithKeywordAttributes函数运行时间：", duration1)
	fmt.Println("===========测试寻找某个范围的区块时间===================")

	duration3 := BTGetBlocksInTimeRange("1697862699", "1697862699")
	duration4_1 := MHGetBlocksInTimeRange("1697862699", "1697862699")
	duration4 := GraphGetBlocksInTimeRange("1697862699", "1697862699")

	// 输出运行时间
	fmt.Println("BTGetBlocksInTimeRange函数运行时间：", duration3)
	fmt.Println("MHGetBlocksInTimeRange函数运行时间：", duration4_1)
	fmt.Println("GetBlocksInTimeRange函数运行时间：", duration4)

	fmt.Println("===========测试溯源某个id区块时间===================")
	duration5 := BTGetsocuid(EXPSouceID)
	duration6_1 := MHGetsocuid(EXPSouceID)
	duration6 := GraphGetsocuid(EXPSouceID)

	// 输出运行时间
	fmt.Println("BTGetsocuid函数运行时间：", duration5)
	fmt.Println("MHGetsocuid函数运行时间：", duration6_1)
	fmt.Println("GraphGetsocuid函数运行时间：", duration6)

	duration7 := BThaveid("137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR")
	duration8_1 := MHhaveid("137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR")
	duration8 := Graphhaveid2("137SkQh4aC2Db1pAXDL29ixjnYjgEzx4bR")

	// 输出运行时间
	Log.Info("======时间汇总====================================")

	Log.Info("BTQueryWithKeywordAttributes函数运行时间：", duration2)
	Log.Info("MHQueryWithKeywordAttributes函数运行时间：", duration21)
	Log.Info("GraphQueryWithKeywordAttributes函数运行时间：", duration1)

	Log.Info("BTGetBlocksInTimeRange函数运行时间：", duration3)
	Log.Info("MHGetBlocksInTimeRange函数运行时间：", duration4_1)
	Log.Info("GraphGetBlocksInTimeRange函数运行时间：", duration4)

	Log.Info("BTGetsocuid函数运行时间：", duration5)
	Log.Info("MHGetsocuid函数运行时间：", duration6_1)
	Log.Info("GraphGetsocuid函数运行时间：", duration6)

	Log.Info("BThaveid函数运行时间：", duration7)
	Log.Info("MHhaveid函数运行时间：", duration8_1)
	Log.Info("Graphhaveid函数运行时间：", duration8)
	//Log.Fatal("==============================================")
}

func BThaveid(myaddress string) (duration21 time.Duration) {
	//func (bc *BlockChain) FindMyUTXO(pubKeyHash []byte) []UTXOInfo {
	//存储所有和目标地址相关的utxo集合
	// var utxos []TXOutput
	startTimeRecode := time.Now()
	var utxoInfos []Transaction
	pubKeyHash := getPubKeyHashFromAddress(myaddress)
	//定义一个存放已经消耗过的所有的utxos的集合(跟指定地址相关的)
	spentUtxos := make(map[string][]int)
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Fatal("getMainBlockLength长度出错")
		err = fmt.Errorf("getMainBlockLength长度出错")
		return
	}
	count := 0
	it := bc.NewIterator()
	count1 := 0
	for {
		//遍历区块
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		//遍历交易
		txs := block.Transactions
		hashget := make(map[*Transaction][]byte)
		for _, tx := range txs {
			hashget[tx] = tx.TxHash
		}
		hashget, _ = getTxValuesByKeys(hashget)
		for tx, _ := range hashget {
			tx = Deserialize_tx(hashget[tx])
		LABEL:

			//1. 遍历output，判断这个output的锁定脚本是否为我们的目标地址
			for outputIndex, output := range tx.TXOutputs {
				if bytes.Equal(output.ScriptPubKeyHash, pubKeyHash) {
					//找到属于目标地址的output
					// utxos = append(utxos, output)
					//if TX != nil {
					utxoInfos = append(utxoInfos, *tx)
					//printtx(TX)
					count++
					//}
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
								//continue
							}
						}
					}

				}

			}

			//++++++++++++++++++++++遍历inputs+++++++++++++++++++++
			// if tx.isCoinbaseTx() {
			// 	//如果是挖矿交易，则不需要遍历inputs
			// 	//fmt.Println("发现挖矿交易，无需遍历inputs")
			// 	continue
			// }
			//遍历输入的  赚钱->花钱  这里把赚的前放到
			for _, input := range tx.TXInputs {
				// if input.PubKey /*付款人的公钥*/ == pubKeyHash /*张三的公钥哈希*/ {
				if bytes.Equal(getPubKeyHashFromPubKey(input.PubKey), pubKeyHash) {
					spentKey := string(input.Txid)
					spentUtxos[spentKey] = append(spentUtxos[spentKey], int(input.Index))
				}
			}
		}
		//退出条件
		if len(block.PrevHash) == 0 {
			break
		}
	}
	// return utxos
	fmt.Println("个数为", count, "遍历次数为", count1)
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return
}

func MHhaveid(myaddress string) (duration21 time.Duration) {
	startTimeRecode := time.Now()
	var utxoInfos []Transaction
	pubKeyHash := getPubKeyHashFromAddress(myaddress)
	//定义一个存放已经消耗过的所有的utxos的集合(跟指定地址相关的)
	spentUtxos := make(map[string][]int)
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Fatal("getMainBlockLength长度出错")
		err = fmt.Errorf("getMainBlockLength长度出错")
		return
	}
	it := bc.NewIterator()
	count1 := 0
	for {
		//遍历区块
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		buf := bytes.NewReader(block.AllBlooms)
		var filter bloom.BloomFilter
		filter.ReadFrom(buf)

		if filter.Test([]byte(myaddress)) {
			//遍历交易
			txs := block.Transactions
			hashget := make(map[*Transaction][]byte)
			for _, tx := range txs {
				hashget[tx] = tx.TxHash
			}
			hashget, _ = getTxValuesByKeys(hashget)
			for tx, _ := range hashget {
				tx = Deserialize_tx(hashget[tx])
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

						if tx != nil {
							utxoInfos = append(utxoInfos, *tx)
							//printtx(TX)
						}

					}

				}

				//++++++++++++++++++++++遍历inputs+++++++++++++++++++++
				if tx.isCoinbaseTx() {
					//如果是挖矿交易，则不需要遍历inputs
					//fmt.Println("发现挖矿交易，无需遍历inputs")
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
					}
				}
			}
		}

		//退出条件
		if len(block.PrevHash) == 0 {
			break
		}
	}
	// return utxos
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return

}

func Graphhaveid(myaddress string) (duration21 time.Duration) {
	count := 0
	startTimeRecode := time.Now()

	var myhaveid []uuid.UUID
	//获取池子中已经使用的
	var usedmap = make(map[uuid.UUID]int)
	/*TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()*/
	needpubhash := getPubKeyHashFromAddress(myaddress)
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "getMyhaveId中GetBlockChainInstance失败"
		Log.Fatal(ms)
		return
	}
	//只有第一次没用过的才属于最终拥有着 前面都是已经花出去了
	var hased = make(map[uuid.UUID]int)

	//遍历区块 找到
	it := bc.NewIterator()
	count1 := 0
	for {
		//遍历区块
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		// startTimeRecode555 := time.Now()
		for uuid_, confirmedBlockList := range block.ConfirmedLists {
			_, ok := hased[uuid_]
			if ok {
				continue
			}
			//使用了才能加这个
			//hased[uuid_] = 1
			blockArray := confirmedBlockList.BlockArray
			finalConfirmedBlock := blockArray[len(blockArray)-1]
			finalblockhash := finalConfirmedBlock.Hash
			sideblockinfo, err := getValueByKey(finalblockhash)
			if err != nil {
				Log.Warn("GetSideUTXOBySourceId出现错误，", err)
			}
			sideblock := Deserialize_tx(sideblockinfo)
			// fmt.Printf("sideblock.TXOutputs[0].ScriptPubKeyHash %x\n", sideblock.TXOutputs[0].ScriptPubKeyHash)
			// fmt.Printf("pubKeyHash %x\n", needpubhash)
			// if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, needpubhash) {
			// 	Log.Fatal("找到相同的了！！！")
			// }
			if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, needpubhash) {
				//Log.Info("==========================================================")
				//感觉并不需要这个usedmap
				_, existss := usedmap[uuid_]
				if existss {
					//Log.Info("该id已经转移:", existss)
					continue
				}
				usedmap[uuid_] = 1
				myhaveid = append(myhaveid, uuid_)
				// endTimeRecode := time.Now()
				// duration21 = endTimeRecode.Sub(startTimeRecode)
				// Log.Info("从特殊地方返回")
				// return //这里当初写的时候为什么加return呢
			}
		}
		// endTimeRecode := time.Now()
		// duration555555 := endTimeRecode.Sub(startTimeRecode555)
		// fmt.Println("=================", duration555555)

		//验证起始点是否是自己的 要求保证这个没有用过
		//fmt.Printf("block.Product.Pubkeyhash\n	%x\n", block.Product.Pubkeyhash)
		//fmt.Printf("blockpubhash\n	%x\n", blockpubhash)
		if bytes.Equal(block.Product.Pubkeyhash, needpubhash) {
			_, ok := hased[block.Product.SourceID]
			if ok {
			} else { //没有使用过
				_, existss := usedmap[block.Product.SourceID]
				if existss {
					//Log.Info("该id已经转移:", existss)
				} else {
					count++
					usedmap[block.Product.SourceID] = 1
					hased[block.Product.SourceID] = 1
					myhaveid = append(myhaveid, block.Product.SourceID)
				}
			}
		}
		if block.PrevHash == nil {
			break
		}
	}

	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	fmt.Println("graph得到个数为", count)
	return
}
func Graphhaveid2(myaddress string) (duration21 time.Duration) {
	count := 0
	startTimeRecode := time.Now()

	var myhaveid []uuid.UUID
	//获取池子中已经使用的
	var usedmap = make(map[uuid.UUID]int)
	/*TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()*/
	needpubhash := getPubKeyHashFromAddress(myaddress)
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "getMyhaveId中GetBlockChainInstance失败"
		Log.Fatal(ms)
		return
	}
	//只有第一次没用过的才属于最终拥有着 前面都是已经花出去了
	var hased = make(map[uuid.UUID]int)
	count1 := 0
	//遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		// startTimeRecode555 := time.Now()
		//hashget := make(map[uuid.UUID][]byte)
		for uuid_, confirmedBlockList := range block.ConfirmedLists {
			_, ok := hased[uuid_]
			if ok {
				continue
			}
			if bytes.Equal(confirmedBlockList.FinalBelongPubKeyHash, needpubhash) {
				usedmap[uuid_] = 1
				hased[uuid_] = 1
				myhaveid = append(myhaveid, uuid_)
				count++
			}
			//使用了才能加这个
			//hased[uuid_] = 1
		}

		// endTimeRecode := time.Now()
		// duration555555 := endTimeRecode.Sub(startTimeRecode555)

		//验证起始点是否是自己的 要求保证这个没有用过
		//fmt.Printf("block.Product.Pubkeyhash\n	%x\n", block.Product.Pubkeyhash)
		//fmt.Printf("blockpubhash\n	%x\n", blockpubhash)
		if bytes.Equal(block.Product.Pubkeyhash, needpubhash) {
			_, ok := hased[block.Product.SourceID]
			if ok {
			} else { //没有使用过
				_, existss := usedmap[block.Product.SourceID]
				if existss {
					//Log.Info("该id已经转移:", existss)
				} else {
					count++
					usedmap[block.Product.SourceID] = 1
					hased[block.Product.SourceID] = 1
					myhaveid = append(myhaveid, block.Product.SourceID)
				}
			}
		}
		if block.PrevHash == nil {
			break
		}
	}

	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	fmt.Println("graph得到个数为", count)
	return
}

func BTGetsocuid(targetSourceId string) (duration21 time.Duration) {
	startTimeRecode := time.Now()
	targetSourceId = strings.TrimSpace(targetSourceId)
	targetid, err := uuid.Parse(targetSourceId)
	if err != nil {
		Log.Fatal("无法将字符串转换为UUID:", err)
		return
	}
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Fatal(ms)
		return
	}
	//根据uuid进行数据溯源
	it := bc.NewIterator()
	var res []Transaction
	count := 0
	for {
		//遍历区块
		block := it.Next()
		count++
		if count >= MaxExpBlockLen {
			break
		}
		if block == nil {
			Log.Fatal("出现空区块")
		}
		txs := block.Transactions
		hashget := make(map[*Transaction][]byte)
		for _, tx := range txs {
			hashget[tx] = tx.TxHash
		}
		hashget, _ = getTxValuesByKeys(hashget)
		for tx, _ := range hashget {
			tx = Deserialize_tx(hashget[tx])
			if tx.TXOutputs[0].SourceID != targetid {
				continue
			}
			//这里直接加入 并没有做判断
			//if TX != nil {
			transaction := *tx
			res = append(res, transaction)
			count++
			//printtx(TX)
			//}
		}
		//fmt.Println(block.PrevHash)
		if block.PrevHash == nil {
			break
		}
	}
	fmt.Println("共转移", count, "次")
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return
}

//
func MHGetsocuid(targetSourceId string) (duration21 time.Duration) {
	startTimeRecode := time.Now()
	targetSourceId = strings.TrimSpace(targetSourceId)
	targetid, err := uuid.Parse(targetSourceId)
	if err != nil {
		Log.Fatal("无法将字符串转换为UUID:", err)
		return
	}
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Fatal(ms)
		return
	}
	//根据uuid进行数据溯源
	it := bc.NewIterator()
	var res []Transaction
	count := 0
	for {
		//遍历区块
		block := it.Next()
		count++
		if count >= MaxExpBlockLen {
			break
		}
		buf := bytes.NewReader(block.AllBlooms)
		var filter bloom.BloomFilter
		filter.ReadFrom(buf)

		if filter.Test([]byte(targetSourceId)) {
			txs := block.Transactions
			hashget := make(map[*Transaction][]byte)
			for _, tx := range txs {
				hashget[tx] = tx.TxHash
			}
			hashget, _ = getTxValuesByKeys(hashget)
			for tx, _ := range hashget {
				tx = Deserialize_tx(hashget[tx])
				if tx.TXOutputs[0].SourceID != targetid {
					continue
				}
				//这里直接加入 并没有做判断
				//if TX != nil {
				transaction := *tx
				res = append(res, transaction)
				count++
				//printtx(TX)
				//}
			}
		}
		//退出条件
		if block.PrevHash == nil {
			break
		}
	}
	fmt.Println("共转移", count, "次")
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return
}

func GraphGetsocuid(targetSourceId string) (duration21 time.Duration) {
	startTimeRecode := time.Now()
	SourceID, err := uuid.Parse(targetSourceId)
	if err != nil {
		fmt.Println("溯源id不合法", err)
		return
	}
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败  应该是区块链文件不存在，请先进行同步"
		Log.Fatal(ms)
		return
	}
	var usedTxto = make(map[string]int)
	count := 0
	//遍历区块 找到
	it := bc.NewIterator()
	count1 := 0
	for {
		//遍历区块
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		for uuid, confirmedBlockList := range block.ConfirmedLists {
			if uuid != SourceID {
				continue
			} else {
				aa := usedTxto[uuid.String()]
				if aa == 1 {
					continue
				}
				usedTxto[uuid.String()] = 1
				blockArray := confirmedBlockList.BlockArray
				finalConfirmedBlock := blockArray[len(blockArray)-1]
				finalblockhash := finalConfirmedBlock.Hash
				//fmt.Printf("找到soceid的最后哈希值：%x\n", finalblockhash)
				sideblockinfo, err := getValueByKey(finalblockhash)
				if err != nil {
					Log.Fatal("DataTraceall的getValueByKey出现错误，", err)
				}
				sideblock := Deserialize_tx(sideblockinfo)

				//printtrace(sideblock)
				//我觉得这个不会进入
				if sideblock.SourceStrat {
					//Log.Fatal("真的进入了")
					blockhash := sideblock.PoI
					mainblockinfo, _ := getValueByKey(blockhash)
					mainblock := Deserialize(mainblockinfo)
					printtraceBlock(*mainblock)
					continue
				}
				for {
					//Log.Warn("=+++++++++++++++++++++++++++++++++++++++++=次")
					preblockhash := sideblock.PoI
					preblockinfo, err := getValueByKey(preblockhash)
					//fmt.Printf("得到=====================%x", sideblockinfo)
					if err != nil {
						Log.Fatal("DataTraceall的getValueByKey出现错误，", err)
					}
					sideblock1 := Deserialize_tx(preblockinfo)
					sideblock = sideblock1
					count++
					//printtrace(sideblock1)
					//到达溯源的第一个节点
					if sideblock.SourceStrat {
						// blockhash := sideblock.PoI
						// mainblockinfo, _ := getValueByKey(blockhash)
						// mainblock := Deserialize(mainblockinfo)
						//printtraceBlock(*mainblock)
						endTimeRecode := time.Now()
						duration21 = endTimeRecode.Sub(startTimeRecode)
						fmt.Println("共转移", count, "次")
						return
					}
				}
			}
		}
		if block.Product.SourceID.String() == targetSourceId {
			exist := usedTxto[block.Product.SourceID.String()]
			if exist != 1 {
				fmt.Println("sourceid：", block.Product.SourceID.String())
				fmt.Println("	该地址还没有转移")
				printtraceBlock(*block)
				endTimeRecode := time.Now()
				duration21 = endTimeRecode.Sub(startTimeRecode)
				return
			}
			count++
		}
		if block.PrevHash == nil {
			Log.Debug("到达创世区块，退出查找")
			break
		}
	}
	fmt.Println("共转移", count, "次")
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return
}
func MHQueryWithKeywordAttributes(str string) (duration21 time.Duration) {
	startTime := time.Now()

	if str == "" {
		str = "转账"
	}
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Fatal("在experiment的queryWithKeywordAttributes获取chain出错：", err)
	}
	it := bc.NewIterator()
	count1 := 0
	for {
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		buf := bytes.NewReader(block.AllBlooms)
		var filter bloom.BloomFilter
		filter.ReadFrom(buf)

		if filter.Test([]byte(str)) {
			//Log.Info("MH发现 存在", str)
			txs := block.Transactions
			hashget := make(map[*Transaction][]byte)
			for _, tx := range txs {
				hashget[tx] = tx.TxHash
			}
			hashget, _ = getTxValuesByKeys(hashget)
			for tx, _ := range hashget {
				tx = Deserialize_tx(hashget[tx])
				txmsg := tx.TXmsg
				words := jieba.Cut(string(txmsg), true)
				flag := false
				for _, wstr := range words {
					if wstr == str {
						flag = true
						break
					}
				}
				if flag {
					//txblockhash, err3 := getValueByKey(ConfirmedBlock.Hash)
					//需要保持相同的场景，正常情况下block都是存储的交易的哈希值，而我在写的时候直接把写到里面了

					fmt.Printf("	%s在	 %x交易中有需要的信息\n      具体消息的内容：%s\n\n\n",
						tx.TXOutputs[0].SourceID.String(), tx.TxHash, string(tx.TXmsg))

				}
			}
		} else {
			//fmt.Printf("The string '%s' does not exist in the Bloom filter.\n", str)
		}
		if block.PrevHash == nil {
			endTime := time.Now()
			duration21 = endTime.Sub(startTime)
			return duration21
		}
	}
	endTime := time.Now()
	duration21 = endTime.Sub(startTime)
	return duration21
}

func GraphQueryWithKeywordAttributes(str string) (duration1 time.Duration) {
	startTime := time.Now()
	if str == "" {
		str = "转账"
	}
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Fatal("在experiment的queryWithKeywordAttributes获取chain出错：", err)
	}
	it := bc.NewIterator()
	count1 := 0
	for {
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		buf := bytes.NewReader(block.AllBlooms)
		var filter bloom.BloomFilter
		filter.ReadFrom(buf)

		if filter.Test([]byte(str)) {

			//打印  为提高速度注释掉   这个_应该是uuid====================
			for _, ConfirmedBlockList := range block.ConfirmedLists {
				//现在到某个uuid组里了
				buf2 := bytes.NewReader(ConfirmedBlockList.Bloom)
				var filter2 bloom.BloomFilter
				filter2.ReadFrom(buf2)
				//如果还存在，那么进入
				if filter2.Test([]byte(str)) {
					//fmt.Printf("	进入uuid小组%s\n\n", uuid.String())
					//遍历相同组的每个ConfirmedBlock
					for _, ConfirmedBlock := range ConfirmedBlockList.BlockArray {
						buf3 := bytes.NewReader(ConfirmedBlock.Bloom)
						var filter3 bloom.BloomFilter
						filter3.ReadFrom(buf3)
						if filter3.Test([]byte(str)) {
							//fmt.Printf("	进入具体的txblock%x\n\n", ConfirmedBlock.Hash)
							/*   打印  为提高速度注释掉====================
							txblockhash, err3 := getValueByKey(ConfirmedBlock.Hash)

							tx := Deserialize_tx(txblockhash)
							if err3 != nil {
								Log.Warn("序列化tx失败")
							}

							fmt.Printf("	%s在	 %x交易中有需要的信息\n      具体消息的内容：%s\n\n\n", uuid.String(), ConfirmedBlock.Hash, string(tx.TXmsg))
							*/
						}
					}
				}
			}
		} else {
			//fmt.Printf("The string '%s' does not exist in the Bloom filter.\n", str)
		}
		if block.PrevHash == nil {
			endTime := time.Now()
			duration1 = endTime.Sub(startTime)
			return
		}
	}
	endTime := time.Now()
	duration1 = endTime.Sub(startTime)
	return

}

/*
   批处理，模拟反序列化区块主体部分内容
*/
func BTQueryWithKeywordAttributes(str string) (duration2 time.Duration) {
	startTime := time.Now()
	if str == "" {
		str = "转账"
	}
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Fatal("在experiment的queryWithKeywordAttributes获取chain出错：", err)
	}
	it := bc.NewIterator()
	count1 := 0
	for {
		block := it.Next()
		count1++
		if count1 >= MaxExpBlockLen {
			break
		}
		txs := block.Transactions
		hashget := make(map[*Transaction][]byte)
		for _, tx := range txs {
			hashget[tx] = tx.TxHash
		}
		hashget, _ = getTxValuesByKeys(hashget)
		for tx, _ := range hashget {
			tx = Deserialize_tx(hashget[tx])
			txmsg := tx.TXmsg
			words := jieba.Cut(string(txmsg), true)
			flag := false
			for _, wstr := range words {
				if wstr == str {
					flag = true
					break
				}
			}
			if flag {
				//txblockhash, err3 := getValueByKey(ConfirmedBlock.Hash)
				//需要保持相同的场景，正常情况下block都是存储的交易的哈希值，而我在写的时候直接把写到里面了
				fmt.Printf("	%s在	 %x交易中有需要的信息\n      具体消息的内容：%s\n\n\n",
					tx.TXOutputs[0].SourceID.String(), tx.TxHash, string(tx.TXmsg))

			}
		}
		if block.PrevHash == nil {
			endTime := time.Now()
			// 计算函数运行时间
			duration2 = endTime.Sub(startTime)
			return
		}
	}
	endTime := time.Now()
	// 计算函数运行时间
	duration2 = endTime.Sub(startTime)
	return
}

// 获取时间范围的区块数组函数  1点到12点   1 2 3  4 5 6  7 8 9 10 11 12 13
func GraphGetBlocksInTimeRange(startTimestr, endTimestr string) (duration21 time.Duration) {
	// startBlock := GetBlocksInTimeBlock(startTime)
	// if startBlock == nil {
	// 	Log.Info("没有比starttime更早的区块了")
	// 	return nil
	// }
	var blocks []*Block
	startTimeRecode := time.Now()
	startTimestamp, err := strconv.ParseInt(startTimestr, 10, 64)
	if err != nil {
		Log.Fatal("转换时间戳字符串失败：", err)
		return
	}
	endTimestamp, err := strconv.ParseInt(endTimestr, 10, 64)
	if err != nil {
		Log.Fatal("转换时间戳字符串失败：", err)
		return
	}
	startTime := uint64(startTimestamp)
	endTime := uint64(endTimestamp)
	count := 0
	//勿删==========================
	//timestamp := uint64(startTime)
	//timeObj := time.Unix(int64(timestamp), 0)
	//endTim := uint64(endTime)
	//timeObj2 := time.Unix(int64(endTim), 0)

	//找到最后一个节点向前找
	endBlock := SkipGetBlocksInTimeBlock(endTime)
	if endBlock == nil {
		Log.Info("没有endBlock更早的区块了  没有找到区块")
		endTimeRecode := time.Now()
		duration21 = endTimeRecode.Sub(startTimeRecode)
		return
	}
	curblock := endBlock
	for {
		if curblock == nil {
			//Log.Info("在", timeObj.String(), "\n到", timeObj2.String(), "\n共找到区块", count, "个")
			break
		}
		if startTime <= curblock.TimeStamp {
			blocks = append(blocks, curblock)
			count++
			//printBlock(*curblock)
		} else {
			//Log.Info("在", timeObj.String(), "\n	到", timeObj2.String(), "\n		共找到区块", count, "个")
			break
		}
		curblock = curblock.BtNext()
	}
	Log.Debug("GraphGetBlocksInTimeRange找到区块个数为", len(blocks))
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return

}

func MHGetBlocksInTimeRange(startTimestr, endTimestr string) (duration21 time.Duration) {
	// startBlock := GetBlocksInTimeBlock(startTime)
	// if startBlock == nil {
	// 	Log.Info("没有比starttime更早的区块了")
	// 	return nil
	// }
	startTimeRecode := time.Now()
	var blocks []*Block
	startTimestamp, err := strconv.ParseInt(startTimestr, 10, 64)
	if err != nil {
		Log.Fatal("转换时间戳字符串失败：", err)
		return
	}
	endTimestamp, err := strconv.ParseInt(endTimestr, 10, 64)
	if err != nil {
		Log.Fatal("转换时间戳字符串失败：", err)
		return
	}
	startTime := uint64(startTimestamp)
	endTime := uint64(endTimestamp)
	count := 0
	// timestamp := uint64(startTime)
	// timeObj := time.Unix(int64(timestamp), 0)
	// endTim := uint64(endTime)
	// timeObj2 := time.Unix(int64(endTim), 0)

	curBlockcur := getfinalBlock()
	curblock := getBlockByhash(curBlockcur)
	forcount := 0
	for {
		forcount++
		if curblock == nil {
			Log.Info("共找到区块", count, "个")
			break
		}
		if startTime <= curblock.TimeStamp && curblock.TimeStamp <= endTime {
			blocks = append(blocks, curblock)
			count++
			//printBlock(*curblock)
		} else if curblock.TimeStamp > endTime {
			curblock = curblock.BtNext()
		} else {
			Log.Info("共找到区块", count, "个")
			break
		}
		curblock = curblock.BtNext()
	}
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return
}

func BTGetBlocksInTimeRange(startTimestr, endTimestr string) (duration21 time.Duration) {
	startTimeRecode := time.Now()
	var blocks []*Block
	startTimestamp, err := strconv.ParseInt(startTimestr, 10, 64)
	if err != nil {
		Log.Fatal("转换时间戳字符串失败：", err)
		return
	}
	endTimestamp, err := strconv.ParseInt(endTimestr, 10, 64)
	if err != nil {
		Log.Fatal("转换时间戳字符串失败：", err)
		return
	}
	startTime := uint64(startTimestamp)
	endTime := uint64(endTimestamp)
	count := 0
	//timestamp := uint64(startTime)
	//timeObj := time.Unix(int64(timestamp), 0)
	//endTim := uint64(endTime)
	//timeObj2 := time.Unix(int64(endTim), 0)

	curBlockcur := getfinalBlock()
	curblock := getBlockByhash(curBlockcur)
	for {
		if curblock == nil {
			Log.Info("共找到区块", count, "个")
			break
		}
		if startTime <= curblock.TimeStamp && curblock.TimeStamp <= endTime {
			blocks = append(blocks, curblock)
			count++
			//printBlock(*curblock)
		} else if curblock.TimeStamp > endTime {
			curblock = curblock.BtNext()
		} else {
			Log.Info("共找到区块", count, "个")
			break
		}
		curblock = curblock.BtNext()
	}
	endTimeRecode := time.Now()
	duration21 = endTimeRecode.Sub(startTimeRecode)
	return
}

// 通过skiplist的哈希找到区块
func SkipGetBlocksInTimeBlock(Time uint64) *Block {
	curBlockcur := getfinalBlock()
	curBlock := getBlockByhash(curBlockcur)
	if curBlock == nil {
		return nil
	}
	if curBlock.TimeStamp <= Time {
		return curBlock
	}
	for {
		/*
			1.1 获取当前的区块 通过三个手段 只要不是nil就行
			     判断是否是超过了时间
				如果超过了 那么就从curblock向前找
				如果没有超过 那么curblock就是此区块
				 对于第二个时间，如果返回了nil，那么前面的就都是
		*/
		//使用最高级前进
		skipblock := getBlockByhash(curBlock.Skiplist[2])
		if skipblock == nil {
			Log.Debug("高级查找跳跃太长，已经失败")
			//使用次高级前进
			skipblock = getBlockByhash(curBlock.Skiplist[1])
			if skipblock == nil {
				//使用高级前进
				Log.Debug("次高级跳跃太长，已经失败")
				skipblock = getBlockByhash(curBlock.Skiplist[0])
			}
		}
		//判断是否为nil  说明只有三步内 直接从本区块向前找
		if skipblock == nil {
			//else去最下面处理
			Log.Debug("高级全部失败 说明就在附近了")
		} else { //否则判断是否超过了目标，没有超过则更新本区块
			if Time < skipblock.TimeStamp {
				Log.Debug("高级小于目标值，继续下一次大的迭代")
				curBlock = skipblock
				continue
			}
			//else去最下面处理
		}
		//到这里说明curblock向前找就可以了，也就是高级全部失效了
		for {
			//遍历区块
			preblock := curBlock.BtNext()
			if preblock == nil { //如果为空了，没有找到就返回nil
				Log.Debug("目标为空，返回nil")
				return nil
			} else if Time < preblock.TimeStamp { //如果小于目标值 那就找下一个
				curBlock = preblock
				continue
			} else { //说明大于等于目标值，想找的就是第一个大于等于目标值的 那就是目标
				Log.Debug("找到目标")
				return preblock
			}
		}
	}
}

//新添加的几个功能函数
// 1. 主链区块的遍历  如果传入空值那么获取最后一个区块
func (block *Block) BtNext() (nextblock *Block) {
	if block == nil {
		blockkey := getfinalBlock()
		blockbyte, err := getValueByKey(blockkey)
		if err != nil {
			Log.Fatal("从bucket获取哈希值失败")
		}
		finalblock := Deserialize(blockbyte)
		return finalblock
	}
	if block.PrevHash == nil {
		return nil
	}
	prehash, err := getValueByKey(block.PrevHash)
	if err != nil {
		return nil
	}
	preblock := Deserialize(prehash)
	return preblock
}

func (block *Block) BtNextMemory() (nextblock *Block) {
	if block == nil {
		bc, err := GetBlockChainInstance()
		defer bc.closeDB()
		if err != nil {
			Log.Fatal("在experiment的queryWithKeywordAttributes获取chain出错：", err)
		}
		it := bc.NewIterator()
		finalblock := it.MemoryNextBlock()
		return finalblock
	}
	if block.PrevHash == nil {
		return nil
	}
	preblock := GetBlockFromMemory(block.PrevHash)
	return preblock
}
