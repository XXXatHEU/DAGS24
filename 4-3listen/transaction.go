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

var SSourceID uuid.UUID //测试用
var PPubKeyHash []byte  //测试用
var PPubey []byte       //测试用

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

//创建普通交易
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
	tx := Transaction{nil, inputs, outputs, uint64(timeStamp)}
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
func (tx *Transaction) sign(priKey *ecdsa.PrivateKey, prevTxs map[string]*Transaction) bool {
	fmt.Println("具体对交易签名sign...")

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

	txCopy := Transaction{tx.TXID, inputs, outputs, tx.TimeStamp}
	return &txCopy
}

//具体校验
func (tx *Transaction) verify(prevTxs map[string]*Transaction) bool {
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

	lines = append(lines, fmt.Sprintf("\n--- Transaction %x:", tx.TXID))

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
		fmt.Printf("Deserialize_tx decode err:", err)
		return nil
	}

	return &tx
}

func transferTraceability(sourceId string, to string) {
	if mywallet == (MyWallet{}) {
		Log.Warn("钱包获取失败，未进入区块链网络")
		return
	}
	sourceId = "958efbde-8e35-4b5e-8df2-14e4f5b8c11a"
	to = "1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7"
	sourceuuid, err := uuid.Parse(sourceId)
	if err != nil {
		fmt.Println("无效的 UUID 字符串")
		return
	}
	txid, index, err := GetUTXOBySourceId(sourceuuid)
	if err != nil {
		Log.Warn("transferTraceability的GetUTXOBySourceId出现错误 ：", err)
		return
	}
	fmt.Printf("txid:%x", txid)
	tx, err2 := CreateTransactionFromUTXO(txid, index, to, sourceuuid)
	if err2 != nil {
		Log.Warn("transferTraceability的CreateTransactionFromUTXO出现错误 ：", err2)
		return
	}
	Log.Info("tx.TXID:\n", tx.TXID)
	fmt.Printf("tx.TXID: %x\n", tx.TXID)
	fmt.Printf("tx.TimeStamp: %x\n", tx.TimeStamp)
	fmt.Printf("tx.TXInputs[0].PubKey %x\n", tx.TXInputs[0].PubKey)
	fmt.Printf("tx.TXInputs[0].ScriptSig %x\n", tx.TXInputs[0].ScriptSig)
	fmt.Printf("tx.TXInputs[0].Index %x\n", tx.TXInputs[0].Index)
	fmt.Printf("tx.TXInputs[0].Txid %x\n", tx.TXInputs[0].Txid)
	fmt.Printf("tx.TXInputs[0].ScriptPubKeyHash %x\n", tx.TXOutputs[0].ScriptPubKeyHash)
	fmt.Printf("tx.TXInputs[0].SourceID %x\n", tx.TXOutputs[0].SourceID)
	SSourceID = tx.TXOutputs[0].SourceID
	PPubey = tx.TXInputs[0].PubKey
	PPubKeyHash = getPubKeyHashFromPubKey(tx.TXInputs[0].PubKey)
	fmt.Println()
	//加入交易池
	TxPoolmutex.Lock()
	TxPool[string(tx.TXID)] = tx
	TxPoolmutex.Unlock()
	//进行广播
	txs := make([]*Transaction, 0)
	txs = append(txs, tx)
	//测试用
	exist := CheckhasSourceID(SSourceID, tx.TXInputs[0].PubKey)
	if exist {
		AddToTransactionPool(tx)
	} else {
		Log.Warn("交易验证没有通过，可能是sourceID所属权限不归属此人或者本节点存在区块链更新延迟")
	}
	PackTxArrTaskAndToChan(SendTX, txs)
	// dddd := "958efbde-8e35-4b5e-8df2-14e4f5b8c11a"
	// uuidValue, err := uuid.Parse(dddd)
	// fmt.Printf("公钥 %x", tx.TXInputs[0].PubKey)

	// cd := CheckhasSourceID(uuidValue, []byte(tx.TXInputs[0].PubKey))
	// if cd {
	// 	Log.Info("成功")
	// }
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
func GetUTXOBySourceId(targetSourceId uuid.UUID) (txid string, index int, err error) {
	bc, err2 := GetBlockChainInstance()
	defer bc.db.Close()
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

//从本人的utxo中 将指定soceid转移给别人 生成一笔交易
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
	tx := Transaction{nil, inputs, outputs, uint64(timeStamp)}

	tx.setHash()
	Log.Info("setHash")
	bc, err := GetBlockChainInstance()
	defer bc.db.Close()
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
	defer bc.db.Close()
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
	PrintDataTraceability(result)
	return result
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
	fmt.Printf("targetSourceId:%x\n", targetSourceId)
	fmt.Printf("pubkey:%x", pubkey)
	bc, err2 := GetBlockChainInstance()
	defer bc.db.Close()
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
