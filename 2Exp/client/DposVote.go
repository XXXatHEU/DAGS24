package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

//投票策略
func votStrategy(data dposJsonStruct) ([]float64, int) {
	//全部的信誉值
	allrepataion := data.GrupALLnodeReputaion
	allstk := data.GrupALLnodeStk
	mystk := data.ReputationDetail.TC.TC
	SelectVersion = data.IntData
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
				Log.Warn("||||||||||||||||||||||长度不一致问题=========================")
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

func SendVoteResult(votes []float64, replydata dposJsonStruct, votecount int) {
	Log.Info("让我进行投票,MyInitNum:", MyInitNum)
	var outmsg dposJsonStruct

	var GetValue = make(map[int]float64) //恶意交易

	//统计处理的交易 发回去
	ValidtxMapMutex.Lock()
	for key, value := range HealthTxMap {
		if HealthTxNum == 0 {
			break
		}
		GetValue[key] += float64(value) / float64(HealthTxNum) * 20
	}
	for key, value := range MalicTxMap {
		if HealthTxNum == 0 {
			break
		}
		GetValue[key] -= float64(value) / float64(HealthTxNum) * 35
	}
	ValidtxMapMutex.Unlock()

	outmsg.Comm = ReplyVote
	outmsg.GroupReplyVote = votes
	outmsg.InitNodeNum = replydata.InitNodeNum
	outmsg.GroupNum = replydata.GroupNum
	outmsg.GroupNode = replydata.GroupNode
	outmsg.GroupReplyVoteCount = votecount
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
