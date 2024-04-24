package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

//定义交易结构
type Transaction struct {
	TXID      []byte     //交易id
	TXInputs  []TXInput  //可以有多个输入
	TXOutputs []TXOutput //可以有多个输出
	TimeStamp uint64     //创建交易的时间
	TxHash    []byte
	//本溯源线上一个节点
	PoI []byte

	//当前主链的最后一个区块(暂时留空)
	CP []byte

	//指向还没有TBP指向的区块
	TBP []byte
	//标识为溯源线的第一个节点  如果是的话就第一个节点
	SourceStrat bool
	TXmsg       []byte
}

type TXInput struct {
	Txid  []byte //这个input所引用的output所在的交易id
	Index int64  //这个input所引用的output在交易中的索引

	// ScriptSig string //付款人对当前交易(新交易，而不是引用的交易)的签名
	ScriptSig []byte //对当前交易的签名
	PubKey    []byte //付款人的公钥
}

type TXOutput struct {
	ScriptPubKeyHash []byte    //收款人的公钥哈希
	SourceID         uuid.UUID //溯源id
}

var SSourceID uuid.UUID  //测试用
var PPubKeyHash []byte   //测试用
var PPubey []byte        //测试用
var lastFindBlock []byte //上次已经遍历到的区块 这样在自动发送交易的时候就不会再遍历前面的了
//由于没有办法直接将地址赋值给TXoutput，所以需要提供一个output的方法
func newTXOutput(address string, SourceID1 uuid.UUID) TXOutput {
	output := TXOutput{SourceID: SourceID1}

	//通过地址获取公钥哈希值
	pubKeyHash := getPubKeyHashFromAddress(address)
	output.ScriptPubKeyHash = pubKeyHash

	return output
}

// # 获取交易ID
// 对交易做哈希处理
func (tx *Transaction) setHash() error {
	//对tx做gob编码得到字节流，做sha256，赋值给TXID
	var buffer bytes.Buffer

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(tx)
	if err != nil {
		fmt.Println("encode err:", err)
		return err
	}

	hash := sha256.Sum256(buffer.Bytes())

	//我们使用tx字节流的哈希值作为交易id
	tx.TXID = hash[:]
	return nil
}

//挖矿奖励
var reward = 12.5

// # 创建挖矿交易
func NewCoinbaseTx(miner /*挖矿人*/ string, data string) *Transaction {
	//特点：没有输入，只有一个输出，得到挖矿奖励
	//挖矿交易需要能够识别出来，没有input，所以不需要签名，
	//挖矿交易不需要签名，所以这个签名字段可以书写任意值，只有矿工有权利写
	//中本聪：写的创世语
	//现在都是由矿池来写，写自己矿池的名字
	input := TXInput{Txid: nil, Index: -1, ScriptSig: nil, PubKey: []byte(data)}

	//创建output
	// output := TXOutput{Value: reward, ScriptPubk: miner}

	//修改 output := newTXOutput(miner, reward)
	SourceID := uuid.New()
	output := newTXOutput(miner, SourceID)

	timeStamp := time.Now().Unix()

	tx := Transaction{
		TXID:      nil,
		TXInputs:  []TXInput{input},
		TXOutputs: []TXOutput{output},
		TimeStamp: uint64(timeStamp),
		PoI:       nil, // 本溯源线上一个节点
		CP:        nil, // 当前主链的最后一个区块
		TBP:       nil, // 指向还没有TBP指向的区块
	}

	tx.setHash()
	return &tx
}

//判断一笔交易是否为挖矿交易
func (tx *Transaction) isCoinbaseTx() bool {
	inputs := tx.TXInputs
	//input个数为1，id为nil，索引为-1
	if len(inputs) == 1 && inputs[0].Txid == nil && inputs[0].Index == -1 {
		return true
	}
	return false
}

//创建普通交易  弃用
// 1. from/*付款人*/,to/*收款人*/,amount输入参数/*金额*/
func NewTransaction(from, to string, SourceID uuid.UUID, bc *BlockChain) *Transaction {
	//钱包就是在这里使用的，from=》钱包里面找到对应的wallet-》私钥-》签名
	wm := NewWalletManager()
	if wm == nil {
		fmt.Println("打开钱包失败!")
		return nil
	}

	// 钱包里面找到对应的wallet
	wallet, ok := wm.Wallets[from]
	if !ok {
		fmt.Println("没有找到付款人地址对应的私钥!")
		return nil
	}

	fmt.Println("找到付款人的私钥和公钥，准备创建交易...")

	priKey := wallet.PriKey //私钥签名阶段使用，暂且注释掉
	pubKey := wallet.PubKey
	//我们的所有output都是由公钥哈希锁定的，所以去查找付款人能够使用的output时，也需要提供付款人的公钥哈希值
	pubKeyHash := getPubKeyHashFromPubKey(pubKey)

	// 2. 遍历账本，找到from满足条件utxo集合（3），返回这些utxo包含的总金额(15)

	//包含所有将要使用的utxo集合
	var spentUTXO = make(map[string][]int64)
	//这些使用utxo包含总金额
	//var retValue float64

	//遍历账本，找到from能够使用utxo集合,以及这些utxo包含的钱
	// spentUTXO, retValue = bc.findNeedUTXO(from, amount)
	spentUTXO, havetheid := bc.findNeedUTXO(pubKeyHash, SourceID)
	// map[0x222] = []int{0}
	// map[0x333] = []int{0,1}
	if !havetheid {
		fmt.Println("当前SourceID终点不为当前节点所有！")
		return nil
	}

	// 3. 如果金额不足，创建交易失败
	// if retValue < amount {
	// 	fmt.Println("金额不足，创建交易失败!")
	// 	return nil
	// }
	var inputs []TXInput
	var outputs []TXOutput

	// 4. 拼接inputs
	// > 遍历utxo集合，每一个output都要转换为一个input(3)
	for txid, indexArray := range spentUTXO {
		//遍历下标, 注意value才是我们消耗的output的下标
		for _, i := range indexArray {
			input := TXInput{Txid: []byte(txid), Index: i, ScriptSig: nil, PubKey: pubKey}
			inputs = append(inputs, input)
		}
	}

	// 5. 拼接outputs
	// > 创建一个属于to的output
	//创建给收款人的output
	output1 := newTXOutput(to, SourceID)
	outputs = append(outputs, output1)

	// > 如果总金额大于需要转账的金额，进行找零：给from创建一个output
	//修改后没有找零操作
	// if retValue > amount {
	// 	// output2 := TXOutput{from, retValue - amount}
	// 	output2 := newTXOutput(from, retValue-amount)
	// 	outputs = append(outputs, output2)
	// }

	timeStamp := time.Now().Unix()

	// 6. 设置哈希，返回
	//tx := Transaction{nil, inputs, outputs, uint64(timeStamp)}
	tx := Transaction{
		TXID:      nil,
		TXInputs:  inputs,
		TXOutputs: outputs,
		TimeStamp: uint64(timeStamp), // 创建交易的时间戳
		PoI:       nil,               // 本溯源线上一个节点
		CP:        nil,               // 当前主链的最后一个区块
		TBP:       nil,               // 指向还没有TBP指向的区块
	}

	tx.setHash()

	if !bc.signTransaction(&tx, priKey) {
		fmt.Println("交易签名失败")
		return nil
	}
	return &tx
}

//实现具体签名动作（copy，设置为空，签名动作）
//参数1：私钥
//参数2：inputs所引用的output所在交易的集合:
// > key :交易id
// > value：交易本身
//大概意思就是将引用的交易的TXOutput放到本交易的TXInput的PubKey，求此时的交易哈希，然后对这个交易哈希签名得到数据
func (tx *Transaction) sign(priKey *ecdsa.PrivateKey, prevTxs map[string]*Transaction) bool {
	fmt.Println("具体对交易签名sign...")
	//这个地方不会执行  因为没有挖矿交易
	if tx.isCoinbaseTx() {
		fmt.Println("找到挖矿交易，无需签名!")
		return true
	}

	//1. 获取交易copy，pubKey，ScriptPubKey字段置空
	txCopy := tx.trimmedCopy()

	//2. 遍历交易的inputs for, 注意，不要遍历tx本身，而是遍历txCopy
	for i, input := range txCopy.TXInputs {
		fmt.Printf("开始对input[%d]进行签名...\n", i)

		prevTx := prevTxs[string(input.Txid)]
		if prevTx == nil {
			return false
		}

		//input引用的output
		output := prevTx.TXOutputs[input.Index]

		// > 获取引用的output的公钥哈希
		//for range是input是副本，不会影响到变量的结构
		// input.PubKey = output.ScriptPubKeyHash
		txCopy.TXInputs[i].PubKey = output.ScriptPubKeyHash

		// > 对copy交易进行签名，需要得到交易的哈希值
		txCopy.setHash()

		// > 将input的pubKey字段置位nil, 还原数据，防止干扰后面input的签名
		txCopy.TXInputs[i].PubKey = nil

		hashData := txCopy.TXID //我们去签名的具体数据

		//> 开始签名
		r, s, err := ecdsa.Sign(rand.Reader, priKey, hashData)
		if err != nil {
			fmt.Println("签名失败!")
			return false
		}
		signature := append(r.Bytes(), s.Bytes()...)

		// > 将数字签名赋值给原始tx
		tx.TXInputs[i].ScriptSig = signature
	}

	fmt.Println("交易签名成功!")
	return true
}

