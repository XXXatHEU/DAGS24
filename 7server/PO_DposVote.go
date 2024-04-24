package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

////////////////////////////////////////下面是dns进行选举过程
//控制选举过程
func SelectControl() {
	//ticker := time.NewTicker(5 * time.Minute)
	ticker := time.NewTicker(time.Duration(SelectInterval) * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C // 每次从 ticker 的通道中读取，等待 5 分钟
		NodeIndexMutex.Lock()
		if NodeIndex < minVotingNodes {
			NodeIndexMutex.Unlock()
			continue
		}
		NodeIndexMutex.Unlock()
		SelectingMutex.Lock() //告知我正在选举，将停止推送应该访问哪个节点(这里退化成需要上一个选举)
		SortAndLaunchNodePoll()
		<-overSelectChan
		SelectingMutex.Unlock()
		fmt.Println("完成锁的释放")
	}
}

/*
	0、获取每个节点所有信息（不加选举锁）
	1、设置分组，将分组放置到全局变量中
	2、通知分组结果
	3、进行投票选举
*/

//整理节点并发起投票
func SortAndLaunchNodePoll() {
	//1、找到能用的节点
	validNodes := GetValidNodes()
	//2、返回能参与选举的节点 并设置好中央委员会节点
	validNodes = DPoSAdminSelect(validNodes)
	GroupNodeNum = len(validNodes)
	fmt.Println("这次参与竞选的人数", GroupNodeNum)
	fmt.Println("参与下层分组成员", validNodes)
	// 使用 strings.Join() 构建带逗号分隔符的字符串
	result := strings.Join(validNodes, ",")
	if len(validNodes) == 0 {
		fmt.Println("无可用在线节点，退出本次投票")
		return
	}
	// 3、请求谱聚类分组，准备要发送的数据
	requestData := map[string]interface{}{
		"rows": result, //删选后的行号
		"row":  "3",    //聚类的数量
	}
	mapWithArrays := AskSpecPy(requestData)
	fmt.Println("谱聚类分组结果", mapWithArrays)
	if mapWithArrays == nil {
		fmt.Println("AskGroupPy出现错误")
		return
	}
	//4、发起投票
	GrupArray = mapWithArrays                                   //赋给全局变量
	GrupArrayVote = make(map[int][]float64, len(mapWithArrays)) //表示投票结果
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
	SelectVersion++
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
			outmsg.IntData = SelectVersion //将这次选举的版本发给他
			outmsg.GrupALLnodeReputaion = GrupallReputaion[i]
			AllNodeReputationMutex.Lock()
			outmsg.ReputationDetail = AllNodeReputation[element]
			AllNodeReputationMutex.Unlock()
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
		UpdateVoteCounts()
	//超时提前结束
	case <-time.After(30 * time.Second):
		fmt.Println("超时，投票协程未收到信号结束信号，直接执行下一步操作")
		UpdateVoteCounts()
	}
}

//我的改进dpos算法的选举过程2 收集选票
func CollectVotes(dposAskStruct dposJsonStruct) {
	Log.Info(dposAskStruct.InitNodeNum, "节点尝试上锁")
	GrupArrayVoteMutex.Lock()
	//更新代币数
	AllNodeReputationMutex.Lock()
	tokens := dposAskStruct.TokenChanges
	for initnum, value := range tokens {
		tempstruct := AllNodeReputation[initnum]
		oldAllNodeReputation[initnum] = tempstruct
		tempstruct.TC.TC += value
		tempstruct.CalcuateReputation()
		AllNodeReputation[initnum] = tempstruct
	}
	AllNodeReputationMutex.Unlock()

	resultArry := dposAskStruct.GroupReplyVote
	groupVotes, groupExists := GrupArrayVote[dposAskStruct.GroupNum]

	if groupExists && len(resultArry) == len(groupVotes) {
		for index, value := range resultArry {
			groupVotes[index] += value
		}
		GroupNodeNum--
	} else {
		//Log.Fatal("收集到了一个不合法的选票")
		fmt.Printf("Mismatch in group array lengths or group number %d does not exist\n", dposAskStruct.GroupNum)
	}
	Log.Info("收集到了一个选票,剩余GroupNodeNum:", GroupNodeNum, "本次选票来自:", dposAskStruct.InitNodeNum)

	//投票结束
	if GroupNodeNum == 0 {
		VoteOverChan <- 1
	}
	GrupArrayVoteMutex.Unlock()
	Log.Info(dposAskStruct.InitNodeNum, "节点释放锁")
}

