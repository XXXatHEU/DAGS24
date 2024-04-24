package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

//处理用户输入命令，完成具体函数的调用
//cli : command line 命令行
type CLI struct {
}

//使用说明，帮助用户正确使用
const Usage = `
正确使用方法：
	create <地址> <交易信息> "创建区块链"
	addBlock <需要写入的的数据> "添加区块"
	print "打印区块链"
	getBalance <地址> "获取余额"
	send <FROM> <TO> <AMOUNT> <MINER> <DATA>
	createWallet "创建钱包"
	listAddress "列举所有的钱包地址"
	printTx "打印区块的所有交易"
	--------------------------
	enter "进入区块链网络"
	moni "模拟打包并发送"
	autosendtx  "启动不断自动生成一笔交易的后台服务"
	stopautoSendtx "关闭自动生成交易后台服务"
	mining "启动后台挖矿"
	createProduct <溯源信息> <溯源产品归属地址> "创建一个溯源产品"
	trace <targetSourceId> "数据溯源"
	traceall  "列出所有的溯源信息"
	zkwallet "更新zk钱包和公司名称"
	printbucket "打印bucket的键"
	txPool "打印交易池内容"
	fetch "发送同步区块链请求"
	rupt  "暂停挖矿协程"
	stopmining "关闭挖矿协程"
	exit "关闭程序"
	cs "测试某个函数"
`

// const Usage1 = "" +
// 	"./block" +
// 	""
var Autosendtx = make(chan int)

func init() {
	Autosendtx = nil //刚开始是关闭的
}
func (cli *CLI) listest(inmsg, outmsg chan string) {
	for {
		select {
		case <-time.After(3 * time.Second):
			outmsg <- "listen发出问候"
			Log.Info("Tlistest监听收到其他节点一个命令 向上传出命令")
		case msg := <-inmsg:
			Log.Info("listest收到主程序命令， 接下来执行命令:", msg)
		}
	}
}
func (cli *CLI) dailtest(inmsg, outmsg chan string) {
	for {
		select {
		case <-time.After(5 * time.Second):
			outmsg <- "dail发出问候"
			Log.Info("dailtest监听收到其他节点一个命令 向上传出命令")
		case msg := <-inmsg:
			Log.Info("dailtest收到主程序命令， 接下来执行命令:", msg)
		}
	}
}

