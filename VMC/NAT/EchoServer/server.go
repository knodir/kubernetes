package main

import (
	//"bytes"
	"fmt"
	// "log"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	CONN_HOST = ""
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

// Prints error message (err) with caller provided string (msg).
// Terminate the program if terminate=true (continue execution otherwise).
func handleError(msg string, err error, terminate bool) {
	// do not take any action if operation succeeded, i.e., err=nil
	if err != nil {
		fmt.Printf("%s: %s\n", msg, err)
		// terminate program if terminate=True
		if terminate {
			os.Exit(0)
		}
	}
}

var fileNum int

func main() {
	// Listen for incoming connections
	l, err := net.Listen(CONN_TYPE, ":"+CONN_PORT)
	handleError("[ERROR] failed to listen to the connection port", err, true)

	// Print environment variables
	env_var := os.Environ()
	for i := 1; i < len(env_var); i++ {
		fmt.Printf("%s\n", env_var[i])
	}

	// Close listener application when application closes
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)

	// open a file for writing latency values
	file, err := os.Create("packet_latency.txt")
	handleError("[ERROR] Could not create stats file", err, true)
	defer file.Close()

	for {
		// Listen for an incoming connection
		conn, err := l.Accept()
		handleError("[ERROR] failed to listen to the incoming port", err, true)

		//Logs an incoming message
		//fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		//Handle connections in a new goroutine
		go handleRequest(conn, file)
	}
}

func handleRequest(conn net.Conn, fd *os.File) {
	// Make a buffer to hold incoming data
	var buf [1600]byte
		
	for true {	
		// Read the incoming connection into the buffer
		reqLen, err := conn.Read(buf[0:])
		if err != nil {
			break
		}
		var timestamp time.Time
		err = (&timestamp).UnmarshalBinary(buf[0:reqLen])
		if err == nil {
			end_time := time.Now()

			// writing order (server_time, client_time, latency)
			_, err = fd.WriteString(strconv.FormatInt(end_time.UnixNano(), 10) + " " + strconv.FormatInt(timestamp.UnixNano(), 10) + " " + strconv.FormatInt(int64(end_time.Sub(timestamp)), 10) + " \n")
			handleError("[ERROR] Could not write stats to file", err, true)
		}
	}
	fmt.Printf("closing conn...\n")
	conn.Close()
}
