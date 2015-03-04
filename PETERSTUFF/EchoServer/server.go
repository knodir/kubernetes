package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
)

const (
	CONN_HOST = ""
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

func main() {
	// Listen for incoming connections
	l, err := net.Listen(CONN_TYPE, ":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error Listening:", err.Error())
		os.Exit(1)
	}

	//Close listener application when application closes
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)
	for {
		// Listen for an incoming connection
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		//Logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		//Handle connections in a new goroutine
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data
	buf := make([]byte, 1024)

	// Read the incoming connection into the buffer
	reqLen, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	// Builds the message
	n := bytes.Index(buf, []byte{0})
	message := "Hi, I received your message! It was " + strconv.Itoa(reqLen) + " bytes long and that's what it said: \"" + string(buf[:n-1]) + "\" ! Honestly I have no clue about what to do with your mesages, so Bye Bye\n"
	// Write the message in the connection channel
	conn.Write([]byte(message))
	// Close the connection when you are done with it
	conn.Close()
}
