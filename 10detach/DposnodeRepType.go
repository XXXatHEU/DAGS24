package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
)

// 节点的综合信誉评价223
type NodeReputation struct {
	Pf    Performance          // 性能指标
	Hp    ConsensusPerformance // 共识参与表现
	Sr    HistoricalSelection  // 被选入委员会的次数占比
	Gu    NodeRecognition      // 暂时为0，可以表示其他未来可能的指标
	TC    TokenHolding         // 节点拥有的代币占比
	Value float64
	Extra ExtraDetail //额外的信息
}

const (
	alpha1 = 1 // Pf的权重
	alpha2 = 1 // Hp的权重
	alpha3 = 1 // Sr的权重
	alpha4 = 1 // Gu的权重
	alpha5 = 6 // TC的权重
)

type ExtraDetail struct {
	Llongitude                float64  //经度
	Llatitude                 float64  //纬度
	HhealthyNodeIdentifier    NodeType //是否正常节点  0是恶意节点
	MmaliciousnessProbability float64  //恶意节点作恶的可能性
}

// 节点性能指标
type Performance struct {
	Delay float64 //延迟
	De    float64 // 延迟得分
	H     float64
	M     float64 //内存大小

	Iio   float64 //io速度
	Hw    float64 // 硬件性能得分
	Ttk   uint    //在线时长
	Ttall uint    //整个区块链网络运行时长
	Od    float64 // 在线时长得分
	Value float64
}

//共识参与表现的权重
const (
	beta1 = 0.4 // De_k的权重
	beta2 = 0.3 // Hw_k的权重
	beta3 = 0.3 // Od_k的权重
)

// 共识参与表现
type ConsensusPerformance struct {
	Vvote    uint    //实际投票票数
	Vvoteall uint    //有机会参与的总投票数
	Ha       float64 // 共识参与程度
	Hht      uint    //正常发送交易数量
	Hhtall   uint    //节点发送的所有交易数
	Ht       float64 // 发送交易有效指标
	Hhb      uint    //被验证接受的区块数
	Hhball   uint    //节点打包的所有区块数
	Hc       float64 // 历史出块占比
	Hhe      uint    //发送的恶意交易和区块数量
	Hheall   uint    //所有交易和打包的所有区块数量
	Hd       float64 // 恶意交易和区块数量
	Value    float64
}

//共识参与表现的权重
const (
	r1 = 0.1 // Ha_k的权重
	r2 = 0.4 // Ht_k的权重
	r3 = 0.2 // Hc_k的权重
	r4 = 0.3 // Hd_k的惩罚权重
)

// 历史选举情况 sr
type HistoricalSelection struct {
	Ssro  uint //选入下层委员会占比
	Ssr1  uint //选入中央委员会占比
	Ssra  uint //参与的轮数
	Value float64
}

//非主链区块认可程度 Gu  直接都默认1
type NodeRecognition struct {
	Ddirect   float64 //直接认可
	Iindirect float64 //间接认可
	Value     float64
}

//代币总数tc
type TokenHolding struct {
	TC    float64 // 节点拥有的代币占比
	Tcall float64
	Value float64
}

func (pf *Performance) CalcuPf() {
	// 计算De_k
	pf.De = math.Exp(-pf.Delay)
	// 计算Hw_k：首先计算原始硬件得分Hk
	Hk := pf.H * 0.5 * pf.M * 0.5 * pf.Iio * 0.01
	// 使用双曲正切函数归一化硬件得分到0和1之间
	pf.Hw = 0.5 * ((math.Exp(Hk)-math.Exp(-Hk))/(math.Exp(Hk)+math.Exp(-Hk)) + 1)
	if math.IsNaN(pf.Hw) {
		pf.Hw = 0
	}

	// 计算Od_k
	if pf.Ttall > 0 {
		t := float64(pf.Ttk) / float64(pf.Ttall)
		pf.Od = 1 / (1 + math.Exp(-t))
	} else {
		pf.Od = 0
	}

	// 计算Pf_i
	pf.Value = beta1*pf.De + beta2*pf.Hw + beta3*pf.Od

}

func (hp *ConsensusPerformance) CalcuHp() {
	// 计算Ha, Ht, Hc, 和 Hd
	if hp.Vvoteall > 0 {
		hp.Ha = float64(hp.Vvote) / float64(hp.Vvoteall)
	} else {
		hp.Ha = 0
	}
	if hp.Hhtall > 0 {
		hp.Ht = float64(hp.Hht) / float64(hp.Hhtall)
	} else {
		hp.Ht = 0
	}
	if hp.Hhball > 0 {
		hp.Hc = float64(hp.Hhb) / float64(hp.Hhball)
	} else {
		hp.Hc = 0
	}
	if hp.Hheall > 0 {
		hp.Hd = -float64(hp.Hhe) / float64(hp.Hheall)
	} else {
		hp.Hd = 0
	}
	// 计算Hp_j
	hp.Value = r1*hp.Ha + r2*hp.Ht + r3*hp.Hc + r4*hp.Hd
}

