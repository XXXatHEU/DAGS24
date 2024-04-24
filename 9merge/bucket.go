package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
)

// 128 256 512 1024 2048/home/kz/2048/
var blockchainDBFile = "blockchain.db"

//const blockchainDBFile = "blockchain.db"

const bucketBlock = "bucketBlock"           //装block的桶  addblock 启动区块
const lastBlockHashKey = "lastBlockHashKey" //用于访问bolt数据库，得到最后一个区块的哈希值
var DBfilemutex sync.Mutex                  //第一个是在大锁，针对bd这个文件上锁
var DBmutex sync.Mutex                      //第二个是小锁
func addBlockToBucket(newBlock *Block) (newBlockHash []byte, err error) {

	err1 := createOrUpdateKeyValue(newBlock.Hash, newBlock.Serialize())
	if err1 != nil {
		err = err1
		return
	}
	err2 := createOrUpdateKeyValue([]byte(lastBlockHashKey), newBlock.Hash)
	if err2 != nil {
		err = err2
		return
	}
	//更新bc的tail，这样后续的AddBlock才会基于我们newBlock追加
	//bc.tail = newBlock.Hash
	newBlockHash = newBlock.Hash
	return
}

func intit() {

	DBmutex.Lock()
	// 检查文件是否存在
	_, err := os.Stat(blockchainDBFile)
	if os.IsNotExist(err) {
		// 文件不存在，释放锁
		DBmutex.Unlock()
		// 等待一段时间再尝试
		Log.Debug("文件不存在，稍后再试")
		time.Sleep(time.Second * 1)
	}
	// 文件存在，尝试打开数据库
	db, err := bolt.Open(blockchainDBFile, 0600, nil)
	if err != nil {
		Log.Warn("getfinalBlock Bucket Open失败,将稍后再试", err)
		DBmutex.Unlock()
	}
	defer db.Close()
	txCount := 0
	blockCount := 0
	startTime := time.Now()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			Log.Fatal("Bucket not found ,buckBlock:", bucketBlock)
			return fmt.Errorf("Bucket %q not found", bucketBlock)
		}

		c := bucket.Cursor()

		// 定位到范围的起始位置000001fe86ede04c9eb838295bda9e92705ba2e394538f296c005d4315a56099
		byteArray := []byte{1, 0, 1, 254, 134, 237, 224, 76, 158, 184, 56, 41, 91, 218, 158, 146, 112, 91, 162, 227, 148, 83, 143, 41, 108, 0, 93, 67, 21, 165, 96, 153}
		startKey := []byte(byteArray)
		c.Seek(startKey)

		// 遍历范围内的键
		for k, value := c.First(); k != nil; k, value = c.Next() {
			fmt.Println(k)
			tx := Deserialize_tx(value)
			//printtx(tx)
			//序列化成tx后，有可能也有问题，因此还需要判断tx的txhash
			if tx == nil || tx.TxHash == nil {
				block := Deserialize(value)
				if block != nil {
					for _, tx := range block.Transactions {
						txCount++
						printtx(tx)
					}
					blockCount++
					mutexBlockMap.Lock()
					MemoryBlockMap[string(block.Hash)] = block
					mutexBlockMap.Unlock()
					fmt.Printf("主链区块个数：%d 侧链区块个数：%d\n", blockCount, txCount)
					if txCount < 100 {

					} else {
						break

					}
				} else {
					Log.Warn("block也为nil")
				}

			} else { //完全是tx区块
				//printtx(tx)
				txCount++
				mutexTxMap.Lock()
				MemoryTxMap[string(tx.TxHash)] = tx
				mutexTxMap.Unlock()
			}

		}

		return nil
	})
	DBmutex.Unlock()
	endTime := time.Now()
	duration1 := endTime.Sub(startTime)
	fmt.Println("持续时间:", duration1)
}
func intit3() {

	DBmutex.Lock()
	// 检查文件是否存在
	_, err := os.Stat(blockchainDBFile)
	if os.IsNotExist(err) {
		// 文件不存在，释放锁
		DBmutex.Unlock()
		// 等待一段时间再尝试
		Log.Debug("文件不存在，稍后再试")
		time.Sleep(time.Second * 1)
	}
	// 文件存在，尝试打开数据库
	db, err := bolt.Open(blockchainDBFile, 0600, nil)
	if err != nil {
		Log.Warn("getfinalBlock Bucket Open失败,将稍后再试", err)
		DBmutex.Unlock()
	}
	defer db.Close()
	txCount := 0
	blockCount := 0
	startTime := time.Now()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			Log.Fatal("Bucket not found ,buckBlock:", bucketBlock)
			return fmt.Errorf("Bucket %q not found", bucketBlock)
		}

		// 遍历键值对
		bucket.ForEach(func(key, value []byte) error {
			fmt.Printf("%x\n", key)
			//尝试序列化成tx区块
			fmt.Println(key)
			tx := Deserialize_tx(value)
			//序列化成tx后，有可能也有问题，因此还需要判断tx的txhash
			if tx == nil || tx.TxHash == nil {
				block := Deserialize(value)
				if block != nil {
					blockCount++
					mutexBlockMap.Lock()
					MemoryBlockMap[string(block.Hash)] = block
					mutexBlockMap.Unlock()
				} else {
					Log.Warn("block也为nil")
				}

			} else { //完全是tx区块
				//printtx(tx)
				txCount++
				mutexTxMap.Lock()
				MemoryTxMap[string(tx.TxHash)] = tx
				mutexTxMap.Unlock()
			}
			fmt.Printf("主链区块个数：%d 侧链区块个数：%d\n", blockCount, txCount)
			return nil
		})

		return nil
	})
	DBmutex.Unlock()
	endTime := time.Now()
	duration1 := endTime.Sub(startTime)
	fmt.Println("持续时间:", duration1)
}

