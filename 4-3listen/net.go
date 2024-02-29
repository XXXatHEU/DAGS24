package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var (
	start_Port      = 10000
	endPort         = 20000
	hasConnAddrsMap = map[string]bool{} //已经连接的地址 哈希表
	hasConnAddrsMu  sync.Mutex          //哈希表操作的锁
	Zookexit        = 0
)

type Command int

const (
	SendGenesisBlock Command = iota
	SendCommonBlock
	SendMinedBlock
	SendTX
	SendDB
	FetchBlocks //请求获取你的本地区块信息
	FetchTX
)

type SendMessage struct {
	Len     int32   // 请求消息的长度
	Cmd     Command // 请求的命令
	Payload []byte  // 请求携带的数据
}

type ResponseMessage struct {
	Len     int32   // 请求消息的长度
	Cmd     Command // 请求的命令
	Payload []byte  // 请求携带的数据
}

//发送消息
func sendRequest(conn net.Conn, req []byte) error {
	// var payloadBytes []byte
	// if req.Payload != nil {
	// 	// 将 Payload 转换为 byte[]
	// 	var ok bool
	// 	payloadBytes, ok = convertPayloadToBytes(req.Payload)
	// 	if !ok {
	// 		Log.Error("sendRequest payload type assertion failed")
	// 	}
	// }

	// req.Len = int32(len(payloadBytes) + 4)

	// buf := make([]byte, 4)
	// binary.BigEndian.PutUint32(buf, uint32(req.Len))
	// binary.BigEndian.PutUint32(buf[0:], uint32(req.Cmd))
	// msg := append(buf, payloadBytes...)

	// 将整个请求消息序列化成 byte 数组并发送给服务器
	_, errWrite := conn.Write(req)
	if errWrite != nil {
		Log.Error("发送数据失败：", errWrite)
	}
	return nil
}

// 根据不同 Payload 类型将其转换为字节数组
/*
func convertPayloadToBytes(payload interface{}) ([]byte, bool) {
	payloadValue := reflect.ValueOf(payload)
	switch payloadValue.Kind() {
	case reflect.Slice:
		// 如果 Payload 是 slice 类型
		elemType := payloadValue.Type().Elem()
		switch elemType {
		case reflect.TypeOf(byte(0)):
			// 如果 slice 的元素类型是 byte，则说明 Payload 是 []byte 类型
			return payload.([]byte), true
		case reflect.TypeOf(Transaction{}):
			// 如果 slice 的元素类型是 Transaction，则说明 Payload 是 []Transaction 类型
			transactions, ok := payload.([]Transaction)
			if !ok {
				return nil, false
			}
			payloadBytes := make([]byte, len(transactions)*10)
			for i, transaction := range transactions {
				binary.BigEndian.PutUint32(payloadBytes[i*10:i*10+4], transaction.ID)
				copy(payloadBytes[i*10+4:i*10+10], transaction.Data)
			}
			return payloadBytes, true
		case reflect.TypeOf(Block{}):
			// 如果 slice 的元素类型是 Block，则说明 Payload 是 []Block 类型
			blocks, ok := payload.([]Block)
			if !ok {
				return nil, false
			}
			payloadBytes := make([]byte, len(blocks)*100)
			for i, block := range blocks {
				binary.BigEndian.PutUint32(payloadBytes[i*100:i*100+4], block.ID)
				copy(payloadBytes[i*100+4:i*100+100], block.Data)
			}
			return payloadBytes, true
		default:
			return nil, false
		}
	default:
		return nil, false
	}
}*/

