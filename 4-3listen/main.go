package main

import (
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/google/uuid"
)

func main() {
	gob.Register(elliptic.P256())
	gob.Register(elliptic.P256())
	gob.Register(&TXInput{})
	gob.Register(&TXOutput{})
	gob.Register(uuid.UUID{})
	gob.Register(&Transaction{})
	gob.Register(elliptic.P256())
	gob.Register(wallet{})

	go func() {
		http.HandleFunc("/", hello)
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			fmt.Println("ListenAndServe Err:", err.Error())
			return
		}
	}()

	cli := CLI{}
	cli.Run()
	//关闭Zkconn连接
	Zkconn.Close()
}
func hello(resp http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(resp, "Hello World, Are You OK?")
}