//负责解析命令的方法
func (cli *CLI) Run() {
	//cmds := os.Args

	osinput := make(chan string)
	inListenChan := make(chan string)
	outListenChan := make(chan string)
	inDailChan := make(chan string)
	outDailChan := make(chan string)
	// miningChan := make(chan bool)

	//读键盘输入命令线程
	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			fmt.Println(line)
			osinput <- line
		}
	}()
	//键盘命令解析线程

	fmt.Println(Usage)
	for {
		select {
		//输入线程有信号发生
		case msg := <-osinput:
			cmds := append(os.Args[:1], strings.Fields(msg)...)
			if len(cmds) < 2 {
				Log.Error("输入参数无效，请检查!")
				fmt.Println(Usage)
				continue
			}
			for i, cmd := range cmds {
				fmt.Printf("cmds[%d]: %s\n", i, cmd)
			}
			cmds[1] = strings.ToLower(cmds[1])
			switch cmds[1] {
			case "enter":
				if enterflag == 1 {
					Log.Warn("区块链已经完成初始化")
				} else {
					enterflag = 1
					addrout := make(chan string)
					go ListenRun(addrout)
					var addres string
					addres = <-addrout
					Zookinit()
					ZookRun(addres)

					go DailRun()
					go Netpoolstart()
					// fmt.Println("执行区块链初始化任务")
					// go cli.listest(inListenChan, outListenChan)
					// go cli.dailtest(inDailChan, outDailChan)
				}

			case "create":
				fmt.Println("创建区块被调用!")
				// if len(cmds) != 4 {
				// 	fmt.Println("输入参数无效，请检查!")
				// 	continue
				// }
				//productmsg := cmds[3]
				productmsg := "这是第一笔交易"
				//address := cmds[2]
				//address := "15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47"
				cli.createBlockChain(productmsg)
			case "stopmining":
				MiningControlStop <- 1
				Log.Info("发送停止挖矿协程")
			case "addblock":
				if len(cmds) != 3 {
					fmt.Println("输入参数无效，请检查!")
					continue
				}
				data := cmds[2] //需要检验个数
				cli.addBlock(data)
			case "s":
				// if len(cmds) != 4 {
				// 	fmt.Println("输入参数无效，请检查!")
				// 	continue
				// }
				to := "cmds[3]"
				//fmt.Println(sourceID, to)
				transferTraceability(to, "测试的一笔交易")
			case "autosendtx":
				Log.Debug("自动发送交易后台服务启动")
				go StartAutosendTX()
			case "stopautoSendtx":
				StopAutosendTX()
			case "print":
				fmt.Println("打印区块被调用!")
				cli.print()
			case "getbalance":
				fmt.Println("获取余额命令被调用!")
				if len(cmds) != 3 {
					fmt.Println("输入参数无效，请检查!")
					continue
				}
				address := cmds[2] //需要检验个数
				cli.getBalance(address)
			case "send":
				fmt.Println("send命令被调用")
				if len(cmds) != 7 {
					fmt.Println("输入参数无效，请检查!")
					continue
				}

				from := cmds[2]
				to := cmds[3]
				//这个是金额，float64，命令接收都是字符串，需要转换
				Source := cmds[4]
				SourceID, err := uuid.Parse(Source)
				if err != nil {
					fmt.Println("溯源id不合法", err)
					continue
				}
				miner := cmds[5]
				data := cmds[6]
				productmsg := cmds[7]
				cli.send(from, to, SourceID, miner, data, productmsg)
			case "createwallet":
				fmt.Println("创建钱包命令被调用!")
				cli.createWallet()
			case "createproduct":
				if len(cmds) != 4 {
					fmt.Println("输入参数无效，请检查!")
					createProductAndToPool(nil, nil)
					continue
				} else {
					ProductData := cmds[2]
					Pubkeyhash := cmds[3]
					createProductAndToPool([]byte(ProductData), []byte(Pubkeyhash))
				}
			case "listaddress":
				fmt.Println("listAddress 被调用")
				cli.listAddress()
			case "printtx":
				cli.printTx()
			case "exit":
				fmt.Println("退出程序")
				return
			case "moni":
				fmt.Println("Test_moni_SendCommonBlocks 被调用")
				Test_moni_SendCommonBlocks()
			case "moni1":
				fmt.Println("GetBlockCountUntilSpecificBlock 被调用")
				BlockExist([]byte("0000353abfa21effb30e49b2fa106a8516a3cefe4b2c3b19cbbefea570ab54d7"))
			case "zk":
				Log.Info("Test_moni_SendCommonBlocks 被调用")
			case "mining":
				if enterflag == 1 {
					Log.Info("开始挖矿")
					go MiningControl()
				} else {
					Log.Warn("未进入区块链网络")
				}

			case "rupt":
				if enterflag != 1 {
					Log.Warn("请先开始挖矿")
				} else {
					InterruptChan <- 1
					Log.Info("打断信号发送成功")
				}
			case "traceall":
				Log.Info("打印所有的溯源节点")
				DataTraceall()
			case "trace":
				targetSourceId := cmds[2]
				Log.Info("打印溯源节点，没有名称的将以地址替代")
				DataTraceability(targetSourceId)
			case "txpool":
				PrintTxPool()
			case "zkwallet":
				if Zookexit == 1 {
					Util(Zkconn)
				} else {
					Zookinit()
					Util(Zkconn)
				}
			case "cs":
				fmt.Println("开始执行命令")
				start := cmds[2]
				GraphGetsocuid(start)
				// start := cmds[2]
				// stop := cmds[3]
				// GetBlocksInTimeRange(start, stop)
			case "time":
				etime()
			case "gre":
				str := cmds[2]
				GraphQueryWithKeywordAttributes(str)
			case "bte":
				str := cmds[2]
				BTQueryWithKeywordAttributes(str)
			case "printbucket":
				getBucketKeys()
			case "fetch":
				fmt.Println("发送同步区块链请求")
				FetchBlocksRequest(nil)

			default:
				fmt.Println("输入参数无效，请检查!")
				fmt.Println(Usage)
			}

		case msg := <-outListenChan:
			fmt.Println("收到listenchan消息: ", msg, "交给dail协程处理 ")
			fmt.Println()
			inDailChan <- msg

		case msg := <-outDailChan:
			fmt.Println("收到dailChan消息: ", msg, "交给dail协程处理 ")
			inListenChan <- msg
		}
	}

	// for {
	// 	var input string
	// 	fmt.Scanln(&input)
	// 	if input == "exit" {
	// 		close(done)
	// 		os.Exit(0)
	// 	}
	// 	c <- input
	// }

	//用户至少输入两个参数
	// if len(cmds) < 2 {
	// 	fmt.Println("输入参数无效，请检查!")
	// 	fmt.Println(Usage)
	// 	return
	// }

	// switch cmds[1] {
	// case "create":
	// 	fmt.Println("创建区块被调用!")
	// 	if len(cmds) != 3 {
	// 		fmt.Println("输入参数无效，请检查!")
	// 		return
	// 	}
	// 	address := cmds[2]
	// 	cli.createBlockChain(address)

	// case "addBlock":
	// 	if len(cmds) != 3 {
	// 		fmt.Println("输入参数无效，请检查!")
	// 		return
	// 	}
	// 	data := cmds[2] //需要检验个数
	// 	cli.addBlock(data)
	// case "print":
	// 	fmt.Println("打印区块被调用!")
	// 	cli.print()
	// case "getBalance":
	// 	fmt.Println("获取余额命令被调用!")
	// 	if len(cmds) != 3 {
	// 		fmt.Println("输入参数无效，请检查!")
	// 		return
	// 	}
	// 	address := cmds[2] //需要检验个数
	// 	cli.getBalance(address)
	// case "send":
	// 	fmt.Println("send命令被调用")
	// 	if len(cmds) != 7 {
	// 		fmt.Println("输入参数无效，请检查!")
	// 		return
	// 	}

	// 	from := cmds[2]
	// 	to := cmds[3]
	// 	//这个是金额，float64，命令接收都是字符串，需要转换
	// 	Source := cmds[4]
	// 	SourceID, err := uuid.Parse(Source)
	// 	if err != nil {
	// 		fmt.Println("溯源id不合法", err)
	// 		return
	// 	}
	// 	miner := cmds[5]
	// 	data := cmds[6]
	// 	cli.send(from, to, SourceID, miner, data)
	// case "createWallet":
	// 	fmt.Println("创建钱包命令被调用!")
	// 	cli.createWallet()
	// case "listAddress":
	// 	fmt.Println("listAddress 被调用")
	// 	cli.listAddress()
	// case "printTx":
	// 	cli.printTx()
	// default:
	// 	fmt.Println("输入参数无效，请检查!")
	// 	fmt.Println(Usage)
	// }
}
