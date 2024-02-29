package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
)

//收到交易后的处理过程
func ReceiveTxArr(message *SendMessage) {
	fmt.Println("----------------------------------")
	Log.Info("进入ReceiveTxArr处理")
	txarry := byteToTransactionArr(message.Payload)

	for index, tx := range txarry {
		fmt.Println("\n\n+++++++++++++++++交易分割+++++++++++++++")
		fmt.Printf("第 %d 个交易的哈希值为: %x\n", index+1, tx.TXID)
		var sourceID uuid.UUID
		var pubkey []byte
		fmt.Printf("tx.TXID: %x\n", tx.TXID)
		//测试用
		// fmt.Printf("tx.TimeStamp: %x\n", tx.TimeStamp)
		// fmt.Printf("tx.TXInputs[0].PubKey %x\n", tx.TXInputs[0].PubKey)
		// fmt.Printf("tx.TXInputs[0].ScriptSig %x\n", tx.TXInputs[0].ScriptSig)
		// fmt.Printf("tx.TXInputs[0].Index %x\n", tx.TXInputs[0].Index)
		fmt.Printf("tx.TXInputs[0].Txid %x\n", tx.TXInputs[0].Txid)
		fmt.Printf("tx.TXInputs[0].ScriptPubKeyHash %x\n", tx.TXOutputs[0].ScriptPubKeyHash)
		fmt.Printf("tx.TXInputs[0].SourceID %x\n", tx.TXOutputs[0].SourceID)
		sourceID = tx.TXOutputs[0].SourceID
		//测试用
		// if tx.TXOutputs[0].SourceID == SSourceID {
		// 	Log.Warn("SSourceID 两者相等")
		// }
		for _, input := range tx.TXInputs {
			//直接打印交易
			fmt.Printf("上一笔交易的id：%x\n", input.Txid)
			fmt.Printf("上一笔交易的下标：%x\n", input.Index)
			fmt.Printf("上一个人的公钥：%x\n", input.PubKey)

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
		for _, output := range tx.TXOutputs {
			//直接打印交易
			fmt.Println("SourceID:", output.SourceID)
			break
		}

		//判断交易的合法性
		exist := CheckhasSourceID(sourceID, pubkey)
		//exist := CheckhasSourceID(SSourceID, PPubey)
		if exist {
			AddToTransactionPool(tx)
		} else {
			Log.Warn("交易验证没有通过，可能是sourceID所属权限不归属此人或者本节点存在区块链更新延迟")
		}
		break
	}
	/*
	 */

}

//这个主要是ReceiveTxArr收到广播的交易后并且验证成功了就会放到这里面
func AddToTransactionPool(tx *Transaction) {
	Log.Debug("向TxPacketPool中添加数据")
	key := string(tx.TXID)

	existflag := false
	//如果存在那么就不继续了
	TxPacketPoolmutex.Lock()
	defer TxPacketPoolmutex.Unlock()
	//放入缓存池
	TxPacketPool[key] = tx
	Log.Debug("TxPacketPool数据有：", len(TxPacketPool))
	if len(TxPacketPool) < TxPacketPoolMin {
		return
	}
	//超过一定数量，放到交易池中
	TxPoolmutex.Lock()
	for key, value := range TxPacketPool {
		TxPool[key] = value
		delete(TxPacketPool, key)
	}

	TxPoolmutex.Unlock()
	if MiningControlExist {
		fmt.Println("监听组发现挖矿控制协程存在，发送停止挖矿命令")
		InterruptChan <- 1
		existflag = true
	}

	Log.Debug("完成交易池更新过程")
	Log.Info("现在交易池中交易的数量:", len(TxPool))

	// if len(TxPool) > 5 {
	// 	//TxPoolcond.Broadcast()
	// 	Log.Info("广播有新的交易生成")
	// }
	Log.Debug("AddToTransactionPoo放弃交易池的锁")
	//如果人家本来在挖矿，那你这里就要给人家重新启动啊
	if existflag {
		MiningTimeStart <- 1
	}
}

//收到别人挖到的区块
func ReceiveMinedBlock(message *SendMessage) {
	Log.Info("进入ReceiveBlockArr处理")
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
	DBmutex.Lock()
	defer DBmutex.Unlock()
	if len(blockarry) == 0 {
		return
	}
	block := blockarry[0]

	exit := BlockExist(block.PrevHash)
	arry := RmBlockArryPointer(blockarry)
	if !exit {
		Log.Warn("区块前哈希不存在 将抛弃该区块")
		return
	} else { //如果区块前哈希存在

		fmt.Printf("前哈希值为:%x", block.PrevHash)
		myLen := GetBlockCountUntilSpecificBlock(block.PrevHash)
		hisLen := len(blockarry)
		Log.Info("myLen:", myLen, "  hisLen：", hisLen)
		if myLen > hisLen { //如果我到那个位置更多，那么抛弃他发来的数据
			Log.Info("我的更多一点，抛弃对方发来的数据")
		} else {
			Log.Info("根据对方发来的数据进行更新")
			RmBlocksUntilSpecBlock(block.PrevHash)
			//更新区块
			AppendBlocksToChain(arry[:])
			Log.Info("区块更新成功")
		}
	}
}

//收到区块后的处理过程  请求的区块
func ReceiveBlockArr(message *SendMessage) {
	Log.Info("进入ReceiveBlockArr处理")
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
	DBmutex.Lock()
	defer DBmutex.Unlock()
	if len(blockarry) == 0 {
		return
	}
	finalBlock := blockarry[len(blockarry)-1]
	exit := BlockExist(finalBlock.Hash)
	reverseBlockArr(blockarry)
	arry := RmBlockArryPointer(blockarry)
	if !exit {
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
func BlockExist(hash1 []byte) (exist bool) {

	hash2 := DBkeyhash(hash1)

	if !isFileExist(blockchainDBFile) {
		fmt.Println("区块链文件不存在！")
		return false
	}

	db, err := bolt.Open(blockchainDBFile, 0600, nil)
	if err != nil {
		Log.Info("BlockExist数据库打开失败")
		return false
	}
	defer db.Close()

	// 开启读事务
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			Log.Warn("Bucket 'block' not found")
			return fmt.Errorf("Bucket 'block' not found")
		}

		value := bucket.Get(hash2)
		if value == nil {
			Log.Info("Block not found")
			return fmt.Errorf("Block not found")
		}

		// 查询结果已经储存在变量 value 中，可以根据需要进行处理
		Log.Info("Block found: %x\n", value)
		exist = true
		return nil
	})

	if err != nil {
		return false
	}

	// 返回查询结果和成功状态
	return
}