func checkType(data []byte) string {
	// 使用反射来检查数据类型
	var block Block
	var transaction Transaction

	if err := json.Unmarshal(data, &block); err == nil {
		fmt.Println("block类型")
		return "block"
	}

	if err := json.Unmarshal(data, &transaction); err == nil {
		fmt.Println("transaction类型")
		return "transaction"
	}
	fmt.Println("未知类型")

	return "unknown"
}

func createOrUpdateKeyValue(key []byte, value []byte) error {
	Log.Debug("尝试获取dbmutex")
	DBmutex.Lock()
	defer DBmutex.Unlock()
	Log.Debug("获取成功")

	// 检查文件是否存在
	// _, err := os.Stat(blockchainDBFile)
	// if os.IsNotExist(err) {
	// 	DBmutex.Unlock()
	// 	Log.Warn("区块链文件不存在，将释放锁==================")
	// 	time.Sleep(time.Second * 1)
	// 	DBmutex.Lock()
	// 	continue
	// }
	// if _, err := os.Stat(blockchainDBFile); os.IsNotExist(err) {
	// 	fmt.Printf("database file %s does not exist", blockchainDBFile)
	// }
	Log.Debug("尝试打开数据库", blockchainDBFile)
	// 文件存在，尝试打开数据库
	db, err := bolt.Open(blockchainDBFile, 0666, nil)
	defer db.Close()
	if err != nil {
		Log.Debug("打开数据库失败\n")
		return err
	}
	Log.Debug("打开数据库成功\n")

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketBlock))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(key), value)
		if err != nil {
			return err
		}

		return nil
	})

	if err == nil {
		// 成功执行写操作，退出循环，返回nil
		Log.Debug("文件更新成功")
		return nil
	} else {

		Log.Fatal("文件更新失败")

		return nil
	}

	// 执行写操作失败，等待一段时间再尝试

}
func createOrSaveTxs(Transactions []*Transaction) error {
	Log.Debug("尝试获取dbmutex")
	DBmutex.Lock()
	defer DBmutex.Unlock()
	Log.Debug("获取成功")
	Log.Debug("尝试打开数据库", blockchainDBFile)
	// 文件存在，尝试打开数据库
	db, err := bolt.Open(blockchainDBFile, 0666, nil)
	defer db.Close()
	if err != nil {
		Log.Debug("打开数据库失败\n")
		return err
	}
	Log.Debug("打开数据库成功\n")

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketBlock))
		if err != nil {
			return err
		}
		//保存到区块里面
		for _, tx := range Transactions {
			err = bucket.Put([]byte(tx.TxHash), tx.Serialize_tx())
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err == nil {
		// 成功执行写操作，退出循环，返回nil
		Log.Debug("文件更新成功")
		return nil
	} else {

		Log.Fatal("文件更新失败")

		return nil
	}

	// 执行写操作失败，等待一段时间再尝试

}