func (tx *Transaction) signByPreScript(priKey *ecdsa.PrivateKey, prescript []byte) bool {
	//fmt.Println("具体对交易签名sign...")

	//1. 获取交易copy，pubKey，ScriptPubKey字段置空
	txCopy := tx.trimmedCopy()

	//2. 遍历交易的inputs for, 注意，不要遍历tx本身，而是遍历txCopy

	//fmt.Printf("开始对input进行签名...\n")

	// > 获取引用的output的公钥哈希
	//for range是input是副本，不会影响到变量的结构
	// input.PubKey = output.ScriptPubKeyHash
	txCopy.TXInputs[0].PubKey = prescript

	// > 对copy交易进行签名，需要得到交易的哈希值
	txCopy.setHash()

	// > 将input的pubKey字段置位nil, 还原数据，防止干扰后面input的签名
	txCopy.TXInputs[0].PubKey = nil

	hashData := txCopy.TXID //我们去签名的具体数据

	//> 开始签名
	r, s, err := ecdsa.Sign(rand.Reader, priKey, hashData)
	if err != nil {
		fmt.Println("签名失败!")
		return false
	}
	signature := append(r.Bytes(), s.Bytes()...)

	// > 将数字签名赋值给原始tx
	tx.TXInputs[0].ScriptSig = signature
	Log.Debug("交易签名成功!")
	//fmt.Println("交易签名成功!")
	return true
}

//trim修剪, 签名和校验时都会使用
func (tx *Transaction) trimmedCopy() *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	//创建一个交易副本，每一个input的pubKey和Sig都设置为空。
	for _, input := range tx.TXInputs {
		input := TXInput{
			Txid:      input.Txid,
			Index:     input.Index,
			ScriptSig: nil,
			PubKey:    nil,
		}
		inputs = append(inputs, input)
	}

	outputs = tx.TXOutputs

	//txCopy := Transaction{tx.TXID, inputs, outputs, tx.TimeStamp}

	txCopy := Transaction{
		TXID:      tx.TXID,
		TXInputs:  inputs,
		TXOutputs: outputs,
		TimeStamp: tx.TimeStamp, // 创建交易的时间戳
		PoI:       tx.PoI,       // 本溯源线上一个节点
		CP:        tx.CP,        // 当前主链的最后一个区块
		TBP:       tx.TBP,       // 指向还没有TBP指向的区块
	}

	return &txCopy
}

//具体校验
func (tx *Transaction) verify(prevTxs map[string]*Transaction) bool {
	Log.Debug("verify具体校验")
	//1. 获取交易副本txCopy
	txCopy := tx.trimmedCopy()
	//2. 遍历交易，inputs，
	for i, input := range tx.TXInputs {
		prevTx := prevTxs[string(input.Txid)]
		if prevTx == nil {
			return false
		}

		//3. 还原数据（得到引用output的公钥哈希）获取交易的哈希值
		output := prevTx.TXOutputs[input.Index]
		txCopy.TXInputs[i].PubKey = output.ScriptPubKeyHash
		txCopy.setHash()

		//清零环境, 设置为nil
		txCopy.TXInputs[i].PubKey = nil

		//具体还原的签名数据哈希值
		hashData := txCopy.TXID
		//签名
		signature := input.ScriptSig
		//公钥的字节流
		pubKey := input.PubKey

		//开始校验
		var r, s, x, y big.Int
		//r,s 从signature截取出来
		r.SetBytes(signature[:len(signature)/2])
		s.SetBytes(signature[len(signature)/2:])

		//x, y 从pubkey截取除来，还原为公钥本身
		x.SetBytes(pubKey[:len(pubKey)/2])
		y.SetBytes(pubKey[len(pubKey)/2:])
		curve := elliptic.P256()
		pubKeyRaw := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

		//进行校验
		res := ecdsa.Verify(&pubKeyRaw, hashData, &r, &s)
		if !res {
			fmt.Println("发现校验失败的input!")
			return false
		}
	}
	//4. 通过tx.ScriptSig, tx.PubKey进行校验
	fmt.Println("交易校验成功!")

	return true
}

