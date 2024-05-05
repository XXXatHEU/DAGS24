package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var (
	path           = "/liveNode"
	addStr         string //127.0.0.1:9900的ip加端口形式
	finalPath      string //最终形成的在zookeeper里面的路径（有/的路径）
	localaddr      []byte //addStr byte[]形式  放在zook中节点的数据
	dialGetLocalip = "8.8.8.8:80"
	hosts          = []string{"192.168.201.16:2181"}
	//hosts          = []string{"192.144.220.80:2181"}
	Zkconn         *zk.Conn
)

func ListenPort() (port int, error error) {
	start := start_Port
	rand.Seed(time.Now().UnixNano()) // 初始化随机种子
	delta := uint16(rand.Intn(9000)) // 生成一个范围在 [0, 100) 的随机数
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

//TODO 这个地方最后应该设置为p2p能够建立起连接的ip和端口
//设置本机的ip和端口
func setLocalAddress(addr string) {
	// conn1, err := net.Dial("udp", dialGetLocalip)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// defer conn1.Close()

	// localAddr := conn1.LocalAddr().(*net.UDPAddr)
	// addStr := net.JoinHostPort(localAddr.IP.String(), strconv.Itoa(localAddr.Port))
	// finalPath = path + "/" + addStr
	// fmt.Println("生成的本机出口ip和端口是: ", addStr)
	// localaddr = []byte(addStr)
	addStr = addr
	finalPath = path + "/" + addStr
	localaddr = []byte(addStr)
	Log.Info("完成zook路径的初始化:", finalPath)
}

//递归的创建路径，如果不存在则先创建前面的路径
//递归的创建路径，如果不存在则先创建前面的路径
//0表示永久，1表示短暂
func createPathRecursively(zkConn *zk.Conn, path string, data []byte, mod int32) error {
	// 分割路径
	components := strings.Split(path, "/")
	var currPath string

	for index, component := range components {
		if component == "" {
			continue
		}
		currPath += "/" + component
		exists, _, err := zkConn.Exists(currPath)
		if err != nil {
			return fmt.Errorf("failed to check existence of %s: %v", currPath, err)
		}

		//对于最后一个节点
		if index == len(components)-1 {
			if !exists {
				zkConn.Create(currPath, nil, mod, zk.WorldACL(zk.PermAll))
				if err != nil {
					return fmt.Errorf("failed to create %s: %v", currPath, err)
				}
				_, err := zkConn.Set(path, data, -1)
				if err != nil {
					return fmt.Errorf("failed to set data for %s: %v", path, err)
				}
				return nil
			} else {
				return fmt.Errorf("failed to create %s: %v", currPath, err)
			}
		} else { //对于其他，那么直接创建路径就可以

			if !exists {
				zkConn.Create(currPath, nil, mod, zk.WorldACL(zk.PermAll))
				if err != nil {
					return fmt.Errorf("failed to create %s: %v", currPath, err)
				}
			}
		}

	}

	// 在最终节点处设置数据
	//_, err := zkConn.Set(path, data, -1)
	return nil
}

// 增
func addLocalAddressToZook() error {

	//setLocalAddress()

	error := createPathRecursively(Zkconn, finalPath, localaddr, 1)
	if error != nil {
		Log.Fatal("创建分支出错:", error)
		return error
	}
	return nil
	// var flags int32 = 0
	// // 获取访问控制权限
	// acls := zk.WorldACL(zk.PermAll)
	// s, err := conn.Create(finalPath, localaddr, flags, acls)
	// if err != nil {
	// 	fmt.Printf("创建失败: %v\n", err)
	// 	return fmt.Errorf("向zookeeper添加本节点ip失败: %v", err)
	// }
	// fmt.Printf("创建: %s 成功", s)
	// return nil
}

// 查  没用到
func get() {
	data, _, err := Zkconn.Get(finalPath)
	if err != nil {
		fmt.Printf("查询%s失败, err: %v\n", finalPath, err)
		return
	}
	Log.Info(finalPath, "的值为", string(data))
}
func getChildren(zkConn *zk.Conn, nodepath string) (map[string]string, error) {
	children, _, err := zkConn.Children(nodepath)
	if err != nil {
		return nil, fmt.Errorf("failed to get children of %s: %v", nodepath, err)
	}
	dataMap := make(map[string]string)
	for _, child := range children {
		data, _, err := zkConn.Get(nodepath + "/" + child)
		if err != nil {
			return nil, fmt.Errorf("failed to get data of %s/%s: %v", nodepath, child, err)
		}
		dataMap[child] = string(data)
	}
	return dataMap, nil
}

//查询所有节点  会返回一个map
func getChildrenWithPathValues() (map[string]string, error) {
	children, _, err := Zkconn.Children(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get children of %s: %v", path, err)
	}
	dataMap := make(map[string]string)
	for _, child := range children {
		data, _, err := Zkconn.Get(path + "/" + child)
		if err != nil {
			return nil, fmt.Errorf("failed to get data of %s/%s: %v", path, child, err)
		}
		dataMap[child] = string(data)
	}
	return dataMap, nil
}

// 删改与增不同在于其函数中的version参数,其中version是用于 CAS支持
// 可以通过此种方式保证原子性
// 改 改就没有必要了  没用到
func modify(conn *zk.Conn) {

	new_data := []byte("hello zookeeper")
	_, sate, _ := conn.Get(finalPath)
	_, err := conn.Set(finalPath, new_data, sate.Version)
	if err != nil {
		fmt.Printf("数据修改失败: %v\n", err)
		return
	}
	fmt.Println("数据修改成功")
}

// 删本节点
func delLocalAddress(conn *zk.Conn) {
	_, sate, _ := conn.Get(finalPath)
	err := conn.Delete(finalPath, sate.Version)
	if err != nil {
		fmt.Printf("数据删除失败: %v\n", err)
		return
	}
	fmt.Println("数据删除成功")
}

func delPathAllChildren(conn *zk.Conn) {
	children, _, err := conn.Children(path)
	if err != nil {
		fmt.Println("获取子节点失败:", err)
		return
	}

	for _, child := range children {
		err = conn.Delete(path+"/"+child, -1)
		if err != nil {
			fmt.Println("删除子节点失败:", err)
		}
	}
}

/*
	本程序主要有几个函数
	 1.setLocalAddress  通过拨号 查到本地的出口ip和端口，这个需要改成区块链的p2p服务的ip和端口
	 2.createPathRecursively 根据传入的path创建节点，如果路径不存在那么会递归的调用创建路径
	    然后将节点值放到上面
	 3.addLocalToZook会调用第一个函数获取ip和端口并调用第二个函数插入数据
	 4.getChildrenWithPathValues获取path下的所有节点，并返回一个map
	 5.delLocalAddress调用后会将finalpath删除

	 注意：关于创建的节点值是否需要维持生命需要斟酌
	       创建连接地址的需要处理


*/
func addZknodefromFile(conn *zk.Conn, nodePath string, nodevalue string, filepath string) {
	lines, err := readTxtFile(filepath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, line := range lines {
		fmt.Println(line)
		fmt.Println("插入zk路径为：", nodePath+"/"+line)
		error := createPathRecursively(conn, nodePath+"/"+line, []byte(line), 0)
		if error != nil {
			fmt.Println("addnode插入数据失败")
		}
	}

}

//会创建路径  在往里添加初始值（wallet的address和name）的时候使用这个
func AddNodeToZK(conn *zk.Conn, nodePath string, nodevalue []byte, mod int32) error {

	fmt.Println(nodePath)
	fmt.Println("插入zk路径为：", nodePath)
	error := createPathRecursively(conn, nodePath, nodevalue, mod)
	if error != nil {
		fmt.Println("addnode插入数据失败:", error)
		return error
	}
	return nil
}
func main1() {

	// 创建zk连接地址

	// 连接zk
	conn, _, err := zk.Connect(hosts, time.Second*5)
	defer conn.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 添加本地节点
	addLocalAddressToZook()

	// 查询 path 目录下的所有节点及其值
	dataMap, err := getChildrenWithPathValues()
	if err != nil {
		log.Fatalf("failed to get children: %v", err)
	}

	// 输出节点和对应值到控制台
	for node, value := range dataMap {
		fmt.Printf("打印所有path下的节点 ： %s: %s\n", node, value)
	}
}