//不允许加锁  2.0修改版可以加锁   设置
// func getfinalBlock() []byte {
// 	bytes, err := getValueByKey([]byte(lastBlockHashKey))
// 	if err != nil {
// 		Log.Fatal("getfinalBlock失败", err)
// 	}
// 	return bytes
// }
//2.0修改版可以加锁   设置
func getfinalBlock() []byte {
	options := &bolt.Options{
		Timeout:  time.Second * 5, // 设置超时时间为5秒
		ReadOnly: true,            // 以只读模式打开数据库
	}

	for {
		DBmutex.Lock()
		// 检查文件是否存在
		_, err := os.Stat(blockchainDBFile)
		if os.IsNotExist(err) {
			// 文件不存在，释放锁
			DBmutex.Unlock()
			// 等待一段时间再尝试
			time.Sleep(time.Second * 10)
			continue
		}

		// 文件存在，尝试打开数据库
		db, err := bolt.Open(blockchainDBFile, 0600, options)
		if err != nil {
			Log.Warn("getfinalBlock Bucket Open失败,将稍后再试", err)
			DBmutex.Unlock()

			// 等待一段时间再尝试
			time.Sleep(time.Second * 2)
			continue
		}
		defer db.Close()

		var value []byte
		err = db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketBlock))
			if bucket == nil {
				Log.Fatal("getfinalBlock   Bucket not found ：", bucketBlock)
				return fmt.Errorf("Bucket %q not found", bucketBlock)
			}

			val := bucket.Get([]byte(lastBlockHashKey))
			if val == nil {
				Log.Warn("getfinalBlock Key not found in bucket ：", lastBlockHashKey)
				return fmt.Errorf("Key %q not found in bucket", []byte(lastBlockHashKey))
			}

			value = make([]byte, len(val))
			copy(value, val)

			return nil
		})

		DBmutex.Unlock()

		if err != nil {
			// 错误处理或返回错误，根据您的需要
			// 等待一段时间再尝试
			time.Sleep(time.Second * 10)
			continue
		}

		// 成功获取数据时，返回value
		return value
	}
}
func getTxValuesByKeys(keys map[*Transaction][]byte) (map[*Transaction][]byte, error) {
	options := &bolt.Options{
		Timeout:  time.Second * 5, // 设置超时时间为5秒
		ReadOnly: true,            // 以只读模式打开数据库
	}
	for {
		DBmutex.Lock()
		// 检查文件是否存在
		_, err := os.Stat(blockchainDBFile)
		if os.IsNotExist(err) {
			// 文件不存在，释放锁
			DBmutex.Unlock()
			// 等待一段时间再尝试
			Log.Debug("文件不存在，稍后再试")
			time.Sleep(time.Second * 1)
			continue
		}
		// 文件存在，尝试打开数据库
		db, err := bolt.Open(blockchainDBFile, 0600, options)
		if err != nil {
			Log.Warn("getfinalBlock Bucket Open失败,将稍后再试", err)
			DBmutex.Unlock()
			// 等待一段时间再尝试
			time.Sleep(time.Second * 2)
			continue
		}
		defer db.Close()
		result := make(map[*Transaction][]byte)
		var value []byte
		err = db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketBlock))
			if bucket == nil {
				Log.Fatal("Bucket not found ,buckBlock:", bucketBlock)
				return fmt.Errorf("Bucket %q not found", bucketBlock)
			}
			for tx, key := range keys {
				val := bucket.Get(key)
				if val == nil {
					Log.Warn("Key not found in bucket, key:")
					fmt.Printf("%x\n", key)
					keys[tx] = nil
					continue
				}
				value = make([]byte, len(val))
				copy(value, val)
				result[tx] = value
			}
			return nil
		})
		DBmutex.Unlock()
		if err != nil && string(err.Error()) == "-1" {
			Log.Warn("在数据库中没有找到该区块")
			return nil, err
		}
		if err != nil {
			// 等待一段时间再尝试
			time.Sleep(time.Second * 1)
			continue
		}
		// 成功获取数据时，返回value
		return result, err
	}
}