//每个连接后都会有的接收响应或者请求的后台协程函数
func recvResponse(conn net.Conn, recvmsgChan chan *SendMessage) error {
	// 读取响应消息头部的长度信息
	for {
		header := make([]byte, 4)
		_, err := io.ReadFull(conn, header)
		if err != nil {
			return err
		}
		length := binary.BigEndian.Uint32(header)
		Log.Info("消息的长度为：", length)
		// 读取整个响应消息
		msg := make([]byte, length-4)
		_, err = io.ReadFull(conn, msg)
		if err != nil {
			return err
		}
		Log.Info("完成数据的读取")
		// 构造 SendMessage 对象，并将其发送到消息通道中
		cmd := Command(int(binary.BigEndian.Uint32(msg[:4])))
		fmt.Println("收到的为：", msg)
		Log.Info("cmd:", Command(cmd))
		payload := msg[4:]
		sendMessage := &SendMessage{
			Len:     int32(length),
			Cmd:     cmd,
			Payload: payload,
		}
		recvmsgChan <- sendMessage
	}

}

// func recvResponse(conn net.Conn, recvmsgChan chan *SendMessage) error {
// 	// 读取响应消息头部的长度信息
// 	for {
// 		header := make([]byte, 4)
// 		_, err := io.ReadFull(conn, header)
// 		if err != nil {
// 			return err
// 		}
// 		length := binary.BigEndian.Uint32(header)
// 		Log.Info("消息的长度为：", length)

// 		// 将接收到的长度信息从网络字节序转换为主机字节序
// 		length = uint32(net.IPv4(byte(header[0]), byte(header[1]), byte(header[2]), byte(header[3])).To4()[0])<<24 |
// 			uint32(net.IPv4(byte(header[0]), byte(header[1]), byte(header[2]), byte(header[3])).To4()[1])<<16 |
// 			uint32(net.IPv4(byte(header[0]), byte(header[1]), byte(header[2]), byte(header[3])).To4()[2])<<8 |
// 			uint32(net.IPv4(byte(header[0]), byte(header[1]), byte(header[2]), byte(header[3])).To4()[3])

// 		// 读取整个响应消息
// 		msg := make([]byte, length-4)
// 		_, err = io.ReadFull(conn, msg)
// 		if err != nil {
// 			return err
// 		}

// 		// 构造 SendMessage 对象，并将其发送到消息通道中
// 		cmd := Command(binary.BigEndian.Uint32(msg[:4]))
// 		payload := msg[4:]
// 		sendMessage := &SendMessage{
// 			Len:     int32(length),
// 			Cmd:     cmd,
// 			Payload: payload,
// 		}
// 		recvmsgChan <- sendMessage
// 	}
// }

// func RecvMsgTransform(recvmsg SendMessage) interface{} {
// 	unpackcmd := RequestTransformPackCmd(recvmsg.Cmd)
// 	if unpackcmd == -1 {
// 		Log.Error("RecvMsgTransform中未处理的命令类型")
// 	}
// 	switch recvmsg.Payload.(type) {
// 	case []byte:
// 		data, err := UnpackMessageData(unpackcmd, recvmsg.Payload.([]byte))
// 		if err != nil {
// 			Log.Error("RecvMsgTransform 转换有效载荷数据失败:", err)
// 		}
// 		return data
// 	default:
// 		Log.Error("不能处理的有效载荷类型")
// 	}
// 	return nil
// }

//辅助类函数，找到没有被利用的端口和ip进行监听
func FindAvailablePort() (port int, error error) {
	start := start_Port
	rand.Seed(time.Now().UnixNano()) // 初始化随机种子
	delta := uint16(rand.Intn(9000)) // 生成一个范围在 [0, 9000) 的随机数
	start += int(delta)              // 将随机数加到 start_Port 上
	for port := start; port <= endPort; port++ {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			// 处理错误
			continue
		}

		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			// 端口已经被占用，继续尝试下一个端口
			continue
		}

		// 成功监听到端口，执行需要的操作
		defer listener.Close()

		// 这里可以返回找到的端口号等信息
		return port, nil
	}

	// 如果循环结束仍然没有找到合适的端口，则返回错误信息
	return 0, errors.New("无法找到可用端口")

}

