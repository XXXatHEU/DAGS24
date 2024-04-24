package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

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
