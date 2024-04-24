package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type PackCommand int

const (
	PackBlock PackCommand = iota
	PackTX
	PackDB //请求获取你的本地区块信息
	PackByte
	UnPackBlock
	UnPackTX
	UnPackDB //请求获取你的本地区块信息
	UnPackByte
)

// 将 .db 格式的数据打包为 []byte 类型
func packDBData(dbData *[]byte) []byte {
	var packedData []byte
	packedData = append(packedData, *dbData...)
	return packedData
}

// 将 []byte 类型的数据直接打包
func packByteArrayData(byteData []byte) []byte {
	return byteData
}

// 将 []TX 类型的数据序列化并打包
func packTxArrayData(txArray []Transaction) ([]byte, error) {
	var packedData []byte
	for _, tx := range txArray {
		serializedData := tx.Serialize_tx()
		packedData = append(packedData, serializedData...)
	}
	return packedData, nil
}

// 将 []Block 类型的数据序列化并打包
func packBlockArrayData(blockArray []Block) ([]byte, error) {
	var packedData []byte
	for _, block := range blockArray {
		serializedData := block.Serialize()
		packedData = append(packedData, serializedData...)
	}
	return packedData, nil
}

// 对不同类型的数据进行分别处理，打包为统一格式的 []byte 数组
func PackMessageData(cmd PackCommand, data interface{}) ([]byte, error) {
	if data == nil {
		return nil, errors.New("空输入")
	}
	switch cmd {
	case PackByte:
		byteData, ok := data.([]byte)
		if !ok {
			return nil, errors.New("invalid []byte data")
		}
		return packByteArrayData(byteData), nil
	case PackDB:
		dbData, ok := data.([]byte)
		if !ok {
			return nil, errors.New("invalid DB data")
		}
		return packDBData(&dbData), nil
	case PackTX:
		txArray, ok := data.([]Transaction)
		if !ok {
			return nil, errors.New("invalid TX array")
		}
		return packTxArrayData(txArray)
	case PackBlock:
		blockArray, ok := data.([]Block)
		if !ok {
			return nil, errors.New("invalid Block array")
		}
		return packBlockArrayData(blockArray)
	default:
		return nil, errors.New("unknown command")
	}
}

func unpackBlockArrayData(packedData []byte) ([]Block, error) {
	var blockArray []Block
	for len(packedData) > 0 {
		// 反序列化一个 Block 对象
		block := Deserialize(packedData)
		if block == nil {
			return nil, errors.New("failed to deserialize block")
		}
		serializedData := block.Serialize()
		blockArray = append(blockArray, *block)

		// 更新已解析的字节数据
		packedData = packedData[len(serializedData):]
	}
	return blockArray, nil
}

// 反序列化 []byte 数据为 *[]byte 类型
func unpackDBData(packedData []byte) (*[]byte, error) {
	dbData := make([]byte, len(packedData))
	copy(dbData, packedData)
	return &dbData, nil
}

// 反序列化 []byte 数据为 []byte 类型
func unpackByteArrayData(packedData []byte) ([]byte, error) {
	byteData := make([]byte, len(packedData))
	copy(byteData, packedData)
	return byteData, nil
}
func unpackTxArrayData(packedData []byte) ([]Transaction, error) {
	var txArray []Transaction
	for len(packedData) > 0 {
		// 反序列化一个 Transaction 对象
		tx := Deserialize_tx(packedData)
		serializedData := tx.Serialize_tx()
		if tx == nil {
			return nil, errors.New("failed to deserialize transaction")
		}
		txArray = append(txArray, *tx)

		// 更新已解析的字节数据
		packedData = packedData[len(serializedData):]
	}
	return txArray, nil
}
func UnpackMessageData(cmd PackCommand, packedData []byte) (interface{}, error) {
	if len(packedData) == 0 {
		return nil, errors.New("空输入")
	}
	switch cmd {
	case UnPackByte:
		return unpackByteArrayData(packedData)
	case UnPackDB:
		return unpackDBData(packedData)
	case UnPackTX:
		return unpackTxArrayData(packedData)
	case UnPackBlock:
		return unpackBlockArrayData(packedData)
	default:
		return nil, errors.New("unknown command")
	}
}

//注意是发送请求的时候使用的   和这个方法弃用
func RequestTransformPackCmd(cmd Command) PackCommand {
	switch cmd {
	case SendGenesisBlock:
		return PackBlock
	case SendCommonBlock:
		return PackBlock
	case SendTX:
		return PackTX
	case FetchBlocks:
		return PackBlock
	case SendDB:
		return PackDB
	}
	return -1
}

//注意是接收的时候用的
func ResponeTransformPackCmd(cmd Command) PackCommand {
	switch cmd {
	case SendGenesisBlock:
		return PackBlock
	case SendCommonBlock:
		return PackBlock
	case SendTX:
		return PackTX
	case FetchBlocks:
		return PackBlock
	case SendDB:
		return PackDB
	}
	return -1
}

