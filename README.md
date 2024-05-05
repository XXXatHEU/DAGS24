# 环境配置

### 1.1 go环境问题

要求go环境最高1.17.1 因为有些密码库不支持更高版本

```go
 //查看GOROOT目录位置 删除所有go其他版本  如果没有就略过
go env | grep GOROOT

//rm -rf 卸载上面查询出来的位置

//下载go
 wget https://golang.org/dl/go1.17.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.17.1.linux-amd64.tar.gz

//修改配置文件  vim /etc/profile

GOROOT=/usr/local/go
GOPROXY=https://goproxy.cn
//path中我想要加上$GOROOT/bin  其他自行配置
PATH=$PATH:$GOROOT/bin

source ~/.bashrc

//重启终端 查看go version
```

### 1.2 zookeeper配置

需要有docker，安装过程百度

将DNSServer_Zookeeper的内容放到根目录里面

赋予下面两个运行权限

/zookServer/zookeeperServer/zookeeper-3.4.10/bin/zkServer.sh

/zookServer/zookeeperServer/startZoonavigator.sh

运行/zookServer/zookeeperServer/startZoonavigator.sh

**修改zook.go中的hosts的ip地址**  端口固定值为2181

### 1.3 尝试运行

如果遇到get下载包超时问题，运行下面代码

```java
go env -w GOPROXY=https://goproxy.io,direct
go env -w GO111MODULE=on
```

编译

```shell
chmod 777 *.sh
rm go.mod -rf
go mod init main
./gomod.sh  ### 有些包下载不下来 单独运行下载
go mod tidy
go build -o blockchain *.go
./blockchain
```

