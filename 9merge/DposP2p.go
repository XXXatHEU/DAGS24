package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
)

var DPosDnsPeerAddr string

var ValidtxMapMutex sync.Mutex      //保护下面两个map  也要加上自己
var HealthTxMap = make(map[int]int) //正常交易 [10]2 说明10发送了2个交易
var MalicTxMap = make(map[int]int)  //恶意交易
var HealthTxNum = 0                 //这一轮正常交易的总数量
var MalicTxNum = 0                  //这一轮恶意交易的总数量

func DposListen() {
	//获得本地监听端口
	localip, _ := GetLocalIp()
	localport, _ := FindAvailablePort()
	DPosDnsPeerAddr = localip + ":" + strconv.Itoa(localport)
	Log.Info("节点获得ip监听地址:", DPosDnsPeerAddr)
	//发送数据
	Log.Info("开始发送节点获得ip监听地址")
	var ValidMessage dposJsonStruct
	ValidMessage.Comm = MyValidAddr
	ValidMessage.InitNodeNum = MyInitNum
	ValidMessage.StringData = DPosDnsPeerAddr
	jsonData, err := json.Marshal(ValidMessage)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		os.Exit(0)
	}
	//发送本节点的数
	connWrite(DNSConn, []byte(jsonData))
	DposDnsOrder(DPosDnsPeerAddr) //里面是for循环
}

func DposDnsOrder(addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Peer Node listening on", addr)
	// 4.接受peer客户端连接
	for {
		Log.Info("节点", MyInitNum, "DPosDnsPeerAddr", DPosDnsPeerAddr, "收到连接请求")
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("接受连接失败:", err)
			continue
		}
		go DposP2phandleConnection(conn) //没有for循环，执行一次就退出
	}
}

//
func DposP2phandleConnection(conn net.Conn) {
	defer conn.Close()
	var nodeChan = make(chan dposJsonStruct)
	//启动读取请求协程
	go func() {
		Log.Info("启动读取请求协程===================================")
		dposAskStruct, errdposGetAskJson := connGetMsgStruct(conn)
		if errdposGetAskJson != nil {
			fmt.Println("DposP2phandleConnection中读取对方发送的数据失败,", errdposGetAskJson)
			return
		} else {
			switch dposAskStruct.Comm {
			case ValidmyData:
				DposValidthetx(nodeChan, dposAskStruct)
			case Old_ValidmyData:
				Old_DposValidthetx(nodeChan, dposAskStruct)
			case PL_ValidmyData:
				PL_DposValidthetx(nodeChan, dposAskStruct)
			case VS_ValidmyData:
				VS_DposValidthetx(nodeChan, dposAskStruct)

			default:
				Log.Debug("DposP2phandleConnection中的没有处理的语句")
				// 如果没有匹配的值，则执行默认语句块
			}
		}
	}()

	//仅执行一次就关闭连接
	select {
	case msg := <-nodeChan:
		jsonBytes := StructToJson(msg)
		connWrite(conn, jsonBytes)
	}
}

//验证交易
func DposValidthetx(nodeChan chan dposJsonStruct, hisTx dposJsonStruct) {
	//定义发送的的请求为初始化
	var validationResult dposJsonStruct
	validationResult.StringData = "节点发送了一笔恶意交易"
	validationResult.Comm = ValidResult
	InitNum := hisTx.InitNodeNum
	InitGrupNum := hisTx.GroupNum
	if hisTx.MyNodeType == healthyNode {
		validationResult.ValidResultData = ValTxSucces
		ValidtxMapMutex.Lock()
		HealthTxMap[InitNum]++
		HealthTxNum++
		ValidtxMapMutex.Unlock()
		Log.Info("DPosDnsPeerAddr", DPosDnsPeerAddr, "表示", InitNum, "节点发送的交易验证成功")
	} else {
		validationResult.ValidResultData = ValTxFaild
		ValidtxMapMutex.Lock()
		MalicTxMap[InitNum]++
		MalicTxNum++
		ValidtxMapMutex.Unlock()
		Log.Info("DPosDnsPeerAddr", DPosDnsPeerAddr, "表示", InitNum, "节点发送了一笔恶意交易", "hisTx MyNodeType", hisTx.MyNodeType, "histx", hisTx)
		NotifyMaliciousNode(InitNum, InitGrupNum) //举报
	}
	nodeChan <- validationResult
}

//举报恶意节点
func NotifyMaliciousNode(InitNum int, InitGrupNum int) {
	var MalicNode dposJsonStruct
	MalicNode.Comm = NotifyMaliciousNodeComm
	MalicNode.InitNodeNum = InitNum
	MalicNode.GroupNum = InitGrupNum
	jsondata := StructToJson(MalicNode)
	connWrite(DNSConn, []byte(jsondata))
}
