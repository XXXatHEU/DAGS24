package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
)

func main22() {
	// 生成一个随机的节点 ID
	nodeID := make([]byte, 16)
	_, err := rand.Read(nodeID)
	if err != nil {
		panic(err)
	}
	nodeIDString := hex.EncodeToString(nodeID)
	fmt.Printf("nodeIDString: %s\n", nodeIDString)
	// 建立连接
	conn, err := net.Dial("udp", "127.0.0.1:9000")
	if err != nil {
		panic(err)
	}

	// 发送握手信息
	versionMessage := NewVersionMessage(nodeIDString, "my-app", "/1.0.0/", 1234567890)
	WriteMessage(conn, versionMessage)

	// 等待对方应答握手信息
	message, err := ReadMessage(conn)
	if err != nil {
		panic(err)
	}
	switch message := message.(type) {
	case *VersionMessage:
		fmt.Printf("Connected to node %v\n", message.getNodeID())
	default:
		panic("Unexpected message type")
	}
}

// Message 定义了消息类型的接口
type Message interface {
	// 序列化后的消息格式
	Serialize() []byte
	// 反序列化为具体的消息类型
	Deserialize(data []byte) (Message, error)
}

// VersionMessage 是一种握手信息，包括节点 ID、应用名称、协议版本和时间戳等信息
type VersionMessage struct {
	NodeID      string
	Application string
	Protocol    string
	Timestamp   int64
}

// NewVersionMessage 用于创建一个 VersionMessage 对象
func NewVersionMessage(nodeID string, application string, protocol string, timestamp int64) *VersionMessage {
	return &VersionMessage{
		NodeID:      nodeID,
		Application: application,
		Protocol:    protocol,
		Timestamp:   timestamp,
	}
}

// getNodeID 返回节点 ID 的字符串表示
func (message *VersionMessage) getNodeID() string {
	return message.NodeID
}

// Serialize 将 VersionMessage 对象序列化为二进制数据
func (message *VersionMessage) Serialize() []byte {
	// ...
	return nil
}

// Deserialize 将二进制数据反序列化为 VersionMessage 对象
func (message *VersionMessage) Deserialize(data []byte) (Message, error) {
	// ...
	return &VersionMessage{}, nil
}

// BlockMessage 是一种区块消息，包括区块哈希值和区块数据等信息
type BlockMessage struct {
	BlockHash []byte
	Data      []byte
}

// NewBlockMessage 用于创建一个 BlockMessage 对象
func NewBlockMessage(blockHash []byte, data []byte) *BlockMessage {
	return &BlockMessage{
		BlockHash: blockHash,
		Data:      data,
	}
}

// NodeID 返回空字符串，因为区块消息没有节点 ID
func (message *BlockMessage) NodeID() string {
	return ""
}

// Serialize 将 BlockMessage 对象序列化为二进制数据
func (message *BlockMessage) Serialize() []byte {
	// ...
	return nil
}

// Deserialize 将二进制数据反序列化为 BlockMessage 对象
func (message *BlockMessage) Deserialize(data []byte) (Message, error) {
	// ...
	return &BlockMessage{}, nil
}

// WriteMessage 将消息写入连接中
func WriteMessage(conn net.Conn, message Message) error {
	switch message := message.(type) {
	case *VersionMessage:
		// 处理 VersionMessage 类型的消息
	case *BlockMessage:
		// 处理 BlockMessage 类型的消息
		payload := message.Serialize()
		header := make([]byte, 5)
		header[0] = 0x02
		binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))
		if _, err := conn.Write(header); err != nil {
			return err
		}
		if _, err := conn.Write(payload); err != nil {
			return err
		}
	default:
		panic("Unexpected message type")
	}

	// ...
	return nil
}

// ReadMessage 从连接中读取消息
func ReadMessage(conn net.Conn) (Message, error) {
	// 读取消息类型和长度
	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	messageType := header[0]
	length := binary.BigEndian.Uint32(header[1:])

	// 根据消息类型读取相应的内容
	var payload []byte
	switch messageType {
	case 0x01:
		payload = make([]byte, length)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return nil, err
		}
		// 反序列化 VersionMessage 对象
		message, err := DeserializeVersionMessage(payload)
		if err != nil {
			return nil, err
		}
		return message, nil
	case 0x02:
		payload = make([]byte, length)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return nil, err
		}
		// 反序列化 BlockMessage 对象
		message, err := DeserializeBlockMessage(payload)
		if err != nil {
			return nil, err
		}
		return message, nil
	default:
		return nil, fmt.Errorf("unsupported message type: 0x%x", messageType)
	}
}

// DeserializeVersionMessage 将二进制数据反序列化为 VersionMessage 对象
func DeserializeVersionMessage(data []byte) (*VersionMessage, error) {
	// ...
	return &VersionMessage{}, nil
}

// DeserializeBlockMessage 将二进制数据反序列化为 BlockMessage 对象
func DeserializeBlockMessage(data []byte) (*BlockMessage, error) {
	// ...
	return &BlockMessage{}, nil
}
