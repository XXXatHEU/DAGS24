package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//定义枚举类型
type dposCommod int

const (
	AskForVali dposCommod = iota //询问应该让谁进行验证
	ReplyAskForVali
	RelpyInitNum     //返回初始序号
	ReplyInitNumData //返回初始序号的基本信息
	AskVote          //让节点进行投票
	ReplyVote        //投票结果
	ValidmyData      //请求验证我的区块
	LaterPost        //一会再来请求我 //接下来这三个都是对数据包里的ReplyAskForvali对应
	BlockSet         //已经被拉黑
	SelfValid        //自己验证就可以了
)
const Flame2fileName string = "../flame2.txt" //存储所有节点类型的数据

type NodeType int

const (
	unhealthyNode NodeType = iota //健康节点
	healthyNode                   //不健康节点
)

//定义传输json时的结构体
type dposJsonStruct struct {
	StringData           string         `json:"stringdata"` //专用于string类型的数据返回
	ReputationDetail     NodeReputation `json:"reputationDetail"`
	Comm                 dposCommod     `json:"comm"`                //命令
	InitNodeNum          int            `json:"data"`                //携带的数据
	GroupPersons         []int          `json:"grouppersons"`        //对应的命令是请求投票AskVote  发送当前组有谁
	GroupNum             int            `json:"groupnum"`            //对应的命令是请求投票AskVote和ReplyVote 发送属于哪个组
	GroupNode            int            `json:"groupnode"`           //对应的命令是请求投票AskVote和ReplyVote 发送是哪个节点
	GroupReplyVote       []float64      `json:"groupreplyvote"`      //对应的命令是ReplyVote 返回投票结果
	GroupReplyVoteCount  int            `json:"groupreplyvotecount"` //对应的命令是ReplyVote 返回投票次数
	GrupALLnodeReputaion int            `json:"grupnodeReputaion"`   //组内共有信誉值
	GrupALLnodeStk       int            `json:"grupALLnodeStk"`      //组内所有的代币
	MyReputation         int            `json:"myReputation"`        //自己的信誉值
	MclicNodeSGrup       map[int][]int  `json:"groupMclicNode"`
	MyNodeType           NodeType       `json:"myNodeType"`
	ReplyForvaliNode     int            `json:"replyAskForvali"` //对应common里面的对 请求验证节点的响应
}

//共有结构体上述

//共有结构体上述
var AskForChan = make(chan dposJsonStruct)

var myGroupNum = 0
var GrupMclic = make(map[int][]int) //每组恶意节点的集合
var inMclieGrup = make(map[int]int) //自己组内恶意节点的集合 key是节点数  value是在组内的下标
var inGrup = make(map[int]bool)     //自己组内所有节点的集合
var nodetype NodeType               //节点类型
var nodeMalic float64               //节点作恶的可能性

var DNSConn net.Conn

var MyInitNum int = -1   //我是第几个节点
var Netdelayms_n float64 //我的网络延迟

func init() {
}

func DposNodeMain() {
	//GetNline(0)
	//()
	//节点命令总程序
	//NetLatency()
	DposClinetOrder()
}

var dnsOrderChan = make(chan string) //向dns里发送命令的管道，写入后select将向dnsConn发送信息

func DposClinetOrder() {

	serverAddr := "127.0.0.1:10001"
	ReadconnChan := make(chan dposJsonStruct)
	readDone := make(chan struct{}) //连接断开

	// 连接服务器
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}
	fmt.Println("节点连接成功")
	DNSConn = conn
	defer conn.Close()

	//子协程接收dnsDpos响应，并将数据给osinput
	go func() {
		for {
			orderMessage, err2 := connGetMsgStruct(conn)
			if err != nil {
				fmt.Println("接受数据时出现异常", err2)
				os.Exit(1)
			}
			RelpyControl(orderMessage)
		}
	}()

	//主动发起请求
	go func() {
		for {
			select {
			case msg := <-dnsOrderChan:
				fmt.Println("收到命令，即将发送数据：", msg)
				connWrite(conn, []byte(msg))

				//_, err := conn.Write([]byte(msg))
				if err != nil {
					fmt.Println("发送命令时发生错误，即将断开连接:", err)
					return
				}

			case <-time.After(time.Second * time.Duration(rand.Intn(10))):
				AskForSomeOneValid(conn)
			}
		}
	}()

	//不断接收请求
	for {
		select {
		case msg := <-ReadconnChan:
			RelpyControl(msg)
		case <-readDone:
			//连接已经断开
			return
		}
	}
}

