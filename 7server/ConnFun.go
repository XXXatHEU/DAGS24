package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func DposDnsOrder() {

	// 1. 监听地址和端口
	addr := "127.0.0.1:10001"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Server listening on", addr)

	//启动的选举模式  由Startup_mode_global变量控制
	if Startup_mode_global == po_dpos_mode {
		go SelectControl()
	}

	if Startup_mode_global == old_dpos_mode {
		go Old_SelectControl()
	}

	if Startup_mode_global == vs_dpos_mode {
		go VS_SelectControl()
	}

	if Startup_mode_global == pl_dpos_mode {
		go PL_SelectControl() //选举线程
	}

	//go VS_SelectControl() //选举线程

	//上面如果要计算信誉值的话需要几秒以后才能做，因为下面做完才能开始进行信誉值的计算

	// 4.接受客户端连接
	for {
		// 接受客户端连接
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		channelIndexMutex.Lock()
		//连接后获取这个节点的各个参数
		go handleConnection(conn, Outchannels[channelIndex], Inchannels[channelIndex], channelIndex)
		channelIndex++
		channelIndexMutex.Unlock()
	}

}

// 连接后的事件处理  包括读取协程等
func handleConnection(conn net.Conn, OutChan chan string, InChan chan string, initnum int) {
	defer conn.Close()
	//1.初始化，向其发送是属于第几个节点
	NodeIndexMutex.Lock()
	//定义发送的的请求为初始化
	var initHi dposJsonStruct
	initHi.Comm = RelpyInitNum
	initHi.InitNodeNum = NodeIndex
	NodeIndex++
	NodeIndexMutex.Unlock()

	var overhandChan = make(chan int, 1)     //停止对本节点的连接
	var Old_overhandChan = make(chan int, 1) //停止对本节点的连接
	var VS_overhandChan = make(chan int, 1)  //停止对本节点的连接
	var PL_overhandChan = make(chan int, 1)  //停止对本节点的连接

	//接收命令
	go func() {
		for {
			dposAskStruct, errdposGetAskJson := connGetMsgStruct(conn, initnum)
			if errdposGetAskJson != nil {
				fmt.Println("dposGetAskJson获取ask参数发生错误")
				return
			} else {
				switch dposAskStruct.Comm {

				case ReplyInitNumData: //返回它的信息
					go ReplyInitNumDataFunc(dposAskStruct)

				case ReplyVote: //投票结果
					go CollectVotes(dposAskStruct)
				case VS_ReplyVote: //vs投票结果
					go VS_CollectVotes(dposAskStruct)
				case PL_ReplyVote: //vs投票结果
					go PL_CollectVotes(dposAskStruct)
				case Old_ReplyVote: //vs投票结果
					go Old_CollectVotes(dposAskStruct)

				case Old_AskForVali: //请求验证信息
					//ReplyAskForValiFunc(conn, dposAskStruct)
					err := Old_ReplyAskForValiFunc(conn, dposAskStruct)
					if err != nil {
						Old_overhandChan <- 1
					}
				case PL_AskForVali: //请求验证信息
					//ReplyAskForValiFunc(conn, dposAskStruct)
					err := PL_ReplyAskForValiFunc(conn, dposAskStruct)
					if err != nil {
						PL_overhandChan <- 1
					}
				case VS_AskForVali: //请求验证信息
					//ReplyAskForValiFunc(conn, dposAskStruct)
					err := VS_ReplyAskForValiFunc(conn, dposAskStruct)
					if err != nil {
						VS_overhandChan <- 1
					}

				case AskForVali: //请求验证信息
					//ReplyAskForValiFunc(conn, dposAskStruct)
					err := ReplyAskForValiFunc(conn, dposAskStruct)
					if err != nil {
						overhandChan <- 1
					}
					//return
				/*
					这里留下了一个奇怪的问题，这个return不能使用
					如果使用在CollectVotes就会死锁
					即使把里面的锁删掉也是这样  不知道为什么
				*/
				case MyValidAddr: //设置节点的p2p监听地址
					SetNodeAddr(conn, dposAskStruct)

				case NotifyMaliciousNodeComm: //处理作恶节点
					go HandleMaliciousNode(dposAskStruct)
				case Old_NotifyMaliciousNodeComm: //处理作恶节点
					go Bashline_HandleMaliciousNode(dposAskStruct)
				case VS_NotifyMaliciousNodeComm: //处理作恶节点
					go Bashline_HandleMaliciousNode(dposAskStruct)
				case PL_NotifyMaliciousNodeComm: //处理作恶节点
					go Bashline_HandleMaliciousNode(dposAskStruct)

				default:
					// 如果没有匹配的值，则执行默认语句块
				}
			}
		}
	}()

	// 将结构体转换为 JSON 字节流
	jsonBytes := StructToJson(initHi)
	connWrite(conn, jsonBytes)

	//向节点发送请求的位置
	for {
		select {
		case msg := <-OutChan:
			connWrite(conn, []byte(msg))
		case <-overhandChan: //连接已经断开
			Log.Info(initnum, "已经被拉黑，与之连接将断开")
			DisconnectedPool[initnum] = true
			return
		case <-Old_overhandChan: //连接已经断开
			Log.Info(initnum, "已经被拉黑，与之连接将断开")
			DisconnectedPool[initnum] = true
			return
		case <-PL_overhandChan: //连接已经断开
			Log.Info(initnum, "已经被拉黑，与之连接将断开")
			DisconnectedPool[initnum] = true
			return
		case <-VS_overhandChan: //连接已经断开
			Log.Info(initnum, "已经被拉黑，与之连接将断开")
			DisconnectedPool[initnum] = true
			return
		}
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
		return
	}
}

