package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var channelIndexMutex2 sync.Mutex
var channelIndex2 = -1

var NodeIndexMutex2 sync.Mutex
var NodeIndex2 = 0

var Outchannels2 = make([]chan string, 3000)
var Inchannels2 = make([]chan string, 3000)

func init() {
	// 创建一个包含70个channel的切片
	// 初始化每个channel
	for i := range Outchannels2 {
		Outchannels2[i] = make(chan string)
		Inchannels2[i] = make(chan string)
	}
}

func Orderinmain() {

	// 1. 监听地址和端口
	addr := "127.0.0.1:10000"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Server listening on", addr)

	// 3. 总控协程  这个函数将接受输入并将输入发送到各个子节点
	go OrderMain()

	// 4.接受客户端连接
	for {
		// 接受客户端连接
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		channelIndexMutex2.Lock()
		channelIndex2++
		go handleConnection2(conn, Outchannels2[channelIndex2], Inchannels2[channelIndex2])
		channelIndexMutex2.Unlock()
	}

}

// 连接后的事件处理
func handleConnection2(conn net.Conn, OutChan chan string, InChan chan string) {
	defer conn.Close()
	//1.初始化，向其发送是属于第几个节点
	NodeIndexMutex2.Lock()
	nodeIndexStr := strconv.Itoa(NodeIndex2)
	fmt.Println("发送节点序号:", nodeIndexStr)
	NodeIndex2++
	_, err := conn.Write([]byte(nodeIndexStr))
	if err != nil {
		fmt.Println("发送命令时发生错误，即将断开连接:", err)
		return
	}
	NodeIndexMutex2.Unlock()

	//启动读取响应协程
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("Error reading:", err)
				break
			}
			receivedData := string(buffer[:n])
			fmt.Println("clinet响应:", receivedData)
		}
	}()
	//总控向节点发送数据
	for {
		select {
		case msg := <-OutChan:
			fmt.Println("收到命令，即将发送数据：", msg)
			_, err := conn.Write([]byte(msg))
			if err != nil {
				fmt.Println("发送命令时发生错误，即将断开连接:", err)
				return
			}

		}
	}
}

//总控部分
func OrderMain() {
	//读键盘输入命令线程
	osinput := make(chan string)

	go func() {
		for {
			fmt.Println("请输出要发送的命令")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			fmt.Println(line)
			osinput <- line
		}
	}()

	for {
		select {
		case msg := <-osinput:
			//运行一个
			fmt.Println("得到键盘输入，即将发送给节点")
			channelIndexMutex2.Lock()
			fields := strings.Fields(msg)
			tempcount := 0
			if strings.ToLower(fields[0]) == "addblock" {
				//for j := 0; j < 2049; j++ {
				for i := 0; i <= channelIndex2; i++ {
					//for k := 0; k < 100; k++ {
					tempchannel := Outchannels2[i]
					go func() {
						select {
						case tempchannel <- msg:
							fmt.Println("数据发送成功！向", i, "发送")
						default:
							fmt.Println("无法发送数据，channel已满或无接收者。")
						}
					}()
					if strings.ToLower(fields[0]) == "addblock" {
						time.Sleep(500 * time.Millisecond)
					}
					tempcount++
					//}
				}
				//}
				fmt.Println("发送完毕")
			} else if strings.ToLower(fields[0]) == "mining" /* || strings.ToLower(fields[0]) == "create"*/ {
				fmt.Println("进入mining==========================================")
				for i := 0; i < 1; i++ {
					tempchannel := Outchannels2[i]
					go func() {
						select {
						case tempchannel <- msg:
							fmt.Println("mining 数据发送成功！", i)
						default:
							fmt.Println("无法发送数据，channel已满或无接收者。")
						}
					}()
					tempcount++
				}

			} else if strings.ToLower(fields[0]) == "enter" {
				for i := 0; i <= channelIndex2; i++ {
					//for k := 0; k < 100; k++ {
					tempchannel := Outchannels2[i]
					go func() {
						select {
						case tempchannel <- msg:
							fmt.Println("数据发送成功！向", i, "发送")
						default:
							fmt.Println("无法发送数据，channel已满或无接收者。")
						}
					}()
					if strings.ToLower(fields[0]) == "enter" {
						//time.Sleep(1000 * time.Millisecond)
					}
					tempcount++
					//}
				}
				fmt.Println("发送完毕")

			} else {
				for i := 0; i <= channelIndex2; i++ {
					tempchannel := Outchannels2[i]
					go func() {
						select {
						case tempchannel <- msg:
							fmt.Println("数据发送成功！")
						default:
							fmt.Println("无法发送数据，channel已满或无接收者。")
						}
					}()
					tempcount++
				}
			}

			fmt.Println("发送成功,个数为：", tempcount)
			channelIndexMutex2.Unlock()
		}
	}
}
