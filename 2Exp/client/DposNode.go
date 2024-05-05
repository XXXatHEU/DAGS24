package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

//共有结构体上述
var AskForChan = make(chan dposJsonStruct)

var myGroupNum = 0
var GrupMclic = make(map[int][]int) //每组恶意节点的集合
var inMclieGrup = make(map[int]int) //自己组内恶意节点的集合 key是节点数  value是在组内的下标
var inGrup = make(map[int]bool)     //自己组内所有节点的集合
var nodetype NodeType               //节点类型
var nodeMalic float64               //节点作恶的可能性

var DNSConn net.Conn

var SelectVersion = -1   //当前版本的类型
var MyInitNum int = -1   //我是第几个节点
var Netdelayms_n float64 //我的网络延迟

func init() {
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

	//获得延迟
	Netdelayms_n, _ = strconv.ParseFloat(fields[2], 64)

	return lineText, node
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

///////////////////////////工具类

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
