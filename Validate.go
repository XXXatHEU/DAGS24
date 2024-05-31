package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
)

//收到交易后的处理过程
func ReceiveTxArr(message *SendMessage) {
	//fmt.Println("----------------------------------")
	//Log.Info("进入ReceiveTxArr处理")
	txarry := byteToTransactionArr(message.Payload)
	Log.Info(len(txarry), "===================================")
	for _, tx := range txarry {
		//fmt.Println("\n\n+++++++++++++++++交易分割+++++++++++++++")
		//fmt.Printf("第 %d 个交易的哈希值为: %x\n", index+1, tx.TXID)
		var sourceID uuid.UUID
		var pubkey []byte
		//fmt.Printf("tx.TXID: %x\n", tx.TXID)
		//fmt.Printf("tx.TXInputs[0].Txid %x\n", tx.TXInputs[0].Txid)
		//fmt.Printf("tx.TXInputs[0].ScriptPubKeyHash %x\n", tx.TXOutputs[0].ScriptPubKeyHash)
		//fmt.Printf("tx.TXInputs[0].SourceID %x\n", tx.TXOutputs[0].SourceID)
		sourceID = tx.TXOutputs[0].SourceID
		for _, input := range tx.TXInputs {
			//直接打印交易
			//fmt.Printf("上一笔交易的id：%x\n", input.Txid)
			//	fmt.Printf("上一笔交易的下标：%x\n", input.Index)
			//	fmt.Printf("上一个人的公钥：%x\n", input.PubKey)
			if pubkey != nil && !bytes.Equal(input.PubKey, pubkey) {
				Log.Fatal("两者竟然不一样")
			}
			pubkey = input.PubKey
			//测试用
			// if bytes.Equal(PPubKeyHash, getPubKeyHashFromPubKey(input.PubKey)) {
			// 	Log.Warn("PPubKeyHash 两者相等")
			// }
			// if bytes.Equal(input.PubKey, PPubey) {
			// 	Log.Warn("PPubey 两者相等")
			// }
			break
		}
		/*for _, output := range tx.TXOutputs {
			//直接打印交易
			fmt.Println("SourceID:", output.SourceID)
			break
		}*/

		//判断交易的合法性
		exist := CheckhasSourceID(sourceID, pubkey)
		//exist := CheckhasSourceID(SSourceID, PPubey)
		if exist {

			Log.Debug("添加进这个交易区块")
			AddToTransactionPool(tx)
		} else {
			Log.Debug("不存在这个交易区块")
			if MiningControlExist { //如果是挖矿节点，那么就不做处理
			} else { //否则的话就听挖矿协程的
				Log.Warn("交易验证没有通过，可能是sourceID所属权限不归属此人或者本节点存在区块链更新延迟")
				FetchBlocksRequest(nil)
			}
		}
	}
	/*
	 */

}

//根据uuid找到区块，判断交易或者区块的时间戳哪个更小，
func CompareBlock(tx *Transaction) bool {
	//这里本来再加上比较输入和输出，没有加上
	defer Log.Debug("跳出比较过程")
	if !isFileExist(blockchainDBFile) {
		Log.Debug("区块链文件不存在！")
		return true
	}
	//如果最后一个区块都不存在那么就直接返回错误
	_, errget := getValueByKey([]byte(lastBlockHashKey))
	if errget != nil {
		return true
	}

	Mainblock, Sideblock, isBlock, noexit := getSourceidBlock(tx.TXOutputs[0].SourceID)
	if noexit { //如果不存在就说明他更加新
		Log.Debug("未找到该引用id，验证通过")
		return true
	}

	if isBlock {
		if Mainblock.TimeStamp < tx.TimeStamp {
			Log.Debug("主链时间戳验证通过")
			return true
		} else {
			fmt.Printf("主链时间戳为: %d,交易验证戳为: %d\n", Mainblock.TimeStamp, tx.TimeStamp)
			Log.Debug("主链时间戳没有验证通过")
			return false
		}
	} else {
		if Sideblock.TimeStamp < tx.TimeStamp {
			fmt.Printf("主链时间戳为: %d,交易验证戳为: %d\n", Sideblock.TimeStamp, tx.TimeStamp)
			Log.Debug("侧链时间戳验证通过")
			return true
		} else {
			Log.Debug("侧链时间戳没有验证通过")
			return false
		}

	}
}

