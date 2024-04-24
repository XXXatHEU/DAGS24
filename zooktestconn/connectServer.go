package main

import (
	"bufio"
	"crypto/elliptic"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var (
	path           = "/liveNode"
	addStr         string
	finalPath      string //最终形成的在zookeeper里面的路径
	localaddr      []byte
	dialGetLocalip = "8.8.8.8:80"
	mywallet       MyWallet
)

func init() {
	mywallet = MyWallet{}
	gob.Register(elliptic.P256())
}

//TODO 这个地方最后应该设置为p2p能够建立起连接的ip和端口
//设置本机的ip和端口
func setLocalAddress() {
	conn1, err := net.Dial("udp", dialGetLocalip)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn1.Close()

	localAddr := conn1.LocalAddr().(*net.UDPAddr)
	addStr := net.JoinHostPort(localAddr.IP.String(), strconv.Itoa(localAddr.Port))
	finalPath = path + "/" + addStr
	fmt.Println("生成的本机出口ip和端口是: ", addStr)
	localaddr = []byte(addStr)
}

//递归的创建路径，如果不存在则先创建前面的路径
//0表示永久，1表示短暂
func createPathRecursively(zkConn *zk.Conn, path string, data []byte, mod int32) error {
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
			zkConn.Create(currPath, nil, mod, zk.WorldACL(zk.PermAll))
			if err != nil {
				return fmt.Errorf("failed to create %s: %v", currPath, err)
			}
		}
	}

	// 在最终节点处设置数据
	//_, err := zkConn.Set(path, data, -1)
	_, err := zkConn.Set(path, data, -1)
	if err != nil {
		return fmt.Errorf("failed to set data for %s: %v", path, err)
	}
	return nil
}

// 增
func addLocalToZook(conn *zk.Conn) error {
	// flags有4种取值：
	// 0:永久，除非手动删除
	// zk.FlagEphemeral = 1:短暂，session断开则该节点也被删除
	// zk.FlagSequence  = 2:会自动在节点后面添加序号
	// 3:Ephemeral和Sequence，即，短暂且自动添加序号
	//setLocalAddress()

	setLocalAddress()
	error := createPathRecursively(conn, finalPath, localaddr, 1)
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

// 删
func delLocalAddress(conn *zk.Conn) {
	_, sate, _ := conn.Get(finalPath)
	err := conn.Delete(finalPath, sate.Version)
	if err != nil {
		fmt.Printf("数据删除失败: %v\n", err)
		return
	}
	fmt.Println("数据删除成功")
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
func main() {

	// 创建zk连接地址
	hosts := []string{"192.144.220.80:2181"}
	// 连接zk
	conn, _, err := zk.Connect(hosts, time.Second*5)
	defer conn.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 添加本地节点
	//addLocalToZook(conn)

	// 查询 path 目录下的所有节点及其值
	// dataMap, err := getChildren(conn, nodepath)
	// if err != nil {
	// 	log.Fatalf("failed to get children: %v", err)
	// }

	// 输出节点和对应值到控制台
	// for node, value := range dataMap {
	// 	fmt.Printf("打印所有path下的节点 ： %s: %s\n", node, value)
	// }

	addZknodefromFile(conn, "/Name/all", "", "name.txt")
	updateWalletAll(conn)
	Getmywallet(conn)
}
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
		fmt.Println("addnode插入数据失败")
		return error
	}
	return nil
}

type Wallet_zk struct {
	Address string `json:"Address"`
	//PriKey  *ecdsa.PrivateKey `json:"PriKey"`
	Wallet []byte `json:"Wallet"` //需要进一步转化
	Name   string `json:"Name"`
}

type MyWallet struct {
	Address string
	walle   *wallet
	Name    string
	myvm    *WalletManager
}

//重要  从zk中获得可以使用的账户
func Getmywallet(conn *zk.Conn) {
	mp, _ := getChildren(conn, "/Wallet/all")
	var walletaddres string
	for key, _ := range mp {
		fmt.Println("key:", key)
		err := AddNodeToZK(conn, "/Wallet/use/"+key, nil, 1)
		if err == nil {
			walletaddres = key
			break
		}
	}
	var useNanme string
	mp2, _ := getChildren(conn, "/Wallet-Name")
	if _, ok := mp2[walletaddres]; !ok { //如果没有的话
		mp, _ := getChildren(conn, "/Name/all")
		for key, _ := range mp {

			err := AddNodeToZK(conn, "/Name/use/"+key, []byte(walletaddres), 0)
			if err == nil {
				useNanme = key
				break
			}
		}
		//将这个放到/Wallet-Name中
		AddNodeToZK(conn, "/Wallet-Name/"+walletaddres, []byte(useNanme), 0)
	} else {
		useNanme = mp2[walletaddres]
	}
	walletjson := mp[walletaddres]
	fmt.Println("address:", walletaddres, "name:", useNanme)

	/*
		1. 将json转换成Wallet_zk
		2. 将Wallet_zk转换成MyWallet
	*/
	wallet_zk2, _ := JsonToWallet_zk([]byte(walletjson))
	mywallet.Address = walletaddres
	mywallet.Name = useNanme
	mywallet.walle = WalletfromByte(wallet_zk2.Wallet)
	mywallet.myvm = GetMyWalletManager(*mywallet.walle, mywallet.Address)
	fmt.Println("mywallet.wm.listAddresses():", mywallet.myvm.listAddresses())

}

//辅助工具类
func Wallet_zkToJson(wallet_zk Wallet_zk) []byte {
	jsonData, err := json.Marshal(wallet_zk)
	if err != nil {
		fmt.Println("Wallet_zkToJson 转换struct到json格式失败:", err)
		return []byte(err.Error())
	}
	return jsonData
}

//辅助工具类
func JsonToWallet_zk(jsonData []byte) (wallet_zk2 Wallet_zk, err2 error) {

	err2 = json.Unmarshal([]byte(jsonData), &wallet_zk2)
	if err2 != nil {
		fmt.Println("转换json到struct格式失败:", err2)
		return
	}
	return wallet_zk2, nil
}

/*
	1.需要将wallet读成二进制
	2.将二进制放到struct里面
	3.将strcut转换成json
	4.将json放到zk中

*/
func updateWalletAll(conn *zk.Conn) {
	wm := NewWalletManager()
	if wm == nil {
		fmt.Println(" NewWalletManager 失败!")
		return
	}

	context := wm.WalletByte()
	wm2 := ZkNewWalletManager()
	wm2.WalletManagerfromByte(context)

	for address, _ := range wm.Wallets {
		wallet_zk := Wallet_zk{
			Address: address,
			Wallet:  GetWalletByte(*wm.Wallets[address]),
		}
		prepath := "/Wallet/all"
		AddNodeToZK(conn, prepath+"/"+address, Wallet_zkToJson(wallet_zk), 0)
	}

	//排序, 升序

}

func readTxtFile(filePath string) ([]string, error) {
	// 打开txt文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	// 逐行读取文件内容
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