func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("\n--- Transaction txid%x:", tx.TXID))
	lines = append(lines, fmt.Sprintf("	 txhash: %x", tx.TxHash))
	lines = append(lines, fmt.Sprintf("	 PoI: %x", tx.PoI))
	lines = append(lines, fmt.Sprintf("	 CP:  %x", tx.CP))
	lines = append(lines, fmt.Sprintf("	 TBP: %x", tx.TBP))
	lines = append(lines, fmt.Sprintf("	 SourceStrat:%t", tx.SourceStrat))
	lines = append(lines, fmt.Sprintf("	 TXmsg:%s", string(tx.TXmsg)))
	for i, input := range tx.TXInputs {

		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("     	  TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("      	  Index:       %d", input.Index))
		lines = append(lines, fmt.Sprintf("      	  Signature: %x", input.ScriptSig))
		lines = append(lines, fmt.Sprintf("      	  PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.TXOutputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("     	   SourceID :  %s", output.SourceID))
		lines = append(lines, fmt.Sprintf("     	   Script: %x", output.ScriptPubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// 绑定Serialize方法， gob编码
func (tx *Transaction) Serialize_tx() []byte {
	var buffer bytes.Buffer

	//创建编码器
	encoder := gob.NewEncoder(&buffer)
	//编码
	err := encoder.Encode(tx)
	if err != nil {
		fmt.Printf("Serialize_tx Encode err:", err)
		return nil
	}

	return buffer.Bytes()
}

// 反序列化，输入[]byte，返回tx
func Deserialize_tx(src []byte) *Transaction {
	var tx Transaction
	//创建解码器
	decoder := gob.NewDecoder(bytes.NewReader(src))
	//解码
	err := decoder.Decode(&tx)
	if err != nil {
		Log.Warn("Deserialize_tx decode err: %s", err)
		//fmt.Printf("%x\n", src)
		return nil
	}
	return &tx
}

func getMyhaveId2() (myhaveid []uuid.UUID) {
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

	myaddress := mywallet.Address
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
	//碰到diffBlock就停止向下查询 防止无意义的查询过程
	diffBlock := lastFindBlock
	flag := 0
	for {
		//遍历区块
		block := it.Next()
		if flag == 0 {
			lastFindBlock = block.Hash
			flag = 1
		}
		Log.Debug("遍历区块")
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
			//fmt.Printf("sideblock.TXOutputs[0].ScriptPubKeyHash %x\n", sideblock.TXOutputs[0].ScriptPubKeyHash)
			//fmt.Printf("pubKeyHash %x\n", blockpubhash)
			if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, blockpubhash) {
				_, existss := usedmap[uuid_]
				if existss {
					Log.Debug("该id已经转移:", existss)
					continue
				}
				myhaveid = append(myhaveid, uuid_)
			}
		}
		//验证起始点是否是自己的 要求保证这个没有用过
		if bytes.Equal(block.Product.Pubkeyhash, blockpubhash) {
			_, ok := hased[block.Product.SourceID]
			if ok {
			} else { //没有使用过
				_, existss := usedmap[block.Product.SourceID]
				if existss {
					Log.Debug("该id已经转移:", existss)
				} else {
					hased[block.Product.SourceID] = 1
					myhaveid = append(myhaveid, block.Product.SourceID)
				}
			}
		}
		if block.PrevHash == nil || bytes.Equal(block.PrevHash, diffBlock) {
			break
		}
	}
	Log.Debug("没有找到该溯源id")
	return
}

func getMyhaveId() (myhaveid []uuid.UUID) {
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

	myaddress := mywallet.Address
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
	//碰到diffBlock就停止向下查询 防止无意义的查询过程
	diffBlock := lastFindBlock
	flag := 0
	for {
		//遍历区块
		block := it.Next()
		if flag == 0 {
			lastFindBlock = block.Hash
			flag = 1
		}
		Log.Debug("遍历区块")
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
			//fmt.Printf("sideblock.TXOutputs[0].ScriptPubKeyHash %x\n", sideblock.TXOutputs[0].ScriptPubKeyHash)
			//fmt.Printf("pubKeyHash %x\n", blockpubhash)
			if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, blockpubhash) {
				_, existss := usedmap[uuid_]
				if existss {
					Log.Debug("该id已经转移:", existss)
					continue
				}
				myhaveid = append(myhaveid, uuid_)
			}
		}
		//验证起始点是否是自己的 要求保证这个没有用过
		if bytes.Equal(block.Product.Pubkeyhash, blockpubhash) {
			_, ok := hased[block.Product.SourceID]
			if ok {
			} else { //没有使用过
				_, existss := usedmap[block.Product.SourceID]
				if existss {
					Log.Debug("该id已经转移:", existss)
				} else {
					hased[block.Product.SourceID] = 1
					myhaveid = append(myhaveid, block.Product.SourceID)
				}
			}
		}
		if block.PrevHash == nil || bytes.Equal(block.PrevHash, diffBlock) {
			break
		}
	}
	Log.Debug("没有找到该溯源id")
	return
}
func StopAutosendTX() {
	if Autosendtx != nil {
		Autosendtx <- 1
		Log.Debug("发送请求成功")
	} else {
		Log.Warn("自动发送交易后台服务未开启")
	}
}

//启动自动发送交易
func StartAutosendTX(cli *CLI) {
	Autosendtx = make(chan int)
	transferTraceability6("", "", cli)
	return
	for {
		select {
		case <-Autosendtx: //发来了取消  cli.addBlock(data)
			Log.Debug("自动发送交易后台协程被销毁")
			return
		case <-time.After(4 * time.Second):
			Log.Debug("自动生成一笔交易并发送")
			transferTraceability6("", "", cli)
			//transferTraceability("", "", cli)
		}

	}
}
func (cli *CLI) StartAutoNewBlock() {
	Log.Debug("StartAutoNewBlock=============")
	for {
		select {
		case <-time.After(4 * time.Second):
			cli.addBlock("自动生成的一个区块")
			Log.Debug("自动生成的一个区块")
		}
	}
}

var count11 = 0
var firstin = false

//下面下面函数的重新修改
func transferTraceability2(to string, msg string, cli *CLI) {
	//1.获取在线的账户
	tostr, _ := getOnlineAddres()
	to = string(tostr)

	//2.获取区块链实例
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Info(ms)
		return
	}
	txs := make([]*Transaction, 0)
	var hased = make(map[uuid.UUID]int)
	var usedmap = make(map[uuid.UUID]int)
	var myhaveid []uuid.UUID

	//txpool里面存放是交易id，为了快速，这里将这个id放到里面（存在不一致性）
	TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()
	//3.遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()
		fmt.Println("区块的ConfirmedLists里的账户有：", len(block.ConfirmedLists))
		//下面是对已经转换一次的进行处理
		for uuid, confirmedBlockList := range block.ConfirmedLists {
			fmt.Println("遍历")
			//如果前面已经用到了 那么就退出
			_, ok := hased[uuid]
			if ok {
				continue
			}
			//没有使用过  这里仅仅判断usedmap里面是否有这个id
			_, existss := usedmap[uuid]
			if existss {
				Log.Debug("该id已经转移:", existss)
				continue
			}
			//到这里说明没有用到
			hased[uuid] = 1
			//新建一个交易区块
			var tx *Transaction
			//反序列化被引用的区块
			blockArray := confirmedBlockList.BlockArray
			finalConfirmedBlock := blockArray[len(blockArray)-1]
			finalblockhash := finalConfirmedBlock.Hash
			sideblockinfo, err := getValueByKey(finalblockhash)
			fmt.Printf("finalblockhash  %x\n", finalblockhash)
			if err != nil {
				Log.Warn("GetSideUTXOBySourceId出现错误，", err)
				continue
			}
			sideblock := Deserialize_tx(sideblockinfo)
			txid := string(sideblock.TXID)
			Blockhash := finalConfirmedBlock.Hash
			//创建交易区块
			tx, err2 = CreateTransactionBlockFromUTXO(txid, 0, to, uuid, Blockhash, nil)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionFromUTXO出现错误 ：", err2)
				continue
			}
			//对一些信息进行设置
			SetTxDetail(tx, msg, to)
			//无用的东西
			SSourceID = tx.TXOutputs[0].SourceID
			Log.Debug("开始进行广播")
			AddToTransactionPool(tx)
			//printtx(tx)
			txs = append(txs, tx)
			myhaveid = append(myhaveid, uuid)
		}
		//fmt.Printf("block.Product.Pubkeyhash:  %x\n", block.Product.Pubkeyhash)
		//fmt.Printf("pubKeyHash:%x\n", pubKeyHash)

		//2.对没有用过的进行处理
		_, ok := hased[block.Product.SourceID]
		if ok {
		} else { //没有使用过   第一个参数是txid，这里类似创世区块，创世区块没有txid
			tx, err2 := CreateTransactionSpec(string(block.Product.StartTxid), 0, to, block.Product.SourceID, block.Hash, nil)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionSpec出现错误 ：", err2)
			}
			_, existss := usedmap[block.Product.SourceID]
			if existss {
				Log.Debug("该id已经转移:", existss)
			} else {
				//对交易设置一些关键信息
				SetTxDetail(tx, msg, to)
				AddToTransactionPool(tx)
				//printtx(tx)
				txs = append(txs, tx)
				hased[block.Product.SourceID] = 1
				myhaveid = append(myhaveid, block.Product.SourceID)
			}
		}

		//退出循环
		if block.PrevHash == nil {
			break
		}

	}
	Log.Info("共广播交易区块:", len(myhaveid))
	return
}

var tempin3 *Block
var tempcount = 0

func transferTraceability6(to string, msg string, cli *CLI) {

	tempcount++
	if tempcount == 20 {
		tempin3 = nil
		tempcount = 0
	}

	//1.获取在线的账户
	tostr, _ := getRandomAddres()
	to = string(tostr)

	//2.获取区块链实例
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Info(ms)
		return
	}
	txs := make([]*Transaction, 0)
	var hased = make(map[uuid.UUID]int)
	var usedmap = make(map[uuid.UUID]int)
	var myhaveid []uuid.UUID
	//获得本账户
	myaddress := mywallet.Address
	Log.Warn(myaddress)

	//txpool里面存放是交易id，为了快速，这里将这个id放到里面（存在不一致性）
	TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()
	//3.遍历区块 找到
	it := bc.NewIterator()
	pretempblock := tempin3
	firstin := true
	for {
		startTime := time.Now()
		//遍历区块
		block := it.MemoryNextBlock()
		if firstin {
			tempin3 = block
			firstin = false
		}
		if pretempblock != nil && bytes.Equal(pretempblock.Hash, block.Hash) {
			Log.Info("遍历上次内容")
			return
		}
		fmt.Println("区块的ConfirmedLists里的账户有：", len(block.ConfirmedLists))
		//下面是对已经转换一次的进行处理
		for uuid, confirmedBlockList := range block.ConfirmedLists {
			//如果前面已经用到了 那么就退出
			_, ok := hased[uuid]
			if ok {
				continue
			}
			//没有使用过  这里仅仅判断usedmap里面是否有这个id
			_, existss := usedmap[uuid]
			if existss {
				Log.Debug("该id已经转移:", existss)
				continue
			}

			//到这里说明没有用到
			hased[uuid] = 1
			//新建一个交易区块
			var tx *Transaction
			//反序列化被引用的区块
			blockArray := confirmedBlockList.BlockArray
			finalConfirmedBlock := blockArray[len(blockArray)-1]
			finalblockhash := finalConfirmedBlock.Hash
			sideblock := GetTxFromMemory(finalblockhash)
			//获取交易的wallet
			txwallet, errget := getWalletFromTx(sideblock.TXOutputs[0].ScriptPubKeyHash)
			if errget != nil {
				Log.Fatal("没有获取到交易对应的wallet")
			} else {
				//fmt.Println("找到账户为", txwallet.Address)
			}
			txid := string(sideblock.TXID)
			//Blockhash := finalConfirmedBlock.Hash
			//Blockhash := sideblock.
			//创建交易区块
			tx, err2 = CreateTransactionBlockFromUTXO(txid, 0, to, uuid, finalblockhash, &txwallet)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionFromUTXO出现错误 ：", err2)
				continue
			}
			fmt.Println(tx)
			//对一些信息进行设置
			SetTxDetail(tx, msg, to)
			//无用的东西
			SSourceID = tx.TXOutputs[0].SourceID
			Log.Debug("开始进行广播")
			AddToTransactionPool(tx)
			//printtx(tx)
			txs = append(txs, tx)
			myhaveid = append(myhaveid, uuid)
		}

		//2.对没有用过的进行处理

		txwallet2, errget2 := getWalletFromTx(block.Product.Pubkeyhash)
		if errget2 != nil {
			Log.Fatal("没有获取到交易对应的wallet")
		} else {
			//fmt.Println("找到账户为", txwallet2.Address)
		}
		_, ok := hased[block.Product.SourceID]
		if ok {
		} else { //没有使用过   第一个参数是txid，这里类似创世区块，创世区块没有txid

			tx, err2 := CreateTransactionSpec(string(block.Product.StartTxid), 0, to, block.Product.SourceID, block.Hash, &txwallet2)
			Log.Debug("第一次转移！")
			fmt.Println(tx)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionSpec出现错误 ：", err2)
			}
			_, existss := usedmap[block.Product.SourceID]
			if existss {
				Log.Debug("该id已经转移:", existss)
			} else {
				//对交易设置一些关键信息
				SetTxDetail(tx, msg, to)
				AddToTransactionPool(tx)
				//printtx(tx)
				txs = append(txs, tx)
				hased[block.Product.SourceID] = 1
				myhaveid = append(myhaveid, block.Product.SourceID)
			}
		}

		//退出循环
		if block.PrevHash == nil {
			break
		}
		endTime := time.Now()
		duration1 := endTime.Sub(startTime)
		Log.Info("时间", duration1)
	}
	Log.Info("共广播交易区块:", len(myhaveid))

	return
	//PackTxArrTaskAndToChan(SendTX, txs)

}