![image-20240505200329152](https://gitee.com/zheshu/typora/raw/master/image-20240505200329152.png)

输入zkwallet回车进行初始化（务必按上面要求修改zook的ip地址）

### 1.4 还有一些配置

（这一步是代码中出现的问题，不想改代码了，所以这里需要手动创建一个节点）

访问zookeeper的可视化面板 网址—— ip:29000

在下面的内容输入 ip:2181

![image-20240505194949154](https://gitee.com/zheshu/typora/raw/master/image-20240505194949154.png)

![image-20240505200832250](https://gitee.com/zheshu/typora/raw/master/image-20240505200832250.png)

然后点进Wallet中新建use节点

![image-20240505200919347](https://gitee.com/zheshu/typora/raw/master/image-20240505200919347.png)

然后回到运行中，敲入enter回车，正常情况应该是下面的形式

![image-20240505201018859](https://gitee.com/zheshu/typora/raw/master/image-20240505201018859.png)



# 第一个实验 1Exp



这里将构建五个区块链，链中每个区块的最小交易数量是不一样的，具体的128、256、512、1024、2048等等

1）针对2048，如果想要大批量生成交易区块，那么应该需要有2048个交易，因此提前预值了2048个区块的区块链文件，在wallet文件中，需要将里面的blockchain.db放到运行环境的根目录中，覆盖原来的文件

![image-20240505213205548](https://gitee.com/zheshu/typora/raw/master/image-20240505213205548.png)

2）将下面的BuildDB函数注释打开（打开后无法再进入控制台，将自动的生成交易）

![image-20240505213312701](https://gitee.com/zheshu/typora/raw/master/image-20240505213312701.png)

3）运行脚本buildDB.sh

 将在/home/kz/中生成五个区块链的运行代码，然后自动运行生成，每个链独立运行



4）时间测量

在生成具体的区块后，修改这个时间测量模式变量 1是不加载到内存模式 2是加载到内存模式

![image-20240505214135078](https://gitee.com/zheshu/typora/raw/master/image-20240505214135078.png)

运行后生成下面日志 1024指的是遍历区块最少交易为1024的区块产生的日志 在最后有具体的时间汇总，然后再到另外的程序中，比如128、256等运行测量

![image-20240505214259638](https://gitee.com/zheshu/typora/raw/master/image-20240505214259638.png)





我这里将blockchain.go中的注释取消掉了，不知道未来会有些什么错误，如果出现错误，将两个注释重新注释掉

![image-20240505212720514](https://gitee.com/zheshu/typora/raw/master/image-20240505212720514.png)







# 第二个实验 2Exp

0、文件夹介绍

client 区块链节点

server  处理投票等服务端

pythonSever 谱聚类处理服务端



1、前提

需要保证三个文件夹中的flame2.txt文件内容一样

需要保证client和server里的DposGlobalType.go文件内容一样

通过控制DposGlobalType.go文件启动模式来测量不同的baseline

![image-20240505220151338](https://gitee.com/zheshu/typora/raw/master/image-20240505220151338.png)

2、运行

运行pythonSever ，接受谱聚类分组请求

单独运行server，将所有的go文件编译即 go build -o DNS *.go

client中运行脚本69.sh ，脚本中将同时创建指定数量的进程节点即区块链节点



# 其他一些过程记录

挖矿控制协程思想

![image-20240505224449535](https://gitee.com/zheshu/typora/raw/master/image-20240505224449535.png)

区块链添加逻辑

![区块链添加逻辑](https://gitee.com/zheshu/typora/raw/master/%E5%8C%BA%E5%9D%97%E9%93%BE%E6%B7%BB%E5%8A%A0%E9%80%BB%E8%BE%91.png)

区块的实现框图

![区块的实现框图](https://gitee.com/zheshu/typora/raw/master/%E5%8C%BA%E5%9D%97%E7%9A%84%E5%AE%9E%E7%8E%B0%E6%A1%86%E5%9B%BE.png)



请求和接收区块的逻辑处理

![请求和接收区块的逻辑处理](https://gitee.com/zheshu/typora/raw/master/%E8%AF%B7%E6%B1%82%E5%92%8C%E6%8E%A5%E6%94%B6%E5%8C%BA%E5%9D%97%E7%9A%84%E9%80%BB%E8%BE%91%E5%A4%84%E7%90%86.png)

交易广播



![交易广播](https://gitee.com/zheshu/typora/raw/master/%E4%BA%A4%E6%98%93%E5%B9%BF%E6%92%AD.png)





![verifyTxconverter](D:\数据溯源\区块链形成计划过程记录\流程\verifyTxconverter.png)

![zookeeper映射](https://gitee.com/zheshu/typora/raw/master/zookeeper%E6%98%A0%E5%B0%84.png)

![广播](D:\数据溯源\区块链形成计划过程记录\流程\广播.png)

![cc.go的思路](https://gitee.com/zheshu/typora/raw/master/cc.go%E7%9A%84%E6%80%9D%E8%B7%AF.png)

![context](https://gitee.com/zheshu/typora/raw/master/context.png)











数据溯源

![数据溯源](https://gitee.com/zheshu/typora/raw/master/%E6%95%B0%E6%8D%AE%E6%BA%AF%E6%BA%90.png)

收到交易

![image-20240505224535042](https://gitee.com/zheshu/typora/raw/master/image-20240505224535042.png)





## 实现

1.修改逻辑，钱包名称为自定义钱包名称，钱包也即用户，新建一千个用户
    十万个节点
2.两个用户溯源实现 
2.发布溯源起始节点函数，无需挖矿即可发布区块，区块奖励为获得这个id
3.

关于p2p通信
  192.144.220.80上维护全节点ip地址列表

使用zookeeper来维护全节点ip的过程
   1.向zookeeper说明自己的ip和端口

总体步骤
  1.获取全节点ip地址
      1）用zookeeper配置中心来维护连接
      2）简单维护，本地节点或得到后判断能否连接成功
  2.获取区块信息
     1）获取区块信息（发送文件文件的形式）
     2）验证区块是否合法（繁琐）
               - 分叉问题
                              - 如何验证是否合法
     ​     ​     ​     3）更新本地数据库
     ​     ​     ​     4）获取收到的交易
     ​     ​     ​     5）更新本地交易池
    ​    ​    3.钱包数据结构如何组织
     ​     ​     ​     1）每个程序的钱包名字如何组织
     ​     ​     ​     2）签名交易和验证过程

  4.能否可视化所有的交易
  5.程序放在docker中启动还是直接运行程序， 
       
结构修改
   1.发布溯源起始点交易函数
   2.修改UTXO获取过程
   3.添加溯源函数
   4.添加溯源交易过程


快速生成区块链网络脚本

      1. 生成指定数目用户
      2. 生成溯源信息

其他信息
  1.区块暂时使用pow共识算法

难点
  1.简单的pow算法会导致分叉问题
       达成共识的时候如何解决

## 实现过程记录

### 1zookconn

完成了多个节点连接zook注册自己的监听ip和端口

监听

并在连接后会主动获取其他节点的ip和端口并连接，主动连接后会首先告知对方自己注册在zook中的ip和端口，对方判断如果已经连接后会断开连接

main的主逻辑以及建立连接后的逻辑用sleep代替

![image-20230609213020848](https://gitee.com/zheshu/typora/raw/master/images/202306092130979.png)

write不能没有前面的go，也就是不能进入其他的函数，这样在这个函数里新建的go不能正常工作，也就是不能正常写操作

### 2modifyBlockStructure

修改transaction.go的代码，直接在源代码上修改了，如果有问题去根目录下的blockchain去拿

NewCoinbaseTx创建挖矿交易，挖矿交易还是给一点钱

NewTransaction创建普通交易，不再寻找utxo，

## 使用zookeeper实现DNS Seeds

`/goworkplace/zookeeperServer/zookeeper-3.4.10/data`目录是持久化数据，

`/goworkplace/zookeeperServer/zookeeper-3.4.10/conf/zoo.cfg`是配置文件 配置上面的目录为持久化数据路径，zk的默认端口号是2181，启动命令在bin目录里面，启动命令为`./zkServer.sh start`



```go
		本程序主要有几个函数
		 1.setLocalAddress  通过拨号 查到本地的出口ip和端口，这个需要改成区块链的p2p服务的ip和端口
		 2.createPathRecursively 根据传入的path创建节点，如果路径不存在那么会递归的调用创建路径
		    然后将节点值放到上面
		 3.addLocalToZook会调用第一个函数获取ip和端口并调用第二个函数插入数据
		 4.getChildrenWithPathValues获取path下的所有节点，并返回一个map
		 5.delLocalAddress调用后会将finalpath删除
```

接下来设计

1. 三个p2p节点完成通信

   - 每个节点维护一个通信池（能否判断已经断开连接？）

   - 当有新节点建立的时候进行通信

     - 新节点进入是旧节点连接新节点还是新节点连接旧节点呢

       （应该是新节点连接旧节点 然后申请获取信息 

   - 当有节点退出怎么办，如何通过发布订阅



在区块链网络中，全节点之间的通信通常需要建立点对点（peer-to-peer，P2P）连接，但并不要求每个全节点都需要直接连接到其他所有节点。

当一个全节点加入到区块链网络中时，它可以通过一些特定的协议（例如 DNS Seed）或者节点列表来发现其他节点的存在，并建立与其中一部分节点的 P2P 连接。

区块链节点的的端口将从9000开始 









listen收交易如何与主线程交互

- 交易池数据结构
- 区块池数据结构
  1. 收到数据后先获取mutex锁，比如区块，向区块放入数据，然后释放锁，返回一个replyblock回复，否则重发
  2. 收到交易后，也返回一个交易回复

dial发逻辑，是群发过程，main中为每个连接维护一个chan数组，通过chan与其进行交互



main中逻辑

1. init，zookper中注册自己的信息，
2. 然后拨号
3. 拨号后，首先要获取其他区块逻辑，更新区块，更新交易
4. 创建监听协程
5. 挖矿，监听协程可以随时打断当前逻辑并修改修改挖矿的信息

## 运行

不同地址代表不同密钥对

这里的转账 并不判断对方地址是否真实存在，即使不存在，如果你能找到一个私钥，根据私钥得到公钥，再从公钥得到哈希，如果能对的上也能进行转账

```shell
编译
     go build  -o blockchain *.go


创建钱包   获得钱包地址
     ./blockchain createWallet
        多运行几次  获得了 几个钱包地址  这里创建的钱包 每个都对应着不同的密钥对  也就是可以代表不同的人 
        12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr
        12xR7FjLiAexBfY9nrSBAyJ4qmiZtXj8Au
        18fhYgcSaRwQm1wUzCwQGRYR4ufuA5PGCQ
        1Mgvneccs26RJd4vnH8S6J4nxT6iiJcVUT

打印所有的钱包地址
   ./blockchain listAddress 

创建创世块   这个地址从上面钱包地址上选一个
     ./blockchain create <地址>
     比如 ./blockchain create 12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr

打印所有交易 只能看到一个输入为-1的交易
    ./blockchain printTx 

打印区块，能看到一个区块
    ./blockchain print  

打印余额 比如12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr
   ./blockchain getBalance <地址>
   即 ./blockchain getBalance 12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr

--------
转账 
   ./blockchain send <FROM> <TO> <AMOUNT> <MINER> <DATA> 
   第四个参数是矿工的意思，他要挖矿，会有一笔挖矿收入给他 ，所以下面即使转账了因为矿工还是指的他因此也会有一笔收入
   比如 ./blockchain send \
             12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr  \
             12xR7FjLiAexBfY9nrSBAyJ4qmiZtXj8Au \
             5 \
             12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr \
             "转了一笔5btc的钱给别人"
查看余额
        ./blockchain getBalance 12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr
        ./blockchain getBalance  12xR7FjLiAexBfY9nrSBAyJ4qmiZtXj8Au
打印交易和打印区块
       ./blockchain printTx
       ./blockchain print  
```



send 发送地址的时候是创建的挖矿交易，然后创建了一个普通交易，
 然后将这两笔交易放到了block里面 AddBlock 

walletmanager.go 的createWallet创建钱包的时候是将密钥对和地址写入到磁盘中 



地址如果是正确的就可以，并非一定在钱包里面

最原始状态(什么都没有)

- 在本地创建一个钱包
- 网络沟通，发现什么都没有 然后创建创世块
- 广播这个创世块

这时候其他节点加入

![image-20230613204420628](https://gitee.com/zheshu/typora/raw/master/images/202306132044696.png)







数据溯源逻辑

一、新建溯源线

  input输入为-1

二、验证交易

  首先判断是否是溯源线起点 

  判断交易的input是否在utox中存在并且没有被花费

三、溯源线遍历

  修改区块链结构，那么交易里面内容到底是什么？

  1）input和output 不是数组了 一个元素  都是地址

​       交易内容:溯源码

2）修改后验证如何验证

​     从input中得到地址，拿到地址后遍历所有区块的

  所有交易，从交易看output是否是这个地址

​    如果是这个地址后看是否是这个溯源码

   (在遍历的时候需要维护这个地址的输出码篮子)

   如果是这个溯源码从输出码篮子里面判断有没有

   花费出去

  判断完成后再for循环维护输出码篮子 

3）这样修改后我如何体现在代码上？

  \- 肯定utxo获取是要改的

  \- 转账金额的时候 这个是函数的入口 主要将这个实现

   的逻辑改了  剩下都是打印的逻辑 都是次要的

   完成溯源的逻辑后就是网络交互的问题了

先来后到吧  在原有的逻辑上线将溯源的逻辑干完 

然后再着重添加交易池和区块池看如何能交互 并达成共识

主要是 

  1）数据的序列化，在区块数据库传送时是用protubuf还是单纯

传入.dat文件？

  2）区块池如何传送，交易池内容如何传送

  3）传送过程中，正好区块发布了，交易池内容删了怎么办

  4）区块的最长链

​     如果仅仅通过时间戳，如果因为网络延迟导致我已经加到

​      比较晚的区块上怎么办，交易在我的交易池上已经没有了



  ![image-20230613210047747](https://gitee.com/zheshu/typora/raw/master/images/202306132100816.png)

几个需要注意的地方，创建交易只能从有对应的私钥节点发布

## 重要时间节点

### 3blockChain

下面进行修改，先修改send函数

调用了NewTransaction，<font face="仿宋" color=red>这里是先验证交易 先验证交易再去发布</font>在NewTransaction中调用findNeedUTXO，

在findNeedUTXO调用FindMyUTXO，根据地址找到所有的utxo

在FindMyUTXO修改，只是如果这个地址已经在output里面了 那么就说明已经用了 那么就直接return



交易结构 我看不用修改，只是将TXoutput中的金额变为id就可以

那么我创建一个交易 

NewCoinbaseTx创建挖矿交易 

- 将TXOutput的转账金额修改为uuid.UUID格式的

- 将NewCoinbaseTx的挖初始矿的奖励修改为uuid
- 注意：：不能单纯的挖矿的奖励
- 

SourceID     uuid.UUID

- newTXOutput函数 



- NewCoinbaseTx挖矿交易

  	SourceID := uuid.New()
  	
  	output := newTXOutput(miner, SourceID)

- NewTransaction创建普通交易

  里面调用的findNeedUTXO函数

  - 调用FindMyUTXO，这里不能重写，就应该按照比特币的思维重新判断，不然会提高溯源的效率，既然

    如此，那么这里也不修改了，原因是这个函数返回这个地址的所有的溯源终点

  调用FindMyUTXO获得所有的utxo，判断是否有需要的utxo

  

```shell
 ./blockchain createWallet 创建钱包
    13m3skaknaZNGnhbbLWwpfUfQRr9w973bx
    184ghPdyqeVDnoByTgta66y7GMPGDa4iWo
./blockchain create 18F3TNk6o5KCfeHw8LddTgyCkWSzwew48o 创建创世块
 ./blockchain printTx  打印所有交易
  ./blockchain print  打印区块
./blockchain getBalance 18F3TNk6o5KCfeHw8LddTgyCkWSzwew48o 打印现在拥有的交易

 ./blockchain send \
             18F3TNk6o5KCfeHw8LddTgyCkWSzwew48o  \
             17TJ8jfxWq4FiNiUC6husUcwdmBjtVzard \
             94087794-ef44-4bc9-8e24-ae78dee02c9d \
             12uHSPzBYkFpLeX5gAKJZ9kiYn7wyz8dbr \
             "转了一笔5btc的钱给别人"
             
  create 18F3TNk6o5KCfeHw8LddTgyCkWSzwew48o
  
  

15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47
17qcyFGbxHg5D8wWzJgH3npgmJ4HUVjoi1
1AHEPXKPNJqdYbJLv7Q5v9jxRqg7uNfNV1
1CPdz1zuZ3SB4vy2j3E3aqkigd77Xi8vYf
1CPngX63QCdyHxtLUfVRbpKZjSMwxK7dVm
1HdeexBdNXtKPNs1GbbiFjNb1tj9uFwMiw
1LjimDvfDQ8Mgspk78aV4wPT7KAk46S6bM
1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7
1Q2SPHP6YZ59eM66EsbUQ67EpYq8Jbnjxh

28482658-e5d3-42bb-8655-ddf9f93c676e
  从 1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7
  到 1CPdz1zuZ3SB4vy2j3E3aqkigd77Xi8vYf
ab35b6d1-fa02-41ee-8241-74ed869463c4
  从1LjimDvfDQ8Mgspk78aV4wPT7KAk46S6bM
  到15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47
1fe86fb1-882d-45a7-9a08-2d007372724c
  从1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7
  到15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47
958efbde-8e35-4b5e-8df2-14e4f5b8c11a
  从1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7
  到15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47
  再到1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7
6a728848-41d2-4739-a1ee-b0f2d377a785
 从1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7
 
 
send 15FRUMr1ZXb21AxasyzZXStFFFAmZb5F47 1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7 958efbde-8e35-4b5e-8df2-14e4f5b8c11a 1PofC1kSGBKiHN3SggHmVBoetrAKwuaNP7 dd


区块哈希值
第五：0000893ca3457f10af7abc381ad99e6bf20fd7e4f937e1dd362d8786ce922448 （）
第四：0000d71601ef2dfe97328c8783b75931b9e29ee8b139c92c3062489fcc3f1d02
第三：0000d558daa001ddf462e7ef4676159933ee6b78dc809b964e711f597659a3fa
第二：00001a4a0de5223dcdda8dbc09990cbeda842049dc74cca183546935ad145766
第一：0000353abfa21effb30e49b2fa106a8516a3cefe4b2c3b19cbbefea570ab54d7

  
 
```

## 

### 4remote

![image-20230614215623863](https://gitee.com/zheshu/typora/raw/master/images/202306142156007.png)



现在姑且认为不会发布重复的交易，也就是每个区块的时间戳不会重复，姑且假设区块广播时间非常小

也就是正在挖矿中，别的节点完成挖矿后能及时通知这个节点

1、规范接口，形成四个线程

​    监听、拨打以及广播、挖矿(挖矿通过命令注入 也就是在cli里面)、监听本地发布交易

如果这样的话，太不可控了，

全部通过cli命令来完成



当有新的区块或者交易生成后 通过监听建立的连接发给另一方，通过主动建立建立的连接发给另一方



2、对cli.go完成io多路复用

3、在4-3listen中对p2p监听和dail和zook进行重置



zook.go启动后寻找空闲端口并进行连接 以使的占用该端口  如果

为了防止冲突，端口从10000开始 找一个随机数并占用 

初始化： 占用端口 连接上zook上 表示该节点开始  （错误 这个开始节点应该在一个函数里面，当启动后才能进行初始化）



overch := make(chan int)   <-overch 等待 

函数签名 overch chan int



![image-20230617145457744](https://gitee.com/zheshu/typora/raw/master/images/202306171454812.png)

总结  4-3 暂时完成了enter功能

### 4-4 remote



特殊的生产者消费者



- 所有消费者等待同一份数据
- 主协程过一段时间后会自动删除数据不会等待所有子协程都读完
- 主协程不知道有多少个子协程

消费者平时在干什么？应该阻塞在这个结构体上



定义发送结构体（完成）



1. 接收来自其他节点的请求，自己就会将所有的区块和交易信息都发给他们，这个应该写成一个单独的函数，
2. 收到交易后做出处理，更新区块和节点信息(先做这个)
3. 区块和交易做出处理







#### 一些一直用的函数

const (

  SendGenesisBlock Command = iota

  SendCommonBlock

  SendTX

  FetchBlocks //请求获取你的本地区块信息

  FetchTX

)

[]byte(addStr)

aa  []byte

const (

  PackBlock PackCommand = iota

  PackTX

  PackDB //请求获取你的本地区块信息

  PackByte

  UnPackBlock

  UnPackTX

  UnPackDB //请求获取你的本地区块信息

  UnPackByte

)





字符比较

```go
		// 将字符串转换为字节数组
		hashBytes, err := hex.DecodeString("0000d558daa001ddf462e7ef4676159933ee6b78dc809b964e711f597659a3fa")
		if err != nil {
			// 转换失败
			fmt.Println("字符串转换为字节数组失败！")
			return
		}
		// 将字节数组赋值给hash
		var hash [32]byte
		copy(hash[:], hashBytes)

		blockArray = append(blockArray, block)
		if bytes.Equal([]byte(block.Hash), hash[:]) {
			{
				fmt.Println("发现相等!")
				break
			}
		}
```





Validate主要用来收到交易或者区块后的处理过程

​    最主要内容 ：

- ReceiveTxArr  收到交易后的处理过程

- ReceiveBlockArr 收到区块后的处理过程

   		

net是接收和发送的模块  

​    最主要内容 

 startDailWork函数 接收处理

 startConnWork 监听建立连接后的处理 （修改后 建立连接后都统一进入startConnWork ）







netpool 网络任务池主要用来发送东西的

 SendtaskChan 是接收任务的chan

AcceptSendTask是后台运行程序，从SendtaskChan接收任务

 gdb 文件

source /usr/local/go/src/runtime/runtime-gdb.py

- 查看当前活动的 Goroutine ： info goroutines 
- 切换到指定的 Goroutine 上下文：goroutine <GoroutineID>
- 查看当前 Goroutine 的堆栈：bt
- 单步执行或继续运行： next 单步执行到下一行代码 continue继续运行程序
- 



### 公钥哈希、公钥、地址

Wallets map[string]*wallet

Address是找到这个wallet的键



公钥哈希 就是在区块上的地址   注意address只是来标识wallet的

listaddress 显示的是address

send的时候output.ScriptPubKeyHash = pubKeyHash 这个pubKeyHash 是getPubKeyHashFromAddress(address)

区块上的是公钥哈希，公钥哈希能够从address得到 getPubKeyHashFromAddress

address可以从公钥哈希尝试得到  pubkeyhashToAddress



transaction.TXInputs[0].PubKey: mywallet.walle.PubKey

output.ScriptPubKeyHash  =   getPubKeyHashFromAddress(address)





## 图区块链

### 需要修改内容

1、结构上修改

2、溯源链交易的生成

   (获取前一个区块 主链逆向查找到最后一个区块   确认  生成交易和区块 )

   （验证这个区块  主链逆向查找到最后一个区块   确认区块）

 3、主链上的内容

   判断自己是否有资格挖主链上的区块    

​    有一个要生成主链区块的交易池  从里面拿出一个溯源信息 产生一个溯源起点

​    验证能看到的所有的区块内容

 			如果使用的是池子里面的那么就串起来判断，有一个不行那就删除

   















**普通区块三个指针**

- **POI(Point of Interest)指向本溯源线上一个节点（不可为空）**
- **CP(Current Pointer)指向当前主链的最后一个区块，用来表示高度（不可为空）**
- **TBP(To Be Proposed)指向还没有TBP指向的区块，用来加快收敛速度（可以为空）**

**主链区块一个指针**

- **MP(Main Chain Previous Pointer) 主链上的区块指向前一个主链区块的指针**
- **SP（Split Pointer）分裂指针(暂缓)**（暂时不加）
- 已经确认的普通区块列表，使用
- 布隆过滤器



待确认队列



区块结构

- POI 溯源线上一个区块哈希

- TBP 没有TBP指向的区块

- cp 指其高度（但是在当前下，这个没什么用   因为某个挖矿区块可能收不到这个交易 进而没法通过验证

  以后可以使用raft，组成一个节点 

  ）



- MP主链结构前一个区块的指针

- 产品条形码对应的集合  也就是二维数组 一维表示产品条形码 二维表示确认的链式结构 【逆向的顺序】 

  ​	确认交易结构

  ​    二维：    溯源id   溯源区块哈希  布隆过滤器

  ​    一维：    溯源id  大的布隆过滤器   （判断是否存在的时候先判断大的是否有 有的话看小的是否有）

     {id,下面结构的数组}

​        [111111] -> {{haxizhi,布隆过滤器}，{haxizhi,布隆过滤器}，{haxizhi,布隆过滤器}，{haxizhi,布隆过滤器}  }

​        [222222]->  {{haxizhi,布隆过滤器}，{haxizhi,布隆过滤器}，{haxizhi,布隆过滤器}，{haxizhi,布隆过滤器}  }

​     最外面有个布隆过滤器

- 三级跳表结构

- 新的溯源产品信息

​        结构： 溯源id、说明



规划：

1. 具体的结构
2. 验证布隆过滤器的使用
3. 跳表结构的使用（跳表溯源函数       ）
4. 区块的序列化和反序列化  





关键词提取（放入布隆过滤器中）：来源方、去向方、
以下是几个支持中文和英文的关键字提取库和工具：

1. github.com/huichen/sego：这是一个用于中文分词和关键字提取的库。它基于字典和规则的方法，具有较高的准确性和性能。
2. github.com/yanyiwu/gojieba：这是一个流行的中文分词库，也支持关键字提取功能。它基于结巴分词算法，可以快速而准确地处理中文文本。
3. github.com/wangbin/jiebago：这是另一个基于结巴分词算法的中文分词库。它提供了关键字提取的功能，可以根据词频和文本特征进行关键字排序。
4. github.com/kljensen/snowball：这个库提供了英文和其他语言的词干提取和关键字提取功能。它支持多种语言，包括中文。
5. github.com/jdkato/prose：这是一个全面的自然语言处理库，支持英文和其他语言的分词、词性标注、实体识别和关键字提取等功能。



具体结构



布隆过滤器的使用

头文件："github.com/willf/bloom"  如果没有需要go get，不要用go mod 然后go tidy 生成

```go
// gitee连接https://github.com/bits-and-blooms/bloom

创建布隆过滤器：

  // 创建布隆过滤器

  m := uint(1000000) // 位数组大小

  k := uint(5)    // 哈希函数数量

  filter := bloom.New(m, k)



//添加元素

filter.Add([]byte("apple"))

//判断元素是否存在

fmt.Println(filter.Test([]byte("apple")))


```





1、思考

Transactions不会变，还是用原来的

针对侧链究竟需要修改什么？

删掉PoI []byte，还是使用PrevHash标识前一个区块。

侧链需要修改什么？

需要修改生成一个区块的时候，的东西



0）添加侧链区块交易池

​     添加主链区块交易池

1）生成侧链区块的时候

2）生成主链区块的时候

3）序列化传输的时候

4）反序列化传输的时候

 

mining -》 MiningControl() 为主链区块挖矿



侧链：

侧链挖矿特别迅速：

不能只关心自己的区块，可以看当前挖矿的交易是否是本地正在挖的，如果是那么停止，否则就是仅仅加到交易池里面





侧链不需要挖矿，放在发布交易的窗口里面，改成发布一个区块。

修改发布交易接口



GetUTXOBySourceId(targetSourceId uuid.UUID)
根据主链的末尾，根据targetSourceId 找ConfirmedLists这个map表，如果找到，取BlockArray最后一个元素的Hash，这个就是最后一个交易的哈希值





```cpp
type Transaction struct {
	TXID      []byte     //交易id
	TXInputs  []TXInput  //可以有多个输入
	TXOutputs []TXOutput //可以有多个输出
	TimeStamp uint64     //创建交易的时间

	//本溯源线上一个节点
	PoI []byte

	//当前主链的最后一个区块(暂时留空)
	CP []byte

	//指向还没有TBP指向的区块
	TBP []byte
}
```



启动步骤

  1. 启动zk 

```sh
#进入zk工作目录
cd /zookServer/zookeeperServer/zookeeper-3.4.10/bin
#执行
  ./zkServer.sh start
```



几个注意

1. 通过hash拿到区块，这个hash如果是手动输入字符串，需要通过[]byte转化，然后通过DBkeyhash函数转化

   但是网络传输的区块不需要进行转化，直接使用就可以。所以GetBlockCountUntilSpecificBlock函数里面并没有转化的过程

2. 获取交易池中的交易会发生阻塞的情况，这时候就不能收到中断命令，因此我将其放入到矿工的工作里面，这样包工头及以上都不会阻塞

3. mining挖矿是传递的是指针类型的block，而且getTxFromTxPool也是获得的是指针类型的数据，那么可能在更新交易池时有问题，也就是给删了，但是还用着

<font face="仿宋" color=red>1. zk启动时 在那个start脚本里面有启动zk可视化的脚本命令 就是docket那个</font>

<font face="仿宋" color=red>2. go版本只能使用go1.17.1，我试过1.16会有切片安全问题，1.19太高，导致公钥加密的时候有问题，暂时定为1.17.1 可能1.17阶段的都可以</font>

### 大量生成交易实现

1. 在/wallet/中预制好各种节点，存放各种值

   账户一:存在

   账户二:存在

   

现在的地址中

生成密钥对，根据私钥和公钥生成地址，然后根据将地址为键，私钥和公钥作为值持久化



生成转账交易的时候

 根据地址从钱包中拿到具体wallet，此时也就获得了公钥和私钥，交易中的所有output都是公钥的哈希值，所以根据这个公钥哈希值找付款人能够使用的output





  （1）数据采集环节：航天宏图、拓尔思；
（2）数据存储环节：易华录、中科曙光、深桑达；
（3）数据加工环节：海天瑞声、科大讯飞、航天宏图、中科星图、海量数据、星环科技、达梦数据、拓尔思等；
（4）数据流通环节：a.数据交易所：安恒信息、广电运通、浙数文化、人民网；b.数据产品/服务提供商：航天宏图、上海钢联、海天瑞声、卓创资讯、山大地纬、慧辰股份；c.数据共享：太极股份、中科江南、博思软件、南威软件等；
（5）数据安全环节：安恒信息、奇安信、深信服、信安世纪、启明星辰、天融信、绿盟科技、美亚柏科、亚信安全、恒为科技、安博通、中新赛克、等。



航天宏图
拓尔思
易华录
中科曙光
深桑达
海天瑞声
科大讯飞
航天宏图
中科星图
海量数据
星环科技
达梦数据
拓尔思
安恒信息
奇安信
深信服
信安世纪
启明星辰
天融信
绿盟科技
美亚柏科
亚信安全
恒为科技
安博通
中新赛克

（4）数据流通环节：a.数据交易所：安恒信息、广电运通、浙数文化、人民网；b.数据产品/服务提供商：航天宏图、上海钢联、海天瑞声、卓创资讯、山大地纬、慧辰股份；c.数据共享：太极股份、中科江南、博思软件、南威软件等；
（5）数据安全环节：安恒信息、奇安信、深信服、信安世纪、启明星辰、天融信、绿盟科技、美亚柏科、亚信安全、恒为科技、安博通、中新赛克、等。

