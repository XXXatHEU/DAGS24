#开机运行
sudo systemctl start docker
/zookServer/zookeeperServer/zookeeper-3.4.10/bin/zkServer.sh start

#启动命令
sudo docker run -d -p 29000:29000 -e HTTP_PORT=29000 --name zoonavigator --restart unless-stopped elkozmon/zoonavigator:latest
#如果已经存在了 那么
sudo docker restart zoonavigator




