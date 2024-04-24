package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

func SetNodeAddr(conn net.Conn, messageStruct dposJsonStruct) {
	NodeAddr := messageStruct.StringData
	InitNodeNum := messageStruct.InitNodeNum
	AllNodeAddrMutex.Lock()
	AllNodeAddr[InitNodeNum] = NodeAddr
	AllNodeAddrMutex.Unlock()
	Log.Info("设置", InitNodeNum, "节点地址成功:", NodeAddr)
}

//节点想知道访问哪个节点去验证
func ReplyAskForValiFunc(conn net.Conn, messageStruct dposJsonStruct) error {
	//1.获取请求信息
	GroupNum := messageStruct.GroupNum
	InitNodeNum := messageStruct.InitNodeNum
	Leaderlist := VoteFianlResult[GroupNum]

	//2.根据节点信息从本地获得应该请求哪个
	//设置一个要返回的数据信息
	var result dposJsonStruct
	result.Comm = ReplyAskForVali
	AllNodeReputationMutex.Lock()
	result.ReputationDetail = AllNodeReputation[InitNodeNum]
	AllNodeReputationMutex.Unlock()

	//2.1判断现在是否正在选举
	//2.1.1如果正在选举，那么返回一个错误信息
	//2.2判断是否只有一个，且该元素是否是它本身
	if len(Leaderlist) == 1 && Leaderlist[0] == InitNodeNum {
		result.ReplyForvaliNode = int(SelfValid)
		resultjson := StructToJson(result)
		connWrite(conn, resultjson)
		return nil
	}

	//2.2判断是否被拉黑
	ExpPoolMutex.Lock()
	_, ok := BlacklistPoolAll[InitNodeNum]
	ExpPoolMutex.Unlock()
	if ok {
		result.ReplyForvaliNode = int(BlockSet)
		resultjson := StructToJson(result)
		connWrite(conn, resultjson)
		return fmt.Errorf("节点已经被拉黑了")
	}

	//2.3判断是否为空，这个空可能是有延迟 或者是 组内已经全被拉黑了 那么随机给它一个验证南街店
	if len(Leaderlist) == 0 {
		fmt.Println("InitNodeNum  ", InitNodeNum, "  GroupNum:", GroupNum, "致命错误,不会出现在一个组里领导者没有的情况", "VoteFianlResult", VoteFianlResult,
			"    GroupNum", GroupNum, "   VoteFianlResult[GroupNum]", VoteFianlResult[GroupNum])
		result.ReplyForvaliNode = int(LaterPost)
		jsonBytes := StructToJson(result)
		connWrite(conn, jsonBytes)
		fmt.Println("发送这个信息完毕")
		return nil
	}
	//2.4用个全局变量来标识下一个应该访问谁，然后让其+1
	//2.4.1先判断是否超过最大值
	NextGroupNodeMutex.Lock()
	NextGroupNode[GroupNum]++
	if NextGroupNode[GroupNum] >= len(VoteFianlResult[GroupNum])-1 {
		NextGroupNode[GroupNum] = 0
	}
	NextIndex := NextGroupNode[GroupNum]                   //下标
	NextIndexValue := VoteFianlResult[GroupNum][NextIndex] //下标对应的值
	NextGroupNodeMutex.Unlock()

	//如果下一个节点是自己 那么就返回自己
	if NextIndexValue == InitNodeNum {
		result.ReplyForvaliNode = int(SelfValid)
	} else {
		//result.ReplyForvaliNode = (int)的LaterPost
		result.ReplyForvaliNode = int(ReplyAskForVali)
		//ip需要重新进行设置  VoteFianlResult[GroupNum][temp]
		AllNodeAddrMutex.Lock()
		addr := AllNodeAddr[NextIndexValue]
		AllNodeAddrMutex.Unlock()
		Log.Info("得到", NextIndexValue, "节点地址成功:", addr)
		result.StringData = addr
	}

	//2.5设置返回的数据
	jsonBytes := StructToJson(result)
	//3.返回结果
	connWrite(conn, jsonBytes)
	return nil
}

