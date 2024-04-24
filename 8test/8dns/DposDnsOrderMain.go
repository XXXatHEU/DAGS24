package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
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

///下面是配置变量
var SelectInterval = 10 //选举间隔时长
var FirstIN = true      //第一次进入

var AdminNum = 2                  //中央委员会成员数据
var AdminList []string            //中央委员会成员
var AdminList2 []string           //备选中央委员会成员
var NomalAdminList []string       //基层委员会成员的所有成员
var SelectingMutex sync.Mutex     //正在选举过程
var VoteOverChan = make(chan int) //发送信号则说明投票结束

var GroupNodeNum = 0 //参与本轮投票的总人数

var GrupMclic = make(map[int][]int) //每组的恶意节点标识

var GrupArrayVoteMutex sync.Mutex   //保护投票记录和GroupNodeNum
var GrupArray = make(map[int][]int) //分组结果 比如[1][9,21,1,34,5]是1组里有[9,21,34,5]这几个节点
var GrupArrayVote map[int][]float64 //投票记录  [1][1,2,3，4]表示9得到的投票值为1，21投票值为2

var GrupindexToIndex = make(map[int]int) //比如[1]= 10 说明1在它的组里下标是10
var GrupallReputaion = make(map[int]int) //每个组的总信誉值 [1] = 10 说明组内总信誉值为10
var Grupallstk = make(map[int]int)       //每个组的总信誉值 [1] = 10 说明组内总信誉值为10
var VoteFianlResult map[int][]int        //最终获胜者
//下一组应该访问谁
var NextGroupNode = make(map[int]int)

//用于保护Outchannels 和 Inchannels
var channelIndexMutex sync.Mutex            //存储addr和对应的账户和账户特定信息的节点
var channelIndex = 0                        //从0开始递增，表明有多少用户加入，即使用户退出也不会减少
var Outchannels = make([]chan string, 5000) //发送消息 通道
var Inchannels = make([]chan string, 5000)  //收到消息 通道

var NodeIndexMutex sync.Mutex
var NodeIndex = 0

var AllNodeReputationMutex sync.Mutex //保护map
//所有节点的信誉struct
var AllNodeReputation = make(map[int]NodeReputation)

var DisconnectedPool = make(map[int]bool) //对方连接断开
var BlacklistPool = make(map[int]bool)    //拉黑池
var ExpPoolMutex sync.Mutex               //上面两个池子用一把锁

//注意 channelIndexMutex和 ExpPoolMutex 需要按顺序加锁
func init() {
	// 创建一个包含70个channel的切片
	// 初始化每个channel
	for i := range Outchannels {
		Outchannels[i] = make(chan string)
		Inchannels[i] = make(chan string)
	}
}

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

	go SelectControl()   //选举线程
	go SimultaeControl() //模拟线程

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
	//先启动读取协程

	//启动读取请求协程
	go func() {
		for {
			dposAskStruct, errdposGetAskJson := connGetMsgStruct(conn, initnum)
			if errdposGetAskJson != nil {
				fmt.Println("dposGetAskJson获取ask参数发生错误")
				return
			} else {
				switch dposAskStruct.Comm {
				case ReplyInitNumData:
					// //统计延迟信息 测试用
					// dposAskJson, _ := json.Marshal(dposAskStruct)
					// InChan <- string(dposAskJson)
					//具体的处理
					//fmt.Println(dposAskStruct.ReputationDetail.extra)
					ReplyInitNumDataFunc(dposAskStruct)

				case ReplyVote: //投票结果
					SpecDPoSSelect2(dposAskStruct)
				case AskForVali:
					ReplyAskForValiFunc(conn, dposAskStruct)
				default:
					// 如果没有匹配的值，则执行默认语句块
				}
			}
		}
	}()

	// 将结构体转换为 JSON 字节流
	jsonBytes, err1 := json.Marshal(initHi)
	if err1 != nil {
		fmt.Println("转换失败:", err1)
		return
	}
	connWrite(conn, jsonBytes)
	// startTime := time.Now()
	// //<-InChan
	// endTime := time.Now()
	// elapsed := endTime.Sub(startTime)

	// fmt.Printf("延迟为%v\n", elapsed)

	//向节点发送请求的位置
	for {
		select {
		case msg := <-OutChan:
			connWrite(conn, []byte(msg))
		}
	}
}

////////////////////////////////////////下面是dns进行选举过程
//控制选举过程
func SelectControl() {
	//ticker := time.NewTicker(5 * time.Minute)
	ticker := time.NewTicker(time.Duration(SelectInterval) * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C // 每次从 ticker 的通道中读取，等待 5 分钟
		NodeIndexMutex.Lock()
		if NodeIndex < 2 {
			NodeIndexMutex.Unlock()
			continue
		}
		NodeIndexMutex.Unlock()
		SelectingMutex.Lock() //告知我正在选举，将停止推送应该访问哪个节点
		InitSpecDPoSSelct()   //初始化设置 进行spec分组 并将分组结果设置全局变量
		SpecDPoSSelect()
		SelectingMutex.Unlock()

	}
}