//请求谱聚类分组 返回分组后的数据
func AskSpecPy(requestData map[string]interface{}) map[int][]int {

	requestDataBytes, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	// 发送POST请求到Python HTTP服务器
	resp, err := http.Post("http://localhost:10002", "application/json", bytes.NewBuffer(requestDataBytes))
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer resp.Body.Close()

	// 读取响应体
	var responseData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return nil
	}
	//fmt.Println(responseData)

	// 解析 responseData["result"] 中的值
	resultMap, ok := responseData["result"].(map[string]interface{})
	if !ok {
		fmt.Println("Error: responseData['result'] is not a map[int][]int")
		return nil
	}
	mapWithArrays := make(map[int][]int)
	for key, value := range resultMap {
		key_int, _ := strconv.Atoi(key)
		//fmt.Println(value)
		if slice, ok := value.(string); ok {
			//fmt.Println("Data 是一个字符串切片:", slice)
			strArray := strings.Split(slice, ",")
			//转换成int类型
			intSlice := make([]int, len(strArray))
			for i, str := range strArray {
				intValue, _ := strconv.Atoi(str)
				intSlice[i] = intValue
			}
			mapWithArrays[key_int] = intSlice
		} else {
			//fmt.Println("Data 不是一个字符串切片")
			return nil
		}
	}
	//fmt.Println(mapWithArrays)
	return mapWithArrays
}

//得到请求的struct结构体形式的数据，如果中间发生错误那么返回nil  server和node有些不同，这里需要知道是哪个initnum以便从池子里面删除
func connGetMsgStruct(conn net.Conn, initnum int) (dposJsonStruct, error) {
	// 读取数据包长度信息
	lenBuffer := make([]byte, 4)
	_, err := io.ReadFull(conn, lenBuffer)
	if err != nil {
		fmt.Println("Error reading:  节点疑似已经断开", err)
		//断开那么就放入到断开池里面
		fmt.Println("上锁")
		ExpPoolMutex.Lock()
		DisconnectedPool[initnum] = true
		ExpPoolMutex.Unlock()
		fmt.Println("解锁===================================")

		return dposJsonStruct{}, err // 返回空的结构体和错误信息
	}
	packetLen := binary.BigEndian.Uint32(lenBuffer)
	// 根据数据包长度读取数据
	buffer := make([]byte, packetLen)
	_, err = io.ReadFull(conn, buffer)
	if err != nil {
		fmt.Println("Error reading data packet:", err)
		os.Exit(1)
		return dposJsonStruct{}, err
	}

	// 将接收到的数据解析为结构体
	var msgStruct dposJsonStruct
	err = json.Unmarshal(buffer, &msgStruct)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		os.Exit(2)
		return dposJsonStruct{}, err
	}
	return msgStruct, nil
}

// 定义结构体来解析 JSON 数据
type ResponseData struct {
	Row  int      `json:"row"`
	Rows []string `json:"rows"`
}

func StructToJson(ValidMessage dposJsonStruct) []byte {
	jsonData, err := json.Marshal(ValidMessage)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		os.Exit(0)
	}
	return jsonData
}

// startTime := time.Now()
// //<-InChan
// endTime := time.Now()
// elapsed := endTime.Sub(startTime)

// fmt.Printf("延迟为%v\n", elapsed)