func transferTraceability5(to string, msg string, cli *CLI) {

	tempcount++
	if tempcount == 20 {
		tempin3 = nil
		tempcount = 0
	}

	//1.获取在线的账户
	tostr, _ := getRandomAddres()
	to = string(tostr)

	//2.获取区块链实例
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Info(ms)
		return
	}
	txs := make([]*Transaction, 0)
	var hased = make(map[uuid.UUID]int)
	var usedmap = make(map[uuid.UUID]int)
	var myhaveid []uuid.UUID
	//获得本账户
	myaddress := mywallet.Address
	Log.Warn(myaddress)

	//txpool里面存放是交易id，为了快速，这里将这个id放到里面（存在不一致性）
	TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()
	//3.遍历区块 找到
	it := bc.NewIterator()
	pretempblock := tempin3
	firstin := true
	for {
		startTime := time.Now()
		//遍历区块
		block := it.MemoryNextBlock()
		txslen := len(txs)
		if firstin {
			tempin3 = block
			firstin = false
		}
		if pretempblock != nil && bytes.Equal(pretempblock.Hash, block.Hash) {
			Log.Info("遍历上次内容")
			return
		}
		fmt.Println("区块的ConfirmedLists里的账户有：", len(block.ConfirmedLists))
		//下面是对已经转换一次的进行处理
		hashpre := make(map[uuid.UUID][]byte)
		for uuid_, confirmedBlockList := range block.ConfirmedLists {
			//如果前面已经用到了 那么就退出
			_, ok := hased[uuid_]
			if ok {
				continue
			}
			//没有使用过  这里仅仅判断usedmap里面是否有这个id
			_, existss := usedmap[uuid_]
			if existss {
				Log.Debug("该id已经转移:", existss)
				continue
			}
			//使用了才能加这个
			//hased[uuid_] = 1
			blockArray := confirmedBlockList.BlockArray
			finalConfirmedBlock := blockArray[len(blockArray)-1]
			finalblockhash := finalConfirmedBlock.Hash
			hashpre[uuid_] = finalblockhash
		}
		hashget, _ := getValuesByKeys(hashpre)
		for uuid_, sideblockinfo := range hashget {
			hased[uuid_] = 1
			var tx *Transaction
			sideblock := Deserialize_tx(sideblockinfo)
			//获取交易的wallet
			txwallet, errget := getWalletFromTx(sideblock.TXOutputs[0].ScriptPubKeyHash)
			if errget != nil {
				Log.Fatal("没有获取到交易对应的wallet")
			} else {
				//fmt.Println("找到账户为", txwallet.Address)
			}
			txid := string(sideblock.TXID)
			//Blockhash := hashpre[uuid_]
			sideblockPOI := sideblock.TxHash
			//创建交易区块
			tx, err2 = CreateTransactionBlockFromUTXO(txid, 0, to, uuid_, sideblockPOI, &txwallet)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionFromUTXO出现错误 ：", err2)
				continue
			}
			//对一些信息进行设置
			SetTxDetail(tx, msg, to)
			//无用的东西
			SSourceID = tx.TXOutputs[0].SourceID
			Log.Debug("开始进行广播")
			AddToTransactionPool(tx)
			//printtx(tx)
			txs = append(txs, tx)
			myhaveid = append(myhaveid, uuid_)
		}

		//2.对没有用过的进行处理
		txwallet2, errget2 := getWalletFromTx(block.Product.Pubkeyhash)
		if errget2 != nil {
			Log.Fatal("没有获取到交易对应的wallet")
		} else {
			//fmt.Println("找到账户为", txwallet2.Address)
		}
		_, ok := hased[block.Product.SourceID]
		if ok {
		} else { //没有使用过   第一个参数是txid，这里类似创世区块，创世区块没有txid

			tx, err2 := CreateTransactionSpec(string(block.Product.StartTxid), 0, to, block.Product.SourceID, block.Hash, &txwallet2)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionSpec出现错误 ：", err2)
			}
			_, existss := usedmap[block.Product.SourceID]
			if existss {
				Log.Debug("该id已经转移:", existss)
			} else {
				//对交易设置一些关键信息
				SetTxDetail(tx, msg, to)
				AddToTransactionPool(tx)
				//printtx(tx)
				txs = append(txs, tx)
				hased[block.Product.SourceID] = 1
				myhaveid = append(myhaveid, block.Product.SourceID)
			}
		}

		//退出循环
		if block.PrevHash == nil {
			break
		}
		endTime := time.Now()
		duration1 := endTime.Sub(startTime)
		Log.Info("时间", duration1)
		Log.Info("添加元素", len(txs)-txslen)

	}
	Log.Info("共广播交易区块:", len(myhaveid))

	//PackTxArrTaskAndToChan(SendTX, txs)
}

func transferTraceability4(to string, msg string, cli *CLI) {

	tempcount++
	if tempcount == 50 {
		tempin3 = nil
		tempcount = 0
	}

	//1.获取在线的账户
	tostr, _ := getRandomAddres()
	to = string(tostr)

	//2.获取区块链实例
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Info(ms)
		return
	}
	txs := make([]*Transaction, 0)
	var hased = make(map[uuid.UUID]int)
	var usedmap = make(map[uuid.UUID]int)
	var myhaveid []uuid.UUID
	//获得本账户
	myaddress := mywallet.Address
	Log.Warn(myaddress)

	//txpool里面存放是交易id，为了快速，这里将这个id放到里面（存在不一致性）
	TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()
	//3.遍历区块 找到
	it := bc.NewIterator()
	pretempblock := tempin3
	firstin := true
	for {
		startTime := time.Now()
		//遍历区块
		block := it.Next()
		txslen := len(txs)
		if firstin {
			tempin3 = block
			firstin = false
		}
		if pretempblock != nil && bytes.Equal(pretempblock.Hash, block.Hash) {
			Log.Info("遍历上次内容")
			return
		}
		fmt.Println("区块的ConfirmedLists里的账户有：", len(block.ConfirmedLists))
		//下面是对已经转换一次的进行处理
		hashpre := make(map[uuid.UUID][]byte)
		for uuid_, confirmedBlockList := range block.ConfirmedLists {
			//如果前面已经用到了 那么就退出
			_, ok := hased[uuid_]
			if ok {
				continue
			}
			//没有使用过  这里仅仅判断usedmap里面是否有这个id
			_, existss := usedmap[uuid_]
			if existss {
				Log.Debug("该id已经转移:", existss)
				continue
			}
			//使用了才能加这个
			//hased[uuid_] = 1
			blockArray := confirmedBlockList.BlockArray
			finalConfirmedBlock := blockArray[len(blockArray)-1]
			finalblockhash := finalConfirmedBlock.Hash
			hashpre[uuid_] = finalblockhash
		}
		hashget, _ := getValuesByKeys(hashpre)
		for uuid_, sideblockinfo := range hashget {
			hased[uuid_] = 1
			var tx *Transaction
			sideblock := Deserialize_tx(sideblockinfo)
			//获取交易的wallet
			txwallet, errget := getWalletFromTx(sideblock.TXOutputs[0].ScriptPubKeyHash)
			if errget != nil {
				Log.Fatal("没有获取到交易对应的wallet")
			} else {
				//fmt.Println("找到账户为", txwallet.Address)
			}
			txid := string(sideblock.TXID)
			Blockhash := hashpre[uuid_]
			//创建交易区块
			tx, err2 = CreateTransactionBlockFromUTXO(txid, 0, to, uuid_, Blockhash, &txwallet)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionFromUTXO出现错误 ：", err2)
				continue
			}
			//对一些信息进行设置
			SetTxDetail(tx, msg, to)
			//无用的东西
			SSourceID = tx.TXOutputs[0].SourceID
			Log.Debug("开始进行广播")
			AddToTransactionPool(tx)
			//printtx(tx)
			txs = append(txs, tx)
			myhaveid = append(myhaveid, uuid_)
		}

		//2.对没有用过的进行处理
		txwallet2, errget2 := getWalletFromTx(block.Product.Pubkeyhash)
		if errget2 != nil {
			Log.Fatal("没有获取到交易对应的wallet")
		} else {
			//fmt.Println("找到账户为", txwallet2.Address)
		}
		_, ok := hased[block.Product.SourceID]
		if ok {
		} else { //没有使用过   第一个参数是txid，这里类似创世区块，创世区块没有txid

			tx, err2 := CreateTransactionSpec(string(block.Product.StartTxid), 0, to, block.Product.SourceID, block.Hash, &txwallet2)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionSpec出现错误 ：", err2)
			}
			_, existss := usedmap[block.Product.SourceID]
			if existss {
				Log.Debug("该id已经转移:", existss)
			} else {
				//对交易设置一些关键信息
				SetTxDetail(tx, msg, to)
				AddToTransactionPool(tx)
				//printtx(tx)
				txs = append(txs, tx)
				hased[block.Product.SourceID] = 1
				myhaveid = append(myhaveid, block.Product.SourceID)
			}
		}

		//退出循环
		if block.PrevHash == nil {
			break
		}
		endTime := time.Now()
		duration1 := endTime.Sub(startTime)
		Log.Info("时间", duration1)
		Log.Info("添加元素", len(txs)-txslen)

	}
	Log.Info("共广播交易区块:", len(myhaveid))

	//PackTxArrTaskAndToChan(SendTX, txs)
}

