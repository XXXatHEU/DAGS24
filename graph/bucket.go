package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

const blockchainDBFile = "blockchain.db"
const bucketBlock = "bucketBlock"           //装block的桶
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
		fmt.Printf("打开数据库失败\n")
		return err
	}
	fmt.Printf("打开数据库成功\n")

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
				Log.Fatal("Bucket %q not found", bucketBlock)
				return fmt.Errorf("Bucket %q not found", bucketBlock)
			}

			val := bucket.Get([]byte(lastBlockHashKey))
			if val == nil {
				//Log.Warn("Key %q not found in bucket", key)
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
			fmt.Println("文件不存在，稍后再试")
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
				Log.Fatal("Bucket %q not found", bucketBlock)
				return fmt.Errorf("Bucket %q not found", bucketBlock)
			}

			val := bucket.Get([]byte(key))
			if val == nil {
				Log.Warn("Key %q not found in bucket", key)
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
		fmt.Println("删除数据库文件失败：", err)
	} else {
		// 删除成功
		fmt.Println("数据库文件已删除！")
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
	fmt.Println("打印所有的键")
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
