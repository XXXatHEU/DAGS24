package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var (
	start_Port      = 9000
	endPort         = 11000
	hasConnAddrsMap = map[string]bool{} //已经连接的地址 哈希表
	hasConnAddrsMu  sync.Mutex          //哈希表操作的锁
)

// type peernodeList struct {
// 	mu           sync.Mutex
// 	CommandValid bool
// 	Command      interface{}
// 	CommandIndex int

// 	// For 2D:
// 	SnapshotValid bool
// 	Snapshot      []byte
// 	SnapshotTerm  int
// 	SnapshotIndex int
// }

func FindAvailablePort() (port int, error error) {
	for port := start_Port; port <= endPort; port++ {
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
func main() {
	// lablog.init()

	// serverId := 1
	// lablog.Debug(serverId, lablog.Server, "这是一条服务器测试日志，服务器 ID 为 %d", serverId)

	// gid := 1
	// serverId1 := 2
	// lablog.ShardDebug(gid, serverId1, lablog.Commit, "这是一条分片日志，分片 ID 为 %d，服务器 ID 为 %d", gid, serverId)

	//获取本节点的ip和端口
	localip, _ := GetLocalIp()
	fmt.Println("本机ip为：", localip)
	localport, _ := FindAvailablePort()
	fmt.Println("可用端口为：", localport)
	addr := localip + ":" + strconv.Itoa(localport)

	//获取到立即占用这个端口  注意：这里应该将注册到zook放在监听之后，防止失效
	go startListening(strconv.Itoa(localport))
	//Work()
	//向zook上注册本机ip加端口
	zkconn, _, err := zk.Connect(hosts, time.Second*5)

	if err != nil {
		fmt.Println("zook连接出错:", err)
		return
	}
	time.Sleep(3 * time.Second)
	setLocalAddress(addr)
	addLocalAddressToZook(zkconn)
	time.Sleep(3 * time.Second)
	dataMap, err := getChildrenWithPathValues(zkconn)
	if err != nil {
		log.Fatalf("获取zook节点信息出错: %v", err)
	}

	// 输出节点和对应值到控制台
	var peerAddrs []string
	for node, value := range dataMap {
		fmt.Printf("打印所有path下的节点 ： %s: %s\n", node, value)
		peerAddrs = append(peerAddrs, value)
	}

	//delPathAllChildren(zkconn)
	for _, value := range peerAddrs {
		fmt.Println(value)
		go startDialing(value)
	}

	defer zkconn.Close()

	//下面模拟主线程的工作
	time.Sleep(120 * time.Second)
}
func Work() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入要监听的端口号：")
	port, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	// 去除端口号中的换行符和空格
	port = strings.TrimSpace(port)

	// 启动P2P监听
	go startListening(port)
	//time.Sleep(2 * time.Second)
	fmt.Print("请输入对方的IP地址和端口号（例如：127.0.0.1:8000）：")
	peerAddr, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	// 去除对方IP地址和端口号中的换行符和空格
	peerAddr = strings.TrimSpace(peerAddr)

	// 启动P2P拨号
	startDialing(peerAddr)
}

// 启动P2P监听
func startListening(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()
	for {
		fmt.Println("正在监听端口:", port)

		conn, err := listener.Accept()

		buf := make([]byte, 256)
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		if n > 0 {
			fmt.Printf("收到消息:%s\np2p>", buf[:n])
		}
		peerAddr := string(buf[:n])

		if err != nil {
			log.Fatal(err)
		}

		if strings.TrimSpace(peerAddr) == addStr {
			fmt.Println("来自本地环回拨号，退出")
			return
		}
		//没有办法在Accept前判断对方ip以及端口并选择性的断开
		//那么hasConnAddrsMap和锁就没有意义  这里还是加上锁吧
		hasConnAddrsMu.Lock()
		//转换成string，并去掉空格和换行符
		_, exists := hasConnAddrsMap[peerAddr]
		if exists {
			hasConnAddrsMu.Unlock()
			fmt.Println("接收到来自已经建立连接的节点拨号，退出")
			return
		}
		hasConnAddrsMap[peerAddr] = true
		hasConnAddrsMu.Unlock()
		fmt.Println("收到来自", peerAddr, "的连接")
		startConnWork(conn)
	}
}

func startConnWork(conn net.Conn) {
	go writeConn(conn)
	go readConn(conn)
	time.Sleep(120 * time.Second)
}

// 读取连接数据
func readConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 256)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		if n > 0 {
			fmt.Printf("收到消息:%s\np2p>", buf[:n])
		}
	}
}

// 写入连接数据
func writeConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("p2p>")
		// 读取标准输入，以换行为读取标志
		data, _ := reader.ReadString('\n')
		conn.Write([]byte(data))
	}
}

// 启动P2P拨号
func startDialing(peerAddr string) {

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
		log.Fatal(err)
	}
	//连接已经建立
	hasConnAddrsMu.Unlock()

	defer conn.Close()

	conn.Write([]byte(addStr))

	fmt.Println("已连接到", conn.RemoteAddr())

	go readConn(conn)
	go writeConn(conn)

	//这里是拨号后的逻辑
	time.Sleep(120 * time.Second)

}