//第三次修改
func transferTraceability3(to string, msg string, cli *CLI) {

	tempcount++
	if tempcount == 20 {
		tempin3 = nil
		tempcount = 0
	}

	//1.获取在线的账户
	tostr, _ := getRandomAddres()
	to = string(tostr)

	//2.获取区块链实例
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Info(ms)
		return
	}
	txs := make([]*Transaction, 0)
	var hased = make(map[uuid.UUID]int)
	var usedmap = make(map[uuid.UUID]int)
	var myhaveid []uuid.UUID
	//获得本账户
	myaddress := mywallet.Address
	Log.Warn(myaddress)

	//txpool里面存放是交易id，为了快速，这里将这个id放到里面（存在不一致性）
	TxPoolmutex.Lock()
	for _, value := range TxPool {
		usedmap[value.TXOutputs[0].SourceID] = 1
	}
	TxPoolmutex.Unlock()
	//3.遍历区块 找到
	it := bc.NewIterator()
	pretempblock := tempin3
	firstin := true
	for {
		startTime := time.Now()
		//遍历区块
		block := it.Next()
		if firstin {
			tempin3 = block
			firstin = false
		}
		if pretempblock != nil && bytes.Equal(pretempblock.Hash, block.Hash) {
			Log.Info("遍历上次内容")
			return
		}
		fmt.Println("区块的ConfirmedLists里的账户有：", len(block.ConfirmedLists))
		//下面是对已经转换一次的进行处理
		for uuid, confirmedBlockList := range block.ConfirmedLists {
			//如果前面已经用到了 那么就退出
			_, ok := hased[uuid]
			if ok {
				continue
			}
			//没有使用过  这里仅仅判断usedmap里面是否有这个id
			_, existss := usedmap[uuid]
			if existss {
				Log.Debug("该id已经转移:", existss)
				continue
			}

			//到这里说明没有用到
			hased[uuid] = 1
			//新建一个交易区块
			var tx *Transaction
			//反序列化被引用的区块
			blockArray := confirmedBlockList.BlockArray
			finalConfirmedBlock := blockArray[len(blockArray)-1]
			finalblockhash := finalConfirmedBlock.Hash
			sideblockinfo, err := getValueByKey(finalblockhash)
			//fmt.Printf("finalblockhash  %x\n", finalblockhash)
			if err != nil {
				Log.Warn("GetSideUTXOBySourceId出现错误，", err)
				continue
			}
			sideblock := Deserialize_tx(sideblockinfo)
			//获取交易的wallet
			txwallet, errget := getWalletFromTx(sideblock.TXOutputs[0].ScriptPubKeyHash)
			if errget != nil {
				Log.Fatal("没有获取到交易对应的wallet")
			} else {
				//fmt.Println("找到账户为", txwallet.Address)
			}
			txid := string(sideblock.TXID)
			Blockhash := finalConfirmedBlock.Hash
			//创建交易区块
			tx, err2 = CreateTransactionBlockFromUTXO(txid, 0, to, uuid, Blockhash, &txwallet)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionFromUTXO出现错误 ：", err2)
				continue
			}
			//对一些信息进行设置
			SetTxDetail(tx, msg, to)
			//无用的东西
			SSourceID = tx.TXOutputs[0].SourceID
			Log.Debug("开始进行广播")
			AddToTransactionPool(tx)
			//printtx(tx)
			txs = append(txs, tx)
			myhaveid = append(myhaveid, uuid)
		}

		//2.对没有用过的进行处理

		txwallet2, errget2 := getWalletFromTx(block.Product.Pubkeyhash)
		if errget2 != nil {
			Log.Fatal("没有获取到交易对应的wallet")
		} else {
			//fmt.Println("找到账户为", txwallet2.Address)
		}
		_, ok := hased[block.Product.SourceID]
		if ok {
		} else { //没有使用过   第一个参数是txid，这里类似创世区块，创世区块没有txid

			tx, err2 := CreateTransactionSpec(string(block.Product.StartTxid), 0, to, block.Product.SourceID, block.Hash, &txwallet2)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionSpec出现错误 ：", err2)
			}
			_, existss := usedmap[block.Product.SourceID]
			if existss {
				Log.Debug("该id已经转移:", existss)
			} else {
				//对交易设置一些关键信息
				SetTxDetail(tx, msg, to)
				AddToTransactionPool(tx)
				//printtx(tx)
				txs = append(txs, tx)
				hased[block.Product.SourceID] = 1
				myhaveid = append(myhaveid, block.Product.SourceID)
			}
		}

		//退出循环
		if block.PrevHash == nil {
			break
		}
		endTime := time.Now()
		duration1 := endTime.Sub(startTime)
		Log.Info("时间", duration1)
	}
	Log.Info("共广播交易区块:", len(myhaveid))

	return
	//PackTxArrTaskAndToChan(SendTX, txs)
}

//对交易区块设置一些关键信息 比如msg、哈希值
func SetTxDetail(tx *Transaction, msg string, to string) {
	hash := sha256.Sum256(tx.Serialize_tx())
	tx.TxHash = hash[:]

	//获取随机标题信息
	tiitle, err5 := randomLineFromFile()
	if err5 != nil {
		Log.Warn("获取随机标题失败")
	}
	timestamp := tx.TimeStamp
	t := time.Unix(int64(timestamp), 0)
	tomsg := "在" + t.String() + "时" + mywallet.Address + "向" + to + "转账"
	if msg == "" {
		msg = "没有传入转账信息，将使用默认值：" + tomsg + "附带消息" + tiitle
	}
	//交易区块的其他信息
	tx.TXmsg = []byte(msg)
}
func SetSpecTxDetail(tx *Transaction) {
	hash := sha256.Sum256(tx.Serialize_tx())
	tx.TxHash = hash[:]

	//获取随机标题信息
	tiitle, err5 := randomLineFromFile()
	if err5 != nil {
		Log.Warn("获取随机标题失败")
	}
	timestamp := tx.TimeStamp
	t := time.Unix(int64(timestamp), 0)
	msg := "在" + t.String() + "时创建。" + "初始溯源区块信息，将使用默认值：" + "附带消息" + tiitle

	//交易区块的其他信息
	tx.TXmsg = []byte(msg)
}

