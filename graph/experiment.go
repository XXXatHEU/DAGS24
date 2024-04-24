package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/willf/bloom"
)

func etime() {
	startTime := time.Now()
	// 调用要测试的函数
	GraphQueryWithKeywordAttributes("15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47")
	// 记录函数结束时间
	endTime := time.Now()
	// 计算函数运行时间
	duration1 := endTime.Sub(startTime)
	// 输出运行时间

	startTime = time.Now()
	// 调用要测试的函数
	BTQueryWithKeywordAttributes("15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration2 := endTime.Sub(startTime)
	// 输出运行时间

	startTime = time.Now()
	// 调用要测试的函数
	MHQueryWithKeywordAttributes("15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration21 := endTime.Sub(startTime)
	// 输出运行时间

	fmt.Printf("GraphQueryWithKeywordAttributes函数运行时间：%s\n", duration1)
	fmt.Printf("BTQueryWithKeywordAttributes函数运行时间：%s\n", duration2)
	fmt.Printf("MHQueryWithKeywordAttributes函数运行时间：%s\n", duration21)
	Log.Info("===========测试寻找某个范围的区块时间===================")
	startTime = time.Now()
	// 调用要测试的函数
	BTGetBlocksInTimeRange("1658628718", "1690164718")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration3 := endTime.Sub(startTime)
	// 输出运行时间

	startTime = time.Now()
	// 调用要测试的函数
	GetBlocksInTimeRange("1658628718", "1690164718")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration4 := endTime.Sub(startTime)

	startTime = time.Now()
	// 调用要测试的函数
	MHGetBlocksInTimeRange("1658628718", "1690164718")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration4_1 := endTime.Sub(startTime)

	// 输出运行时间
	fmt.Printf("BTGetBlocksInTimeRange函数运行时间：%s\n", duration3)
	fmt.Printf("GetBlocksInTimeRange函数运行时间：%s\n", duration4)
	fmt.Printf("MHGetBlocksInTimeRange函数运行时间：%s\n", duration4_1)

	Log.Info("===========测试溯源某个id区块时间===================")
	startTime = time.Now()
	// 调用要测试的函数
	BTGetsocuid("91ebf83e-9015-4ad6-8d4f-58141f9c5b56")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration5 := endTime.Sub(startTime)
	// 输出运行时间
	fmt.Printf("BTGetsocuid函数运行时间：================================%s\n", duration5)
	startTime = time.Now()
	// 调用要测试的函数
	GraphGetsocuid("91ebf83e-9015-4ad6-8d4f-58141f9c5b56")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration6 := endTime.Sub(startTime)

	startTime = time.Now()
	// 调用要测试的函数
	MHGetsocuid("91ebf83e-9015-4ad6-8d4f-58141f9c5b56")
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration6_1 := endTime.Sub(startTime)

	// 输出运行时间
	fmt.Printf("BTGetsocuid函数运行时间：%s\n", duration5)
	fmt.Printf("GraphGetsocuid函数运行时间：%s\n", duration6)
	fmt.Printf("MHGetsocuid函数运行时间：%s\n", duration6_1)

	startTime = time.Now()
	// 调用要测试的函数
	BThaveid(mywallet.Address)
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration7 := endTime.Sub(startTime)
	// 输出运行时间

	startTime = time.Now()
	// 调用要测试的函数
	Graphhaveid(mywallet.Address)
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration8 := endTime.Sub(startTime)

	startTime = time.Now()
	// 调用要测试的函数
	MHhaveid(mywallet.Address)
	// 记录函数结束时间
	endTime = time.Now()
	// 计算函数运行时间
	duration8_1 := endTime.Sub(startTime)

	// 输出运行时间

	fmt.Printf("GraphQueryWithKeywordAttributes函数运行时间：%s\n", duration1)
	fmt.Printf("BTQueryWithKeywordAttributes函数运行时间：%s\n", duration2)
	fmt.Printf("MHQueryWithKeywordAttributes函数运行时间：%s\n", duration21)

	fmt.Printf("BTGetBlocksInTimeRange函数运行时间：%s\n", duration3)
	fmt.Printf("GetBlocksInTimeRange函数运行时间：%s\n", duration4)
	fmt.Printf("MHGetBlocksInTimeRange函数运行时间：%s\n", duration4_1)

	fmt.Printf("BTGetsocuid函数运行时间：%s\n", duration5)
	fmt.Printf("GraphGetsocuid函数运行时间：%s\n", duration6)
	fmt.Printf("MHGetsocuid函数运行时间：%s\n", duration6_1)

	fmt.Printf("BThaveid函数运行时间：%s\n", duration7)
	fmt.Printf("Graphhaveid函数运行时间：%s\n", duration8)
	fmt.Printf("MHhaveid函数运行时间：%s\n", duration8_1)
}

func BThaveid(myaddress string) (utxoInfos []Transaction) {
	//func (bc *BlockChain) FindMyUTXO(pubKeyHash []byte) []UTXOInfo {
	//存储所有和目标地址相关的utxo集合
	// var utxos []TXOutput

	pubKeyHash := getPubKeyHashFromAddress(myaddress)
	//定义一个存放已经消耗过的所有的utxos的集合(跟指定地址相关的)
	spentUtxos := make(map[string][]int)
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Warn("getMainBlockLength长度出错")
		err = fmt.Errorf("getMainBlockLength长度出错")
		return
	}
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
					TX := getTxByhash(tx.TxHash)
					if TX != nil {
						utxoInfos = append(utxoInfos, *TX)
						printtx(TX)
					}

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

func MHhaveid(myaddress string) (utxoInfos []Transaction) {
	pubKeyHash := getPubKeyHashFromAddress(myaddress)
	//定义一个存放已经消耗过的所有的utxos的集合(跟指定地址相关的)
	spentUtxos := make(map[string][]int)
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		Log.Warn("getMainBlockLength长度出错")
		err = fmt.Errorf("getMainBlockLength长度出错")
		return
	}
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()
		buf := bytes.NewReader(block.AllBlooms)
		var filter bloom.BloomFilter
		filter.ReadFrom(buf)

		if filter.Test([]byte(myaddress)) {
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
						TX := getTxByhash(tx.TxHash)
						if TX != nil {
							utxoInfos = append(utxoInfos, *TX)
							printtx(TX)
						}

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
	return utxoInfos

}

func Graphhaveid(myaddress string) (myhaveid []uuid.UUID) {
	if mywallet == (MyWallet{}) {
		Log.Warn("钱包获取失败，未进入区块链网络")
		return
	}
	//获取池子中已经使用的
	var usedmap = make(map[uuid.UUID]int)
	TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()

	Log.Warn(myaddress)
	blockpubhash := getPubKeyHashFromAddress(myaddress)
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "getMyhaveId中GetBlockChainInstance失败"
		Log.Warn(ms)
		return
	}

	var hased = make(map[uuid.UUID]int)
	//遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()
		for uuid_, confirmedBlockList := range block.ConfirmedLists {
			hased[uuid_] = 1
			blockArray := confirmedBlockList.BlockArray
			finalConfirmedBlock := blockArray[len(blockArray)-1]
			finalblockhash := finalConfirmedBlock.Hash
			sideblockinfo, err := getValueByKey(finalblockhash)
			if err != nil {
				Log.Warn("GetSideUTXOBySourceId出现错误，", err)
			}
			sideblock := Deserialize_tx(sideblockinfo)
			fmt.Printf("sideblock.TXOutputs[0].ScriptPubKeyHash %x\n", sideblock.TXOutputs[0].ScriptPubKeyHash)
			fmt.Printf("pubKeyHash %x\n", blockpubhash)
			if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, blockpubhash) {
				_, existss := usedmap[uuid_]
				if existss {
					Log.Info("该id已经转移:", existss)
					continue
				}
				myhaveid = append(myhaveid, uuid_)
				return
			}
			if err != nil {
				Log.Warn("GetSideUTXOBySourceId出现错误，", err)
			}

		}
		//验证起始点是否是自己的 要求保证这个没有用过
		fmt.Printf("block.Product.Pubkeyhash\n	%x\n", block.Product.Pubkeyhash)
		fmt.Printf("blockpubhash\n	%x\n", blockpubhash)
		if bytes.Equal(block.Product.Pubkeyhash, blockpubhash) {

			_, ok := hased[block.Product.SourceID]
			if ok {
			} else { //没有使用过
				_, existss := usedmap[block.Product.SourceID]
				if existss {
					Log.Info("该id已经转移:", existss)
				} else {
					hased[block.Product.SourceID] = 1
					myhaveid = append(myhaveid, block.Product.SourceID)
				}
			}
		}
		if block.PrevHash == nil {
			break
		}

	}

	Log.Debug("没有找到该溯源id")

	return
}

