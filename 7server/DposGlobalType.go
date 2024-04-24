package main

const Flame2fileName string = "./flame3.txt" //存储所有节点类型的数据
var PL_Knode = 11                            //委员会中人的个数
var Old_Knode = 15                           //委员会中人的个数
var K_Knode = 10                             //委员会中人的个数
var PO_Knode = 8                             //委员会中人的个数
var VS_Knode = 12                            //委员会中人的个数
var AdminNum = 10                            //中央委员会成员数据
var recognition_probability = 0.000000
var reconition = 0.01

var SelectInterval = 2 //选举间隔时长 秒

var minVotingNodes = 99 //配置节点个数 如果注册的节点达不到就先暂停选举

var Startup_mode_global = po_dpos_mode

//定义启动共识机制的模式
type Startup_mode int

const (
	voidStartup_mode Startup_mode = iota
	po_dpos_mode
	old_dpos_mode
	vs_dpos_mode
	pl_dpos_mode
)

//定义枚举类型1
type dposCommod int

const (
	AskForVali dposCommod = iota //询问应该让谁进行验证
	Old_AskForVali
	PL_AskForVali
	VS_AskForVali
	K_AskForVali

	ReplyAskForVali
	Old_ReplyAskForVali
	PL_ReplyAskForVali
	VS_ReplyAskForVali
	K_ReplyAskForVali

	RelpyInitNum     //返回初始序号
	ReplyInitNumData //返回初始序号的基本信息
	MyValidAddr      //返回我的地址

	AskVote //让节点进行投票
	PL_AskVote
	VS_AskVote
	Old_AskVote
	K_AskVote

	VS_ReplyVote  //投票结果
	PL_ReplyVote  //投票结果
	ReplyVote     //投票结果
	Old_ReplyVote //投票结果
	K_ReplyVote   //投票结果

	ValidmyData //请求验证我的区块
	Old_ValidmyData
	PL_ValidmyData
	VS_ValidmyData
	K_ValidmyData

	ValidResult     //验证结果
	Old_ValidResult //验证结果
	PL_ValidResult  //验证结果
	VS_ValidResult  //验证结果
	K_ValidResult   //验证结果

	LaterPost  //一会再来请求我 //接下来这四个个都是对数据包里的ReplyAskForvali对应
	BlockSet   //已经被拉黑
	SelfValid  //自己验证就可以了
	OldVersion //版本过时

	NotifyMaliciousNodeComm     //通知恶意节点
	Old_NotifyMaliciousNodeComm //通知恶意节点
	PL_NotifyMaliciousNodeComm  //通知恶意节点
	VS_NotifyMaliciousNodeComm  //通知恶意节点
	K_NotifyMaliciousNodeComm   //通知恶意节点

)

type NodeType int

const (
	void00        NodeType = iota //置空 避免  下面在载入的时候会+1避免0的情况
	unhealthyNode                 //不健康节点
	healthyNode                   //健康节点
)

//定义传输json时的结构体
type dposJsonStruct struct {
	StringData           string         `json:"stringdata"` //专用于string类型的数据返回
	IntData              int            `json:"intData"`    //专用于int类型的数据返回
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

	ReplyForvaliNode     int `json:"replyAskForvali"`    //对应common里面的对 请求验证节点的响应
	Old_ReplyForvaliNode int `json:"oldreplyAskForvali"` //对应common里面的对 请求验证节点的响应
	PL_ReplyForvaliNode  int `json:"plreplyAskForvali"`  //对应common里面的对 请求验证节点的响应
	VS_ReplyForvaliNode  int `json:"vsreplyAskForvali"`  //对应common里面的对 请求验证节点的响应
	K_ReplyForvaliNode   int `json:"KreplyAskForvali"`   //对应common里面的对 请求验证节点的响应

	ValidResultData ValidTxStatus   `json:"validresult"`  //验证结果的响应
	TokenChanges    map[int]float64 `json:"tokenchanges"` //代币变化 int为序号 后面为变化值

	VS_PersonVoteList map[int][]int   `json:"vs_personvotelist"` //vs投票名单 [1][1,2,3,4]表明序号1，赞同为1,弃权2，反对3 至于4暂时定为得分
	VS_MclicNodeSGrup map[int]float64 `json:"vs_groupMclicNode"` //vs恶意节点名单
	VS_GroupReplyVote map[int][]int   `json:"vs_GroupReplyVote"` //对应的命令是ReplyVote 返回投票结果

	PL_PersonVoteList map[int][]int   `json:"pl_personvotelist"` //PL投票名单 [1][1,2,3,4]表明序号1，非常赞同，赞同为1，非常反对，反对
	PL_MclicNodeSGrup map[int]float64 `json:"pl_groupMclicNode"` //PL恶意节点名单
	PL_GroupReplyVote map[int][]int   `json:"pl_GroupReplyVote"` //对应的命令是ReplyVote 返回投票结果

	Old_PersonVoteList map[int][]int   `json:"old_personvotelist"` //PL投票名单 [1][1,2,3,4]表明序号1，非常赞同，赞同为1，非常反对，反对
	Old_MclicNodeSGrup map[int]float64 `json:"old_groupMclicNode"` //PL恶意节点名单
	Old_GroupReplyVote map[int][]int   `json:"old_GroupReplyVote"` //对应的命令是ReplyVote 返回投票结果

	K_PersonVoteList map[int][]int   `json:"k_personvotelist"` //PL投票名单 [1][1,2,3,4]表明序号1，非常赞同，赞同为1，非常反对，反对
	K_MclicNodeSGrup map[int]float64 `json:"k_groupMclicNode"` //PL恶意节点名单
	K_GroupReplyVote map[int][]int   `json:"k_GroupReplyVote"` //对应的命令是ReplyVote 返回投票结果
}

//验证交易结果
type ValidTxStatus int

const (
	vodi0 ValidTxStatus = iota
	ValTxFaild
	ValTxSucces
	ValTxBlock //被拉黑
)
