package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

type ProductType struct {
	SourceID    uuid.UUID //溯源id
	StartTxid   []byte    //溯源产品的交易id（这个产品就会是新生成的）
	ProductData []byte    //溯源产品的信息
	Pubkeyhash  []byte    //冗余信息 归属人
}

func (product *ProductType) setProductHash() error {
	//对tx做gob编码得到字节流，做sha256，赋值给TXID
	var buffer bytes.Buffer

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(product)
	if err != nil {
		fmt.Println("encode err:", err)
		return err
	}

	hash := sha256.Sum256(buffer.Bytes())

	//我们使用tx字节流的哈希值作为交易id
	product.StartTxid = hash[:]
	return nil
}

func createProduct(ProductData []byte, Pubkeyhash []byte) ProductType {
	var product ProductType
	sourceID := uuid.New()
	product.SourceID = sourceID
	err2 := product.setProductHash()
	if err2 != nil {
		Log.Fatal("创建一个起始区块的product时进行settxid出错:", err2)
	}

	if ProductData == nil {
		ProductData = []byte("由于没有输入信息，因此这是模拟生成的交易信息")
	}
	product.ProductData = ProductData
	//这里我设计失误了
	// if Pubkeyhash == nil {
	// 	var err error
	// 	Pubkeyhash, err = getOnlineAddres()
	// 	if err != nil {
	// 		Log.Fatal("createProduct出错: ", err)
	// 	}
	// }
	product.Pubkeyhash = getPubKeyHashFromAddress(mywallet.Address)
	return product
}

func createProductAndToPool(ProductData []byte, Pubkeyhash []byte) {
	product := createProduct(ProductData, Pubkeyhash)
	//fmt.Printf("---%x,%x,%s", product.ProductData, product.Pubkeyhash, product.SourceID)
	ProductPoolMu.Lock()

	ProductPool = append([]ProductType{product}, ProductPool...)
	for i, p := range ProductPool {
		fmt.Printf("第%d个产品id： %s\n", i, p.SourceID)
	}
	ProductPoolMu.Unlock()
	PackProductAndToChan(Productcomm, &product)
}

func PopFromProductPool() (ProductType, error) {
	ProductPoolMu.Lock()
	defer ProductPoolMu.Unlock()
	if len(ProductPool) == 0 {
		return ProductType{}, errors.New("ProductPool is empty")
	}
	// Get the element at index 0
	product := ProductPool[0]
	// Remove the element from the slice
	ProductPool = ProductPool[1:]

	return product, nil
}

//获得在线人的账户，除非等到没有人的时候才会转给自己
func getOnlineAddres() (to []byte, err error) {
	if enterflag == 0 {
		err = fmt.Errorf("请先进入区块链网络")
		return
	}
	myaddr := mywallet.Address
	mp2, _ := getChildren(Zkconn, "/Wallet/use")
	var addresses []string
	for addr := range mp2 {
		addresses = append(addresses, addr)
		//fmt.Println("addresses", addresses)
	}
	// 生成一个随机数
	randomNumber, err := rand.Int(rand.Reader, big.NewInt(int64(len(mp2))))
	if err != nil {
		Log.Info("生成随机数失败")
	}

	// 将生成的随机数转换为int类型
	randIndex := int(randomNumber.Int64())
	Log.Debug("在线账户人数：", len(mp2))
	// 确保随机索引不等于mywallet.Address
	if addresses[randIndex] == myaddr {
		randIndex = (randIndex + 1) % len(mp2)
	}
	to = []byte(addresses[randIndex])
	fmt.Printf("本账户为%x\n转账目标人选取：%x\n", myaddr, to)
	return
}

//随机获取一个转账账户
func getRandomAddres() (to []byte, err error) {

	wm := NewWalletManager()
	if wm == nil {
		fmt.Println(" NewWalletManager 失败!")
		return
	}

	addresses := wm.listAddresses()
	// for i, address := range addresses {
	// 	fmt.Printf("第%d个： %s\n", i, address)
	// }
	// 生成一个随机数
	randomNumber, err := rand.Int(rand.Reader, big.NewInt(int64(len(addresses))))
	if err != nil {
		Log.Info("生成随机数失败")
	}

	// 将生成的随机数转换为int类型
	randIndex := int(randomNumber.Int64())
	to = []byte(addresses[randIndex])
	Log.Debug("转账目标人选取：", to)
	return
}
