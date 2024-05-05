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



# 其他一些说明



