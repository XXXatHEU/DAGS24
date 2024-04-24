package main

import (
	"fmt"
	"net"
)

func PL_DposP2phandleConnection(conn net.Conn) {
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
			case PL_ValidmyData:
				PL_DposValidthetx(nodeChan, dposAskStruct)
			default:
				Log.Debug("PL_DposP2phandleConnection中的没有处理的语句")
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
func PL_DposValidthetx(nodeChan chan dposJsonStruct, hisTx dposJsonStruct) {
	//定义发送的的请求为初始化
	var validationResult dposJsonStruct
	validationResult.StringData = "节点发送了一笔恶意交易"
	validationResult.Comm = PL_ValidResult
	InitNum := hisTx.InitNodeNum
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
		PL_NotifyMaliciousNode(InitNum) //举报
	}
	nodeChan <- validationResult
}

//举报恶意节点
func PL_NotifyMaliciousNode(InitNum int) {
	var MalicNode dposJsonStruct
	MalicNode.Comm = PL_NotifyMaliciousNodeComm
	MalicNode.InitNodeNum = InitNum
	jsondata := StructToJson(MalicNode)
	connWrite(DNSConn, []byte(jsondata))
}
