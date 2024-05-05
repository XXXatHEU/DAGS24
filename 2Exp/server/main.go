package main

import (
	"time"
)

func main() {
	go Orderinmain()
	go DposDnsOrder()
	//一直sleep
	for {
		time.Sleep(time.Second * 1000)
	}
}
