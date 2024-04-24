package main

import (
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
	create <地址> "创建区块链"
	addBlock <需要写入的的数据> "添加区块"
	print "打印区块链"
	getBalance <地址> "获取余额"
	send <FROM> <TO> <AMOUNT> <MINER> <DATA>
	createWallet "创建钱包"
	listAddress "列举所有的钱包地址"
	printTx "打印区块的所有交易"
	enter "进入区块链网络"
	exit "关闭程序"
`

// const Usage1 = "" +
// 	"./block" +
// 	""

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
	enterflag := 0
	osinput := make(chan string)
	inListenChan := make(chan string)
	outListenChan := make(chan string)
	inDailChan := make(chan string)
	outDailChan := make(chan string)
	// miningChan := make(chan bool)

	//读键盘输入命令线程
	go func() {
		for {
			var input string
			fmt.Scanln(&input)
			osinput <- input
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
			switch cmds[1] {
			case "enter":
				if enterflag == 1 {
					Log.Warn("区块链已经完成初始化")
				} else {
					enterflag = 1
					fmt.Println("执行区块链初始化任务")
					go cli.listest(inListenChan, outListenChan)
					go cli.dailtest(inDailChan, outDailChan)
				}

			case "create":
				fmt.Println("创建区块被调用!")
				if len(cmds) != 3 {
					fmt.Println("输入参数无效，请检查!")
					return
				}
				address := cmds[2]
				cli.createBlockChain(address)

			case "addBlock":
				if len(cmds) != 3 {
					fmt.Println("输入参数无效，请检查!")
					return
				}
				data := cmds[2] //需要检验个数
				cli.addBlock(data)
			case "print":
				fmt.Println("打印区块被调用!")
				cli.print()
			case "getBalance":
				fmt.Println("获取余额命令被调用!")
				if len(cmds) != 3 {
					fmt.Println("输入参数无效，请检查!")
					return
				}
				address := cmds[2] //需要检验个数
				cli.getBalance(address)
			case "send":
				fmt.Println("send命令被调用")
				if len(cmds) != 7 {
					fmt.Println("输入参数无效，请检查!")
					return
				}

				from := cmds[2]
				to := cmds[3]
				//这个是金额，float64，命令接收都是字符串，需要转换
				Source := cmds[4]
				SourceID, err := uuid.Parse(Source)
				if err != nil {
					fmt.Println("溯源id不合法", err)
					return
				}
				miner := cmds[5]
				data := cmds[6]
				cli.send(from, to, SourceID, miner, data)
			case "createWallet":
				fmt.Println("创建钱包命令被调用!")
				cli.createWallet()
			case "listAddress":
				fmt.Println("listAddress 被调用")
				cli.listAddress()
			case "printTx":
				cli.printTx()
			case "exit":
				fmt.Println("退出程序")
				return

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
