package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	strEcho := "Hello"
	servAddr := os.Args[1]
	fmt.Printf("ADDRESS OF SERVICE IS: %s\n", servAddr)
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		fmt.Printf("ResolveTCPAddr failed\n")
		os.Exit(1)
	}
	fmt.Printf("Resolved TCP Addr\n")
	for true {
		<-time.After(time.Second * 1)
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			fmt.Printf("Dial failed\n")
			os.Exit(1)
		}

		_, err = conn.Write([]byte(strEcho))
		if err != nil {
			fmt.Printf("Write to server failed\n")
			os.Exit(1)
		}

		fmt.Printf("write to server = %s\n", strEcho)

		reply := make([]byte, 1024)

		_, err = conn.Read(reply)
		if err != nil {
			fmt.Printf("Write to server failed\n")
			os.Exit(1)
		}

		fmt.Printf("reply from server=%s\n", string(reply))

		conn.Close()
	}
}