/*
	0、获取每个节点所有信息（不加选举锁）
	1、设置分组，将分组放置到全局变量中
	2、通知分组结果
	3、进行投票选举
*/

func InitSpecDPoSSelct() {
	/*
		1. 谱聚类分组，将前一次的领导节点形成领导者选择节点
		2. 对每一组形成通知协程，向各自组内节点发送通知消息，并收集投票结果
	*/
	//分组需要判断当前节点的数量有多少,需要剔除被拉黑节点和其他一些节点

}

//我的改进dpos算法的选举过程
func SpecDPoSSelect() {
	//找到能用的节点
	validNodes := GetValidNodes()
	//返回能参与选举的节点 并设置好中央委员会节点
	validNodes = SpecDPoSAdminSelect(validNodes)
	GroupNodeNum = len(validNodes)
	fmt.Println()
	fmt.Println("参与下层分组成员", validNodes)
	// 使用 strings.Join() 构建带逗号分隔符的字符串
	result := strings.Join(validNodes, ",")
	if len(validNodes) == 0 {
		fmt.Println("无可用在线节点，退出本次投票")
		return
	}
	// 准备要发送的数据
	requestData := map[string]interface{}{
		"rows": result, //删选后的行号
		"row":  "3",    //聚类的数量
	}
	mapWithArrays := AskGroupPy(requestData)
	fmt.Println("谱聚类分组结果", mapWithArrays)

	if mapWithArrays == nil {
		fmt.Println("AskGroupPy出现错误")
		return
	}

	//这里假设已经分组发送分组完成，这里设置分组数量为3，以后也直接指定分组数量就可以
	/*
		1{1,2,5,7,9,16,17}
		2{3,4,6,8,18,19}
		3{10,11,12,20,21,22}
		4{13,14,15,23,24,25,26,27} 竞选领导者节点
	*/
	GrupArray = mapWithArrays //赋给全局变量
	GrupArrayVote = make(map[int][]float64, len(mapWithArrays))
	GrupMclic = make(map[int][]int)
	for key, value := range GrupArray {
		// 为新的键创建一个切片，长度与原始切片相同，但所有元素为0 用来表示投票结果
		votes := make([]float64, len(value))
		GrupArrayVote[key] = votes

		//下面是找到每组的恶意节点 放到GrupMclic
		AllNodeReputationMutex.Lock()
		for index, node := range value {
			//更新组内的总信誉值
			GrupallReputaion[key] += int(AllNodeReputation[index].Value)
			Grupallstk[key] += int(AllNodeReputation[index].TC.TC)
			//更新GrupindexToIndex
			GrupindexToIndex[node] = index
			//更新GrupMclic
			if AllNodeReputation[node].Extra.HhealthyNodeIdentifier == unhealthyNode {
				GrupMclic[key] = append(GrupMclic[key], node)
				//fmt.Println("恶意节点作恶的可能性", AllNodeReputation[node].Extra.MmaliciousnessProbability)
			}
		}
		//fmt.Println("节点个数为", len(value), "恶意节点个数为", len(GrupMclic[key]), "节点为", GrupMclic[key])
		AllNodeReputationMutex.Unlock()
	}

	fmt.Println() // 换行
	fmt.Println("对参与节点发起投票")
	// 遍历每个组 发送数据  i是每个组
	for i, group := range mapWithArrays {
		//再遍历一遍，遍历当前组的每个节点 将总数据发送给每个节点
		for _, element := range group {
			//fmt.Printf("节点%d是归属于%d组的，内部成员有%d\n", element, i, group)
			var outmsg dposJsonStruct
			outmsg.Comm = AskVote
			outmsg.GroupPersons = group
			outmsg.GroupNum = i
			outmsg.GroupNode = element
			outmsg.GrupALLnodeStk = Grupallstk[i]
			outmsg.GrupALLnodeReputaion = GrupallReputaion[i]
			outmsg.ReputationDetail = AllNodeReputation[element]
			outmsg.MclicNodeSGrup = GrupMclic //恶意节点整理
			//outmsg.Data = "123456"
			// 将结构体转换为 JSON 字节流
			jsonBytes, err1 := json.Marshal(outmsg)
			if err1 != nil {
				fmt.Println("转换失败:", err1)
				return
			}
			Outchannels[element] <- string(jsonBytes)
		}
	}
	fmt.Println("等待票的收集")
	fmt.Println() // 换行
	//然后阻塞等待分组完成，然后才会退出释放锁
	select {
	//全部收集齐了
	case <-VoteOverChan:
		fmt.Println("全员票数收集完成，投票协程被唤醒")
		SpecDPoSSelect3()
	//超时提前结束
	case <-time.After(30 * time.Second):
		fmt.Println("超时，投票协程未收到信号结束信号，直接执行下一步操作")
		SpecDPoSSelect3()
	}
}