//辅助类函数，返回本连接的ip和端口
func ReturnIpwithPort(conn net.Conn) (remoteIP string, remotePort int) {
	remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
	remoteIP = remoteAddr.IP.String()
	remotePort = remoteAddr.Port
	return remoteIP, remotePort
}

//辅助类函数
func GetLocalIp() (ip string, returnerr error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	// 遍历所有网络接口，查找符合条件的 IP 地址
	for _, iface := range ifaces {
		// 忽略 loopback 接口和非 up 状态的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取接口的地址信息
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 遍历接口的地址信息，查找符合条件的 IP 地址
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// 忽略非 IPv4 地址
			if ipnet.IP.To4() == nil {
				continue
			}

			// 判断 IP 地址是否在本地网络中
			if ipnet.Contains(net.ParseIP("127.0.0.1")) || ipnet.Contains(net.ParseIP("::1")) {
				continue
			}

			// 找到符合条件的 IP 地址，返回结果
			return ipnet.IP.String(), nil
		}
	}
	return "", errors.New("无法找到可用ip")
}

//关键函数，用来监听端口的后台程序
func ListenRun(addrout chan string) {
	//获得本地监听端口
	localip, _ := GetLocalIp()
	localport, _ := FindAvailablePort()
	addr := localip + ":" + strconv.Itoa(localport)
	Log.Info("节点获得ip监听地址:", addr)
	addrout <- addr
	overch := make(chan int)
	//等待监听结束才会return
	go startListening(strconv.Itoa(localport), overch)
	<-overch
	Log.Warn("监听协程已经停止")
	return
}
func Zookinit() {
	if Zookexit == 1 {
		return
	}
	var err error
	Zkconn, _, err = zk.Connect(hosts, time.Second*5)

	if err != nil {
		Log.Error("zook连接出错:", err)
		return
	} else {
		Log.Info("zook已经成功连接")
		Zookexit = 1
	}
}

func ZookRun(addres string) {
	Getmywallet(Zkconn)
	//这里并不会defer Zkconn.Close()防止出现变故
	//zookrun有义务维护本节点在zook中的心跳信息
	setLocalAddress(addres)
	addLocalAddressToZook()
	get()
}

func KeepWalletZk() {
	if Zkconn == nil {
		Log.Warn("zk没有初始化，需要先进入区块链网络！")
		return
	}
	//1. 首先
}

//拨号的入口，从zk中获取数据只会运行一次，并运行着维护与zk的心跳，当本函数退出后，其他新加入节点就无法获知本节点的地址
func DailRun() {
	dataMap, err := getChildrenWithPathValues()
	if err != nil {
		log.Fatalf("获取zook节点信息出错: %v", err)
	}
	var peerAddrs []string
	for node, value := range dataMap {
		fmt.Printf("打印所有path下的节点 ： %s: %s\n", node, value)
		peerAddrs = append(peerAddrs, value)
	}
	//waitgroup同步协程
	var Wg sync.WaitGroup
	//delPathAllChildren(zkconn)
	for _, value := range peerAddrs {
		Wg.Add(1)
		go startDialing(value, &Wg)
	}
	Wg.Wait()
	Log.Info("DailRun创建的协程已经全部结束")
}

//为防止广播风暴，这里暂时设计成只广播给我自身主动拨打的节点
func Broadcast_Listen() {

}

// 启动P2P监听
func startListening(port string, overch chan int) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	for {
		Log.Info("正在监听端口:", port)

		conn, err := listener.Accept()

		buf := make([]byte, 256)
		n, err := conn.Read(buf)
		if err != nil {
			Log.Error(err)
		}
		if n > 0 {
			fmt.Printf("收到消息:%s\np2p>", buf[:n])
		}
		//下面这几行在判断是否是自己拨打自己
		peerAddr := string(buf[:n])

		if strings.TrimSpace(peerAddr) == addStr {
			fmt.Println("来自本地环回拨号，退出")
			return
		}
		//没有办法在Accept前判断对方ip以及端口并选择性的断开
		//那么hasConnAddrsMap和锁就没有意义  这里还是加上锁吧
		//hasConnAddrsMu.Lock()
		//转换成string，并去掉空格和换行符
		_, exists := hasConnAddrsMap[peerAddr]
		if exists {
			//hasConnAddrsMu.Unlock()
			fmt.Println("接收到来自已经建立连接的节点拨号，退出")
			return
		}
		hasConnAddrsMap[peerAddr] = true

		Log.Info("收到来自", peerAddr, "的连接")
		defer conn.Close()
		go startConnWork(conn)
	}
	//这个并没有起作用  需要后面再改  或许直接程序退出就可以了
	overch <- 1
}

