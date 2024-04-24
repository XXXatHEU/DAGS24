package main

import (
	"sync"
)

///下面是配置变量

var FirstIN = true //第一次进入

var AdminList []string                   //中央委员会成员
var AdminList2 []string                  //备选中央委员会成员
var NomalAdminList []string              //基层委员会成员的所有成员
var SelectingMutex sync.Mutex            //正在选举过程
var VoteOverChan = make(chan int, 10000) //发送信号则说明投票结束

var GrupArrayVoteMutex sync.Mutex   //保护投票记录和GroupNodeNum
var GroupNodeNum = 0                //参与本轮投票的总人数
var GrupArray = make(map[int][]int) //分组结果 比如[1][9,21,1,34,5]是1组里有[9,21,34,5]这几个节点
var GrupArrayVote map[int][]float64 //投票记录  [1][1,2,3，4]表示9得到的投票值为1，21投票值为2  对比上面
var GrupMclic = make(map[int][]int) //每组的恶意节点标识  淘汰历届被拉黑的节点

var GrupindexToIndex = make(map[int]int) //比如[1]= 10 说明1在它的组里下标是10
var GrupallReputaion = make(map[int]int) //每个组的总信誉值 [1] = 10 说明组内总信誉值为10
var Grupallstk = make(map[int]int)       //每个组的总信誉值 [1] = 10 说明组内总信誉值为10
var VoteFianlResult map[int][]int        //最终获胜者  （应该存每个人对应的组里的获胜者） [1]:{2,3,4} 序号1对应的获胜组的值为2,3,4
//下一组应该访问谁
var NextGroupNode = make(map[int]int)
var NextGroupNodeMutex sync.Mutex

//用于保护Outchannels 和 Inchannels
var channelIndexMutex sync.Mutex            //存储addr和对应的账户和账户特定信息的节点
var channelIndex = 0                        //从0开始递增，表明有多少用户加入，即使用户退出也不会减少
var Outchannels = make([]chan string, 5000) //发送消息 通道
var Inchannels = make([]chan string, 5000)  //收到消息 通道

var overSelectChan = make(chan int, 1) //选举完成
var NodeIndexMutex sync.Mutex
var NodeIndex = 0

var SelectVersion = -1 //版本类型

var AllNodeReputationMutex sync.Mutex                   //保护map
var AllNodeReputation = make(map[int]NodeReputation)    //所有节点的信誉struct
var oldAllNodeReputation = make(map[int]NodeReputation) //上一轮的所有节点的信誉struct

var LeftMalicNodeMutex sync.Mutex         //保护map 剩余恶意节点
var LeftMalicNode = make(map[int]float64) //剩余恶意节点 里面是恶意节点的恶意行为概率
var AllNodeAddrMutex sync.Mutex           //保护map
var AllNodeAddr = make(map[int]string)    //所有节点的地址

var ExpPoolMutex sync.Mutex                        //下面两个池子用一把锁
var DisconnectedPool = make(map[int]bool)          //对方连接断开
var BlacklistPoolAll = make(map[int]bool)          //所有拉黑节点存放
var BlacklistPooltime = make(map[int]int)          //所有拉黑节点存放 里面是时间
var BlacklistPoolTemp = make(map[int]int)          //拉黑池本轮节点
var maliciousBehaviorCountPool = make(map[int]int) //节点恶意行为数量池子

//注意 channelIndexMutex和 ExpPoolMutex 需要按顺序加锁
func init() {
	// 创建一个包含70个channel的切片
	// 初始化每个channel
	for i := range Outchannels {
		Outchannels[i] = make(chan string)
		Inchannels[i] = make(chan string)
	}
}
