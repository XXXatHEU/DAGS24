package main

import "time"

func main() {
	//这是与协调服务通信
	go DposNodeMain()

	//告知dns dpos监听端口验证
	go DposListen()

	//一直sleep
	for {
		time.Sleep(time.Second * 1000)
	}
}
