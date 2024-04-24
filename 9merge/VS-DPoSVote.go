package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

//投票策略
func VS_votStrategy(data dposJsonStruct) map[int][]int {
	//全部的信誉值
	SelectVersion = data.IntData
	VS_PersonVoteList := data.VS_PersonVoteList
	VS_GrupMclic := data.VS_MclicNodeSGrup //所有的恶意节点
	//myReputatil := data.ReputationDetail
	//var resultArry map[int][]int
	//全局恶意节点map
	//下标对应
	fmt.Println("MyInitNum", MyInitNum, "  data.GroupPersons", data.GroupPersons)

	GrupMclic = data.MclicNodeSGrup
	for index, num := range GrupMclic[data.GroupNum] {
		inMclieGrup[num] = index
	}
	for _, num := range data.GroupPersons {
		inGrup[num] = true
	}
	/*
		如果是恶意节点
		 	作恶就是对诚实节点投反对票			对恶意节点投非常赞同
			诚实就是对诚实节点投赞同或者弃权	对恶意节点投反对
		如果是诚实节点
			随机投票
		（要有一些概率）
	*/

	for nodeNum, _ := range VS_PersonVoteList {
		Probability := data.ReputationDetail.Extra.MmaliciousnessProbability
		temptype := nodetype
		// 设置随机数种子
		rand.Seed(time.Now().UnixNano())
		// 生成随机数
		randomNumber := rand.Float64()
		//如果我是恶意节点 在作恶概率下 有可能这次不选择作恶，那么在这决定是否作恶
		if nodetype == unhealthyNode && randomNumber > Probability { //不作恶
			temptype = healthyNode
		}

		if temptype == unhealthyNode { //如果我是恶意节点
			//对于诚实节点 投反对票
			if _, ok := VS_GrupMclic[nodeNum]; ok {
				VS_PersonVoteList[nodeNum][2] = 1
			} else { //对于恶意节点投赞成票
				VS_PersonVoteList[nodeNum][0] = 1
			}
		} else { //如果我是诚实节点
			rand.Seed(time.Now().UnixNano())
			// 生成0到2范围内的随机数
			randomNumber := rand.Intn(3)
			VS_PersonVoteList[nodeNum][randomNumber] = 1
		}
	}
	fmt.Println(VS_PersonVoteList)
	return VS_PersonVoteList
}

func VS_SendVoteResult(vs_PersonVoteList map[int][]int, getdata dposJsonStruct) {
	Log.Info("让我进行投票,MyInitNum:", MyInitNum)
	var outmsg dposJsonStruct

	var GetValue = make(map[int]float64) //恶意交易

	ValidtxMapMutex.Lock()
	for key, value := range HealthTxMap {
		GetValue[key] = float64(value) / float64(HealthTxNum) * 20
	}
	for key, value := range MalicTxMap {
		GetValue[key] -= float64(value) / float64(HealthTxNum) * 35
	}
	ValidtxMapMutex.Unlock()

	outmsg.Comm = VS_ReplyVote
	outmsg.VS_GroupReplyVote = vs_PersonVoteList
	outmsg.InitNodeNum = MyInitNum
	//还未使用
	outmsg.TokenChanges = GetValue
	// 将结构体转换为 JSON 字节流
	jsonBytes, err1 := json.Marshal(outmsg)
	if err1 != nil {
		fmt.Println("转换失败:", err1)
		return
	}
	connWrite(DNSConn, []byte(jsonBytes))
	Log.Info("让我进行投票完毕,MyInitNum:", MyInitNum)
}