//收到别人挖到的区块
func ReceiveMinedBlock(message *SendMessage, getmsg chan []byte) {
	if trystopMing() { //修改后简单粗暴，收到就暂停
		defer activeMing()
	}
	DBfilemutex.Lock()
	defer DBfilemutex.Unlock()
	Log.Debug("进入ReceiveBlockArr处理收到的区块")
	blockarry := byteToBlockArray(message.Payload)

	//判断交易是否过期
	for _ /*index*/, block := range blockarry {
		//Log.Debug("\n\n+++++++++++++++++ 区块分割 +++++++++++++++")
		//fmt.Printf("第 %d 个区块的哈希值为: %x\n", index+1, block.Hash)
		for _, tx := range block.Transactions {
			//直接打印交易
			//Log.Debug(tx.TXOutputs[0].SourceID)
			if !CompareBlock(tx) {
				return
			}
		}
	}

	if len(blockarry) == 0 {
		return
	}
	block := blockarry[0]
	arry := RmBlockArryPointer(blockarry)

	//判断这个区块的前一个区块是否存在
	exit, fileexit := BlockExist(block.PrevHash)
	if fileexit && block.PrevHash == nil {
		Log.Debug("发现创世块广播，更新本地")
		count := 0
		for _, block := range blockarry {
			count++
			if count == 2 {
				Log.Fatal("逻辑设计错误，挖矿广播只能广播一个区块")
			}
			RemoveTxpooltx(block)
		}
		AppendBlocksToChain(arry[:])
		saveConfirmedSideBlock(*block)
		Log.Info("区块更新成功")
		//更新交易池
		return
	}

	if !exit {
		//尝试去拉取对方的区块，或许有惊喜
		FetchBlocksRequest(getmsg)
		Log.Warn("区块前哈希不存在 将抛弃该区块")
		return
	} else { //如果区块前哈希存在
		//fmt.Printf("前哈希值为:%x", block.PrevHash)
		myLen := GetBlockCountUntilSpecificBlock(block.PrevHash)
		hisLen := len(blockarry)
		Log.Info("myLen:", myLen, "  hisLen：", hisLen)

		//判断到特定区块哪个更长，如果我更长那么就抛弃这个区块，否则就加入
		if myLen > hisLen { //如果我到那个位置更多，那么抛弃他发来的数据
			Log.Info("我的更多一点，抛弃对方发来的数据")
			pullok(getmsg) //让他来请求我
		} else if myLen == hisLen { //区块一样长 什么都不做

		} else { //他的更长
			//更新交易池
			count := 0
			for _, block := range blockarry {
				count++
				if count == 2 {
					Log.Fatal("逻辑设计错误，挖矿广播只能广播一个区块")
				}
				RemoveTxpooltx(block)
			}
			Log.Debug("根据对方发来的数据进行更新")
			//删除我的更短的区块
			RmBlocksUntilSpecBlock(block.PrevHash)
			saveConfirmedSideBlock(*block)
			//更新区块
			AppendBlocksToChain(arry[:])
			Log.Info("区块更新成功，加入发来的区块")
		}
	}
}

//收到区块后的处理过程  请求的区块
func ReceiveBlockArr(message *SendMessage) {
	if trystopMing() { //修改后简单粗暴，收到就暂停
		defer activeMing()
	}
	DBfilemutex.Lock()
	defer DBfilemutex.Unlock()
	Log.Info("进入ReceiveBlockArr收到区块处理函数")
	blockarry := byteToBlockArray(message.Payload)

	for index, block := range blockarry {
		fmt.Println("\n\n+++++++++++++++++ 区块分割 +++++++++++++++")
		fmt.Printf("第 %d 个区块的哈希值为: %x\n", index+1, block.Hash)
		for _, tx := range block.Transactions {
			//直接打印交易
			fmt.Println(tx)
		}
	}
	/*
	 */

	if len(blockarry) == 0 {
		return
	}
	finalBlock := blockarry[len(blockarry)-1]
	exit, _ := BlockExist(finalBlock.Hash)
	reverseBlockArr(blockarry)
	arry := RmBlockArryPointer(blockarry)
	if !exit {
		//更新交易池
		for _, block := range blockarry {
			RemoveTxpooltx(block)
		}
		Log.Info("我的区块为空，将更新区块")
		//更新区块
		AppendBlocksToChain(arry)

	} else {

		fmt.Printf("哈希值为:%x", finalBlock.Hash)
		myLen := GetBlockCountUntilSpecificBlock(finalBlock.Hash)
		hisLen := len(blockarry)
		Log.Info("myLen:", myLen, "  hisLen：", hisLen)
		if myLen > hisLen { //如果我到那个位置更多，那么抛弃他发来的数据
			Log.Info("我的更多一点，抛弃对方发来的数据")
		} else {
			//更新交易池
			for _, block := range blockarry {
				RemoveTxpooltx(block)
			}
			Log.Info("根据对方发来的数据进行更新")
			RmBlocksUntilSpecBlock(finalBlock.Hash)
			//更新区块
			AppendBlocksToChain(arry[1:])

			Log.Info("区块更新成功")
		}
	}
}