//节点想知道访问哪个节点去验证
func Old_ReplyAskForValiFunc(conn net.Conn, messageStruct dposJsonStruct) error {
	//1.获取请求信息
	InitNodeNum := messageStruct.InitNodeNum
	Leaderlist := Old_VoteFianlResult

	//2.根据节点信息从本地获得应该请求哪个
	//设置一个要返回的数据信息
	var result dposJsonStruct
	result.Comm = Old_ReplyAskForVali
	AllNodeReputationMutex.Lock()
	result.ReputationDetail = AllNodeReputation[InitNodeNum]
	AllNodeReputationMutex.Unlock()

	// //2.2判断是否被拉黑  不判断自己是否被拉黑
	// ExpPoolMutex.Lock()
	// _, ok := BlacklistPoolAll[InitNodeNum]
	// ExpPoolMutex.Unlock()
	// if ok {
	// 	result.Old_ReplyForvaliNode = int(BlockSet)
	// 	resultjson := StructToJson(result)
	// 	connWrite(conn, resultjson)
	// 	return fmt.Errorf("节点已经被拉黑了")
	// }

	//2.2判断是否只有一个，且该元素是否是它本身
	if _, ok := Leaderlist[InitNodeNum]; ok {
		if len(Leaderlist) == 1 {
			result.Old_ReplyForvaliNode = int(SelfValid)
			resultjson := StructToJson(result)
			connWrite(conn, resultjson)
			return nil
		}
	}

	//2.3判断是否为空，这个空可能是有延迟 或者是 组内已经全被拉黑了 那么随机给它一个验证南街店
	if len(Leaderlist) == 0 {
		fmt.Println("领导胜利者节点数组为空")
		result.Old_ReplyForvaliNode = int(LaterPost)
		jsonBytes := StructToJson(result)
		connWrite(conn, jsonBytes)
		fmt.Println("发送这个信息完毕")
		return nil
	}

	//全部获取并返回  不管什么行为

	//从胜利者节点中挑选一个节点 返回
	rand.Seed(time.Now().UnixNano())
	// 从map中获取所有的key并存入slice
	keys := make([]int, 0, len(Leaderlist))
	for k := range Leaderlist {
		keys = append(keys, k)
	}
	randomNum := keys[rand.Intn(len(keys))]

	AllNodeAddrMutex.Lock()
	addr := AllNodeAddr[randomNum]
	AllNodeAddrMutex.Unlock()
	Log.Info("Old中", randomNum, "节点地址成功:", addr)
	result.StringData = addr
	result.Old_ReplyForvaliNode = int(Old_ReplyAskForVali)
	// //如果下一个节点是自己 那么就返回自己
	// if randomNum == InitNodeNum {
	// 	result.Old_ReplyForvaliNode = int(SelfValid)
	// } else {
	// 	//result.ReplyForvaliNode = (int)的LaterPost
	// 	result.Old_ReplyForvaliNode = int(Old_ReplyAskForVali)
	// 	//ip需要重新进行设置  VoteFianlResult[GroupNum][temp]
	// 	AllNodeAddrMutex.Lock()
	// 	addr := AllNodeAddr[randomNum]
	// 	AllNodeAddrMutex.Unlock()
	// 	Log.Info("Old中", randomNum, "节点地址成功:", addr)
	// 	result.StringData = addr
	// }

	//2.5设置返回的数据
	jsonBytes := StructToJson(result)
	//3.返回结果
	connWrite(conn, jsonBytes)
	return nil
}

//节点想知道访问哪个节点去验证
func PL_ReplyAskForValiFunc(conn net.Conn, messageStruct dposJsonStruct) error {
	//1.获取请求信息
	InitNodeNum := messageStruct.InitNodeNum
	Leaderlist := PL_VoteFianlResult

	//2.根据节点信息从本地获得应该请求哪个
	//设置一个要返回的数据信息
	var result dposJsonStruct
	result.Comm = PL_ReplyAskForVali
	AllNodeReputationMutex.Lock()
	result.ReputationDetail = AllNodeReputation[InitNodeNum]
	AllNodeReputationMutex.Unlock()

	// //2.2判断是否被拉黑  不判断自己是否被拉黑
	// ExpPoolMutex.Lock()
	// _, ok := BlacklistPoolAll[InitNodeNum]
	// ExpPoolMutex.Unlock()
	// if ok {
	// 	result.PL_ReplyForvaliNode = int(BlockSet)
	// 	resultjson := StructToJson(result)
	// 	connWrite(conn, resultjson)
	// 	return fmt.Errorf("节点已经被拉黑了")
	// }

	//2.2判断是否只有一个，且该元素是否是它本身
	if _, ok := Leaderlist[InitNodeNum]; ok {
		if len(Leaderlist) == 1 {
			result.PL_ReplyForvaliNode = int(SelfValid)
			resultjson := StructToJson(result)
			connWrite(conn, resultjson)
			return nil
		}
	}

	//2.3判断是否为空，这个空可能是有延迟 或者是 组内已经全被拉黑了 那么随机给它一个验证南街店
	if len(Leaderlist) == 0 {
		fmt.Println("领导胜利者节点数组为空")
		result.PL_ReplyForvaliNode = int(LaterPost)
		jsonBytes := StructToJson(result)
		connWrite(conn, jsonBytes)
		fmt.Println("发送这个信息完毕")
		return nil
	}

	//全部获取并返回  不管什么行为

	//从胜利者节点中挑选一个节点 返回
	rand.Seed(time.Now().UnixNano())
	// 从map中获取所有的key并存入slice
	keys := make([]int, 0, len(Leaderlist))
	for k := range Leaderlist {
		keys = append(keys, k)
	}
	randomNum := keys[rand.Intn(len(keys))]

	AllNodeAddrMutex.Lock()
	addr := AllNodeAddr[randomNum]
	AllNodeAddrMutex.Unlock()
	Log.Info("PL中", randomNum, "节点地址成功:", addr)
	result.StringData = addr
	result.PL_ReplyForvaliNode = int(PL_ReplyAskForVali)
	// //如果下一个节点是自己 那么就返回自己
	// if randomNum == InitNodeNum {
	// 	result.PL_ReplyForvaliNode = int(SelfValid)
	// } else {
	// 	//result.ReplyForvaliNode = (int)的LaterPost
	// 	result.PL_ReplyForvaliNode = int(PL_ReplyAskForVali)
	// 	//ip需要重新进行设置  VoteFianlResult[GroupNum][temp]
	// 	AllNodeAddrMutex.Lock()
	// 	addr := AllNodeAddr[randomNum]
	// 	AllNodeAddrMutex.Unlock()
	// 	Log.Info("PL中", randomNum, "节点地址成功:", addr)
	// 	result.StringData = addr
	// }

	//2.5设置返回的数据
	jsonBytes := StructToJson(result)
	//3.返回结果
	connWrite(conn, jsonBytes)
	return nil
}

