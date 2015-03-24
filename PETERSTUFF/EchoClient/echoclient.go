package main

import (
	"fmt"
	"net"
	"os"
	//"strings"
	"time"
)

func main() {
	strEcho := "Hello"
	servAddr := "198.162.52.35:3333"
	/*
		env_var := os.Environ()
		for i := 1; i < len(env_var); i++ {
			fmt.Printf("%s\n", env_var[i])
			if strings.Contains(env_var[i], "ECHOMASTERSERVICE_PORT=") {
				result := strings.Split(env_var[i], "tcp://")
				servAddr = result[1]
			}
		}
	*/
	fmt.Printf("ADDRESS OF SERVICE IS: %s\n", servAddr)
	//	if servAddr != "" {

	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		fmt.Printf("ResolveTCPAddr failed\n")
		os.Exit(1)
	}
	fmt.Printf("Resolved TCP Addr\n")
	for true {
		<-time.After(time.Second * 5)
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
	//	}
}