type Element struct {
	Row   int // 行下标
	Col   int // 列下标
	Value int // 值
}

//我的改进dpos算法的选举过程3 统计每组最高的几个 完成选举
func UpdateVoteCounts() {

	fmt.Println("综合票数", GrupArrayVote)
	fmt.Println("上锁之前")
	GrupArrayVoteMutex.Lock()
	GroupNodeNum = math.MaxInt64
	result := GrupArrayVote
	GrupArrayVoteMutex.Unlock()
	fmt.Println("解锁")

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

		//fmt.Printf("维度 %d 中最大的 %d 个值的下标为: ", key, N)

		for i := 0; i < PO_Knode && i < len(pairs); i++ {
			VoteFianlResult[key] = append(VoteFianlResult[key], GrupArray[key][pairs[i].Index])
		}

		fmt.Println()
	}
	//更新全局变量来标识下一个应该访问谁 全局变量
	fmt.Println()
	fmt.Println("=============================================")

	fmt.Println("选举版本:", SelectVersion, "  选举最终结果")

	/*

		筛选出上轮被淘汰的成员



	*/
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
	fmt.Println()
	fmt.Println("上轮被拉黑节点：")
	for key, value := range BlacklistPoolTemp {
		fmt.Println("     ", key, "节点    存活轮数", value)
	}
	fmt.Println()
	fmt.Println("所有被拉黑节点：")
	for key, value := range BlacklistPooltime {
		AllNodeReputationMutex.Lock()
		reputationDetail := AllNodeReputation[key]
		fmt.Println("     ", key, "节点    存活轮数", value, " 作恶可能性:", reputationDetail.Extra.MmaliciousnessProbability)
		AllNodeReputationMutex.Unlock()
	}
	fmt.Println()

	LeftMalicNodeMutex.Lock()
	if len(LeftMalicNode) <= 0 {
		fmt.Println("无运行恶意节点了")
	} else {
		fmt.Println("剩余运行恶意节点：")
		for key, value := range LeftMalicNode {
			fmt.Println("     ", key, "节点    存活轮数", SelectVersion, " 作恶可能性:", value)

		}
	}
	LeftMalicNodeMutex.Unlock()
	//上一轮
	fmt.Println("上一轮信誉值变化：")
	AllNodeAddrMutex.Lock()
	for nodenum, oldStruct := range oldAllNodeReputation {
		newStruct := AllNodeReputation[nodenum]
		//代币变化
		TcChange := newStruct.TC.TC - oldStruct.TC.TC
		//信誉值变化
		reputationChange := newStruct.Value - oldStruct.Value

		fmt.Println(nodenum, "节点", " 代币变化为 ", TcChange, "  信誉值变化 ", reputationChange, "现在代币数量为 ", newStruct.TC.TC, "  信誉为 ", newStruct.Value)
	}
	//剩余剩余数量变化

	repuationS := make([]float64, 2000)
	for nodenum := 0; nodenum < NodeIndex; nodenum++ {
		AllNodeReputationMutex.Lock()
		nodeStruct := AllNodeReputation[nodenum]
		repuationS[nodenum] = nodeStruct.Value
		AllNodeReputationMutex.Unlock()
	}
	SetExcelRowValue("./zXlxs/Po_Repution.xlsx", SelectVersion, repuationS)

	//代币剩余数量变化
	// 创建一个空切片，用于存储100个值
	rowsVludeS := make([]float64, 2000)
	for nodenum := 0; nodenum < NodeIndex; nodenum++ {
		nodeStruct := AllNodeReputation[nodenum]
		rowsVludeS[nodenum] = nodeStruct.TC.TC
	}
	SetExcelRowValue("./zXlxs/PO_Token_Fluctuation.xlsx", SelectVersion, rowsVludeS)

	healthnodenum := 0
	numtotal := 0
	//本轮进入代理节点占比 Secure_Proxy_Node_Distribution.xlsx
	for _, nodenums := range VoteFianlResult {
		for i := 0; i < len(nodenums); i++ {
			numtotal++
			if AllNodeReputation[nodenums[i]].Extra.HhealthyNodeIdentifier == healthyNode {
				healthnodenum++
			}
		}
	}
	for i := 0; i < len(AdminList); i++ {
		intnum, _ := strconv.Atoi(AdminList[i])
		numtotal++
		if AllNodeReputation[intnum].Extra.HhealthyNodeIdentifier == healthyNode {
			healthnodenum++
		}
	}
	floatSlice := []float64{}
	floatSlice = append(floatSlice, float64(healthnodenum)/float64(numtotal)*100) // 向切片中添加一个元素 3.14
	fmt.Println("numtotal:", numtotal, "  healthnodenum:", healthnodenum, floatSlice[0])
	SetExcelRowValue("./zXlxs/Secure_Proxy_Node_Distribution_PO.xlsx", SelectVersion, floatSlice)

	//清零
	oldAllNodeReputation = make(map[int]NodeReputation)

	AllNodeAddrMutex.Unlock()

	BlacklistPoolTemp = make(map[int]int)
	overSelectChan <- 1
	fmt.Println("=============================================")
}

