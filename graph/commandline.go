package main

import (
	"fmt"

	"github.com/google/uuid"
)

func (cli *CLI) addBlock(data string) {
	cli.addBlockChainopp("手动增加区块")
}

func (cli *CLI) createBlockChain(productmsg string) {
	if mywallet == (MyWallet{}) {
		Log.Warn("钱包获取失败，未进入区块链网络")
		return
	}
	// if !isValidAddress(address) {
	// 	fmt.Println("传入地址无效，无效地址为:", address)
	// 	return
	// }
	address := mywallet.Address
	//创建并发送
	err := CreateBlockChain(address, productmsg)
	if err != nil {
		fmt.Println("CreateBlockChain failed:", err)
		return
	}

	fmt.Println("执行完毕!")
}

//手动增加区块
func (cli *CLI) addBlockChainopp(productmsg string) {
	if mywallet == (MyWallet{}) {
		Log.Warn("钱包获取失败，未进入区块链网络")
		return
	}
	// if !isValidAddress(address) {
	// 	fmt.Println("传入地址无效，无效地址为:", address)
	// 	return
	// }
	address := mywallet.Address
	Log.Debug("手动增加区块")
	err := addBlockChain1(address, productmsg)
	if err != nil {
		fmt.Println("CreateBlockChain failed:", err)
		return
	}
	fmt.Println("执行完毕!")
}

//给不是从cli过来的打印区块
func printBlock(block Block) {
	Log.Warn("ConfirmedLists数量为:", len(block.ConfirmedLists))
	for uid, lists := range block.ConfirmedLists {

		fmt.Printf("ConfirmedLists验证交易区块:%s\n", uid.String())
		for _, txhash := range lists.BlockArray {
			fmt.Printf("ConfirmedLists验证交易区块内交易哈希%x\n", txhash.Hash)
		}
	}
	fmt.Printf("	----------------------\n")
	fmt.Printf("		绑定溯源id:\n			%s\n", string(block.Product.SourceID.String()))
	fmt.Printf("		绑定溯源pubkeyhash:\n			%x\n", block.Product.Pubkeyhash)
	address, err := pubkeyhashToAddress(block.Transactions[0].TXOutputs[0].ScriptPubKeyHash)
	if err == nil {
		fmt.Printf("		本地对区块解析后，product归属账户地址:\n			%s\n", address) //矿工写入的数据
	} else {
		fmt.Printf("		本地未能解析product归属公钥哈希归属地址，ScriptPubKeyHash:\n			%x\n", block.Transactions[0].TXOutputs[0].ScriptPubKeyHash) //矿工写入的数据
	}
	fmt.Printf("		溯源信息:\n			%s\n", string(block.Product.ProductData))
	fmt.Printf("	----------------------\n")
}
func printtraceBlock(block Block) {
	address, err := pubkeyhashToAddress(block.Transactions[0].TXOutputs[0].ScriptPubKeyHash)
	if err == nil {
		fmt.Printf("		溯源起点为账户地址:%s\n", address) //矿工写入的数据
	} else {
		fmt.Printf("		本地未能解析product归属公钥哈希归属地址，ScriptPubKeyHash:\n			%x\n", block.Transactions[0].TXOutputs[0].ScriptPubKeyHash) //矿工写入的数据
	}
	fmt.Printf("		溯源起点信息: %s\n", string(block.Product.ProductData))

	fmt.Printf("	----------------------\n")
}

