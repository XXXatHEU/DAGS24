package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"
)

//共有结构体上述
var PL_AskForChan = make(chan dposJsonStruct)

var PL_myGroupNum = 0
var PL_GrupMclic = make(map[int][]int) //每组恶意节点的集合
var PL_inMclieGrup = make(map[int]int) //自己组内恶意节点的集合 key是节点数  value是在组内的下标
var PL_inGrup = make(map[int]bool)     //自己组内所有节点的集合
//var nodetype NodeType               //节点类型
//var nodeMalic float64               //节点作恶的可能性

//var DNSConn net.Conn

//var SelectVersion = -1   //当前版本的类型
//var MyInitNum int = -1   //我是第几个节点
//var Netdelayms_n float64 //我的网络延迟

/////////////////////////////主动发起请求

//询问我该向哪个集群进行验证
func PL_AskForValidation() string {
	testjson := dposJsonStruct{
		//Data: "John",
		Comm:        PL_AskForVali,
		InitNodeNum: MyInitNum,
		IntData:     SelectVersion, //选举版本
	}
	// 将全局结构体变量转换为 JSON 格式
	jsonData, err := json.Marshal(testjson)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return ""
	}
	// 打印 JSON 格式的数据
	return string(jsonData)
}

func PL_ReplyForvaliNodeControl(replydata dposJsonStruct) {
	PL_AskForChan <- replydata
}

//找谁进行验证 （发送给谁）
func PL_AskForSomeOneValid(conn net.Conn) {

	var repuationDeatil NodeReputation
	//如果版本错误，说明是使用的上次的分组版本
	var dataStruct dposJsonStruct
	//不断尝试发送验证请求 直到告诉我问谁
	for {
		//如果还没开始投票 就先停止发送
		if SelectVersion == -1 {
			time.Sleep(time.Second)
			continue
		}

		//1.找谁进行验证 要求返回ip
		msg := PL_AskForValidation()

		connWrite(conn, []byte(msg))

		//阻塞等待返回  需要返回ip
		dataStruct = <-PL_AskForChan
		nodeComm := dataStruct.PL_ReplyForvaliNode
		fmt.Println("返回数据===============================", nodeComm)
		//如果是自己进行验证，那么就不需要再验证
		if nodeComm == int(SelfValid) {
			fmt.Println("自己验证自己")
			return
		} else if nodeComm == int(BlockSet) {
			continue
			//Log.Fatal("自己已经被拉黑，退出", "  我的序号MyInitNum:", MyInitNum, "  我监听的", DPosDnsPeerAddr)
			//os.Exit(0) 没有拉黑选项
		} else if nodeComm == int(PL_ReplyAskForVali) {
			str := dataStruct.StringData
			fmt.Println("获得到了所要的信息", str)
			repuationDeatil = dataStruct.ReputationDetail
			break
		} else if nodeComm == int(LaterPost) {
			fmt.Println("稍后再试")
			time.Sleep(time.Second * 5)
			continue
		} else {
			fmt.Println("什么命令也不是")
			time.Sleep(time.Second * 5)
		}
	}

	//发起连接，等待其返回数据，返回后就关掉
	serverAddr3 := dataStruct.StringData
	fmt.Println("找", serverAddr3, "进行验证")
	//2.连接尝试验证
	connValid, err := net.Dial("tcp", serverAddr3)
	if err != nil {
		Log.Warn("连接验证节点失败：", err)
		return
	}
	defer connValid.Close()

	fmt.Println("节点连接成功")

	//3.1组装一个数据包 并发送
	var ValidMessage dposJsonStruct
	ValidMessage.Comm = PL_ValidmyData
	ValidMessage.MyNodeType = nodetype
	ValidMessage.InitNodeNum = MyInitNum

	Probability := repuationDeatil.Extra.MmaliciousnessProbability

	// 设置随机数种子
	rand.Seed(time.Now().UnixNano())
	// 生成随机数
	randomNumber := rand.Float64()
	// 如果随机数小于0.5，则生成0，否则生成1
	if randomNumber < Probability {
		ValidMessage.MyNodeType = nodetype
	} else {
		ValidMessage.MyNodeType = healthyNode
	}
	Log.Warn("randomNumber:", randomNumber, "  Probability:", Probability, " ValidMessage.MyNodeType", ValidMessage.MyNodeType, "  nodetype", nodetype)

	//3.2将全局结构体变量转换为 JSON 格式
	jsonData := StructToJson(ValidMessage)
	//3.3发送数据
	connWrite(connValid, []byte(jsonData))
	//3.4等待返回数据
	replyMessageStrcut, err := connGetMsgStruct(connValid)
	if err != nil {
		fmt.Println("AskForSomeOneValid的connGetMsgStruct出现错误")
	}
	if replyMessageStrcut.ValidResultData == ValTxSucces {
		Log.Info("对方节点对本验证成功")
	} else if replyMessageStrcut.ValidResultData == ValTxFaild {
		Log.Info("作恶被发现!!!!!")
	} else if replyMessageStrcut.ValidResultData == ValTxBlock { //没有使用
		Log.Fatal("节点已经被拉黑")
	} else if replyMessageStrcut.ValidResultData == vodi0 {
		Log.Fatal("返回了一个异常值，没有注入值  ===========", replyMessageStrcut.StringData, replyMessageStrcut)
	}

}
