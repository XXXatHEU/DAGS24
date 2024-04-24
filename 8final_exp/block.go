package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/willf/bloom"
)

//区块类型  主要在侧链溯源中，需要判断前面这个区块是什么类型，如果是MainblockType就停止溯源
//暂时没用
type BlockType int

const (
	//主链区块
	MainblockType BlockType = iota
	//侧链区块
	SideblockType
)

// 定义区块结构
// 第一阶段: 先实现基础字段：前区块哈希，哈希，数据
// 第二阶段: 补充字段：Version，时间戳，难度值等
type Block struct {
	//版本号
	Version uint64

	//Blocktype BlockType

	// 前区块哈希
	PrevHash []byte

	//交易的根哈希值
	MerkleRoot []byte

	//时间戳
	TimeStamp uint64

	//难度值, 系统提供一个数据，用于计算出一个哈希值
	Bits uint64

	//随机数，挖矿要求的数值
	Nonce uint64

	// 哈希, 为了方便，我们将当前区块的哈希放入Block中
	Hash []byte

	//数据
	// Data []byte
	//一个区块可以有很多交易(交易的集合)  但是主链区块只有一个溯源交易的起点交易
	Transactions []*Transaction

	/////////////////////////////////溯源区块链//////////////////////////////
	// //本溯源线上一个节点
	// PoI []byte

	// //当前主链的最后一个区块(暂时留空)
	// CP []byte

	// //指向还没有TBP指向的区块
	// TBP []byte

	////////////////////////////主链区块////////////////////////
	MP        []byte      //主链上的区块指向前一个主链区块的指针
	Product   ProductType //溯源产品
	SP        [][]byte    //分裂指针  暂时没有添加功能
	AllBlooms []byte
	//先验证AllBlooms是否有，如果有遍历ConfirmedLists所有的 找有没有
	ConfirmedLists      map[uuid.UUID]ConfirmedBlockListType // key为溯源的uuid
	Skiplist            [3][]byte                            //三级列表  第一个元素向前三步、第二个元素向前六步、第三个元素向前九步
	ExtraMessage        string                               //额外信息  用于布隆过滤器的一些信息
	ExtratargetSourceId string
	Extramyaddress      string
}

/*
	AllBlooms是所有的确认的区块里面的词条形成的布隆过滤器
	ConfirmedLists里面包含了所有确认的溯源数组，每个ConfirmedBlockListType里面有某一个溯源产品的确认区块s，

	因此先验证AllBlooms是否有，如果有
	  遍历ConfirmedLists的每个Bloom，验证是否有，如果有进去BlockArray遍历所有的Bloom，拿到Hash，从数据库拿出具体信息来

*/

type ConfirmedBlockType struct {
	Hash  []byte //区块的哈希值  实际上是交易的哈希值
	Bloom []byte //区块包含的信息
}

//先验证AllBlooms是否有，如果有遍历ConfirmedBlockListType 找有没有
type ConfirmedBlockListType struct { //对于map而言，每个uuid都有个ConfirmedBlockListType  里面的FinalBelongPubKeyHash的就是最终所有权
	BlockArray            []ConfirmedBlockType //对上面ConfirmedBlockType进行封装
	Bloom                 []byte               //BlockArray里面包含的词条的布隆过滤器
	FinalBelongPubKeyHash []byte               //收款人的公钥哈希
}

