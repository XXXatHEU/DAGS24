package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

type ProductType struct {
	SourceID    uuid.UUID //溯源id
	ProductData []byte    //溯源产品的信息
	Pubkeyhash  []byte    //冗余信息 归属人
}

func createProduct(ProductData []byte, Pubkeyhash []byte) ProductType {
	var product ProductType
	sourceID := uuid.New()
	product.SourceID = sourceID
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
		fmt.Println("addresses", addresses)
	}
	// 生成一个随机数
	randomNumber, err := rand.Int(rand.Reader, big.NewInt(int64(len(mp2))))
	if err != nil {
		Log.Info("生成随机数失败")
	}

	// 将生成的随机数转换为int类型
	randIndex := int(randomNumber.Int64())
	fmt.Println("randIndex", randIndex)
	fmt.Println("len(mp2)", len(mp2))
	// 确保随机索引不等于mywallet.Address
	if addresses[randIndex] == myaddr {
		randIndex = (randIndex + 1) % len(mp2)
	}
	fmt.Println("randIndex", randIndex)
	to = []byte(addresses[randIndex])
	fmt.Printf("%s", to)
	Log.Debug("地址选取：", to, "|")
	return
}