func getValueByString(keys map[string][]byte) (map[string][]byte, error) {
	options := &bolt.Options{
		Timeout:  time.Second * 5, // 设置超时时间为5秒
		ReadOnly: true,            // 以只读模式打开数据库
	}
	for {
		Log.Debug("UpdateMemory3-1")
		DBmutex.Lock()
		// 检查文件是否存在
		_, err := os.Stat(blockchainDBFile)
		if os.IsNotExist(err) {
			// 文件不存在，释放锁
			DBmutex.Unlock()
			// 等待一段时间再尝试
			Log.Debug("文件不存在，稍后再试")
			time.Sleep(time.Second * 1)
			continue
		}
		// 文件存在，尝试打开数据库
		Log.Debug("UpdateMemory3-2")
		db, err := bolt.Open(blockchainDBFile, 0600, options)
		if err != nil {
			Log.Warn("getfinalBlock Bucket Open失败,将稍后再试", err)
			DBmutex.Unlock()
			// 等待一段时间再尝试
			time.Sleep(time.Second * 2)
			continue
		}
		defer db.Close()
		result := make(map[string][]byte)
		var value []byte
		Log.Warn("UpdateMemory3-3")
		err = db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketBlock))
			if bucket == nil {
				Log.Fatal("Bucket not found ,buckBlock:", bucketBlock)
				return fmt.Errorf("Bucket %q not found", bucketBlock)
			}
			Log.Debug(len(keys))
			for stringKey, key := range keys {
				Log.Debug("UpdateMemory3-5")
				val := bucket.Get(key)
				if val == nil {
					Log.Warn("Key not found in bucket, key:")
					fmt.Printf("%x\n", key)
					keys[stringKey] = nil
					continue
				}
				value = make([]byte, len(val))
				copy(value, val)
				result[stringKey] = value
			}
			return nil
		})
		Log.Debug("UpdateMemory3-4")
		DBmutex.Unlock()
		if err != nil && string(err.Error()) == "-1" {
			Log.Warn("在数据库中没有找到该区块")
			return nil, err
		}
		if err != nil {
			// 等待一段时间再尝试
			time.Sleep(time.Second * 1)
			continue
		}
		// 成功获取数据时，返回value
		return result, err
	}
}

func getValueByString2(keys map[string][]byte) (map[string][]byte, error) {
	options := &bolt.Options{
		Timeout:  time.Second * 5, // 设置超时时间为5秒
		ReadOnly: true,            // 以只读模式打开数据库
	}
	//for {
	DBmutex.Lock()
	// 检查文件是否存在
	_, err := os.Stat(blockchainDBFile)
	if os.IsNotExist(err) {
		// 文件不存在，释放锁
		DBmutex.Unlock()
		// 等待一段时间再尝试
		Log.Fatal("文件不存在，稍后再试")
	}
	numArrays := 200
	slices := splitMapIntoArrays(keys, numArrays)
	/*
		for _, slice := range slices {
			fmt.Println("======================================================")
			for key, _ := range slice {
				//mergedMap2[key] = value
				fmt.Println(key)
			}
		}*/
	//fmt.Println(len(keys))

	// 文件存在，尝试打开数据库
	db, err := bolt.Open(blockchainDBFile, 0600, options)
	if err != nil {
		DBmutex.Unlock()
		Log.Fatal("getfinalBlock Bucket Open失败,将稍后再试", err)
	}
	defer db.Close()
	var value []byte
	var wg sync.WaitGroup
	for i := 0; i < len(slices); i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			err := db.View(func(tx *bolt.Tx) error {
				// 在只读事务中执行读取操作
				bucket := tx.Bucket([]byte(bucketBlock))
				if bucket == nil {
					Log.Fatal("Bucket not found ,buckBlock:", bucketBlock)
					return fmt.Errorf("Bucket %q not found", bucketBlock)
				}
				//fmt.Println(j, "号进程处理", len(slices[j]))
				for stringKey, key := range slices[j] {
					val := bucket.Get(key)
					if val == nil {
						Log.Warn("Key not found in bucket, key:")
						fmt.Printf("%x\n", key)
						keys[stringKey] = nil
						continue
					}
					value = make([]byte, len(val))
					copy(value, val)
					// 使用互斥锁将结果存入 mergedMap
					slices[j][stringKey] = value
				}
				return nil
			})

			if err != nil {
				Log.Warn("Error:", err)
			}
		}(i)
	}

	wg.Wait()
	Log.Debug("getValueByString协程完成")
	mergedMap2 := make(map[string][]byte)
	for _, slice := range slices {
		for key, value := range slice {
			mergedMap2[key] = value
			//fmt.Println(key)
		}
	}
	DBmutex.Unlock()
	// 此处不需要再次初始化 mergedMap，直接返回即可
	return mergedMap2, err
	//}
}

//底层是调用的getValueByString
func getValuesByKeys(keys map[uuid.UUID][]byte) (map[uuid.UUID][]byte, error) {
	newMap := make(map[string][]byte)
	// 遍历原始 map 并进行转换
	for key, value := range keys {
		// 将 uuid.UUID 转换为 string
		keyStr := key.String()
		// 将数据复制到新的 map
		newMap[keyStr] = value
	}
	newMap, err := getValueByString(newMap)
	for key, _ := range keys {
		keys[key] = newMap[key.String()]
	}
	return keys, err
}