//中央委员会设置 运行完这个函数后 中央委员会已经选择完毕 返回的是能够参与选举的基层群众
/*
  AdminList + AdminList2
  valdNodes 包含了 AdminList + AdminList2 + 基层admin + 普通节点
	               temp 	   temp2

*/

//获得中央节点
func DPoSAdminSelect(valdNodes []string) []string {
	if len(valdNodes) <= AdminNum+5 {
		fmt.Println("要求节点数至少为AdminNum + 5 ：", AdminNum+5, "现在有", len(valdNodes))

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
	if len(NomalAdminList) < AdminNum {
		AdminList = NomalAdminList
	} else {
		AdminList = NomalAdminList[len(NomalAdminList)-AdminNum:]
		AdminList2 = NomalAdminList[:len(NomalAdminList)-AdminNum]
	}

	combinedMap := make(map[string]bool) //不是平民的节点 上一届
	combined := append(append(append(AdminList, AdminList2...), temp...), temp2...)
	// 遍历 combined 切片并添加到 map 中
	for _, v := range combined {
		combinedMap[v] = true
	}

	var userlist []string //存储能参与选举的节点（去掉这一届的中央节点）
	for _, v := range valdNodes {
		if _, ok := combinedMap[v]; ok {

		} else {
			userlist = append(userlist, v)
		}
	}
	userlist = append(append(temp, temp2...), userlist...)

	//再来一遍，删除拉黑和断开连接的节点
	channelIndexMutex.Lock()
	ExpPoolMutex.Lock()
	var filteredUserlist []string
	for _, user := range userlist {
		i, _ := strconv.Atoi(user)
		if !DisconnectedPool[i] && !BlacklistPoolAll[i] {
			filteredUserlist = append(filteredUserlist, user)
		}
	}
	// 更新 userlist
	userlist = filteredUserlist
	ExpPoolMutex.Unlock()
	channelIndexMutex.Unlock()
	return userlist
}

//////////////////////////下面是响应函数

//收到节点hi的响应处理函数
func ReplyInitNumDataFunc(nodeReputation dposJsonStruct) {
	//全局存储
	AllNodeReputationMutex.Lock()
	if nodeReputation.ReputationDetail.Extra.HhealthyNodeIdentifier == unhealthyNode {
		LeftMalicNode[nodeReputation.InitNodeNum] = nodeReputation.ReputationDetail.Extra.MmaliciousnessProbability
	}
	AllNodeReputation[nodeReputation.InitNodeNum] = nodeReputation.ReputationDetail
	AllNodeReputationMutex.Unlock()
}

//分组需要判断当前节点的数量有多少,需要剔除被拉黑节点和失去连接连接的节点 并更新全局变量GroupNodeNum 参与本次选举的人数
func GetValidNodes() []string {

	var validNodes []string
	channelIndexMutex.Lock()
	//然后将能够正常进行选举的过程发给python函数
	ExpPoolMutex.Lock()
	for i := 0; i < NodeIndex; i++ {
		//执行剔除过程，这里需要剃掉不能选举和拉黑节点
		if DisconnectedPool[i] || BlacklistPoolAll[i] {
		} else {
			validNodes = append(validNodes, strconv.Itoa(i))
		}
	}

	ExpPoolMutex.Unlock()
	channelIndexMutex.Unlock()
	return validNodes
}
