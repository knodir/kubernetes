// Code adopted from PETERSTUFF/Firewall/test

package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"strconv"
)

func usage() {
	fmt.Printf("run as -> ./echoclient 198.162.52.217:3333 100 <- to send message every 100 microseconds.\n")
}

func main() {

	if len(os.Args) != 3 {
		usage()
		os.Exit(1)
	}

	strEcho := "Hello"
	servAddr := os.Args[1]
	fmt.Printf("ADDRESS OF SERVICE IS: %s\n", servAddr)
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		fmt.Printf("ResolveTCPAddr failed\n")
		os.Exit(1)
	}

	freqInMicrosec, _ := strconv.Atoi(os.Args[2])

	fmt.Printf("Resolved TCP Addr\n")
	for true {
		<-time.After(time.Microsecond * time.Duration(freqInMicrosec))
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