func getValueByKey(key []byte) ([]byte, error) {
	options := &bolt.Options{
		Timeout:  time.Second * 5, // 设置超时时间为5秒
		ReadOnly: true,            // 以只读模式打开数据库
	}

	for {
		DBmutex.Lock()
		// 检查文件是否存在
		_, err := os.Stat(blockchainDBFile)
		if os.IsNotExist(err) {
			// 文件不存在，释放锁
			DBmutex.Unlock()
			// 等待一段时间再尝试
			Log.Debug("文件不存在，稍后再试")
			time.Sleep(time.Second * 1)
			continue
		}
		// 文件存在，尝试打开数据库
		db, err := bolt.Open(blockchainDBFile, 0600, options)
		if err != nil {
			Log.Warn("getfinalBlock Bucket Open失败,将稍后再试", err)
			DBmutex.Unlock()
			// 等待一段时间再尝试
			time.Sleep(time.Second * 2)
			continue
		}
		defer db.Close()

		var value []byte
		err = db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketBlock))
			if bucket == nil {
				Log.Fatal("Bucket not found ,buckBlock:", bucketBlock)
				return fmt.Errorf("Bucket %q not found", bucketBlock)
			}

			val := bucket.Get([]byte(key))
			if val == nil {
				Log.Warn("Key not found in bucket,key:")
				fmt.Printf("key :%x\n", key)
				return fmt.Errorf("-1")
				//return fmt.Errorf("Key %q not found in bucket", []byte(key))
			}

			value = make([]byte, len(val))
			copy(value, val)

			return nil
		})
		DBmutex.Unlock()
		if err != nil && string(err.Error()) == "-1" {
			Log.Warn("在数据库中没有找到该区块")
			return nil, err
		}
		if err != nil {
			// 等待一段时间再尝试
			time.Sleep(time.Second * 1)
			continue
		}
		// 成功获取数据时，返回value
		return value, err
	}
}
func deletedbfile() {
	DBmutex.Lock()
	err := os.Remove(blockchainDBFile)
	DBmutex.Unlock()
	if err != nil {
		// 删除失败
		Log.Debug("删除数据库文件失败：", err)
	} else {
		// 删除成功
		Log.Debug("数据库文件已删除！")
	}
}

/*
func getValueByKey(key []byte) ([]byte, error) {
	options := &bolt.Options{
		Timeout:  time.Second * 5, // 设置超时时间为5秒
		ReadOnly: true,            // 以只读模式打开数据库
	}
	DBmutex.Lock()
	defer DBmutex.Unlock()
	//两个功能：
	// 1. 如果区块链不存在，则创建，同时返回blockchain的示例
	db, err := bolt.Open(blockchainDBFile, 0600, options)
	defer db.Close()
	if err != nil {
		Log.Fatal("Bucket Open失败")
		return nil, err
	}

	var value []byte
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			Log.Fatal("Bucket %q not found", bucketBlock)
			return fmt.Errorf("Bucket %q not found", bucketBlock)
		}

		val := bucket.Get([]byte(key))
		if val == nil {
			//Log.Warn("Key %q not found in bucket", key)
			return fmt.Errorf("Key %q not found in bucket", key)
		}

		value = make([]byte, len(val))
		copy(value, val)

		return nil
	})

	return value, err
}*/

func deleteKey(key []byte) error {
	DBmutex.Lock()
	defer DBmutex.Unlock()

	db, err := bolt.Open(blockchainDBFile, 0666, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			return fmt.Errorf("Bucket %q not found", bucketBlock)
		}

		err := bucket.Delete([]byte(key))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func getBucketKeys() {
	DBmutex.Lock()
	defer DBmutex.Unlock()

	db, err := bolt.Open(blockchainDBFile, 0666, nil)
	defer db.Close()
	if err != nil {
		Log.Fatal("getBucketKeys打开文件失败")
	}

	//	var keys []string
	Log.Debug("打印所有的键")
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketBlock))
		if bucket == nil {
			return fmt.Errorf("Bucket %q not found", bucketBlock)
		}
		count := 1
		err := bucket.ForEach(func(k, v []byte) error {

			fmt.Printf("%d: %x\n", count, k)
			count++
			//keys = append(keys, string(k))
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})

	// for index, key := range keys {
	// 	fmt.Printf("%d: %s\n", index, key)
	// 	fmt.Println(key)
	// }
}
