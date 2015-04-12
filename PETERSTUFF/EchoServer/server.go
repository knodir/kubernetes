package main

import (
	//"bytes"
	"fmt"
	"log"
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

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var fileNum int

func main() {
	// Listen for incoming connections
	l, err := net.Listen(CONN_TYPE, ":"+CONN_PORT)
	checkErr(err)

	// Print environment variables
	env_var := os.Environ()
	for i := 1; i < len(env_var); i++ {
		fmt.Printf("%s\n", env_var[i])
	}

	//Close listener application when application closes
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)

	for {
		// Listen for an incoming connection
		conn, err := l.Accept()
		checkErr(err)

		fileNum++
		newFileName := "latency_throughput_" + strconv.Itoa(fileNum) + ".txt"
		//Logs an incoming message
		//fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		//Handle connections in a new goroutine
		go handleRequest(conn, newFileName)
	}
}

func handleRequest(conn net.Conn, filename string) {
	// Make a buffer to hold incoming data
	var buf [1600]byte
	var num_packets int
	var first_packet bool = false
	var start_time time.Time

	// open a file for writing latency values
	file, err := os.Create(filename)
	checkErr(err)
	defer file.Close()

	for true {
		if first_packet == false {
			first_packet = true
			start_time = time.Now()
		}
		num_packets = num_packets + 1
		fmt.Println(num_packets)

		// Read the incoming connection into the buffer
		reqLen, err := conn.Read(buf[0:])
		if err != nil {
			break
		}
		var timestamp time.Time
		err = (&timestamp).UnmarshalBinary(buf[0:reqLen])
		if err == nil {
			end_time := time.Now()

			_, err = file.WriteString(strconv.FormatInt(end_time.UnixNano(), 10) + " " + strconv.FormatInt(timestamp.UnixNano(), 10) + " " + strconv.FormatInt(int64(end_time.Sub(timestamp)), 10) + " " + strconv.FormatFloat(float64(num_packets)*1000000000/float64(end_time.Sub(start_time)), 'f', 2, 64) + "\n")
			checkErr(err)
		}
	}
	fmt.Printf("closing conn...\n")
	conn.Close()
}
