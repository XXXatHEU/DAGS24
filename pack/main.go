package main

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/google/uuid"
)

type Block struct {
	//版本号
	Version uint64

	// 前区块哈希
	PrevHash []byte

	//交易的根哈希值
	MerkleRoot []byte

	//时间戳
	TimeStamp uint64

	//难度值, 系统提供一个数据，用于计算出一个哈希值
	Bits uint64

	//随机数，挖矿要求的数值
	Nonce uint64

	// 哈希, 为了方便，我们将当前区块的哈希放入Block中
	Hash []byte

	//数据
	// Data []byte
	//一个区块可以有很多交易(交易的集合)
	Transactions []*Transaction
}

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

func init() {
	gob.Register(&Transaction{})
	gob.Register(&TXInput{})
	gob.Register(&TXOutput{})
}

// 将Block数组转换为byte数组
func blockArrayToByte(blockArr []*Block) []byte {
	var buffer bytes.Buffer

	// 注册需要编码的类型
	gob.Register(&Block{})
	gob.Register(&Transaction{})

	// 创建编码器
	encoder := gob.NewEncoder(&buffer)

	// 遍历Block数组，对每个Block进行编码
	for _, block := range blockArr {
		err := encoder.Encode(block)
		if err != nil {
			panic(err)
		}
	}

	return buffer.Bytes()
}

// 将byte数组恢复为Block数组
func byteToBlockArray(byteArr []byte) []*Block {
	var result []*Block

	// 注册需要解码的类型
	gob.Register(&Block{})
	gob.Register(&Transaction{})

	// 创建解码器
	buffer := bytes.NewBuffer(byteArr)
	decoder := gob.NewDecoder(buffer)

	// 循环解码，直到无法解码为止
	for {
		var block Block
		err := decoder.Decode(&block)
		if err != nil {
			break
		}
		result = append(result, &block)
	}

	return result
}

// 将Transaction数组序列化为字节数组
func transactionArrToByte(txs []*Transaction) []byte {
	var buffer bytes.Buffer

	// 创建编码器
	encoder := gob.NewEncoder(&buffer)

	// 编码Transaction数组
	err := encoder.Encode(txs)
	if err != nil {
		panic(err)
	}

	return buffer.Bytes()
}

// 将字节数组反序列化为Transaction数组
func byteToTransactionArr(byteArr []byte) []*Transaction {
	var txs []*Transaction

	// 创建解码器
	buffer := bytes.NewBuffer(byteArr)
	decoder := gob.NewDecoder(buffer)

	// 解码Transaction数组
	err := decoder.Decode(&txs)
	if err != nil {
		panic(err)
	}

	return txs
}

/*
func main() {
	// 创建两笔交易
	tx1 := &Transaction{
		TXID: []byte("tx1"),
		TXInputs: []TXInput{
			{
				Txid:      []byte("prev_tx1"),
				Index:     0,
				ScriptSig: []byte("sig1"),
			},
			{
				Txid:      []byte("prev_tx2"),
				Index:     1,
				ScriptSig: []byte("sig2"),
			},
		},
		TXOutputs: []TXOutput{
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh1"),
			},
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh2"),
			},
		},
	}

	tx2 := &Transaction{
		TXID: []byte("tx11"),
		TXInputs: []TXInput{
			{
				Txid:      []byte("prev_tx1"),
				Index:     0,
				ScriptSig: []byte("sig1"),
			},
			{
				Txid:      []byte("prev_tx2"),
				Index:     1,
				ScriptSig: []byte("sig2"),
			},
		},
		TXOutputs: []TXOutput{
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh1"),
			},
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh2"),
			},
		},
	}

	// 创建两个区块
	block1 := &Block{
		Version:      1,
		PrevHash:     []byte("0"),
		MerkleRoot:   []byte("root1"),
		TimeStamp:    111111,
		Bits:         256,
		Nonce:        123456,
		Hash:         []byte("block1"),
		Transactions: []*Transaction{tx1},
	}

	block2 := &Block{
		Version:      2,
		PrevHash:     []byte("block1"),
		MerkleRoot:   []byte("root2"),
		TimeStamp:    222222,
		Bits:         256,
		Nonce:        654321,
		Hash:         []byte("block2"),
		Transactions: []*Transaction{tx2},
	}
	aa := []*Block{block1, block2}
	// 将Block数组序列化为字节数组
	byteArr := blockArrayToByte(aa)

	// 输出字节数组
	//fmt.Printf("Serialized block array: %x\n", byteArr)

	// 将字节数组恢复为Block数组
	result := byteToBlockArray(byteArr)
	// 输出恢复后的Block数组
	fmt.Println("Deserialized block array: ", string(result[0].Transactions[0].TXID))
}
*/

func main() {
	tx1 := &Transaction{
		TXID: []byte("tx1"),
		TXInputs: []TXInput{
			{
				Txid:      []byte("prev_tx1"),
				Index:     0,
				ScriptSig: []byte("sig1"),
				PubKey:    []byte("key1"),
			},
			{
				Txid:      []byte("prev_tx2"),
				Index:     1,
				ScriptSig: []byte("sig2"),
				PubKey:    []byte("key2"),
			},
		},
		TXOutputs: []TXOutput{
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh1"),
			},
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh2"),
			},
		},
		TimeStamp: 123456,
	}

	tx2 := &Transaction{
		TXID: []byte("tx2"),
		TXInputs: []TXInput{
			{
				Txid:      []byte("prev_tx3"),
				Index:     0,
				ScriptSig: []byte("sig3"),
				PubKey:    []byte("key3"),
			},
			{
				Txid:      []byte("prev_tx4"),
				Index:     1,
				ScriptSig: []byte("sig4"),
				PubKey:    []byte("key4"),
			},
		},
		TXOutputs: []TXOutput{
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh3"),
			},
			{
				SourceID:         uuid.New(),
				ScriptPubKeyHash: []byte("pkh4"),
			},
		},
		TimeStamp: 789012,
	}

	txs := []*Transaction{tx1, tx2}

	// 序列化Transaction数组
	txBytes := transactionArrToByte(txs)

	// 反序列化Transaction数组
	txs2 := byteToTransactionArr(txBytes)

	// 验证结果是否正确
	//fmt.Printf("txs: %#v\n", txs)
	//fmt.Printf("txBytes: %x\n", txBytes)
	fmt.Printf("txs2: %#v\n", string(txs2[0].TXID))
}