/*
	发送方知道自己要发什么，所以提供命令和msg
	打包后，发给SendtaskChan
*/
//这是第一个包装，只是将载荷包上了
//注意长度
//PackBlcokArrTaskAndToChan 和交易的 可以看到只是改了一个地方
func PackBlcokArrTaskAndToChan(cmd Command, blockArr []*Block) {
	// 序列化 Payload 字段
	payloadBuf := new(bytes.Buffer)
	err := binary.Write(payloadBuf, binary.BigEndian, blockArrayToByte(blockArr))
	if err != nil {
		Log.Error("序列化 Payload 失败：", err)
		return
	}
	payloadBytes := payloadBuf.Bytes()

	buf := make([]byte, 0)

	// 将 payloadBytes 添加到 buf 中
	buf = append(buf, payloadBytes...)

	// 序列化 Cmd 字段
	var cmdBytes [4]byte
	binary.BigEndian.PutUint32(cmdBytes[:], uint32(cmd))
	fmt.Println("cmd:", uint32(cmd))
	buf = append(cmdBytes[:], buf...)

	// 序列化 Len 字段
	var lenBytes [4]byte
	binary.BigEndian.PutUint32(lenBytes[:], uint32(len(buf)+4)) //这里并没有加入自己的4个长度
	Log.Info("消息长度为：", uint32(len(buf)+4))
	buf = append(lenBytes[:], buf...)
	//fmt.Println(buf)
	// 将 buf 发送到消息队列中
	SendtaskChan <- buf
}
func PackTxArrTaskAndToChan(cmd Command, txs []*Transaction) {
	// 序列化 Payload 字段
	payloadBuf := new(bytes.Buffer)
	err := binary.Write(payloadBuf, binary.BigEndian, transactionArrToByte(txs))
	if err != nil {
		Log.Error("序列化 Payload 失败：", err)
		return
	}
	payloadBytes := payloadBuf.Bytes()

	buf := make([]byte, 0)

	// 将 payloadBytes 添加到 buf 中
	buf = append(buf, payloadBytes...)

	// 序列化 Cmd 字段
	var cmdBytes [4]byte
	binary.BigEndian.PutUint32(cmdBytes[:], uint32(cmd))
	fmt.Println("cmd:", uint32(cmd))
	buf = append(cmdBytes[:], buf...)

	// 序列化 Len 字段
	var lenBytes [4]byte
	binary.BigEndian.PutUint32(lenBytes[:], uint32(len(buf)+4)) //这里并没有加入自己的4个长度
	Log.Info("消息长度为：", uint32(len(buf)+4))
	buf = append(lenBytes[:], buf...)
	fmt.Println(buf)
	// 将 buf 发送到消息队列中

	SendtaskChan <- buf
	//Tesett(buf)
}

//原来的  备份
func PackTxArrTaskAndToChanPRE(cmd Command, txs []*Transaction) {

	req := SendMessage{
		Cmd:     cmd,
		Payload: transactionArrToByte(txs),
	}

	// 将req转换成字节数组
	buf := bytes.Buffer{}
	err := binary.Write(&buf, binary.BigEndian, req)
	if err != nil {
		panic(err)
	}
	reqBytes := buf.Bytes()

	// 设置Len字段
	reqLen := int32(len(reqBytes))
	req.Len = reqLen

	// 将SendMessage结构体转换为字节数组并发送
	buf.Reset()
	err = binary.Write(&buf, binary.BigEndian, &req)
	if err != nil {
		panic(err)
	}
	// packedReq := buf.Bytes()

	// task := SendTask{
	// 	TaskData: packedReq,
	// }
	// SendtaskChan <- task
}

//pack一些其他命令什么的
func PackMessage(cmd Command, str string) []byte {
	req := SendMessage{
		Cmd:     cmd,
		Payload: []byte(str),
	}
	buf := bytes.Buffer{}
	err := binary.Write(&buf, binary.BigEndian, req)
	if err != nil {
		panic(err)
	}
	reqBytes := buf.Bytes()

	// 设置Len字段
	reqLen := int32(len(reqBytes))
	req.Len = reqLen

	// 将SendMessage结构体转换为字节数组并发送
	buf.Reset()
	err = binary.Write(&buf, binary.BigEndian, &req)
	if err != nil {
		panic(err)
	}
	packedReq := buf.Bytes()
	return packedReq
}

func unPackTask(packedReq []byte) (msg SendMessage) {
	req := &SendMessage{}
	gob.Register(elliptic.P256())
	// 使用 bytes.Buffer 将 packedReq 写入缓冲区
	buf := bytes.NewBuffer(packedReq)

	// 使用 binary.Read 反序列化缓冲区中的数据到 req 中
	err := binary.Read(buf, binary.BigEndian, req)
	if err != nil {
		panic(err)
	}
	return *req
}

// func PackTxArrTaskAndToChan(cmd Command, txs []*Transaction) {
// 	req := SendMessage{
// 		Cmd:     cmd,
// 		Payload: msg,
// 	}

// 	task := SendTask{
// 		TaskData: req,
// 	}
// 	SendtaskChan <- task
// }

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
func init() {
	gob.Register(&Block{})
	gob.Register(&Transaction{})
	gob.Register(&TXInput{})
	gob.Register(&TXOutput{})
	gob.Register(&TXOutput{})
	gob.Register(elliptic.P256())
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

	gob.Register(&Transaction{})
	gob.Register(elliptic.P256())
	gob.Register(&TXInput{})
	gob.Register(&TXOutput{})
	gob.Register(&TXOutput{})
	gob.Register(uuid.UUID{})
	encoder := gob.NewEncoder(&buffer)
	for _, tx := range txs {
		// 编码Transaction数组
		err := encoder.Encode(tx)
		if err != nil {
			panic(err)
		}
	}

	return buffer.Bytes()
}

// 将字节数组反序列化为Transaction数组
func byteToTransactionArr(byteArr []byte) []*Transaction {
	var txs []*Transaction

	gob.Register(&Transaction{})
	gob.Register(elliptic.P256())
	gob.Register(&TXInput{})
	gob.Register(&TXOutput{})
	gob.Register(&TXOutput{})
	gob.Register(uuid.UUID{})
	// 创建解码器 uuid.UUID
	buffer := bytes.NewBuffer(byteArr)
	decoder := gob.NewDecoder(buffer)

	// 解码Transaction数组
	for {
		var tx Transaction
		err := decoder.Decode(&tx)
		if err != nil {
			break
		}
		txs = append(txs, &tx)
	}

	return txs
}