//我的改进dpos算法的选举过程2 收集选票
func SpecDPoSSelect2(dposAskStruct dposJsonStruct) {
	GrupArrayVoteMutex.Lock()
	defer GrupArrayVoteMutex.Unlock()
	resultArry := dposAskStruct.GroupReplyVote
	groupVotes, groupExists := GrupArrayVote[dposAskStruct.GroupNum]

	if groupExists && len(resultArry) == len(groupVotes) {
		for index, value := range resultArry {
			groupVotes[index] += value
		}
		GroupNodeNum--
	} else {
		fmt.Printf("Mismatch in group array lengths or group number %d does not exist\n", dposAskStruct.GroupNum)
	}

	//投票结束
	if GroupNodeNum <= 0 {
		VoteOverChan <- 1
	}
}

type Element struct {
	Row   int // 行下标
	Col   int // 列下标
	Value int // 值
}

//我的改进dpos算法的选举过程3 统计每组最高的几个
func SpecDPoSSelect3() {
	fmt.Println("综合票数", GrupArrayVote)

	GrupArrayVoteMutex.Lock()
	GroupNodeNum = math.MaxInt64
	result := GrupArrayVote
	GrupArrayVoteMutex.Unlock()
	VoteFianlResult = make(map[int][]int)
	// 遍历每个维度
	for key, values := range result {
		// 定义一个结构体切片来存储值和对应的下标
		type Pair struct {
			Value float64
			Index int
		}

		// 初始化一个 Pair 切片，并为每个值分配对应的下标
		pairs := make([]Pair, len(values))
		for i, v := range values {
			pairs[i] = Pair{v, i}
		}

		// 对 Pair 切片进行排序，以找到最大的 N 个值及其下标
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Value > pairs[j].Value
		})

		// 输出维度 key 中最大的 N 个值的下标
		N := 2 // 指定要找到的最大值的个数
		fmt.Printf("维度 %d 中最大的 %d 个值的下标为: ", key, N)

		for i := 0; i < N && i < len(pairs); i++ {
			fmt.Printf("%d ", pairs[i].Index)
			VoteFianlResult[key] = append(VoteFianlResult[key], GrupArray[key][pairs[i].Index])
		}

		fmt.Println()
	}
	//更新全局变量来标识下一个应该访问谁 全局变量
	fmt.Println()
	fmt.Println("=============================================")
	fmt.Println("最终结果")
	//清空基层委员会成员
	NomalAdminList = nil
	for key1, values1 := range VoteFianlResult {
		fmt.Printf("组号: %d, 入选基层委员会成员节点: %v\n", key1, values1)
		for _, value := range values1 {
			NomalAdminList = append(NomalAdminList, strconv.Itoa(value))
		}
	}
	fmt.Println("中央委员会成员", AdminList)
	if AdminList2 != nil {
		fmt.Println("中央委员会备用成员节点", AdminList2)
	}
	fmt.Println("=============================================")

}

//中央委员会设置 运行完这个函数后 中央委员会已经选择完毕 返回的是能够参与选举的基层群众
/*
  AdminList + AdminList2
  valdNodes 包含了 AdminList + AdminList2 + 基层admin + 普通节点
	               temp 	   temp2

*/

func SpecDPoSAdminSelect(valdNodes []string) []string {
	if len(valdNodes) <= AdminNum {
		fmt.Println("要求节点数至少为：", AdminNum)

		os.Exit(0)
	}
	//第一次选举过程
	if FirstIN {
		//从valdNodes中获取几个作为中央委员会成员
		AdminList = valdNodes[len(valdNodes)-AdminNum:]
		valdNodes = valdNodes[:len(valdNodes)-AdminNum]
		FirstIN = false
		return valdNodes
	} //不是第一次才执行下面

	//从基层挑选AdminNum个进入AdminList
	//将AdminList的成员放回到基层，将上一届基层放到中央
	temp := AdminList
	temp2 := AdminList2

	fmt.Println("len(NomalAdminList)  ", len(NomalAdminList), "  AdminNum", AdminNum)
	//中央为后几个节点
	AdminList = NomalAdminList[len(NomalAdminList)-AdminNum:]
	//中央备选为前几个节点
	AdminList2 = NomalAdminList[:len(NomalAdminList)-AdminNum]
	//能选举的节点

	combinedMap := make(map[string]bool)
	combined := append(append(append(AdminList, AdminList2...), temp...), temp2...)
	// 遍历 combined 切片并添加到 map 中
	for _, v := range combined {
		combinedMap[v] = true
	}

	var userlist []string
	for _, v := range valdNodes {
		if _, ok := combinedMap[v]; ok {

		} else {
			userlist = append(userlist, v)
		}
	}
	userlist = append(append(temp, temp2...), userlist...)
	return userlist
}