func BTGetsocuid(targetSourceId string) []Transaction {
	targetSourceId = strings.TrimSpace(targetSourceId)
	targetid, err := uuid.Parse(targetSourceId)
	if err != nil {
		fmt.Println("无法将字符串转换为UUID:", err)
		return nil
	}
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Warn(ms)
		return nil
	}
	//根据uuid进行数据溯源
	it := bc.NewIterator()
	var res []Transaction
	for {
		//遍历区块
		block := it.Next()
		if block == nil {
			Log.Fatal("出现空区块")
		}
		for _, tx := range block.Transactions {
			if tx.TXOutputs[0].SourceID != targetid {
				continue
			}
			//这里直接加入 并没有做判断
			TX := getTxByhash(tx.TxHash)
			if TX != nil {
				transaction := *TX
				res = append(res, transaction)
				printtx(TX)
			}
		}
		//fmt.Println(block.PrevHash)
		if block.PrevHash == nil {
			break
		}
	}
	return res
}

//
func MHGetsocuid(targetSourceId string) []Transaction {
	targetSourceId = strings.TrimSpace(targetSourceId)
	targetid, err := uuid.Parse(targetSourceId)
	if err != nil {
		fmt.Println("无法将字符串转换为UUID:", err)
		return nil
	}
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Warn(ms)
		return nil
	}
	//根据uuid进行数据溯源
	it := bc.NewIterator()
	var res []Transaction
	for {
		//遍历区块
		block := it.Next()

		buf := bytes.NewReader(block.AllBlooms)
		var filter bloom.BloomFilter
		filter.ReadFrom(buf)

		if filter.Test([]byte(targetSourceId)) {
			for _, tx := range block.Transactions {
				if tx.TXOutputs[0].SourceID != targetid {
					continue
				}
				//这里直接加入 并没有做判断
				TX := getTxByhash(tx.TxHash)
				if TX != nil {
					transaction := *TX
					res = append(res, transaction)
					printtx(TX)
				}

			}
		}
		//退出条件
		if block.PrevHash == nil {
			break
		}
	}
	return res
}