func (sr *HistoricalSelection) CalcuSr() {
	if sr.Ssra == 0 {
		// 避免除以0的情况
		log.Println("参与的轮数不能为0，无法进行计算。")
		return
	}
	// 计算得出的Sr_j值
	selectionRatio := float64(sr.Ssro+sr.Ssr1) / float64(sr.Ssra)
	sr.Value = math.Log(selectionRatio + 1)
}

// 计算节点j的代币占比TC_j^q
func (th *TokenHolding) CalculateTokenShare() {
	if th.Tcall > 0 { // 确保分母不为0
		th.Value = float64(th.TC) / float64(th.Tcall)
	} else {
		th.Value = 0
	}
}
func (node *NodeReputation) CalcuateReputation() {
	node.Pf.CalcuPf()
	node.Hp.CalcuHp()
	node.Sr.CalcuSr()
	node.TC.CalculateTokenShare()
	node.Value = alpha1*node.Pf.Value + alpha2*node.Hp.Value + alpha3*node.Sr.Value + alpha4*node.Gu.Value + alpha5*node.TC.Value
}

func SetNodeDetail(lineText []string) {
	// var node NodeReputation
	// node.extra.latitude = lineText[0]

}
func loadNodeReputation(line []string, nr *NodeReputation) error {
	if len(line) < 21 {
		fmt.Println("insufficient data: expected at least 21 elements")
		return fmt.Errorf("insufficient data: expected at least 21 elements")
	}

	// 解析纬度
	latitude, err := strconv.ParseFloat(line[0], 64)
	if err != nil {
		return err
	}
	nr.Extra.Llatitude = latitude

	// 解析经度
	longitude, err := strconv.ParseFloat(line[1], 64)
	if err != nil {
		return err
	}
	nr.Extra.Llongitude = longitude

	// 延迟
	nr.Pf.Delay, err = strconv.ParseFloat(line[2], 64)
	if err != nil {
		return err
	}
	//哈希率
	nr.Pf.H, err = strconv.ParseFloat(line[3], 64)
	if err != nil {
		return err
	}
	//内存大小
	nr.Pf.M, err = strconv.ParseFloat(line[4], 64)
	if err != nil {
		return err
	}
	//io速度
	nr.Pf.Iio, err = strconv.ParseFloat(line[5], 64)
	if err != nil {
		return err
	}
	//在线时长
	value, err := strconv.ParseUint(string(line[6]), 10, 32)
	if err != nil {
		return err
	}
	nr.Pf.Ttk = uint(value)

	// 实际投票次数
	vote1, err := strconv.ParseUint(line[7], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Vvote = uint(vote1)
	//有机会参与投票次数
	voteall, err := strconv.ParseUint(line[8], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Vvoteall = uint(voteall)
	//正常发送交易数量
	ht, err := strconv.ParseUint(line[9], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Hht = uint(ht)
	//节点发送的所有交易数量
	htall, err := strconv.ParseUint(line[10], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Hhtall = uint(htall)
	//被验证接受的区块数
	hb, err := strconv.ParseUint(line[11], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Hhb = uint(hb)
	//所有区块数量
	hball, err := strconv.ParseUint(line[12], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Hhball = uint(hball)
	//发送的恶意交易和区块数量
	he, err := strconv.ParseUint(line[13], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Hhe = uint(he)
	//所有交易和打包的所有区块数量
	heall, err := strconv.ParseUint(line[14], 10, 32)
	if err != nil {
		return err
	}
	nr.Hp.Hheall = uint(heall)

	//选入下层委员会占比
	sro, err := strconv.ParseUint(line[15], 10, 32)
	if err != nil {
		return err
	}
	nr.Sr.Ssro = uint(sro)
	//选入中央委员会占比
	sr1, err := strconv.ParseUint(line[16], 10, 32)
	if err != nil {
		return err
	}
	nr.Sr.Ssr1 = uint(sr1)
	//参与的轮数
	sra, err := strconv.ParseUint(line[17], 10, 32)
	if err != nil {
		return err
	}
	nr.Sr.Ssra = uint(sra)

	// 解析代币持有情况
	tc, err := strconv.ParseUint(line[18], 10, 64)
	if err != nil {
		return err
	}
	nr.TC.TC = float64(tc)

	//解析恶意节点的信息
	healthy, err := strconv.Atoi(line[19])
	if err != nil {
		return err
	}
	//为了避免枚举 初始值 为0的干扰 1为恶意节点 2为健康节点
	nr.Extra.HhealthyNodeIdentifier = NodeType(healthy + 1)
	//解析恶意节点做的的可能性
	nr.Extra.MmaliciousnessProbability, err = strconv.ParseFloat(line[20], 64)
	if err != nil {
		return err
	}
	// 这里我们就假设一个Tcall的值
	nr.TC.Tcall = 1000    // 假设总代币数量为100万
	nr.Pf.Ttall = 10      //总运行时间
	nr.Gu.Ddirect = 1.0   // 给一个假设值
	nr.Gu.Iindirect = 1.0 // 给一个假设值
	return nil
}
