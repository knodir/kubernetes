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

var file *os.File
var num_packets int
var first_packet bool = false
var start_time time.Time

func main() {
	// Listen for incoming connections
	l, err := net.Listen(CONN_TYPE, ":"+CONN_PORT)
	checkErr(err)

	// open a file for writing latency values
	file, err = os.Create("latency_throughput.txt")
	checkErr(err)

	// Print environment variables
	env_var := os.Environ()
	for i := 1; i < len(env_var); i++ {
		fmt.Printf("%s\n", env_var[i])
	}

	//Close listener application when application closes
	defer l.Close()
	defer file.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)

	for {
		// Listen for an incoming connection
		conn, err := l.Accept()
		checkErr(err)
		num_packets = 0
		first_packet = false
		//Logs an incoming message
		//fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		//Handle connections in a new goroutine
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data
	var buf [1600]byte
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
			// Builds the message
			//n := bytes.Index(buf, []byte{0})
			//message := "Hi, I received your message! It was " + strconv.Itoa(reqLen) + " bytes long and that's what it said: \"" + string(buf[:n-1]) + "\" ! Honestly I have no clue about what to do with your mesages, so Bye Bye\n"
			// Write the message in the connection channel
			//conn.Write([]byte(message))

			// Close the connection when you are done with it
		}
	}
	fmt.Printf("closing conn...\n")
	conn.Close()
}
