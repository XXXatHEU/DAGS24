package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

func uintToByte(num uint64) []byte {
	var buffer bytes.Buffer
	//使用二进制编码
	// Write(w io.Writer, order ByteOrder, data interface{}) error
	err := binary.Write(&buffer, binary.LittleEndian, &num)
	if err != nil {
		fmt.Println("binary.Write err :", err)
		return nil
	}

	return buffer.Bytes()
}

//判断文件是否存在
func isFileExist(filename string) bool {
	// func Stat(name string) (FileInfo, error) {
	_, err := os.Stat(filename)

	//os.IsExist不要使用，不可靠
	if os.IsNotExist(err) {
		return false
	}

	return true
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

func Util(zkConn *zk.Conn) {
	//将本地的名字上传
	addZknodefromFile(zkConn, "/Name/all", "", "name.txt")
	//更新本地钱包
	updateWalletAll(zkConn)
}

func pubkeyhashToAddress(str []byte) (raddress string, err error) {
	wm := NewWalletManager()
	if wm == nil {
		fmt.Println(" NewWalletManager 失败!")
		return
	}
	for address, wallet_ := range wm.Wallets {
		//priKey := wallet_.PriKey //私钥签名阶段使用，暂且注释掉
		pubKey := wallet_.PubKey
		//我们的所有output都是由公钥哈希锁定的，所以去查找付款人能够使用的output时，也需要提供付款人的公钥哈希值
		pubKeyHash := getPubKeyHashFromPubKey(pubKey)
		if bytes.Equal(pubKeyHash, str) {
			raddress = address
			return
		}
	}
	err = fmt.Errorf("没找到")
	return
}

var WalletMap = make(map[string]MyWallet)

//从区块中交易里面拿到wallet
func getWalletFromTx(txPubKeyHash []byte) (txWallet MyWallet, err error) {
	walletget, ok := WalletMap[string(txPubKeyHash)]
	if ok {
		return walletget, nil
	}
	wm := NewWalletManager()

	//var txWallet MyWallet
	if wm == nil {
		fmt.Println(" NewWalletManager 失败!")
		return
	}
	for address, wallet_ := range wm.Wallets {
		//priKey := wallet_.PriKey //私钥签名阶段使用，暂且注释掉
		pubKey := wallet_.PubKey
		pubKeyHash := getPubKeyHashFromPubKey(pubKey)
		if bytes.Equal(pubKeyHash, txPubKeyHash) {
			txWallet.Address = address
			txWallet.walle = wallet_
			txWallet.myvm = wm
			txWallet.Name = "getWalletFromTx函数虚拟生成"
			WalletMap[string(txPubKeyHash)] = txWallet
			return
		}
	}
	err = fmt.Errorf("没找到")
	return
}

func randomLineFromFile() (string, error) {
	filename := "./msg.txt"
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := []string{}

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("file '%s' is empty", filename)
	}

	rand.Seed(time.Now().UnixNano())
	randomLine := lines[rand.Intn(len(lines))]
	return randomLine, nil
}

func splitMapIntoArrays(inputMap map[string][]byte, numArrays int) []map[string][]byte {
	// 计算 map 的长度
	mapLength := len(inputMap)
	// 计算每个切片的大小
	sliceSize := numArrays
	if mapLength < numArrays {
		sliceSize = 1
	}

	// 创建存储切片的一维切片
	result := make([]map[string][]byte, numArrays)

	// 初始化每个切片
	for i := 0; i < numArrays; i++ {
		result[i] = make(map[string][]byte, sliceSize)
	}

	// 将 map 数据划分到切片中
	index := 0
	for key, value := range inputMap {
		// 计算当前键值对应该放在哪个切片
		sliceIndex := index % sliceSize
		// 创建 map（如果尚未创建）
		// if result[sliceIndex] == nil {
		// 	result[sliceIndex] = make(map[string][]byte, sliceSize)
		// }
		// 将键值对存入切片
		result[sliceIndex][key] = value
		index++
	}

	return result
}

func test22222() {
	inputMap := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
		"key4": []byte("value4"),
		"key5": []byte("value5"),
		"key6": []byte("value6"),
	}

	// 将 map 切分成 3 个切片
	numArrays := 20
	slices := splitMapIntoArrays(inputMap, numArrays)

	// 打印每个切片的内容
	for i, slice := range slices {
		fmt.Printf("Slice %d:\n", i)
		for key, value := range slice {
			fmt.Printf("  %s: %s\n", key, string(value))
		}
		fmt.Println()
	}
}

func SetExpSouceid() (number int) {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(blockchainDBFile)
	socuidmap := make(map[int]string)
	socuidmap[128] = "7df4e90f-0781-4e31-b53d-a10c77a81361"
	socuidmap[256] = "9556f3ce-a4cb-48c8-a366-15fd89826345"
	socuidmap[512] = "a49dc293-dbf1-41e2-aacf-e4677d6dcc8c"
	socuidmap[1024] = "510a03b3-2a4a-4a3e-9e90-8bcebbeb5c69"
	socuidmap[2048] = "ba02296f-57c0-460e-9763-a4bc456ab83b"
	if len(matches) > 1 {
		// 提取到的数字在第一个匹配组中
		numberStr := matches[1]

		// 将提取到的数字字符串转换为整数
		number, err := strconv.Atoi(numberStr)
		EXPSouceID = socuidmap[number]
		if err == nil {
			fmt.Printf("区块内交易数量是: %d,选取倒数一万个区块的socuid%s\n", number, EXPSouceID)
		} else {
			fmt.Println("无法将数字字符串转换为整数")
		}
	} else {
		fmt.Println("未找到数字")
	}
	return
}

//缩短区块链到指定实验长度
func truncateChain() {
	bc, err := GetBlockChainInstance()
	if err != nil {
		Log.Debug("print err:", err)
		return
	}
	count := 0
	//调用迭代器，输出blockChain
	it := bc.NewIterator()
	for {
		count++
		fmt.Println(count)
		//调用Next方法，获取区块，游标左移
		block := it.MemoryNextBlock()

		//退出条件
		if block.PrevHash == nil {
			Log.Debug("区块链遍历结束!")
			break
		}
	}
	bc.closeDB()
	bc2, err := GetBlockChainInstance()
	if err != nil {
		Log.Debug("print err:", err)
		return
	}
	defer bc2.closeDB()
	//调用迭代器，输出blockChain
	it2 := bc2.NewIterator()
	//如果区块长度大于最大实验长度，那么就进行截取
	if count > MaxExpBlockLen {
		var block2 *Block
		truncateLen := count - MaxExpBlockLen
		count1 := 0
		for {
			count1++
			fmt.Println(count)
			//调用Next方法，获取区块，游标左移
			block2 = it2.MemoryNextBlock()
			if count1 == truncateLen {
				break
			}
		}
		err2 := createOrUpdateKeyValue([]byte(lastBlockHashKey), block2.Hash)
		if err2 != nil {
			err = err2
			return
		}
	}
}