//监听建立连接后的控制主模块
func startConnWork(conn net.Conn) {
	// go writeConn(conn)
	// go readConn(conn)
	Log.Info("startConnWork")
	up := make(chan int)
	Down := make(chan int)
	//注册 当有新区块的时候会被提醒
	index := AddDialGoPool(up, Down)
	Log.Info("startConnWork工作协程等待被唤醒 ", index)
	getmsg := make(chan []byte)
	go GetSendTaskfromPool(Down, getmsg)
	recvmsgChan := make(chan *SendMessage)
	go recvResponse(conn, recvmsgChan)
	for {
		select {
		//有新区块需要广播
		case msg := <-getmsg:
			Log.Info("startConnWork已经被唤醒，触发发送")
			err := sendRequest(conn, msg)
			if err != nil {
				Log.Warn("广播中本节点发送失败:", err)
			} else {
				ip, port := ReturnIpwithPort(conn)
				Log.Info("向 ", ip, "：", port, " 发送信息成功")
			}
		case recvResMsg := <-recvmsgChan:
			Log.Debug(recvResMsg)
			switch recvResMsg.Cmd {
			case SendTX:
				Log.Info("接收到SendTX")
				ReceiveTxArr(recvResMsg)
			case SendCommonBlock:
				Log.Info("接收到SendCommonBlock")
				ReceiveBlockArr(recvResMsg)
			case SendMinedBlock:
				Log.Info("接收到MinedBlock广播的区块")
				ReceiveMinedBlock(recvResMsg)
			case SendGenesisBlock:
				Log.Info("接收到SendGenesisBlock")
			default:
				Log.Info("在startConnWork收到未命名命令:", recvResMsg.Cmd)
			}

		}

	}
	Log.Warn("本conn节点关闭")
	conn.Close()
}

// 堵塞，等待接收数据，未来需要抛弃
func readConn(conn net.Conn, overch chan int) {
	defer conn.Close()

	buf := make([]byte, 25600)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			Log.Error("read错误", err)
			overch <- 1
		}
		if n > 0 {
			fmt.Printf("收到消息:%s\np2p>", buf[:n])
		}
	}
}

// 写入连接数据，未来需要抛弃
func writeConn(conn net.Conn, overch chan int) {
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("p2p>")
		// 读取标准输入，以换行为读取标志
		data, _ := reader.ReadString('\n')
		_, err := conn.Write([]byte(data))
		if err != nil {
			overch <- 1
			//这个可能不会显示 因为一旦父协程读到就直接退出
			Log.Fatal("写入失败，对方已经断开连接！")
		}
	}
}

//辅助函数 包装请求区块和交易并发起命令的函数
func ZipWithToFetchBlocks(conn net.Conn) {
	req := PackMessage(FetchBlocks, "")

	error := sendRequest(conn, req)
	if error != nil {
		ip, port := ReturnIpwithPort(conn)
		Log.Info("向", ip, ":", port, "发送请求区块命令成功")
	}
}

//辅助函数
func ZipWithToFetchTX(conn net.Conn) {
	//""可以传送一个数组表示我现在有什么
	req := PackMessage(FetchTX, "")

	error := sendRequest(conn, req)
	if error != nil {
		ip, port := ReturnIpwithPort(conn)
		Log.Info("向", ip, ":", port, "发送请求区块命令成功")
	}
}