//将将[]*Block赋值给[]Block
func RmBlockArryPointer(blockarr []*Block) (arr []Block) {
	for _, b := range blockarr {
		arr = append(arr, *b)
	}
	return
}

//翻转数组
func reverseBlockArr(blockarry []*Block) {
	n := len(blockarry)
	for i := 0; i < n/2; i++ {
		blockarry[i], blockarry[n-1-i] = blockarry[n-1-i], blockarry[i]
	}
}

//针对hash字符处理问题，这里专门写个函数来处理
//注意GetBlockCountUntilSpecificBlock RmBlocksUntilSpecBlock  AppendBlocksToChain 没有使用这个函数
func DBkeyhash(hash1 []byte) (hash2 []byte) {
	return hash1
	hashstr := string(hash1)
	// 将字符串转换为字节数组
	hashBytes, err := hex.DecodeString(hashstr)
	if err != nil {
		// 转换失败
		Log.Warn("DBkeyhash 字符串转换为字节数组失败！")
		return
	}
	var hash [32]byte
	copy(hash[:], hashBytes)
	hash2 = hash[:]
	return
}

//判断是否有这个区块
func BlockExist(hash1 []byte) (exist bool, fileexit bool) {

	hash2 := DBkeyhash(hash1)

	if !isFileExist(blockchainDBFile) {
		fileexit = true
		fmt.Println("区块链文件不存在！")
		return
	}

	// 开启读事务
	value, err3 := getValueByKey(hash2)
	if err3 != nil {
		exist = false
		return
	}
	if value == nil {
		Log.Info("Block not found")
		fmt.Printf("区块哈希值为：%x\n", hash2)
		exist = false
		return
	} else {
		exist = true
		return
	}

}

//最后一个到特定区块有几个区块
func GetBlockCountUntilSpecificBlock(hash1 []byte) (count int) {
	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_PackBlocksArray:", err)
		return
	}
	defer bc.closeDB()

	//调用迭代器，输出blockChain
	it := bc.NewIterator()
	count = 0
	for {
		//调用Next方法，获取区块，游标左移
		block := it.Next()
		if block == nil {
			Log.Warn("发现致命错误，在收到别人的区块后查看长度时发现问题")
			Log.Warn("在区块遍历时发生错误")
			break
		}
		hashBytes := hash1
		var hash [32]byte
		copy(hash[:], hashBytes)
		if bytes.Equal([]byte(block.Hash), hash[:]) {
			Log.Debug("发现相等!")
			break
		}
		count++
		if block.PrevHash == nil {
			Log.Debug("区块链遍历结束!")
			break
		}
	}
	Log.Debug("最后一个到特定区块区块个数为:", count)
	return
}

//删除到特定区块中间的区块
func RmBlocksUntilSpecBlock(hash1 []byte) {
	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_PackBlocksArray:", err)
		return
	}

	defer bc.closeDB()

	//调用迭代器，输出blockChain
	it := bc.NewIterator()
	for {
		//调用Next方法，获取区块，游标左移
		block := it.Next()
		hashstr := string(hash1)
		// 将字符串转换为字节数组
		hashBytes, err := hex.DecodeString(hashstr)
		if err != nil {
			// 转换失败
			fmt.Println("字符串转换为字节数组失败！")
			return
		}
		fmt.Printf("Hash :---%x-----\n", block.Hash)
		var hash [32]byte
		copy(hash[:], hashBytes)
		if bytes.Equal([]byte(block.Hash), hash[:]) {
			{
				fmt.Println("发现相等!")
				// bc.db.Update(func(tx *bolt.Tx) error {
				// 	bucket := tx.Bucket([]byte(bucketBlock))
				// 	bucket.Put([]byte(lastBlockHashKey), []byte(block.Hash))
				// 	return nil
				// })
				createOrUpdateKeyValue([]byte(lastBlockHashKey), []byte(block.Hash))

				break
			}
		} else { //删除
			deleteKey(block.Hash)
			// err := bc.db.Update(func(tx *bolt.Tx) error {
			// 	b := tx.Bucket([]byte(bucketBlock))
			// 	return b.Delete(block.Hash)
			// })
			// if err != nil {
			// 	fmt.Println("删除区块失败:", err)
			// 	return
			// }
		}
		if block.PrevHash == nil {
			fmt.Println("区块链遍历结束!")
			//这里说明直接全删完了  那么直接把文件给删了
			deletedbfile()
			break
		}

	}
	return
}

