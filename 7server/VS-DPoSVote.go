package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

var vs_GroupNodeNum = 0                        //参与本轮投票的总人数
var VS_GrupArrayVoteMutex sync.Mutex           //保护投票记录和GroupNodeNum
var VS_GrupArrayVote = make(map[int][]int)     //投票得分
var VS_GrupMclic = make(map[int]float64)       //恶意节点标识 [1]0.1  后者表示作恶概率
var VS_VoteOverChan = make(chan int, 10000)    //发送信号则说明投票结束
var VS_VoteFianlResult = make(map[int]float64) //最终获胜者
var VS_overSelectChan = make(chan int, 1)      //选举完成
var VS_SelectingMutex sync.Mutex               //正在选举过程
func VS_SelectControl() {
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
		VS_SelectingMutex.Lock() //告知我正在选举，将停止推送应该访问哪个节点(这里退化成需要上一个选举)
		VS_SortAndLaunchNodePoll()
		<-VS_overSelectChan
		VS_SelectingMutex.Unlock()
		fmt.Println("完成锁的释放")
	}
}

//1、整理节点并发起投票
func VS_SortAndLaunchNodePoll() {
	//1、找到能用的节点
	validNodes := GetValidNodes()
	vs_GroupNodeNum = len(validNodes)
	fmt.Println("这次参与竞选的人数", vs_GroupNodeNum)
	fmt.Println("参与下层分组成员", validNodes)

	VS_GrupArrayVote = make(map[int][]int)
	VS_GrupArrayVote_temp := make(map[int][]int)
	for _, nodenum := range validNodes {
		// 为新的键创建一个切片，长度与原始切片相同，但所有元素为0 用来表示投票结果
		votes := make([]int, 4) //3个选项 第四个是最后得分
		numint, _ := strconv.Atoi(nodenum)
		VS_GrupArrayVote_temp[int(numint)] = votes
		//找出恶意节点来

		if AllNodeReputation[numint].Extra.HhealthyNodeIdentifier == unhealthyNode {
			VS_GrupMclic[numint] = AllNodeReputation[numint].Extra.MmaliciousnessProbability
		}
	}
	VS_GrupArrayVote = VS_GrupArrayVote_temp
	fmt.Println() // 换行
	fmt.Println("对参与节点发起投票")
	SelectVersion++
	//遍历当前组的每个节点 将总数据发送给每个节点
	for _, SomenodeNum := range validNodes {
		var outmsg dposJsonStruct
		outmsg.Comm = VS_AskVote
		outmsg.VS_PersonVoteList = VS_GrupArrayVote_temp
		outmsg.IntData = SelectVersion //将这次选举的版本发给他
		SomenodeNum, _ := strconv.Atoi(SomenodeNum)
		AllNodeReputationMutex.Lock()
		outmsg.ReputationDetail = AllNodeReputation[SomenodeNum]
		AllNodeReputationMutex.Unlock()
		outmsg.VS_MclicNodeSGrup = VS_GrupMclic //恶意节点整理
		jsonBytes := StructToJson(outmsg)
		Log.Info("SomenodeNum", SomenodeNum)
		Outchannels[SomenodeNum] <- string(jsonBytes)
	}

	fmt.Println("等待票的收集")
	fmt.Println() // 换行
	//然后阻塞等待分组完成，然后才会退出释放锁
	select {
	//全部收集齐了
	case <-VS_VoteOverChan:
		fmt.Println("全员票数收集完成，投票协程被唤醒")
		VS_UpdateVoteCounts()
	//超时提前结束
	case <-time.After(30 * time.Second):
		fmt.Println("超时，投票协程未收到信号结束信号，直接执行下一步操作")
		VS_UpdateVoteCounts()
	}
}

//2、收集选票
func VS_CollectVotes(dposAskStruct dposJsonStruct) {
	Log.Info(dposAskStruct.InitNodeNum, "节点尝试上锁")
	VS_GrupArrayVoteMutex.Lock()
	//上一轮  更新代币数
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

	//统计选票
	resultArry := dposAskStruct.VS_GroupReplyVote
	for nodenum, values := range resultArry {
		for index, val := range values {
			VS_GrupArrayVote[nodenum][index] += val
		}
	}

	vs_GroupNodeNum--
	fmt.Println("vs_GroupNodeNum", vs_GroupNodeNum)
	//投票结束
	if vs_GroupNodeNum == 0 {
		VS_VoteOverChan <- 1
	}
	VS_GrupArrayVoteMutex.Unlock()
	Log.Info(dposAskStruct.InitNodeNum, "节点释放锁")
}