func transferTraceability(to string, msg string, cli *CLI) {
	if mywallet == (MyWallet{}) {
		Log.Warn("钱包获取失败，未进入区块链网络")
		return
	}
	//获取本账户有的soceid
	mp := getMyhaveId()
	Log.Warn("本账户共有数量=========", len(mp))

	//return
	if len(mp) == 0 {
		Log.Warn("本账户没有溯源产品")
		count11++
		if count11 == 100 {
			FetchBlocksRequest(nil)
			count11 = 0
			return
		} else if count11 > 500 { //不会执行
			cli.addBlock("自动生成的一个区块")
			count11 = 0
			return
		}

		return
	}

	/*if firstin {
		for _, sourceuuid := range mp {
			fmt.Println(sourceuuid)
		}
		return
	}*/

	//将本账户拥有的全部发完
	txs := make([]*Transaction, 0)
	for _, sourceuuid := range mp {
		firstin = true
		Log.Info("选取sourceuuid：", sourceuuid)
		//if to == "" {
		tostr, err := getOnlineAddres()
		to = string(tostr)
		if err != nil {
			Log.Fatal("获取转账人出错")
		}

		Log.Debug("==============将向", to, "转账==================\n")
		//}

		Log.Info("开始生成交易区块过程")
		txid, preBlockhash, isStart, premainhash, returnerr := GetSideUTXOBySourceId(sourceuuid)
		if returnerr != nil {
			Log.Warn("transferTraceability的GetSideUTXOBySourceId出现错误 ：", returnerr)
			//FetchBlocksRequest(nil)
			continue
		}

		var tx *Transaction
		var err2 error
		//判断是否是起点，是起点和不是起点创建的交易区块不一样
		if isStart { //还没有用过  是起点
			Log.Debug("执行没有用过交易的逻辑")

			tx, err2 = CreateTransactionSpec(txid, 0, to, sourceuuid, premainhash, nil)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionSpec出现错误 ：", err2)
			}
		} else { //不是起点
			//fmt.Printf("txid:%x", txid)
			//根据得到的信息 创建一个交易区块  第二个参数是引用的交易的0下标
			tx, err2 = CreateTransactionBlockFromUTXO(txid, 0, to, sourceuuid, preBlockhash, nil)
			if err2 != nil {
				Log.Warn("transferTraceability的CreateTransactionFromUTXO出现错误 ：", err2)
				return
			}
		}
		hash := sha256.Sum256(tx.Serialize_tx())
		tx.TxHash = hash[:]
		//获取随机标题信息
		/*tiitle, err5 := randomLineFromFile()
		if err5 != nil {
			Log.Warn("获取随机标题失败")
		}*/
		tiitle := "暂时关闭附加信息"
		timestamp := tx.TimeStamp
		t := time.Unix(int64(timestamp), 0)
		tomsg := "在" + t.String() + "时" + mywallet.Address + "向" + to + "转账"
		if msg == "" {
			msg = "没有传入转账信息，将使用默认值：" + tomsg + "附带消息" + tiitle
		}
		//交易区块的其他信息
		tx.TXmsg = []byte(msg)
		SSourceID = tx.TXOutputs[0].SourceID
		Log.Debug("开始加入交易池")
		//加入交易池  这个应该没有用
		/*
			TxPoolmutex.Lock()
			TxPool[string(tx.TXID)] = tx
			TxPoolmutex.Unlock()*/
		//进行广播
		Log.Debug("开始进行广播")

		//printtx(tx)
		txs = append(txs, tx)
		//检查是否有这个区块的所有权
		exist := CheckhasSourceID(SSourceID, tx.TXInputs[0].PubKey)
		if exist {
			AddToTransactionPool(tx)
		} else {
			FetchBlocksRequest(nil)
			Log.Warn("交易验证没有通过，可能是sourceID所属权限不归属此人或者本节点存在区块链更新延迟")
		}

	}
	PackTxArrTaskAndToChan(SendTX, txs)
}

/*
  GetSideUTXOBySourceId 获得侧链的utxo
  根据传入的targetSourceId  从主链出发，找到最后一个包含这个交易的主链区块，然后逆向查找判断是否是这个人的

*/

////////////////获得侧链的utxo使用的这个函数
//如果isStart那说明引用的主链上交易，startblock就是主链上的这个区块，否则返回交易id，引用的交易区块哈希
func GetSideUTXOBySourceId(targetSourceId uuid.UUID) (txid string, Blockhash []byte, isStart bool, premainhash []byte, returnerr error) {
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Info(ms)
		returnerr = errors.New(ms)
		return
	}
	//Address := mywallet.Address
	pubKeyHash := getPubKeyHashFromAddress(mywallet.Address)
	/*fmt.Printf("待转账人的Address : %s\n", Address)
	fmt.Printf("待转账人的AddressHash : %x\n", getPubKeyHashFromAddress(Address))
	fmt.Printf("待转账人的pubKeyHash : %x\n", pubKeyHash)*/
	//遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()
		for uuid, confirmedBlockList := range block.ConfirmedLists {
			if uuid != targetSourceId {
				continue
			} else {
				blockArray := confirmedBlockList.BlockArray
				finalConfirmedBlock := blockArray[len(blockArray)-1]
				finalblockhash := finalConfirmedBlock.Hash
				sideblockinfo, err := getValueByKey(finalblockhash)
				if err != nil {
					Log.Warn("GetSideUTXOBySourceId出现错误，", err)
				}
				sideblock := Deserialize_tx(sideblockinfo)
				//	fmt.Printf("sideblock.TXOutputs[0].ScriptPubKeyHash %x\n", sideblock.TXOutputs[0].ScriptPubKeyHash)
				//	fmt.Printf("pubKeyHash %x\n", pubKeyHash)
				if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, pubKeyHash) {
					txid = string(sideblock.TXID)
					Blockhash = finalblockhash
					return
				} else {
					Log.Debug("找到该溯源终点 该id不属于此账户")
					returnerr = fmt.Errorf("找到该溯源终点 该id不属于此账户")
					return
				}
			}
		}
		//fmt.Printf("block.Product.Pubkeyhash:  %x\n", block.Product.Pubkeyhash)
		//fmt.Printf("pubKeyHash:%x\n", pubKeyHash)
		if block.Product.SourceID == targetSourceId && bytes.Equal(block.Product.Pubkeyhash, pubKeyHash) {
			premainhash = block.Hash
			txid = string(block.Hash)
			isStart = true
			Log.Debug("验证成功")
			return
		} else if block.Product.SourceID == targetSourceId {
			//因为已经到最端点了 还不相等 直接退出
			//fmt.Printf("找到该溯源起点 该id不属于此账户,该pubkeyhash为%x,本账户为%x\n", block.Product.Pubkeyhash, pubKeyHash)
			returnerr = fmt.Errorf("该id不属于此账户")
			return
		}

		//退出循环
		if block.PrevHash == nil {
			break
		}

	}
	Log.Debug("没有找到该溯源id")
	returnerr = fmt.Errorf("该id不属于此账户")
	return
}

//测试程序 可以删除
func Tesett(buffff []byte) {

	header := make([]byte, 4)
	header = buffff[0:4]
	buffff = buffff[4:]
	length := binary.BigEndian.Uint32(header)
	Log.Info("消息的长度为：", length)
	// 读取整个响应消息
	msg := make([]byte, length-4)
	msg = buffff
	Log.Info("完成数据的读取")
	// 构造 SendMessage 对象，并将其发送到消息通道中
	cmd := Command(int(binary.BigEndian.Uint32(msg[:4])))
	fmt.Println("收到的为：", msg)
	Log.Info("cmd:", Command(cmd))
	payload := msg[4:]
	sendMessage := &SendMessage{
		Len:     int32(length),
		Cmd:     cmd,
		Payload: payload,
	}
	ReceiveTxArr(sendMessage)

}

//判断我的账户是否有这个sourceId
//返回交易的id和在交易out中的索引
/////////////修改成返回交易区块的交易哈希
func GetUTXOBySourceId(targetSourceId uuid.UUID) (txid string, index int, err error) {
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Warn(ms)
		err = errors.New(ms)
		return
	}
	pubKeyHash := getPubKeyHashFromPubKey(mywallet.walle.PubKey)
	spentUTXO, havetheid := bc.findNeedUTXO(pubKeyHash, targetSourceId)
	// map[0x222] = []int{0}
	// map[0x333] = []int{0,1}

	if !havetheid {
		fmt.Println("当前SourceID终点不为当前节点所有！")
		err = errors.New("当前SourceID终点不为当前节点所有！")
		return
	}
	if len(spentUTXO) > 0 {
		Log.Info("通过验证，可以转账")
	}
	for key, _ := range spentUTXO {
		txid = string(key)
		index = 0 //溯源链的交易 outUTXO一定是一个 也就是转出的人一定是一个人
		return
	}
	return
}

/////////////根据某人公钥在公链上是否拥有这个targetSourceId
func hasAmount(targetSourceId uuid.UUID, pubKeyHash []byte) (exit bool, returnerr error) {
	Log.Debug("验证转账人在链上是否此SourceId")
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Warn(ms)
		returnerr = errors.New(ms)
		return
	}
	//遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()

		Log.Debug("遍历区块的已完成确认区块")
		for uuid, confirmedBlockList := range block.ConfirmedLists {
			if uuid != targetSourceId {
				continue
			} else {
				Log.Debug("发现终点，验证是否属于此人")
				blockArray := confirmedBlockList.BlockArray
				finalConfirmedBlock := blockArray[len(blockArray)-1]
				finalblockhash := finalConfirmedBlock.Hash
				sideblockinfo, err := getValueByKey(finalblockhash)
				if err != nil {
					Log.Warn("GetSideUTXOBySourceId出现错误，", err)
					returnerr = fmt.Errorf("GetSideUTXOBySourceId出现错误")
				}
				sideblock := Deserialize_tx(sideblockinfo)
				if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, pubKeyHash) {
					Log.Debug("验证通过，终点属于此人")
					exit = true
					return
				} else {
					Log.Debug("验证未通过，该id不属于此账户")
					returnerr = fmt.Errorf("该id不属于此账户")
					return
				}
			}
		}
		//fmt.Printf("%x\n", block.Product.Pubkeyhash)
		//fmt.Printf("%x\n", pubKeyHash)
		Log.Debug("验证此交易来源是否是在区块上")
		if block.Product.SourceID == targetSourceId && bytes.Equal(block.Product.Pubkeyhash, pubKeyHash) {
			Log.Debug("在区块上，交易验证通过")
			exit = true
			return
		} else if block.Product.SourceID == targetSourceId {
			fmt.Printf("用户Pubkeyhash：%x\n", block.Product.Pubkeyhash)
			fmt.Printf("交易归属于Pubkeyhash：%x\n")
			fmt.Println("在区块上，但是交易验证没有通过")

			//因为已经到最端点了 还不相等 直接退出
			exit = false
			returnerr = fmt.Errorf("该id不属于此账户")
			return
		}
		if block.PrevHash == nil {
			Log.Debug("到达创世区块，没有通过验证")
			Log.Info("该id不属于此账户")
			break
		}

	}
	Log.Info("该id不属于此账户")
	exit = false
	returnerr = fmt.Errorf("该id不属于此账户")
	return
}