//我的模拟各种参数的控制
func SimultaeControl() {
	/*
		1、获得正常节点
		2、向其发送命令
		3、接收返回的命令
		    两个线程接收信号
	*/
	//validNodes := GetValidNodes()

}

//////////////////////////下面是响应函数

//收到节点hi的响应处理函数
func ReplyInitNumDataFunc(nodeReputation dposJsonStruct) {
	//全局存储
	AllNodeReputationMutex.Lock()
	AllNodeReputation[nodeReputation.InitNodeNum] = nodeReputation.ReputationDetail
	AllNodeReputationMutex.Unlock()
}

//节点想知道访问哪个节点去验证
func ReplyAskForValiFunc(conn net.Conn, messageStruct dposJsonStruct) {
	//1.获取请求信息
	GroupNum := messageStruct.GroupNum
	InitNodeNum := messageStruct.InitNodeNum
	Leaderlist := VoteFianlResult[InitNodeNum]
	//2.根据节点信息从本地获得应该请求哪个
	//设置一个要返回的数据信息
	var result dposJsonStruct
	result.Comm = ReplyAskForVali
	//2.1判断现在是否正在选举
	//2.1.1如果正在选举，那么返回一个错误信息
	// if !SelectingMutex.TryLock() {
	// 	result.Comm = LaterPost //下次再请求
	// }
	//2.2判断是否只有一个，且该元素是否是它本身
	if len(Leaderlist) == 1 && Leaderlist[0] == InitNodeNum {
		result.ReplyForvaliNode = int(SelfValid)
	}
	//2.3判断是否为空，这个空就是致命错误
	if len(Leaderlist) == 0 {
		fmt.Println("致命错误,不会出现在一个组里领导者没有的情况")
	}
	//2.4用个全局变量来标识下一个应该访问谁，然后让其+1
	//2.4.1先判断是否超过最大值
	NextGroupNode[GroupNum]++
	if NextGroupNode[GroupNum] >= len(VoteFianlResult[GroupNum])-1 {
		GroupNodeNum = 0
	}

	temp := NextGroupNode[GroupNum]
	if temp == InitNodeNum {
		result.ReplyForvaliNode = int(SelfValid)

	} else {
		//result.ReplyForvaliNode = (int)的LaterPost
		result.ReplyForvaliNode = int(ReplyAskForVali)
		//ip需要重新进行设置  VoteFianlResult[GroupNum][temp]
		addr := "127.0.0.1:10001"
		result.StringData = addr
	}

	//2.5设置返回的数据
	jsonBytes, err1 := json.Marshal(result)
	if err1 != nil {
		fmt.Println("转换失败:", err1)
		return
	}

	//3.返回结果
	connWrite(conn, jsonBytes)
}

///////////////////////////////////////////下面是工具函数

//得到请求的struct结构体形式的数据，如果中间发生错误那么返回nil  server和node有些不同，这里需要知道是哪个initnum以便从池子里面删除
func connGetMsgStruct(conn net.Conn, initnum int) (dposJsonStruct, error) {
	// 读取数据包长度信息
	lenBuffer := make([]byte, 4)
	_, err := io.ReadFull(conn, lenBuffer)
	if err != nil {
		fmt.Println("Error reading:  节点疑似已经断开", err)
		//断开那么就放入到断开池里面
		ExpPoolMutex.Lock()
		DisconnectedPool[initnum] = true
		ExpPoolMutex.Unlock()
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

//请求谱聚类分组 返回分组后的数据
func AskGroupPy(requestData map[string]interface{}) map[int][]int {

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

//分组需要判断当前节点的数量有多少,需要剔除被拉黑节点和失去连接连接的节点 并更新全局变量GroupNodeNum 参与本次选举的人数
func GetValidNodes() []string {

	var validNodes []string
	channelIndexMutex.Lock()
	//然后将能够正常进行选举的过程发给python函数
	ExpPoolMutex.Lock()
	for i := 0; i < NodeIndex; i++ {
		//执行剔除过程，这里需要剃掉不能选举和拉黑节点
		if DisconnectedPool[i] || BlacklistPool[i] {
		} else {
			validNodes = append(validNodes, strconv.Itoa(i))
		}
	}

	ExpPoolMutex.Unlock()
	channelIndexMutex.Unlock()
	return validNodes
}