// func ZipWithToSendGenesisBlock(conn net.Conn, msg []byte) {
// 	error := sendRequest(conn, FetchBlocks, msg)
// 	if error != nil {
// 		ip, port := ReturnIpwithPort(conn)
// 		Log.Info("向", ip, ":", port, "发送创世区块成功")
// 	}
// }
// func ZipWithToSendCommonBlock(conn net.Conn, msg []byte) {
// 	error := sendRequest(conn, SendCommonBlock, msg)
// 	if error != nil {
// 		ip, port := ReturnIpwithPort(conn)
// 		Log.Info("向", ip, ":", port, "发送普通区块命令成功")
// 	}
// }
// func ZipWithToSendTX(conn net.Conn, msg []byte) {
// 	error := sendRequest(conn, SendTX, msg)
// 	if error != nil {
// 		ip, port := ReturnIpwithPort(conn)
// 		Log.Info("向", ip, ":", port, "发送普通交易命令成功")
// 	}
// }

//dail建立连接后的控制主模块
func startDailWork(conn net.Conn) {

	/*
		这个函数应该能做到：
		  1. 主动发起请求更新区块和交易（这个什么时候做呢？到达这个函数先做一次，以后收到一个命令就去做 ）
		  2. 监听来自其他节点区块和交易 并更新自己的区块和交易（这里可能有多种不同的请求命令）
		  3. 主动发起广播自己的区块和交易(不可以！！ 为了避免广播风暴设计成只有接电话的人能够广播自己的区块)

	*/
	//ZipWithToFetchBlocks(conn)
	recvmsgChan := make(chan *SendMessage)
	go recvResponse(conn, recvmsgChan)
	for {
		select {
		case recvResMsg := <-recvmsgChan:
			fmt.Println(recvResMsg)
			switch recvResMsg.Cmd {
			case SendTX:
				Log.Info("接收到SendTX")
				Log.Info("接收到SendTX")
				ReceiveTxArr(recvResMsg)
			case SendCommonBlock:
				Log.Info("接收到SendCommonBlock-")
				ReceiveBlockArr(recvResMsg)

			case SendGenesisBlock:
				Log.Info("接收到SendGenesisBlock")
			default:
				Log.Info("在startDailWork收到未命名命令:", recvResMsg.Cmd)
			}
		}
	}
}

// 对某个地址启动P2P拨号的入口函数
func startDialing(peerAddr string, Wg *sync.WaitGroup) {
	defer Wg.Done()
	if peerAddr == addStr {
		fmt.Println("跳过连接自己节点的过程")
		return
	}
	//需要判断是否已经连接 根据已连接的数组来判断
	hasConnAddrsMu.Lock()
	fmt.Println("打印已经连接的节点信息")
	for key, value := range hasConnAddrsMap {
		fmt.Println(key, value)
	}
	_, exists := hasConnAddrsMap[peerAddr]
	if exists {
		fmt.Println("连接已经建立，将退出本次建立过程")
		hasConnAddrsMu.Unlock()
		return
	}
	/*

	   rf.mu.Lock()
	   defer rf.mu.Unlock()
	*/
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {

		Log.Error("Dial出错", err)
		return
	}
	//连接已经建立
	hasConnAddrsMu.Unlock()

	defer conn.Close()

	conn.Write([]byte(addStr))

	fmt.Println("已连接到", conn.RemoteAddr())
	overReadCh1 := make(chan int)
	overWriteCh2 := make(chan int)
	go startConnWork(conn)
	// go readConn(conn, overReadCh1)
	// go writeConn(conn, overWriteCh2)
	select {
	//只要有一个就断开连接
	case <-overReadCh1:
		Log.Error("readConn已经断开")
		return
	case <-overWriteCh2:
		Log.Error("writeConn已经断开")
		return
	}

	Log.Error("startDialing 本连接即将断开.......")
	//这里是拨号后的逻辑
	time.Sleep(5 * time.Second)

	//告知父协程本协程已经结束

}
