package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"
)

func DposNodeMain() {
	DposClinetOrder()
}

var dnsOrderChan = make(chan string) //向dns里发送命令的管道，写入后select将向dnsConn发送信息

func DposClinetOrder() {

	serverAddr2 := "127.0.0.1:10001"
	ReadconnChan1 := make(chan dposJsonStruct)
	readDone1 := make(chan struct{}) //连接断开

	// 连接服务器
	conn, err := net.Dial("tcp", serverAddr2)
	if err != nil {
		Log.Warn("DposClinetOrder连接dns服务器失败:", err)
		os.Exit(1)
	}
	fmt.Println("节点连接成功")
	DNSConn = conn
	defer conn.Close()

	//////////////////////////////////主动接受请求
	go func() {
		for {
			orderMessage, err2 := connGetMsgStruct(conn)
			if err != nil {
				fmt.Println("DposClinetOrder接受数据时出现异常", err2)
				os.Exit(1)
			}
			RelpyControl(orderMessage)
		}
	}()

	//////////////////////////////////////从本地接受生成的请求
	go func() {
		for {
			select {
			//向dns发送消息的接口
			case msg := <-dnsOrderChan:
				fmt.Println("DposClinetOrder收到命令，即将发送数据：", msg)
				connWrite(conn, []byte(msg))
				if err != nil {
					fmt.Println("DposClinetOrder发送命令时发生错误，即将断开连接:", err)
					return
				}
			case <-time.After(time.Second * time.Duration(rand.Intn(30))): //周期性发送命令
				//请求做个验证
				//go Old_AskForSomeOneValid(conn)
				//go VS_AskForSomeOneValid(conn)
				// go VS_AskForSomeOneValid(conn)
				if SelectVersion == -1 {
					time.Sleep(20 * time.Second)
					continue
				}
				go PL_AskForSomeOneValid(conn)
				//AskForSomeOneValid(conn)
			}
		}
	}()

	//////////////////////////////////对请求做出处理
	for {
		select {
		case msg := <-ReadconnChan1:
			RelpyControl(msg) //对于别的节点发来的信息  全由这个地方处理
		case <-readDone1:
			return
		}
	}
}

//处理请求
func RelpyControl(replydata dposJsonStruct) {
	switch replydata.Comm {

	//初始化序号
	case RelpyInitNum: //如果是发来的通知我的序号是多少（初始化）
		Mydata, node := InitMyDetail(replydata)
		MyInitNum = replydata.InitNodeNum
		Log.Warn("MyInitNum ", MyInitNum, " 我的信誉值为", node.Value, "我的节点类型", nodetype, "我的作恶可能性", nodeMalic)
		//TimeLatency() //RelpyInitNum中TimeLatency需要放到后面
		ReplyMyDetail(Mydata, &node, replydata)

		//投票
	case AskVote: //让我进行投票po-dpos
		votes, votecount := votStrategy(replydata)
		SendVoteResult(votes, replydata, votecount)
	case VS_AskVote:
		myvoteresult := VS_votStrategy(replydata)
		VS_SendVoteResult(myvoteresult, replydata)
	case PL_AskVote:
		myvoteresult := PL_votStrategy(replydata)
		PL_SendVoteResult(myvoteresult, replydata)
	case Old_AskVote:
		myvoteresult := Old_votStrategy(replydata)
		Old_SendVoteResult(myvoteresult, replydata)

		//想要知道找谁进行验证
	case ReplyAskForVali: // 请求去验证
		go ReplyForvaliNodeControl(replydata)
	case Old_ReplyAskForVali: // 请求去验证
		go Old_ReplyForvaliNodeControl(replydata)
	case PL_ReplyAskForVali: // 请求去验证
		go PL_ReplyForvaliNodeControl(replydata)
	case VS_ReplyAskForVali: // 请求去验证
		go VS_ReplyForvaliNodeControl(replydata)
	case ValidResult:
		fmt.Println("ValidResult=======", replydata.Comm)
	default:
		fmt.Println("发来了命令，这个命令处理功能还没有设置", replydata.Comm)
		// 如果没有匹配的值，则执行默认语句块
	}
}

//延迟
func TimeLatency() {

	//网络延迟设置
	delay := time.Duration(Netdelayms_n) * time.Millisecond // 创建time.Duration对象，表示延迟的时间
	startTime := time.Now()
	<-time.After(delay) // 阻塞等待延迟时间过去
	actualDelay := time.Since(startTime)
	fmt.Printf("use time.After() delay %f ms, real delay %v\n", Netdelayms_n, actualDelay.Milliseconds())

}

//得到请求的struct结构体形式的数据，如果中间发生错误那么返回nil
func connGetMsgStruct(conn net.Conn) (dposJsonStruct, error) {
	// 读取数据包长度信息
	lenBuffer := make([]byte, 4)
	_, err := io.ReadFull(conn, lenBuffer)
	if err != nil {
		fmt.Println("Error reading:  节点疑似已经断开", err)
		os.Exit(1)
		return dposJsonStruct{}, err // 返回空的结构体和错误信息
	}
	packetLen := binary.BigEndian.Uint32(lenBuffer)
	// 根据数据包长度读取数据
	buffer := make([]byte, packetLen)
	_, err = io.ReadFull(conn, buffer)
	if err != nil {
		Log.Warn("在connGetMsgStruct发生致命错误:", err)
		return dposJsonStruct{}, err
	}

	// 将接收到的数据解析为结构体
	var msgStruct dposJsonStruct
	err = json.Unmarshal(buffer, &msgStruct)
	if err != nil {
		fmt.Println("connGetMsgStruct中Error unmarshalling JSON:", err)
		os.Exit(2)
		return dposJsonStruct{}, err
	}
	return msgStruct, nil
}

//网络延迟功能的测试
func NetLatency() {
	tryCnt := 1000
	delayms_n := 1000 // 延迟时间，单位毫秒
	for i := 0; i < tryCnt; i++ {
		delay := time.Duration(delayms_n) * time.Millisecond // 创建time.Duration对象，表示延迟的时间
		startTime := time.Now()
		<-time.After(delay) // 阻塞等待延迟时间过去
		actualDelay := time.Since(startTime)
		fmt.Printf("use time.After() delay %d ms, real delay %v\n", delayms_n, actualDelay.Milliseconds())
	}
}

//防止粘包的发送接口
func connWrite(conn net.Conn, data []byte) {
	// 计算数据包的长度
	packetLength := make([]byte, 4)
	binary.BigEndian.PutUint32(packetLength, uint32(len(data)))
	// 合并头部和数据包内容
	packet := append(packetLength, data...)
	// 发送数据包到连接中
	_, err := conn.Write(packet)
	if err != nil {
		fmt.Println("发送命令时发生错误，即将断开连接:", err)
		os.Exit(0)
		return
	}
}

func StructToJson(ValidMessage dposJsonStruct) []byte {
	jsonData, err := json.Marshal(ValidMessage)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		os.Exit(0)
	}
	return jsonData
}