//节点想知道访问哪个节点去验证
func VS_ReplyAskForValiFunc(conn net.Conn, messageStruct dposJsonStruct) error {
	//1.获取请求信息
	InitNodeNum := messageStruct.InitNodeNum
	Leaderlist := VS_VoteFianlResult

	//2.根据节点信息从本地获得应该请求哪个
	//设置一个要返回的数据信息
	var result dposJsonStruct
	result.Comm = VS_ReplyAskForVali
	AllNodeReputationMutex.Lock()
	result.ReputationDetail = AllNodeReputation[InitNodeNum]
	AllNodeReputationMutex.Unlock()

	// //2.2判断是否被拉黑  不判断自己是否被拉黑
	// ExpPoolMutex.Lock()
	// _, ok := BlacklistPoolAll[InitNodeNum]
	// ExpPoolMutex.Unlock()
	// if ok {
	// 	result.PL_ReplyForvaliNode = int(BlockSet)
	// 	resultjson := StructToJson(result)
	// 	connWrite(conn, resultjson)
	// 	return fmt.Errorf("节点已经被拉黑了")
	// }

	//2.2判断是否只有一个，且该元素是否是它本身
	if _, ok := Leaderlist[InitNodeNum]; ok {
		if len(Leaderlist) == 1 {
			result.VS_ReplyForvaliNode = int(SelfValid)
			resultjson := StructToJson(result)
			connWrite(conn, resultjson)
			return nil
		}
	}

	//2.3判断是否为空，这个空可能是有延迟 或者是 组内已经全被拉黑了 那么随机给它一个验证南街店
	if len(Leaderlist) == 0 {
		fmt.Println("领导胜利者节点数组为空")
		result.VS_ReplyForvaliNode = int(LaterPost)
		jsonBytes := StructToJson(result)
		connWrite(conn, jsonBytes)
		fmt.Println("发送这个信息完毕")
		return nil
	}

	//全部获取并返回  不管什么行为

	//从胜利者节点中挑选一个节点 返回
	rand.Seed(time.Now().UnixNano())
	// 从map中获取所有的key并存入slice
	keys := make([]int, 0, len(Leaderlist))
	for k := range Leaderlist {
		keys = append(keys, k)
	}
	randomNum := keys[rand.Intn(len(keys))]

	AllNodeAddrMutex.Lock()
	addr := AllNodeAddr[randomNum]
	AllNodeAddrMutex.Unlock()
	Log.Info("VS中", randomNum, "节点地址成功:", addr)
	result.StringData = addr
	result.VS_ReplyForvaliNode = int(VS_ReplyAskForVali)
	// //如果下一个节点是自己 那么就返回自己
	// if randomNum == InitNodeNum {
	// 	result.PL_ReplyForvaliNode = int(SelfValid)
	// } else {
	// 	//result.ReplyForvaliNode = (int)的LaterPost
	// 	result.PL_ReplyForvaliNode = int(PL_ReplyAskForVali)
	// 	//ip需要重新进行设置  VoteFianlResult[GroupNum][temp]
	// 	AllNodeAddrMutex.Lock()
	// 	addr := AllNodeAddr[randomNum]
	// 	AllNodeAddrMutex.Unlock()
	// 	Log.Info("PL中", randomNum, "节点地址成功:", addr)
	// 	result.StringData = addr
	// }

	//2.5设置返回的数据
	jsonBytes := StructToJson(result)
	//3.返回结果
	connWrite(conn, jsonBytes)
	return nil
}