//对于发来的命令进行总的处理
func RelpyControl(replydata dposJsonStruct) {
	switch replydata.Comm {
	case RelpyInitNum: //如果是发来的通知我的序号是多少
		Mydata, node := InitMyDetail(replydata)
		MyInitNum = replydata.InitNodeNum
		TimeLatency() //RelpyInitNum中TimeLatency需要放到后面
		ReplyMyDetail(Mydata, &node, replydata)
	case AskVote: //让我进行投票
		//投票
		votes, votecount := votStrategy(replydata)
		SendVoteResult(votes, replydata, votecount)
	case ReplyAskForVali: //
		ReplyForvaliNodeControl(replydata)
	default:
		fmt.Println("发来了命令，这个命令处理功能还没有设置", replydata.Comm)
		// 如果没有匹配的值，则执行默认语句块
	}
}

/////////////////////////////////核心运行代码
//投票策略
/*
	赞成票
	   先投自己一票，另外两票随机可能投，可能投一张，可能投两张，投的人是随机的，
	反对票
	  可能投，可能不投，但是不会投自己

*/

//投票策略
func votStrategy(data dposJsonStruct) ([]float64, int) {
	//全部的信誉值
	allrepataion := data.GrupALLnodeReputaion
	allstk := data.GrupALLnodeStk
	mystk := data.ReputationDetail.TC.TC
	weight := float64(mystk) / float64(allstk)

	fmt.Println(MyInitNum, "   ", allrepataion, "    allstk", allstk, " ", mystk)
	//全局恶意节点map
	//下标对应
	fmt.Println("MyInitNum", MyInitNum, "  data.GroupPersons", data.GroupPersons)
	//更新自己的组号
	myGroupNum = data.GroupNum

	GrupMclic = data.MclicNodeSGrup
	for index, num := range GrupMclic[data.GroupNum] {
		inMclieGrup[num] = index
	}
	for _, num := range data.GroupPersons {
		inGrup[num] = true
	}

	votecount := 0 //投票总数
	// 创建与 GroupPersons 数组大小相同的投票数组
	votes := make([]float64, len(data.GroupPersons))
	fmt.Println("votes的大小", len(data.GroupPersons))
	//先投自己一票
	for i, num := range data.GroupPersons {
		if num == MyInitNum {
			votes[i] += weight
			votecount++
		}
	}
	if MyInitNum < 5 {
		fmt.Println(MyInitNum, "操作前", votes)
	}
	//恶意节点投票策略
	if nodetype == unhealthyNode {
		fmt.Println(" MyInitNum:", MyInitNum, "     GrupMclic[MyInitNum]", inMclieGrup[MyInitNum], "    GrupMclic", GrupMclic)
		index1, index2, erroo := votStrategyMclic(data.GroupPersons, GrupMclic[myGroupNum])

		if erroo != nil {
			if inMclieGrup[index1] >= len(votes) || inMclieGrup[index2] >= len(votes) {
				fmt.Println("||||||||||||||||||||||长度不一致问题=========================")
			} else {
				votes[inMclieGrup[index1]] += weight
				votes[inMclieGrup[index2]] += weight
			}

		}
	} else { //普通节点投票策略
		voteCount := rand.Intn(3) // 随机确定投票数量，范围为 0 到 2
		//fmt.Println("普通节点投票数量:", voteCount)
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < voteCount; i++ {
			//随机投票
			randIndex := rand.Intn(len(votes))
			votes[randIndex] += weight
			votecount++
		}
	}

	//投反对票
	// 生成0到1之间的随机数
	randomNumber := rand.Float64() * 1 // 乘以1，范围变为0到1
	if randomNumber >= -1 {            //现在改为一定投反对票
		//遍历一遍，如果是0那么就给他反对票
		// 生成一个随机的起始索引
		startIndex := rand.Intn(len(votes))
		// 使用 range 遍历切片，从随机起始索引开始
		//flag := 0 //为了避免随机不到，如果没有投，那么在从头捋一遍，发现0就给他投反对票
		for index, vote := range votes[startIndex:] {
			_, ok := inMclieGrup[index]
			if ok && nodetype == unhealthyNode {
				continue
			}
			//如果不在自己组里，那么跳过
			// _, ok = inGrup[index]
			// if !ok {
			// 	continue
			// }

			if vote == 0 {
				votes[index] = votes[index] - weight
				//flag = 1
				votecount++
				break
			}
		}
	}
	fmt.Println("MyInitNum", MyInitNum, "投票结果：", votes)
	return votes, votecount
}