func (cli *CLI) print() {
	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("print err:", err)
		return
	}

	defer bc.closeDB()
	count := 0
	//调用迭代器，输出blockChain
	it := bc.NewIterator()
	for {
		count++
		//调用Next方法，获取区块，游标左移
		block := it.Next()

		fmt.Printf("\n++++++++++++++++++++++\n")
		fmt.Printf("	Version : %d\n", block.Version)
		fmt.Printf("	PrevHash : %x\n", block.PrevHash)
		fmt.Printf("	MerkleRoot : %x\n", block.MerkleRoot)
		fmt.Printf("	TimeStamp : %d\n", block.TimeStamp)
		fmt.Printf("	Bits : %d\n", block.Bits)
		fmt.Printf("	Nonce : %d\n", block.Nonce)
		fmt.Printf("	Hash : %x\n", block.Hash)
		fmt.Printf("	Data : %x\n", block.Transactions[0].TXInputs[0].ScriptSig) //矿工写入的数据
		fmt.Printf("	----------------------\n")
		fmt.Printf("		绑定溯源id:\n			%s\n", string(block.Product.SourceID.String()))
		fmt.Printf("		绑定溯源pubkeyhash:\n			%x\n", block.Product.Pubkeyhash)
		address, err := pubkeyhashToAddress(block.Transactions[0].TXOutputs[0].ScriptPubKeyHash)
		if err == nil {
			fmt.Printf("		本地对区块解析后，product归属账户地址:\n			%s\n", address) //矿工写入的数据
		} else {
			fmt.Printf("		本地未能解析product归属公钥哈希归属地址，ScriptPubKeyHash:\n			%x\n", block.Transactions[0].TXOutputs[0].ScriptPubKeyHash) //矿工写入的数据
		}
		fmt.Printf("		额外信息地址信息:\n			%s\n", block.Extramyaddress)
		fmt.Printf("		额外信息socureid:\n			%s\n", block.ExtratargetSourceId)
		fmt.Printf("		溯源信息:\n			%s\n", string(block.Product.ProductData))
		fmt.Printf("	----------------------\n")
		//通过验证交易区块
		for uid, lists := range block.ConfirmedLists {
			fmt.Printf("验证交易区块:%s\n", uid.String())
			for txhash := range lists.BlockArray {
				fmt.Printf("%x\n", txhash)
			}
		}
		for i := 0; i < len(block.Skiplist); i++ {
			fmt.Printf("Skiplist[%d]: %x\n", i, block.Skiplist[i])
		}
		// fmt.Printf("Data : %s\n", block.Data)

		pow := NewProofOfWork(block)
		fmt.Printf("IsValid: %v\n", pow.IsValid())

		//退出条件
		if block.PrevHash == nil {
			fmt.Println("区块链遍历结束!")
			break
		}
	}
	Log.Info("区块总个数为:", count)

}

func (cli *CLI) getBalance(address string) {
	if !isValidAddress(address) {
		fmt.Println("传入地址无效，无效地址为:", address)
		return
	}

	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("getBalance err:", err)
		return

	}

	defer bc.closeDB()

	//得到查询地址的公钥哈希
	pubKeyHash := getPubKeyHashFromAddress(address)

	//获取所有相关的utxo集合
	utxoinfos := bc.FindMyUTXO(pubKeyHash)
	//total := 0.0
	flag := 0
	for _, utxo := range utxoinfos {
		if flag == 0 {
			fmt.Println(address, "拥有的货物为:", utxo.TXOutput.SourceID)
			flag++
		} else {
			fmt.Println(utxo.TXOutput.SourceID)
		}
		//这两种方式都可以使用
		// total += utxo.Value

	}
	if flag == 0 {

		fmt.Printf("'%s'当前地址并未拥有货物\n", address)
	}
}

func (cli *CLI) send(from, to string, SourceID uuid.UUID, miner, data string, prodcuctmsg string) {
	if !isValidAddress(from) {
		fmt.Println("传入from无效，无效地址为:", from)
		return
	}

	if !isValidAddress(to) {
		fmt.Println("传入to无效，无效地址为:", to)
		return
	}

	if !isValidAddress(miner) {
		fmt.Println("传入miner无效，无效地址为:", miner)
		return
	}

	bc, err := GetBlockChainInstance()

	if err != nil {
		fmt.Println("send err:", err)
		return
	}

	defer bc.closeDB()

	//创建挖矿交易
	coinbaseTx := NewCoinbaseTx(miner, data)

	//创建txs数组，将有效交易添加进来
	txs := []*Transaction{coinbaseTx}

	//创建普通交易
	tx := NewTransaction(from, to, SourceID, bc)
	if tx != nil {
		fmt.Println("找到一笔有效的转账交易!")
		txs = append(txs, tx)
	} else {
		fmt.Println("注意，找到一笔无效的转账交易, 不添加到区块!")
	}

	//调用AddBlock
	err = bc.AddBlock(txs, prodcuctmsg)
	if err != nil {
		fmt.Println("添加区块失败，转账失败!")
	}

	fmt.Println("添加区块成功，转账成功!")
}

func (cli *CLI) createWallet() {
	wm := NewWalletManager()
	if wm == nil {
		fmt.Println("createWallet失败!")
		return
	}
	address := wm.createWallet()

	if len(address) == 0 {
		fmt.Println("创建钱包失败！")
		return
	}

	fmt.Println("新钱包地址为:", address)
}

func (cli *CLI) listAddress() {
	wm := NewWalletManager()
	if wm == nil {
		fmt.Println(" NewWalletManager 失败!")
		return
	}

	addresses := wm.listAddresses()
	for i, address := range addresses {
		fmt.Printf("第%d个： %s\n", i, address)
	}
}

func (cli *CLI) printTx() {
	bc, err := GetBlockChainInstance()
	if err != nil {
		fmt.Println("getBalance err:", err)
		return
	}

	defer bc.closeDB()

	it := bc.NewIterator()
	for {
		block := it.Next()
		fmt.Println("\n\n+++++++++++++++++ 区块分割 +++++++++++++++")

		for _, tx := range block.Transactions {
			//直接打印交易
			fmt.Println(tx)
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
}