func GraphGetsocuid(address string) {
	SourceID, err := uuid.Parse(address)
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
	//遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()

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
				printtrace(sideblock)
				if sideblock.SourceStrat {
					blockhash := sideblock.PoI
					mainblockinfo, _ := getValueByKey(blockhash)
					mainblock := Deserialize(mainblockinfo)
					printtraceBlock(*mainblock)
					continue
				}
				for {
					Log.Warn("=+++++++++++++++++++++++++++++++++++++++++=次")
					preblockhash := sideblock.PoI
					preblockinfo, err := getValueByKey(preblockhash)
					//fmt.Printf("得到=====================%x", sideblockinfo)
					if err != nil {
						Log.Fatal("DataTraceall的getValueByKey出现错误，", err)
					}
					sideblock1 := Deserialize_tx(preblockinfo)
					sideblock = sideblock1
					printtrace(sideblock1)
					//printtx(sideblock)
					//到达溯源的第一个节点
					if sideblock.SourceStrat {
						// blockhash := sideblock.PoI
						// mainblockinfo, _ := getValueByKey(blockhash)
						// mainblock := Deserialize(mainblockinfo)
						//printtraceBlock(*mainblock)
						return
					}
				}
			}
		}
		exist := usedTxto[block.Product.SourceID.String()]
		if exist != 1 {
			fmt.Println("sourceid：", block.Product.SourceID.String())
			fmt.Println("	该地址还没有转移")
			printtraceBlock(*block)
			return
		}
		if block.PrevHash == nil {
			Log.Debug("到达创世区块，退出查找")
			break
		}
	}

}
func MHQueryWithKeywordAttributes(str string) {
	if str == "" {
		str = "转账"
	}
	bc, err := GetBlockChainInstance()
	if err != nil {
		Log.Fatal("在experiment的queryWithKeywordAttributes获取chain出错：", err)
	}
	it := bc.NewIterator()
	for {
		block := it.Next()

		buf := bytes.NewReader(block.AllBlooms)
		var filter bloom.BloomFilter
		filter.ReadFrom(buf)

		if filter.Test([]byte(str)) {
			//Log.Info("MH发现 存在", str)
			txs := block.Transactions
			for _, tx := range txs {
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
					txblockhash, err3 := getValueByKey(tx.TxHash)
					tx := Deserialize_tx(txblockhash)
					if err3 != nil {
						Log.Warn("序列化tx失败")
					}
					fmt.Printf("	%s在	 %x交易中有需要的信息\n      具体消息的内容：%s\n\n\n",
						tx.TXOutputs[0].SourceID.String(), tx.TxHash, string(tx.TXmsg))

				}
			}
		} else {
			//fmt.Printf("The string '%s' does not exist in the Bloom filter.\n", str)
		}
		if block.PrevHash == nil {
			return
		}
	}
}

