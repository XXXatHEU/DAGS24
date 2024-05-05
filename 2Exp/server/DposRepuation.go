package main

import "fmt"

const MaxmaliciousBehaviorCount = 3

//对作恶行为的处理  下面是作恶次数是否已经超过了规定值，如果超过了规定值就拉黑并且从领导者节点中删除
func HandleMaliciousNode(dposAskStruct dposJsonStruct) {
	nodeNum := dposAskStruct.InitNodeNum
	gruopNum := dposAskStruct.GroupNum
	ExpPoolMutex.Lock()
	maliciousBehaviorCountPool[nodeNum]++
	if maliciousBehaviorCountPool[nodeNum] >= MaxmaliciousBehaviorCount {
		BlacklistPoolAll[nodeNum] = true
		BlacklistPooltime[nodeNum] = SelectVersion
		BlacklistPoolTemp[nodeNum] = SelectVersion
	}
	ExpPoolMutex.Unlock()
	Log.Info(nodeNum, "节点因为作恶被拉黑")
	NextGroupNodeMutex.Lock()
	//从领导者节点中删除这个元素
	// 创建一个空的切片，用于存储不为num的元素
	var newSlice []int
	// 遍历原始切片，将不为1的元素添加到新切片中
	for _, v := range VoteFianlResult[gruopNum] {
		if v != nodeNum {
			newSlice = append(newSlice, v)
		}
	}
	VoteFianlResult[gruopNum] = newSlice
	fmt.Println("删除后", nodeNum, "VoteFianlResult", VoteFianlResult)
	NextGroupNodeMutex.Unlock()

	LeftMalicNodeMutex.Lock()
	delete(LeftMalicNode, nodeNum)
	LeftMalicNodeMutex.Unlock()

}

//对作恶行为的处理  下面是作恶次数是否已经超过了规定值，如果超过了规定值就拉黑并且从领导者节点中删除
func Bashline_HandleMaliciousNode(dposAskStruct dposJsonStruct) {
	return //没有拉黑的选项
	nodeNum := dposAskStruct.InitNodeNum
	ExpPoolMutex.Lock()
	defer ExpPoolMutex.Unlock()
	maliciousBehaviorCountPool[nodeNum]++
	if maliciousBehaviorCountPool[nodeNum] >= MaxmaliciousBehaviorCount {
		BlacklistPoolAll[nodeNum] = true
		BlacklistPooltime[nodeNum] = SelectVersion
		BlacklistPoolTemp[nodeNum] = SelectVersion
	} else {
		return
	}

	Log.Info(nodeNum, "节点因为作恶被拉黑  --Old_HandleMaliciousNode")

	delete(Old_VoteFianlResult, nodeNum)

	LeftMalicNodeMutex.Lock()
	delete(LeftMalicNode, nodeNum)
	LeftMalicNodeMutex.Unlock()

}