/////////////根据某人公钥在公链上是否拥有这个targetSourceId
func hasAmount2Memory(targetSourceId uuid.UUID, pubKeyHash []byte) (exit bool, returnerr error) {
	Log.Debug("验证转账人在链上是否此SourceId")
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Warn(ms)
		returnerr = errors.New(ms)
		return
	}
	//遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.MemoryNextBlock()

		Log.Debug("遍历区块的已完成确认区块")
		for uuid, confirmedBlockList := range block.ConfirmedLists {
			if uuid != targetSourceId {
				continue
			} else {
				Log.Debug("发现终点，验证是否属于此人")
				blockArray := confirmedBlockList.BlockArray
				finalConfirmedBlock := blockArray[len(blockArray)-1]
				finalblockhash := finalConfirmedBlock.Hash
				sideblock := GetTxFromMemory(finalblockhash)
				if bytes.Equal(sideblock.TXOutputs[0].ScriptPubKeyHash, pubKeyHash) {
					Log.Debug("验证通过，终点属于此人")
					exit = true
					return
				} else {
					Log.Debug("验证未通过，该id不属于此账户")
					returnerr = fmt.Errorf("该id不属于此账户")
					return
				}
			}
		}
		//fmt.Printf("%x\n", block.Product.Pubkeyhash)
		//fmt.Printf("%x\n", pubKeyHash)
		Log.Debug("验证此交易来源是否是在区块上")
		if block.Product.SourceID == targetSourceId && bytes.Equal(block.Product.Pubkeyhash, pubKeyHash) {
			Log.Debug("在区块上，交易验证通过")
			exit = true
			return
		} else if block.Product.SourceID == targetSourceId {
			fmt.Printf("用户Pubkeyhash：%x\n", block.Product.Pubkeyhash)
			fmt.Printf("交易归属于Pubkeyhash：%x\n")
			fmt.Println("在区块上，但是交易验证没有通过")

			//因为已经到最端点了 还不相等 直接退出
			exit = false
			returnerr = fmt.Errorf("该id不属于此账户")
			return
		}
		if block.PrevHash == nil {
			Log.Debug("到达创世区块，没有通过验证")
			Log.Info("该id不属于此账户")
			break
		}

	}
	Log.Info("该id不属于此账户")
	exit = false
	returnerr = fmt.Errorf("该id不属于此账户")
	return
}

////////////////////////创建一个交易区块  正常模式下  本区块引用的是前面侧链的区块
//mywallet
func CreateTransactionBlockFromUTXO(txid string, index int, to string, sourceID uuid.UUID, prepoint []byte, txwallet *MyWallet) (*Transaction, error) {

	var inputs []TXInput
	var outputs []TXOutput
	if txwallet == nil {
		txwallet = &mywallet
	}
	// 4. 拼接inputs
	// > 遍历utxo集合，每一个output都要转换为一个input(3)

	input := TXInput{Txid: []byte(txid), Index: int64(index), ScriptSig: nil, PubKey: txwallet.walle.PubKey}
	inputs = append(inputs, input)

	// 5. 拼接outputs
	// > 创建一个属于to的output
	//创建给收款人的output
	output1 := newTXOutput(to, sourceID)
	outputs = append(outputs, output1)

	timeStamp := time.Now().Unix()

	// 6. 设置哈希，返回
	//tx := Transaction{nil, inputs, outputs, uint64(timeStamp)}

	tx := Transaction{
		TXID:        nil,
		TXInputs:    inputs,
		TXOutputs:   outputs,
		TimeStamp:   uint64(timeStamp), // 创建交易的时间戳
		PoI:         []byte(prepoint),  // 本溯源线上一个节点
		CP:          nil,               // 当前主链的最后一个区块
		TBP:         nil,               // 指向还没有TBP指向的区块
		SourceStrat: false,
	}

	tx.setHash()
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	//打开bc失败
	if err != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Warn(ms)
		return &tx, errors.New(ms)
	}
	//签名
	if !bc.signTransaction(&tx, txwallet.walle.PriKey) {
		fmt.Println("交易签名失败")
		return &tx, errors.New("交易签名失败")
	}
	return &tx, nil
}

////////////引用没有验证的区块，这里仅验证侧链上的区块

////////////////////////创建一个特殊交易区块  引用的是主链上的区块
func CreateTransactionSpec(txid string, index int, to string, sourceID uuid.UUID, premainhash []byte, txwallet *MyWallet) (*Transaction, error) {

	var inputs []TXInput
	var outputs []TXOutput
	if txwallet == nil {
		txwallet = &mywallet
	}
	// 4. 拼接inputs
	// > 遍历utxo集合，每一个output都要转换为一个input(3)

	input := TXInput{Txid: []byte(txid), Index: int64(index), ScriptSig: nil, PubKey: txwallet.walle.PubKey}
	inputs = append(inputs, input)

	// 5. 拼接outputs
	// > 创建一个属于to的output
	//创建给收款人的output
	output1 := newTXOutput(to, sourceID)
	outputs = append(outputs, output1)

	timeStamp := time.Now().Unix()

	// 6. 设置哈希，返回
	//tx := Transaction{nil, inputs, outputs, uint64(timeStamp)}

	tx := Transaction{
		TXID:        nil, //在下面的setHash()会设置txid
		TXInputs:    inputs,
		TXOutputs:   outputs,
		TimeStamp:   uint64(timeStamp), // 创建交易的时间戳
		PoI:         premainhash,       // 本溯源线上一个节点
		CP:          nil,               // 当前主链的最后一个区块
		TBP:         nil,               // 指向还没有TBP指向的区块
		SourceStrat: true,
	}

	tx.setHash()
	Log.Debug("打开区块链实例")
	//签名
	//获取主链区块的pubscrpthash
	MainBlock := GetBlockFromMemory(tx.PoI)
	/*pretxBlockinfo, err := getValueByKey(tx.PoI)
	if err != nil {
		Log.Warn("signTransaction的getValueByKey出错", err)
	}
	MainBlock := Deserialize(pretxBlockinfo)*/

	preScriptPubKeyHash := MainBlock.Transactions[0].TXOutputs[0].ScriptPubKeyHash
	tx.signByPreScript(txwallet.walle.PriKey, preScriptPubKeyHash)
	return &tx, nil
}

//从本人的utxo中 将指定soceid转移给别人 生成一笔交易  没有修改前面的指针
func CreateTransactionFromUTXO(txid string, index int, to string, sourceID uuid.UUID) (*Transaction, error) {

	var inputs []TXInput
	var outputs []TXOutput

	// 4. 拼接inputs
	// > 遍历utxo集合，每一个output都要转换为一个input(3)

	input := TXInput{Txid: []byte(txid), Index: int64(index), ScriptSig: nil, PubKey: mywallet.walle.PubKey}
	inputs = append(inputs, input)

	// 5. 拼接outputs
	// > 创建一个属于to的output
	//创建给收款人的output
	output1 := newTXOutput(to, sourceID)
	outputs = append(outputs, output1)

	timeStamp := time.Now().Unix()

	// 6. 设置哈希，返回
	tx := Transaction{
		TXID:      nil,
		TXInputs:  inputs,
		TXOutputs: outputs,
		TimeStamp: uint64(timeStamp), // 创建交易的时间戳
		PoI:       nil,               // 本溯源线上一个节点
		CP:        nil,               // 当前主链的最后一个区块
		TBP:       nil,               // 指向还没有TBP指向的区块
	}

	tx.setHash()
	Log.Info("setHash")
	bc, err := GetBlockChainInstance()
	defer bc.closeDB()
	if err != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Warn(ms)
		return &tx, errors.New(ms)
	}
	if !bc.signTransaction(&tx, mywallet.walle.PriKey) {
		fmt.Println("交易签名失败")
		return &tx, errors.New("交易签名失败")
	}
	return &tx, nil
}
func DataTraceability(targetSourceId string) [][]byte {
	Zookinit()
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
	var result [][]byte //最终结果
	for {
		//遍历区块
		block := it.Next()
		var res [][]byte
		for _, tx := range block.Transactions {
			if tx.TXOutputs[0].SourceID != targetid {
				continue
			}
			//这里直接加入 并没有做判断
			res = append(res, tx.TXOutputs[0].ScriptPubKeyHash)
		}
		for i := len(res) - 1; i >= 0; i-- {
			result = append(result, res[i])
		}
		//退出条件
		if len(block.PrevHash) == 0 {
			break
		}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	for _, row := range result {
		fmt.Printf("%x\n", row)
	}

	//PrintDataTraceability(result)
	return result
}

// func SideBlockNext(nowhash []byte){
// 		sideblockinfo, err := getValueByKey(nowhash)
// 		block = Deserialize(sideblockinfo)
// 		currentHash = block.PrevHash //游标左移
// }
//溯源所有的线
func DataTraceall() {

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
			aa := usedTxto[uuid.String()]
			if aa == 1 {
				continue
			}

			usedTxto[uuid.String()] = 1
			fmt.Println("更换uuid---------------------------------------")
			fmt.Println("sourceid：", uuid.String())
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
					blockhash := sideblock.PoI
					mainblockinfo, _ := getValueByKey(blockhash)
					mainblock := Deserialize(mainblockinfo)
					printtraceBlock(*mainblock)
					break
				}
			}
		}
		exist := usedTxto[block.Product.SourceID.String()]
		if exist != 1 {
			fmt.Println("sourceid：", block.Product.SourceID.String())
			fmt.Println("	该地址还没有转移")
			printtraceBlock(*block)
		}
		if block.PrevHash == nil {
			Log.Debug("到达创世区块，退出查找")
			break
		}
	}

}