func createBlock(txs []*Transaction, prevHash []byte, msg string, verifytxsmap map[uuid.UUID][]*Transaction) Block {
	pubkeyhash := getPubKeyHashFromAddress(mywallet.Address)
	fmt.Printf("NewCoinbaseTx address================= %x\n", pubkeyhash)

	b := Block{
		Version:    0,
		PrevHash:   prevHash,
		MerkleRoot: nil, //随意写的
		TimeStamp:  uint64(time.Now().Unix()),

		Bits:  0, //随意写的, 这个值是uint64，它并不是难度值，而是可以推算出难度值
		Nonce: 0, //随意写的
		Hash:  nil,
		// Data:  []byte(data),
		//Transactions: txs,
		MP: prevHash,
		//Product:        nil,
		//AllBlooms:      nil,
		//ConfirmedLists: nil,
		//Skiplist: nil,
	}
	//填充product
	product, err := PopFromProductPool()
	if err != nil {
		product = createProduct(nil, nil)
	}

	input := TXInput{Txid: nil, Index: -1, ScriptSig: nil, PubKey: mywallet.walle.PubKey}
	output := newTXOutput(mywallet.Address, product.SourceID)

	specifTX := Transaction{
		TXID:      nil,
		TXInputs:  []TXInput{input},
		TXOutputs: []TXOutput{output},
		TimeStamp: uint64(b.TimeStamp),
		PoI:       nil, // 本溯源线上一个节点
		CP:        nil, // 当前主链的最后一个区块
		TBP:       nil, // 指向还没有TBP指向的区块
	}

	SetSpecTxDetail(&specifTX)
	txs = append(txs, &specifTX)
	//realtxs := []*Transaction{&specifTX}

	b.Transactions = txs
	b.Product = product

	//填充梅克尔根值
	b.HashTransactionMerkleRoot()
	fmt.Printf("merkleRoot:%x\n", b.MerkleRoot)

	//计算哈希值
	// b.setHash()  最开始版本残留，应该从pow中得出来
	//如果不是创世区块会添加剩下的内容
	extraMessage := "extraMessage : "
	if prevHash != nil {
		// 创建布隆过滤器
		m := uint(10000)             // 位数组大小
		k := uint(2)                 // 哈希函数数量
		allBlooms := bloom.New(m, k) //全局布隆过滤器

		var confirmedLists = make(map[uuid.UUID]ConfirmedBlockListType) // 要放到区块里面的内容

		//如果是同一个uuid，verifytxsmap里面的元素是按照时间顺序来的，越靠后交易越晚
		for key, TXS := range verifytxsmap {
			Log.Debug("对verifytxsmap进行组合key:", key)
			//新建一个布隆过滤器存放所有uuid一样的元素
			txsbloom := bloom.New(5000, 2) //当前uuid的布隆过滤器
			var confirmedBlockList ConfirmedBlockListType
			for _, tx := range TXS {
				txbloom := bloom.New(500, 2) //当前区块的布隆过滤器
				txmsg := tx.TXmsg            //分词放到布隆过滤器中
				words := jieba.Cut(string(txmsg), true)

				// 获得输入pubkey 输入公钥哈希 输入addredd
				inpubkey := tx.TXInputs[0].PubKey
				inpubkeyhash := getPubKeyHashFromPubKey(inpubkey)
				inaddredd, _ := pubkeyhashToAddress(inpubkeyhash)

				//extraMessage = extraMessage + (" inpubkey" + string(inpubkey) + " inpubkeyhash:" + string(inpubkeyhash) + " inaddredd" + inaddredd)

				words = append(words, string(inpubkey))
				words = append(words, string(inpubkeyhash))
				words = append(words, string(inaddredd))

				//获得输出pubkey  输出公钥哈希 输出addredd
				outpubkeyhash := tx.TXOutputs[0].ScriptPubKeyHash
				outaddred, _ := pubkeyhashToAddress(outpubkeyhash)

				//extraMessage = extraMessage + ("  outpubkeyhash:" + string(outpubkeyhash) + "  outaddred" + string(outaddred))
				words = append(words, string(outpubkeyhash))
				words = append(words, string(outaddred))

				//获得交易id  溯源id
				txid := tx.TXID
				targetid := tx.TXOutputs[0].SourceID
				b.Extramyaddress = outaddred
				b.ExtratargetSourceId = targetid.String()
				extraMessage += targetid.String()
				//extraMessage = extraMessage + (" txid:" + string(txid) + " targetid:" + targetid.String())
				words = append(words, string(txid))
				words = append(words, targetid.String())

				for _, str := range words {
					txbloom.Add([]byte(str))
					txsbloom.Add([]byte(str))
					allBlooms.Add([]byte(str))
				}
				//真正添加到布隆过滤器里面  前面加到word中我怕有问题

				var temp ConfirmedBlockType
				var buf bytes.Buffer
				txbloom.WriteTo(&buf)
				temp.Bloom = buf.Bytes()
				temp.Hash = tx.TxHash
				confirmedBlockList.BlockArray = append(confirmedBlockList.BlockArray, temp)
				confirmedBlockList.FinalBelongPubKeyHash = tx.TXOutputs[0].ScriptPubKeyHash
			}
			var buf bytes.Buffer
			txsbloom.WriteTo(&buf)
			confirmedBlockList.Bloom = buf.Bytes()
			confirmedLists[key] = confirmedBlockList
		}
		Log.Debug("extraMessage====================================", extraMessage)
		b.ExtraMessage = extraMessage
		b.ConfirmedLists = confirmedLists
		var buf bytes.Buffer
		allBlooms.WriteTo(&buf)
		b.AllBlooms = buf.Bytes()

		//创建跳表 有三个哈希值
		//第一个哈希值是指向前2个
		bc, err := GetBlockChainInstance()
		defer bc.closeDB()
		if err != nil {
			Log.Error("NewBlock出错:", err)
		}
		it := bc.NewIterator()
		count := 0
		for {
			//遍历区块
			block := it.MemoryNextBlock()
			if block == nil {
				break
			}
			count = count + 1
			if count == 3 { //3
				b.Skiplist[0] = block.Hash
			} else if count == 6 { //6
				b.Skiplist[1] = block.Hash
			} else if count == 9 { //9
				b.Skiplist[2] = block.Hash
			}
			if block.PrevHash == nil {
				break
			}

		}

	} else {

	}
	return b
}

