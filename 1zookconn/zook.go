package main

import (
	"fmt"
	"log"
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
	hosts          = []string{"192.144.220.80:2181"}
)

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

}

//递归的创建路径，如果不存在则先创建前面的路径
func createPathRecursively(zkConn *zk.Conn, path string, data []byte) error {
	// 分割路径
	components := strings.Split(path, "/")
	var currPath string

	for _, component := range components {
		if component == "" {
			continue
		}
		currPath += "/" + component
		exists, _, err := zkConn.Exists(currPath)
		if err != nil {
			return fmt.Errorf("failed to check existence of %s: %v", currPath, err)
		}
		if !exists {
			fmt.Println("路径不存在，即将创建路径")
			// flags有4种取值：
			// 0:永久，除非手动删除
			// zk.FlagEphemeral = 1:短暂，session断开则该节点也被删除
			// zk.FlagSequence  = 2:会自动在节点后面添加序号
			// 3:Ephemeral和Sequence，即，短暂且自动添加序号
			_, err := zkConn.Create(currPath, nil, 1, zk.WorldACL(zk.PermAll))
			if err != nil {
				return fmt.Errorf("failed to create %s: %v", currPath, err)
			}
		}
	}

	// 在最终节点处设置数据
	fmt.Println("正在向zook中插入节点 path: ", path, "data: ", string(data))
	_, err := zkConn.Set(path, data, -1)
	if err != nil {
		return fmt.Errorf("failed to set data for %s: %v", path, err)
	}
	return nil
}

// 增
func addLocalAddressToZook(conn *zk.Conn) error {

	//setLocalAddress()

	error := createPathRecursively(conn, finalPath, localaddr)
	if error != nil {
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
func get(conn *zk.Conn) {
	data, _, err := conn.Get(finalPath)
	if err != nil {
		fmt.Printf("查询%s失败, err: %v\n", finalPath, err)
		return
	}
	fmt.Printf("%s 的值为 %s\n", finalPath, string(data))
}

//查询所有节点  会返回一个map
func getChildrenWithPathValues(zkConn *zk.Conn) (map[string]string, error) {
	children, _, err := zkConn.Children(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get children of %s: %v", path, err)
	}
	dataMap := make(map[string]string)
	for _, child := range children {
		data, _, err := zkConn.Get(path + "/" + child)
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
	addLocalAddressToZook(conn)

	// 查询 path 目录下的所有节点及其值
	dataMap, err := getChildrenWithPathValues(conn)
	if err != nil {
		log.Fatalf("failed to get children: %v", err)
	}

	// 输出节点和对应值到控制台
	for node, value := range dataMap {
		fmt.Printf("打印所有path下的节点 ： %s: %s\n", node, value)
	}
}