//将区块数组中的区块都加入到数据库
func AppendBlocksToChain(blockarry []Block) {
	count := 0
	for _, block := range blockarry {
		addBlockToBucket(&block)
	}

	Log.Info("共插入数据", count)

}

//工具函数  将验证通过的区块中的交易（按照交易id）从交易池中删除（不安全，不再使用）
func delRecvBlockTxFromPool2222(block Block) {
	TxPoolmutex.Lock()
	defer TxPoolmutex.Unlock()
	for _, tx := range block.Transactions {
		txid := tx.TXID
		if _, ok := (TxPool)[string(txid)]; ok {
			//如果存在那么就删除
			delete(TxPool, string(txid))
			Log.Info("删除TxPool中一笔交易")
		}
	}
}

//请求区块的命令
func FetchBlocksRequest(getmsg chan []byte) {
	//如果conn为空 那么将进行广播要求获得最新的
	Log.Warn("主动请求区块")
	PackCommondAndToChan(FetchBlocks, getmsg)
}

//应该是让他来请求我
func pullok(getmsg chan []byte) {
	PackCommondAndToChan(Pullok, getmsg)
}

/*
	type DBfileMessage struct {
	DB            []byte // 数据库
	lastBlockHash string // 最后一个区块的哈希值
	MainblockLen  int    // 主链区块的长度
}

*/

//响应请求同步区块的处理函数
func FetchBlocksResponse(getmsg chan []byte) {
	Log.Info("响应请求，将打包区块并进行返回数据")
	DBfilemutex.Lock()
	defer DBfilemutex.Unlock()
	DBmutex.Lock()
	content, err := ioutil.ReadFile(blockchainDBFile)
	DBmutex.Unlock()
	if err != nil {
		Log.Warn("FetchBlocksResponse 中ioutil.ReadFile err:", err)
		return
	}
	len, err := getMainBlockLength()
	if err != nil {
		Log.Warn(err)
	}
	var message DBfileMessage
	message.DB = content
	message.lastBlockHash = getfinalBlock()
	message.MainblockLen = len
	PackDBfileMessageTaskAndToChan(ResponeFetchBlocks, message, getmsg)
}

//收到响应的处理函数
func GetFetchBlocksResponse(message *SendMessage, getmsg chan []byte) {
	if trystopMing() { //修改后简单粗暴，收到就暂停
		defer activeMing()
	}
	DBfilemutex.Lock()
	defer DBfilemutex.Unlock()
	Log.Info("响应请求，将打包区块并进行返回数据")
	dbfileMessage := byteToDBfileMessage(message.Payload)
	len, err := getMainBlockLength()
	if err != nil {
		Log.Warn(err)
	}
	//如果文件一样大或者不如我大，停止下一步操作
	Log.Info("mylen 的长度为：", len)
	Log.Info("dbfileMessage.MainblockLen：", dbfileMessage.MainblockLen)
	if len > dbfileMessage.MainblockLen {
		Log.Warn("我的区块更长，放弃更新")
		pullok(getmsg)
		return
	}
	// else if len == dbfileMessage.MainblockLen {
	// 	Log.Info("他发来了一样长的区块，还是看时间吧，看以后谁更长")
	// 	return
	// }
	DBmutex.Lock()
	err2 := os.WriteFile(blockchainDBFile, dbfileMessage.DB, 0644)
	DBmutex.Unlock()
	if err2 != nil {
		Log.Fatal("根据其他节点的区块链文件更新本地区块链失败")
		return
	}
	Log.Info("区块链同步成功")

}