// 创建一个区块（提供一个方法）  这个是手动添加的 依托的是createBlock
// 输入：数据，前区块的哈希值
// 输出：区块
func NewBlock(txs []*Transaction, prevHash []byte, msg string, verifytxsmap map[uuid.UUID][]*Transaction) *Block {

	b := createBlock(txs, prevHash, msg, verifytxsmap)

	//将POW集成到Block中

	pow := NewProofOfWork(&b) //这个函数在blockchain.go中
	hash, nonce := pow.Run()
	Log.Debug("从挖矿处返回")
	b.Hash = hash
	b.Nonce = nonce
	return &b
}

//适配网络模式的NewBlock方法
func MiningNewBlock(txs []*Transaction, prevHash []byte, verifytxsmap map[uuid.UUID][]*Transaction) (*Block, *ProofOfWork) {
	//无法使用，应该是输入被另一个协程接收了
	// reader := bufio.NewReader(os.Stdin)
	// fmt.Println("请输入这个溯源产品的详细信息:")
	// input, _ := reader.ReadString('\n')
	// fmt.Println("您输入的字符串是:", input)
	input := "溯源产品的起点数据，暂时没有设置"
	b := createBlock(txs, prevHash, input, verifytxsmap)

	//填充梅克尔根值
	b.HashTransactionMerkleRoot()
	fmt.Printf("merkleRoot:%x\n", b.MerkleRoot)

	//将POW集成到Block中
	pow := NewProofOfWork(&b) //这个函数在blockchain.go中
	// hash, nonce := pow.Run()
	// b.Hash = hash
	// b.Nonce = nonce

	return &b, pow
}

// 绑定Serialize方法， gob编码
func (b *Block) Serialize() []byte {
	var buffer bytes.Buffer

	//创建编码器
	gob.Register(bloom.BloomFilter{})
	gob.Register(&Transaction{})
	encoder := gob.NewEncoder(&buffer)

	//编码
	err := encoder.Encode(b)
	if err != nil {
		Log.Fatal("Serialize失败:", err)
		return nil
	}
	Log.Debug("完成序列化")

	return buffer.Bytes()
}

// 反序列化，输入[]byte，返回block
func Deserialize(src []byte) *Block {
	var block Block

	//创建解码器
	decoder := gob.NewDecoder(bytes.NewReader(src))
	//解码
	err := decoder.Decode(&block)
	if err != nil {
		fmt.Printf("decode err:", err)
		return nil
	}

	return &block
}

// 简易梅克尔根，把所有的交易拼接到一起，做哈希处理，最终赋值给block.MerKleRoot
func (block *Block) HashTransactionMerkleRoot() {
	//遍历所有的交易，求出交易哈希值

	var info [][]byte

	for _, tx := range block.Transactions {
		//将所有的哈希值拼接到一起，做sha256处理
		txHashValue := tx.TXID //[]byte
		info = append(info, txHashValue)
	}

	value := bytes.Join(info, []byte{})
	hash := sha256.Sum256(value)

	//讲hash值赋值MerKleRoot字段
	block.MerkleRoot = hash[:]
}