//最后一个到特定区块有几个区块
func GetBlockCountUntilSpecificBlock(hash1 []byte) (count int) {
	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_PackBlocksArray:", err)
		return
	}

	defer bc.db.Close()

	//调用迭代器，输出blockChain
	it := bc.NewIterator()
	count = 0
	for {
		//调用Next方法，获取区块，游标左移
		block := it.Next()
		hashBytes := hash1
		//hashBytes := DBkeyhash(hash1)

		fmt.Printf("Hash :---%x-----\n", block.Hash)
		var hash [32]byte
		copy(hash[:], hashBytes)
		if bytes.Equal([]byte(block.Hash), hash[:]) {
			{
				fmt.Println("发现相等!")
				break
			}
		}
		count++
		if block.PrevHash == nil {
			fmt.Println("区块链遍历结束!")
			break
		}

	}
	Log.Info("最后一个到特定区块区块个数为:", count)
	return
}

//删除到特定区块中间的区块
func RmBlocksUntilSpecBlock(hash1 []byte) {
	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_PackBlocksArray:", err)
		return
	}

	defer bc.db.Close()

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
				bc.db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket([]byte(bucketBlock))
					bucket.Put([]byte(lastBlockHashKey), []byte(block.Hash))
					return nil
				})

				break
			}
		} else { //删除
			err := bc.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(bucketBlock))
				return b.Delete(block.Hash)
			})
			if err != nil {
				fmt.Println("删除区块失败:", err)
				return
			}
		}
		if block.PrevHash == nil {
			fmt.Println("区块链遍历结束!")
			//这里说明直接全删完了  那么直接把文件给删了
			err := os.Remove(blockchainDBFile)
			if err != nil {
				// 删除失败
				fmt.Println("删除数据库文件失败：", err)
			} else {
				// 删除成功
				fmt.Println("数据库文件已删除！")
			}
			break
		}

	}
	return
}

//将区块数组中的区块都加入到数据库
func AppendBlocksToChain(blockarry []Block) {
	//如果不存在会自动创建
	db, err := bolt.Open(blockchainDBFile, 0600, nil)
	if err != nil {
		Log.Fatal("创建或者打开数据库失败")
	}
	defer db.Close()
	count := 0
	for _, block := range blockarry {
		err = db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketBlock))
			if bucket == nil {
				//创建bucket
				var err error
				bucket, err = tx.CreateBucket([]byte(bucketBlock))
				if err != nil {
					return err
				}
			}
			// 1. 检查数据库中是否已经存在该区块
			// if bucket.Get(block.Hash) != nil {
			// 	Log.Fatal("发现逻辑错误，在加入区块时发现区块已经存在")
			// 	return nil
			// }

			// 2. 如果该区块不存在，将其写入到数据库中
			blockdata := block.Serialize()
			bucket.Put(block.Hash, blockdata)
			count++
			// 更新最后区块哈希值到数据库
			bucket.Put([]byte(lastBlockHashKey), block.Hash)
			//更新交易池
			delRecvBlockTxFromPool(block)
			return nil
		})
		if err != nil {
			Log.Fatal("插入数据库失败")
		}
	}

	Log.Info("共插入数据", count)

}

//工具函数  将验证通过的区块中的交易从交易池中删除
func delRecvBlockTxFromPool(block Block) {
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
func RequestBlockArr() {

}

//请求交易的命令
func RequestTXArr() {

}

//测试
func Test_moni_SendCommonBlocks() {

	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_PackBlocksArray:", err)
		return
	}

	defer bc.db.Close()
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

func Test_moni_SendTx() {

	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("Test_tx:", err)
		return
	}

	defer bc.db.Close()
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