//将产品加到池子中
func ReceiveProduct(message *SendMessage) {
	Log.Info("响应请求，将打包区块并进行返回数据")
	product := byteToproduct(message.Payload)
	ProductPoolMu.Lock()
	ProductPool = append(ProductPool, *product)
	Log.Info("添加到池子中成功，现在拥有产品数量:", len(ProductPool))
	for i, p := range ProductPool {
		fmt.Printf("第%d个产品id： %s\n", i, p.SourceID)
	}
	ProductPoolMu.Unlock()
}

//测试
func Test_moni_SendCommonBlocks() {

	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_PackBlocksArray:", err)
		return
	}

	defer bc.closeDB()
	var blockArray []*Block

	//调用迭代器，输出blockChain
	it := bc.NewIterator()
	i := 1
	for {
		//调用Next方法，获取区块，游标左移
		block := it.Next()

		// fmt.Printf("\n++++++++++++++++++++++\n")
		// fmt.Printf("Version : %d\n", block.Version)
		// fmt.Printf("PrevHash : %x\n", block.PrevHash)
		// fmt.Printf("MerkleRoot : %x\n", block.MerkleRoot)
		// fmt.Printf("TimeStamp : %d\n", block.TimeStamp)
		// fmt.Printf("Bits : %d\n", block.Bits)
		// fmt.Printf("Nonce : %d\n", block.Nonce)
		fmt.Printf("Hash :---%x-----\n", block.Hash)

		// 将字符串转换为字节数组
		hashBytes, err := hex.DecodeString("0000e208ab1df69ae7119e588ff9813f5183fb1257c032d8b4898ff67ef6fb76")
		if err != nil {
			// 转换失败
			fmt.Println("字符串转换为字节数组失败！")
			return
		}
		// 将字节数组赋值给hash
		var hash [32]byte
		copy(hash[:], hashBytes)

		blockArray = append(blockArray, block)

		if bytes.Equal([]byte(block.Hash), hash[:]) {
			{
				fmt.Println("发现相等!")
				break
			}
		}
		// fmt.Printf("Data : %s\n", block.Data)
		// fmt.Printf("Data : %s\n", block.Transactions[0].TXInputs[0].ScriptSig) //矿工写入的数据

		// pow := NewProofOfWork(block)
		// fmt.Printf("IsValid: %v\n", pow.IsValid())
		i++
		//退出条件

		if block.PrevHash == nil {
			fmt.Println("区块链遍历结束!")
			break
		}

	}

	Log.Info("找到了区块个数：", i)
	PackBlcokArrTaskAndToChan(SendCommonBlock, blockArray)
}

func Test_moni_SendTx() {

	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_tx:", err)
		return
	}

	defer bc.closeDB()
	var blockArray []*Block

	//调用迭代器，输出blockChain
	it := bc.NewIterator()
	i := 1
	for {
		//调用Next方法，获取区块，游标左移
		block := it.Next()

		// fmt.Printf("\n++++++++++++++++++++++\n")
		// fmt.Printf("Version : %d\n", block.Version)
		// fmt.Printf("PrevHash : %x\n", block.PrevHash)
		// fmt.Printf("MerkleRoot : %x\n", block.MerkleRoot)
		// fmt.Printf("TimeStamp : %d\n", block.TimeStamp)
		// fmt.Printf("Bits : %d\n", block.Bits)
		// fmt.Printf("Nonce : %d\n", block.Nonce)
		fmt.Printf("Hash :---%x-----\n", block.Hash)

		// 将字符串转换为字节数组
		hashBytes, err := hex.DecodeString("0000353abfa21effb30e49b2fa106a8516a3cefe4b2c3b19cbbefea570ab54d7")
		if err != nil {
			// 转换失败
			fmt.Println("字符串转换为字节数组失败！")
			return
		}
		// 将字节数组赋值给hash
		var hash [32]byte
		copy(hash[:], hashBytes)

		blockArray = append(blockArray, block)

		if bytes.Equal([]byte(block.Hash), hash[:]) {
			{
				fmt.Println("发现相等!")
				break
			}
		}
		// fmt.Printf("Data : %s\n", block.Data)
		// fmt.Printf("Data : %s\n", block.Transactions[0].TXInputs[0].ScriptSig) //矿工写入的数据

		// pow := NewProofOfWork(block)
		// fmt.Printf("IsValid: %v\n", pow.IsValid())
		i++
		//退出条件

		if block.PrevHash == nil {
			fmt.Println("区块链遍历结束!")
			break
		}

	}

	Log.Info("找到了区块个数：", i)
	PackBlcokArrTaskAndToChan(SendCommonBlock, blockArray)
}