func GraphQueryWithKeywordAttributes(str string) {
	if str == "" {
		str = "转账"
	}
	bc, err := GetBlockChainInstance()
	if err != nil {
		Log.Fatal("在experiment的queryWithKeywordAttributes获取chain出错：", err)
	}
	it := bc.NewIterator()
	for {
		block := it.Next()

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
			return
		}
	}

}
func BTQueryWithKeywordAttributes(str string) {
	if str == "" {
		str = "转账"
	}
	bc, err := GetBlockChainInstance()
	if err != nil {
		Log.Fatal("在experiment的queryWithKeywordAttributes获取chain出错：", err)
	}
	it := bc.NewIterator()
	for {
		block := it.Next()
		txs := block.Transactions
		for _, tx := range txs {
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
				txblockhash, err3 := getValueByKey(tx.TxHash)
				tx := Deserialize_tx(txblockhash)
				if err3 != nil {
					Log.Warn("序列化tx失败")
				}
				fmt.Printf("	%s在	 %x交易中有需要的信息\n      具体消息的内容：%s\n\n\n",
					tx.TXOutputs[0].SourceID.String(), tx.TxHash, string(tx.TXmsg))

			}
		}
		if block.PrevHash == nil {
			return
		}
	}
}

// 获取时间范围的区块数组函数  1点到12点   1 2 3  4 5 6  7 8 9 10 11 12 13
func GetBlocksInTimeRange(startTimestr, endTimestr string) (blocks []*Block) {
	// startBlock := GetBlocksInTimeBlock(startTime)
	// if startBlock == nil {
	// 	Log.Info("没有比starttime更早的区块了")
	// 	return nil
	// }
	startTimestamp, err := strconv.ParseInt(startTimestr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串失败：", err)
		return
	}
	endTimestamp, err := strconv.ParseInt(endTimestr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串失败：", err)
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
	endBlock := GetBlocksInTimeBlock(endTime)
	if endBlock == nil {
		Log.Info("没有endBlock更早的区块了  没有找到区块")
		return nil
	}
	curblock := endBlock
	for {
		if curblock == nil {
			//Log.Info("在", timeObj.String(), "\n到", timeObj2.String(), "\n共找到区块", count, "个")
			return
		}
		if startTime <= curblock.TimeStamp {
			blocks = append(blocks, curblock)
			count++
			printBlock(*curblock)
		} else {
			//Log.Info("在", timeObj.String(), "\n	到", timeObj2.String(), "\n		共找到区块", count, "个")
			return
		}
		curblock = curblock.BtNext()
	}

}

func MHGetBlocksInTimeRange(startTimestr, endTimestr string) (blocks []*Block) {
	// startBlock := GetBlocksInTimeBlock(startTime)
	// if startBlock == nil {
	// 	Log.Info("没有比starttime更早的区块了")
	// 	return nil
	// }
	startTimestamp, err := strconv.ParseInt(startTimestr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串失败：", err)
		return
	}
	endTimestamp, err := strconv.ParseInt(endTimestr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串失败：", err)
		return
	}
	startTime := uint64(startTimestamp)
	endTime := uint64(endTimestamp)
	count := 0
	timestamp := uint64(startTime)
	timeObj := time.Unix(int64(timestamp), 0)
	endTim := uint64(endTime)
	timeObj2 := time.Unix(int64(endTim), 0)

	curBlockcur := getfinalBlock()
	curblock := getBlockByhash(curBlockcur)
	for {
		if curblock == nil {
			Log.Info("在", timeObj.String(), "\n到", timeObj2.String(), "\n共找到区块", count, "个")
			return
		}
		if startTime <= curblock.TimeStamp && curblock.TimeStamp <= endTime {
			blocks = append(blocks, curblock)
			count++
			printBlock(*curblock)
		} else {
			Log.Info("在", timeObj.String(), "\n	到", timeObj2.String(), "\n		共找到区块", count, "个")
			return
		}
		curblock = curblock.BtNext()
	}

}

func BTGetBlocksInTimeRange(startTimestr, endTimestr string) (blocks []*Block) {
	// startBlock := GetBlocksInTimeBlock(startTime)
	// if startBlock == nil {
	// 	Log.Info("没有比starttime更早的区块了")
	// 	return nil
	// }
	startTimestamp, err := strconv.ParseInt(startTimestr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串失败：", err)
		return
	}
	endTimestamp, err := strconv.ParseInt(endTimestr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串失败：", err)
		return
	}
	startTime := uint64(startTimestamp)
	endTime := uint64(endTimestamp)
	count := 0
	timestamp := uint64(startTime)
	timeObj := time.Unix(int64(timestamp), 0)
	endTim := uint64(endTime)
	timeObj2 := time.Unix(int64(endTim), 0)

	curBlockcur := getfinalBlock()
	curblock := getBlockByhash(curBlockcur)
	for {
		if curblock == nil {
			Log.Info("在", timeObj.String(), "\n到", timeObj2.String(), "\n共找到区块", count, "个")
			return
		}
		if startTime <= curblock.TimeStamp && curblock.TimeStamp <= endTime {
			blocks = append(blocks, curblock)
			count++
			printBlock(*curblock)
		} else {
			Log.Info("在", timeObj.String(), "\n	到", timeObj2.String(), "\n		共找到区块", count, "个")
			return
		}
		curblock = curblock.BtNext()
	}

}

// 通过skiplist的哈希找到区块
func GetBlocksInTimeBlock(Time uint64) *Block {
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
		//到这里说明curblock向前找就可以了
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

func getBlockByhash(curBlockcur []byte) *Block {
	//curBlockcur := getfinalBlock()
	curBlockhash, err := getValueByKey(curBlockcur)
	if err != nil {
		//Log.Warn(err)
		return nil

	}
	if curBlockhash == nil {
		return nil
	}
	curBlock := Deserialize(curBlockhash)
	return curBlock
}

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
		Log.Warn("BtNext 出错 ：========================", err)
	}
	preblock := Deserialize(prehash)
	return preblock
}