//找到socurceid的最后一个区块 先判断noexit，如果是yes的话就是没有
// 如果是在main区块上那么isBlock就是true，Mainblock就有值
//如果是false，那么Sideblock就有值
func getSourceidBlock(UUid uuid.UUID) (Mainblock Block, Sideblock Transaction, isBlock bool, noexit bool) {
	defer fmt.Println("跳出getSourceidBlock函数比较过程")
	fmt.Println("进入getSourceidBlock函数比较过程")
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "GetSideUTXOBySourceId中GetBlockChainInstance失败"
		Log.Warn(ms)
		return
	}
	var usedTxto = make(map[string]int)
	//遍历区块 找到
	it := bc.NewIterator()
	for {
		//遍历区块
		block := it.Next()

		if block == nil { //这个地方可能有问题
			noexit = true
			return
		}

		for uuid, confirmedBlockList := range block.ConfirmedLists {
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
			Sideblock = *sideblock
			return
		}
		if block.Product.SourceID == UUid {
			Log.Debug("在主链上找到此uuid")
			Mainblock = *block
			isBlock = true
			return
		}
		if block.PrevHash == nil {
			noexit = true
			return
		}
	}

}

//打印数据溯源数据
func PrintDataTraceability(res [][]byte) {
	nodePath := "/Wallet-Name"
	address_hashpubkey := "/Wallet/address-hashpubkey"
	mp, err := getChildren(Zkconn, address_hashpubkey)
	if err != nil {
		Log.Warn("PrintDataTraceability的getChildren出错")
		return
	}
	for _, val := range res {
		flag := 0
		for address, value := range mp {
			if string(val) == string(value) {
				path := nodePath + "/" + address
				data, _, err := Zkconn.Get(path)
				if err != nil {
					Log.Warn("PrintDataTraceability 的Get出错", err)
					break
				}
				fmt.Println(data)
				flag = 1
				break
			}
		}
		if flag == 0 {
			fmt.Println(string(val))
		}
	}
}

//这个是验证其他人是否有那个
func CheckhasSourceID(targetSourceId uuid.UUID, pubkey []byte) bool {
	Log.Info("需要验证的消息为:")
	//fmt.Printf("targetSourceId:%x\n", targetSourceId)
	//fmt.Printf("pubkey:%x", pubkey)
	bc, err2 := GetBlockChainInstance()
	defer bc.closeDB()
	if err2 != nil {
		ms := "checkSourceIdInAccount中GetBlockChainInstance失败"
		Log.Warn(ms)
		return false
	}

	pubKeyHash := getPubKeyHashFromPubKey(pubkey)
	spentUTXO, havetheid := bc.findNeedUTXO(pubKeyHash, targetSourceId)
	// map[0x222] = []int{0}
	// map[0x333] = []int{0,1}

	if !havetheid {
		fmt.Println("该SourceID没有找到")
		return false
	}
	if len(spentUTXO) > 0 {
		Log.Info("通过验证")
		return true
	}
	return false
}

//简易打印，只打印是谁转给谁
func printtxfromto(transaction *Transaction) {

	toaddress, err111 := pubkeyhashToAddress(transaction.TXOutputs[0].ScriptPubKeyHash)
	if err111 != nil {
		fmt.Println("本地没有此公钥哈希对应的地址")
	}
	frompubkey := transaction.TXInputs[0].PubKey
	fromaddress, _ := pubkeyhashToAddress(getPubKeyHashFromPubKey(frompubkey))
	Log.Info("从", fromaddress, "转到", toaddress)
	//fmt.Printf("指向的前一块区块: %x\n", transaction.PoI)
}

func printtx(transaction *Transaction) {
	fmt.Println("Transaction:")
	fmt.Printf("	transaction的hash: %x\n", transaction.TxHash)
	fmt.Printf("	PoI: %x\n", transaction.PoI)
	fmt.Printf("	TXID: %x\n", transaction.TXID)
	// fmt.Println("TimeStamp:", transaction.TimeStamp)
	fmt.Println("	SourceStrat:", transaction.SourceStrat)
	fmt.Printf("	TXmsg: %s\n", string(transaction.TXmsg))

	fmt.Println("\nTXInputs:")
	for _, input := range transaction.TXInputs {
		// fmt.Println("	Input", i+1)
		fmt.Printf("	使用的Txid: %x\n", input.Txid)
		// fmt.Println("	Index:", input.Index)
		// fmt.Printf("	ScriptSig: %x\n", input.ScriptSig)
		// fmt.Printf("	PubKey: %x\n", input.PubKey)
	}

	fmt.Println("\nTXOutputs:")
	for _, output := range transaction.TXOutputs {
		// fmt.Println("	Output", i+1)
		// fmt.Printf("	ScriptPubKeyHash: %x\n", output.ScriptPubKeyHash)
		fmt.Println("	SourceID:", output.SourceID)
	}

	toaddress, err111 := pubkeyhashToAddress(transaction.TXOutputs[0].ScriptPubKeyHash)
	if err111 != nil {
		fmt.Println("本地没有此公钥哈希对应的地址")
	}
	//fmt.Println("======================", toaddress, "  ========================")

	frompubkey := transaction.TXInputs[0].PubKey
	fromaddress, _ := pubkeyhashToAddress(getPubKeyHashFromPubKey(frompubkey))
	fmt.Printf("从%s转到%s\n", fromaddress, toaddress)

	// pubkeyhash := transaction.TXInputs[0].PubKey
	// fmt.Printf("pubkeyhash%x\n", pubkeyhash)
	// address, err := pubkeyhashToAddress(pubkeyhash)
	// fmt.Printf("%x", address)
	// if err != nil {

	// 	PubKeyHash := transaction.TXOutputs[0].ScriptPubKeyHash
	// 	address2, err2 := pubkeyhashToAddress(PubKeyHash)
	// 	if err2 != nil {
	// 		fmt.Printf("从%s转到%s\n", address, address2)
	// 	}
	// }
	fmt.Printf("---------------------------------------------------\n")
}

func printtrace(transaction *Transaction) {

	toaddress, err111 := pubkeyhashToAddress(transaction.TXOutputs[0].ScriptPubKeyHash)
	if err111 != nil {
		fmt.Println("     本地没有此公钥哈希对应的地址")
	}
	//fmt.Println("======================", toaddress, "  ========================")

	frompubkey := transaction.TXInputs[0].PubKey
	fromaddress, _ := pubkeyhashToAddress(getPubKeyHashFromPubKey(frompubkey))
	timeObj := time.Unix(int64(transaction.TimeStamp), 0)
	formattedTime := timeObj.Format("2006-01-02 15:04:05")
	fmt.Printf("		本hash%x\n", transaction.TxHash)
	fmt.Println("		是否是开始节点：", transaction.SourceStrat)
	fmt.Printf("		指向的前指针%x\n", transaction.PoI)
	fmt.Printf("		交易确认时间%s\n", formattedTime)
	fmt.Printf("		从%s\n		  转到%s\n", fromaddress, toaddress)
	fmt.Printf("		交易信息为: %s\n", string(transaction.TXmsg))

	// pubkeyhash := transaction.TXInputs[0].PubKey
	// fmt.Printf("pubkeyhash%x\n", pubkeyhash)
	// address, err := pubkeyhashToAddress(pubkeyhash)
	// fmt.Printf("%x", address)
	// if err != nil {

	// 	PubKeyHash := transaction.TXOutputs[0].ScriptPubKeyHash
	// 	address2, err2 := pubkeyhashToAddress(PubKeyHash)
	// 	if err2 != nil {
	// 		fmt.Printf("从%s转到%s\n", address, address2)
	// 	}
	// }
	fmt.Printf("		---------------------------------------------------\n")
}

func saveSideBlockToBucket(tx *Transaction) {
	hash := tx.Serialize_tx()
	Log.Debug("saveSideBlockToBucket序列化完成")
	//这么没有求sum256，是为了一致性，怕有问题
	createOrUpdateKeyValue(tx.TxHash, hash)
}
