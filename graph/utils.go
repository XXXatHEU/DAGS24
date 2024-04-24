package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
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