// 3、统计每组最高的几个 完成选举
func VS_UpdateVoteCounts() {

	fmt.Println("综合票数", VS_GrupArrayVote)
	fmt.Println("上锁之前")
	VS_GrupArrayVoteMutex.Lock()
	vs_GroupNodeNum = math.MaxInt64
	VS_GrupArrayVoteMutex.Unlock()
	fmt.Println("解锁")

	var voteScores = make(map[int]float64)
	for key, values := range VS_GrupArrayVote {
		favor := values[0]
		abstention := values[1]
		against := values[2]
		voteScores[key] = Vs_dpos(favor, abstention, against)
	}

	fmt.Println()
	fmt.Println("=============================================")
	fmt.Println("选举版本:", SelectVersion, "  选举最终结果")
	VS_VoteFianlResult = make(map[int]float64)
	//遍历打印并放到最终获胜里面
	keyList, ScoreList := topKMax(voteScores, VS_Knode) //前面是键 后面是得分
	for ListIndex, key := range keyList {
		VS_VoteFianlResult[key] = ScoreList[ListIndex]
		fmt.Println("	节点", key, "对应的模糊值得分", ScoreList[ListIndex])
	}

	//上一轮
	fmt.Println("上一轮代币变化：")
	AllNodeAddrMutex.Lock()
	for nodenum, oldStruct := range oldAllNodeReputation {
		newStruct := AllNodeReputation[nodenum]
		//代币变化
		TcChange := newStruct.TC.TC - oldStruct.TC.TC
		fmt.Println(nodenum, "节点", " 代币变化为 ", TcChange, " 现在代币数量为 ", newStruct.TC.TC)
	}
	//代币剩余数量变化
	// 创建一个空切片，用于存储100个值
	rowsVludeS := make([]float64, 2000)
	for nodenum := 0; nodenum < NodeIndex; nodenum++ {
		nodeStruct := AllNodeReputation[nodenum]
		rowsVludeS[nodenum] = nodeStruct.TC.TC
	}
	SetExcelRowValue("./zXlxs/VS_Token_Fluctuation.xlsx", SelectVersion, rowsVludeS)

	healthnodenum := 0
	numtotal := 0
	//本轮进入代理节点占比 Secure_Proxy_Node_Distribution.xlsx
	for nodenum, _ := range VS_VoteFianlResult {
		numtotal++
		if AllNodeReputation[nodenum].Extra.HhealthyNodeIdentifier == healthyNode {
			healthnodenum++
		}
		fmt.Println(AllNodeReputation[nodenum].Extra.HhealthyNodeIdentifier, "  ", nodenum)
	}
	floatSlice := []float64{}
	floatSlice = append(floatSlice, float64(healthnodenum)/float64(numtotal)*100) // 向切片中添加一个元素 3.14
	fmt.Println("numtotal:", numtotal, "  healthnodenum:", healthnodenum, floatSlice[0])
	SetExcelRowValue("./zXlxs/Secure_Proxy_Node_Distribution_VS.xlsx", SelectVersion, floatSlice)

	//清零
	oldAllNodeReputation = make(map[int]NodeReputation)

	AllNodeAddrMutex.Unlock()

	VS_overSelectChan <- 1
	fmt.Println("=============================================")
}

//没有拉黑行为 只看断开行为就可以
func VS_GetValidNodes() []string {
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

func VSt_A(favor int, total int) float64 {
	return float64(favor) / float64(total)
}

func VSf_A(against int, total int) float64 {
	return float64(against) / float64(total)
}

func Vs_dpos(favor int, abstention int, against int) float64 {
	lambda := 1
	ta := VSt_A(favor, favor+abstention+against)
	fa := VSf_A(against, favor+abstention+against)
	return ta + 0.5*(1+(ta-fa)/(float64(ta)+float64(fa)+2*float64(lambda)))*(1-ta-fa)
}

// 获取 map 中前 k 个最大的值及其对应的键
func topKMax(scores map[int]float64, k int) ([]int, []float64) {
	type kv struct {
		Key   int
		Value float64
	}

	var ss []kv
	for k, v := range scores {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	var topKValues []float64
	var topKKeys []int
	for i := 0; i < k && i < len(ss); i++ {
		topKValues = append(topKValues, ss[i].Value)
		topKKeys = append(topKKeys, ss[i].Key)
	}

	return topKKeys, topKValues
}
