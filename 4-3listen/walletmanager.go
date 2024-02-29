package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/samuel/go-zookeeper/zk"
)

var (
	mywallet MyWallet
)

// 负责对外，管理生成的钱包（公钥私钥）
//私钥1->公钥-》地址1
//私钥2->公钥-》地址2
//私钥3->公钥-》地址3
//私钥4->公钥-》地址4
type WalletManager struct {
	//定义一个map来管理所有的钱包
	//key:地址
	//value:wallet结构(公钥，私钥)
	Wallets map[string]*wallet
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

//创建walletManager结构
func NewWalletManager() *WalletManager {
	//创建一个, Wallets map[string]*wallet
	var wm WalletManager
	gob.Register(elliptic.P256())
	//分配空间，一定要分配，否则没有空间
	wm.Wallets = make(map[string]*wallet)

	//从本地加载已经创建的钱包,写入Wallets结构
	if !wm.loadFile() {
		return nil
	}

	//返回 walletManager
	return &wm
}

func ZkNewWalletManager() *WalletManager {
	//创建一个, Wallets map[string]*wallet
	var wm WalletManager

	//分配空间，一定要分配，否则没有空间
	wm.Wallets = make(map[string]*wallet)

	//返回 walletManager
	return &wm
}

func (wm *WalletManager) createWallet() string {
	// 创建秘钥对
	w := newWalletKeyPair()
	if w == nil {
		fmt.Println("newWalletKeyPair 失败!")
		return ""
	}

	// 获取地址
	address := w.getAddress()

	//把地址和wallet写入map中: Wallets map[string]*wallet
	wm.Wallets[address] = w //<<<--------- 重要

	// 将秘钥对写入磁盘
	if !wm.saveFile() {
		return ""
	}

	// 返回给cli新地址
	return address

}

const walletFile = "wallet.dat"

func (wm *WalletManager) saveFile() bool {
	//data ????
	//使用gob对wm进行编码
	var buffer bytes.Buffer

	//未注册接口函数
	// encoder.Encode err: gob: type not registered for interface: elliptic.p256Curve
	//注册一下接口函数，这样gob才能够正确的编码
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(wm)

	if err != nil {
		fmt.Println("encoder.Encode err:", err)
		return false
	}

	//将walletManager写入磁盘
	err = ioutil.WriteFile(walletFile, buffer.Bytes(), 0600)
	if err != nil {
		fmt.Println("ioutil.WriteFile err:", err)
		return false
	}
	return true
}
func (wm *WalletManager) WalletByte() []byte {
	//data ????
	//使用gob对wm进行编码
	var buffer bytes.Buffer

	//未注册接口函数
	// encoder.Encode err: gob: type not registered for interface: elliptic.p256Curve
	//注册一下接口函数，这样gob才能够正确的编码
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(wm)

	if err != nil {
		fmt.Println("encoder.Encode err:", err)

	}
	return buffer.Bytes()
}

//只是单纯的从wallet中获得byte
func GetWalletByte(wa wallet) []byte {
	//data ????
	//使用gob对wm进行编码
	var buffer bytes.Buffer

	//未注册接口函数
	// encoder.Encode err: gob: type not registered for interface: elliptic.p256Curve
	//注册一下接口函数，这样gob才能够正确的编码
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(wa)

	if err != nil {
		fmt.Println("encoder.Encode err:", err)

	}
	return buffer.Bytes()
}
func WalletfromByte(content []byte) *wallet {

	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(content))

	//解密赋值，赋值给wm:====>map

	var walle = &wallet{
		PriKey: nil,
		PubKey: []byte{},
	}

	err := decoder.Decode(walle)
	if err != nil {
		fmt.Println("WalletfromByte 解析出错 将退出程序")
		os.Exit(1)
		return walle
	}
	return walle
}

//读取wallet.dat文件，加载wm中
func (wm *WalletManager) loadFile() bool {
	//判断文件是否存在
	if !isFileExist(walletFile) {
		fmt.Println("文件不存在,无需加载!")
		return true
	}

	//读取文件
	content, err := ioutil.ReadFile(walletFile)
	if err != nil {
		fmt.Println("ioutil.ReadFile err:", err)
		return false
	}

	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(content))

	//解密赋值，赋值给wm:====>map
	err = decoder.Decode(wm)
	if err != nil {
		fmt.Println("decoder.Decode err:", err)
		return false
	}
	return true
}

//读取wallet.dat文件，加载wm中
func (wm *WalletManager) WalletManagerfromByte(content []byte) bool {

	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(content))

	//解密赋值，赋值给wm:====>map
	err := decoder.Decode(wm)
	if err != nil {
		fmt.Println("WalletManagerfromByte decoder.Decode err:", err)
		return false
	}
	return true
}

func (wm *WalletManager) listAddresses() []string {
	var addresses []string
	for address := range wm.Wallets {
		addresses = append(addresses, address)
	}

	//排序, 升序
	sort.Strings(addresses)

	return addresses
}

func GetMyWalletManager(wall wallet, str string) *WalletManager {
	var wm WalletManager

	//分配空间，一定要分配，否则没有空间
	wm.Wallets = make(map[string]*wallet)
	wm.Wallets[str] = &wall

	//返回 walletManager
	return &wm
}

//重要  从zk中获得可以使用的账户  看图的介绍
func Getmywallet(conn *zk.Conn) {
	//拿到所有的账户
	mp, _ := getChildren(conn, "/Wallet/all")
	var walletaddres string
	//遍历所有的账户
	for key, _ := range mp {
		fmt.Println("key:", key)
		//如果某个值能插入到zk中，说明没有使用 并且标明自己占用  然后打破循环
		err := AddNodeToZK(conn, "/Wallet/use/"+key, nil, 1)
		if err == nil {
			walletaddres = key
			break
		}
	}
	var useNanme string
	//从Wallet-Name有账户和公司名称的映射
	mp2, _ := getChildren(conn, "/Wallet-Name")
	if _, ok := mp2[walletaddres]; !ok { //如果没有的话  就从Name拿到所有  尝试去创建一个映射
		mp, _ := getChildren(conn, "/Name/all")
		for key, _ := range mp { //还是尝试插入，如果插入成功说明就可以使用这个名字了

			err := AddNodeToZK(conn, "/Name/use/"+key, []byte(walletaddres), 0)
			if err == nil {
				useNanme = key
				break
			}
		}
		//将这个放到/Wallet-Name中
		AddNodeToZK(conn, "/Wallet-Name/"+walletaddres, []byte(useNanme), 0)
	} else { //如果有的话直接使用
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
	Log.Info("已经获得了本地钱包，本地钱包地址：", mywallet.myvm.listAddresses())
	Log.Info("本地钱包所属：", mywallet.Name)
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
		Log.Warn(" NewWalletManager 失败!")
		return
	}

	context := wm.WalletByte()
	wm2 := ZkNewWalletManager()
	wm2.WalletManagerfromByte(context)

	for address, wallet := range wm.Wallets {
		wallet_zk := Wallet_zk{
			Address: address,
			Wallet:  GetWalletByte(*wm.Wallets[address]),
		}
		prepath := "/Wallet/all"
		AddNodeToZK(conn, prepath+"/"+address, Wallet_zkToJson(wallet_zk), 0)
		pubkey := "/Wallet/address-hashpubkey"
		AddNodeToZK(conn, pubkey+"/"+address, wallet.PubKey, 0)
	}

	//排序, 升序

}
