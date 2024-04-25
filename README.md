# 环境配置

### 1.1 go环境问题

要求go环境最高1.17.1 因为有些密码库不支持更高版本

```go
 //查看GOROOT目录位置 删除所有go其他版本  如果没有就略过
go env | grep GOROOT

//rm -rf 卸载上面查询出来的位置

//下载go
 wget https://golang.org/dl/go1.17.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.16.6.linux-amd64.tar.gz

//修改配置文件  vim /etc/profile

GOROOT=/usr/local/go
GOPROXY=https://goproxy.cn
PATH=$PATH:$JAVA_HOME/bin:$JRE_HOME/bin:$GOROOT/bin

source ~/.bashrc

```

### 1.2 zookeeper配置

将DNSServer_Zookeeper的内容放到根目录里面

运行/zookServer/zookeeperServer/startZoonavigator.sh



修改zook.go中的hosts的ip地址





