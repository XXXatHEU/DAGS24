package main

import (
	"fmt"
	"net"
	"os"
)

type p2pListener struct {
	listener *net.UDPConn
}

func newP2PListener(localPort int) (*p2pListener, error) {
	addr := net.UDPAddr{
		Port: localPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return nil, err
	}
	return &p2pListener{listener: conn}, nil
}

func (l *p2pListener) listen() {
	defer l.listener.Close()
	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := l.listener.ReadFromUDP(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from UDP: %v\n", err)
			continue
		}
		fmt.Printf("Received %d bytes from %v: %v\n", n, remoteAddr, string(buf[:n]))
	}
}

func main() {
	listener, err := newP2PListener(9000)
	if err != nil {
		panic(err)
	}
	fmt.Println("Started P2P listener on port 9000")
	listener.listen()
}