//投票策略恶意节点策略  返回在自己组内选取的节点数
func votStrategyMclic(votes []int, mclic []int) (int, int, error) {

	rand.Seed(time.Now().UnixNano())

	// 检查mclic是否至少有一个元素
	if len(mclic) < 1 {
		fmt.Println("数组mclic必须至少包含一个索引")
		return 0, 0, fmt.Errorf("ddd")
	}

	// 随机从数组mclic中选取两个索引，它们可以是相同的  这里只是下标
	index1 := rand.Intn(len(mclic))
	index2 := rand.Intn(len(mclic))

	// 给数组votes对应索引位置的值加一
	// // 输出结果
	// fmt.Printf("随机选取的mclic两个索引：%d 和 %d\n", index1, index2)
	// fmt.Printf("索引对应的votes的位置：%d 和 %d\n", mclic[index1], mclic[index2])

	return mclic[index1], mclic[index2], nil
}

/////////////////////////////主动发起请求

//询问我该向哪个集群进行验证
func AskForValidation() string {
	testjson := dposJsonStruct{
		//Data: "John",
		Comm:        AskForVali,
		GroupNum:    myGroupNum,
		InitNodeNum: MyInitNum,
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

///////////////////////////////////////////////对别人的请求做出处理

//初始化我这个节点的特征
func InitMyDetail(Message dposJsonStruct) (string, NodeReputation) {

	/*
		1、获取我是第几个节点
		2、从文件中获取我的信息
		3、计算综合值
		4、返回结果
	*/

	fmt.Println("更新我的序号值", Message.InitNodeNum)
	lineText := GetNline(Message.InitNodeNum)
	if lineText == "" {
		fmt.Println("从属性文件中得到的数据为空")
		os.Exit(0)
	}
	//切割数据
	fields := strings.Fields(lineText)
	var node NodeReputation
	loadNodeReputation(fields, &node)
	node.CalcuateReputation()
	//节点类型（全局）
	nodetype = node.Extra.HhealthyNodeIdentifier
	//节点作恶可能性（全局）
	nodeMalic = node.Extra.MmaliciousnessProbability
	fmt.Println("node.Gu", node.Gu.Value)
	fmt.Println("node.Pf", node.Pf.Value)
	fmt.Println("node.Sr", node.Sr.Value)
	fmt.Println("node.TC", node.TC.Value)
	fmt.Println("node.Hp", node.Hp.Value)
	fmt.Println("我的信誉值为", node.Value, "我的节点类型", nodetype, "我的作恶可能性", nodeMalic)

	//获得延迟
	Netdelayms_n, _ = strconv.ParseFloat(fields[2], 64)

	return lineText, node
}

func ReplyForvaliNodeControl(replydata dposJsonStruct) {
	AskForChan <- replydata
}

func ReplyMyDetail(linetext string, node *NodeReputation, replydata dposJsonStruct) {
	var outmsg dposJsonStruct
	outmsg.Comm = ReplyInitNumData
	outmsg.StringData = linetext
	nodeCopy := *node // 创建 node 所指向内容的副本
	// 现在，使用节点副本作为中间变量来设置 outmsg 的 ReputationDetail
	outmsg.ReputationDetail = nodeCopy
	outmsg.InitNodeNum = replydata.InitNodeNum
	// 将结构体转换为 JSON 字节流
	jsonBytes, err1 := json.Marshal(outmsg)
	if err1 != nil {
		fmt.Println("转换失败:", err1)
		return
	}
	connWrite(DNSConn, []byte(jsonBytes))
}

func SendVoteResult(votes []float64, replydata dposJsonStruct, votecount int) {
	var outmsg dposJsonStruct
	outmsg.Comm = ReplyVote
	outmsg.GroupReplyVote = votes
	outmsg.InitNodeNum = replydata.InitNodeNum
	outmsg.GroupNum = replydata.GroupNum
	outmsg.GroupNode = replydata.GroupNode
	outmsg.GroupReplyVoteCount = votecount
	// 将结构体转换为 JSON 字节流
	jsonBytes, err1 := json.Marshal(outmsg)
	if err1 != nil {
		fmt.Println("转换失败:", err1)
		return
	}
	connWrite(DNSConn, []byte(jsonBytes))
}

//找谁进行验证 （发送给谁）
func AskForSomeOneValid(conn net.Conn) {
	//1.找谁进行验证 要求返回ip
	msg := AskForValidation()
	connWrite(conn, []byte(msg))
	//阻塞等待返回  需要返回ip
	dataStruct := <-AskForChan

	nodeComm := dataStruct.ReplyForvaliNode
	//如果是自己进行验证，那么就不需要再验证
	if nodeComm == int(SelfValid) {
		fmt.Println("自己验证自己")
		return
	} else if nodeComm == int(ReplyAskForVali) {
		str := dataStruct.StringData
		fmt.Println("活的到的要请求的信息", str)
		return
	}
	//发起连接，等待其返回数据，返回后就关掉
	serverAddr := dataStruct.StringData

	//2.连接尝试验证
	connValid, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}
	fmt.Println("节点连接成功")
	defer connValid.Close()

	//3.循环组包，这里重新组包避免被已刷新组号
	for {
		//3.1组装一个数据包 并发送
		var ValidMessage dposJsonStruct
		ValidMessage.Comm = ValidmyData
		ValidMessage.MyNodeType = nodetype
		//3.2将全局结构体变量转换为 JSON 格式
		jsonData, err := json.Marshal(ValidMessage)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			os.Exit(0)
		}
		//3.3发送数据
		connWrite(connValid, []byte(jsonData))
		//3.4等待返回数据
		replyMessageStrcut, err := connGetMsgStruct(conn)
		if err != nil {
			fmt.Println("AskForSomeOneValid的connGetMsgStruct出现错误")
		}
		//3.5对返回结果进行处理，如果忙再重试，否则就说明验证通过
		if replyMessageStrcut.Comm == LaterPost {
			//3.5.1稍后再试
			time.Sleep(time.Second)
			//3.5.2被拉黑
		} else if replyMessageStrcut.Comm == BlockSet {
			os.Exit(0) //节点直接退出
			//3.5.3正常结束
		} else if SelfValid == replyMessageStrcut.Comm {
			//直接验证通过，执行广播函数
		} else {
			//发起验证函数
			return
		}
	}
}

///////////////////////////工具类

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

//获取第n行的数据，并将第n行的数据按照空格分割后放入数组返回
func GetNline(lineNumber int) string {
	// 打开文件
	file, err := os.Open(Flame2fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentLine := 0

	var lineText string
	for scanner.Scan() {
		if currentLine == lineNumber {
			lineText = scanner.Text()
			break
		}
		currentLine++
	}

	if lineText == "" {
		fmt.Println("没有找到指定的行，或者文件行数不足")
		return ""
	}

	// 使用strings.Fields按空格分割字符串
	//fields := strings.Fields(lineText)

	// 输出分割后的数组
	fmt.Println(lineText)
	return lineText
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
