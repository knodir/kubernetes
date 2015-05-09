// Code adopted from PETERSTUFF/Firewall/test

package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"flag"
	// "strconv"
)

func usage() {
	fmt.Printf("This program requires 4 arguments. \n" + 
		"\t --dst: destination server ip:port \n" +
		"\t --total: total number of messages to send \n" +
		"e.g., $ ./echoclient --dst=198.162.52.126:3333 --total=10000\n" +
		"to send 10K messages to 198.162.52.126:3333." +
		"Note: each message is sent on a separate TCP connection which will have different port number. This emulates TCP connection from different clients.\n")

	// fmt.Printf("This program requires 4 arguments. \n" + 
	// 	"\t --dst: destination server ip:port \n" +
	// 	"\t --freq: number of messages per second \n" +
	// 	"\t --inc: per second acceleration of messages for each thread \n" +
	// 	"\t --total: total number of messages to send \n" +
	// 	"\t --threads: number of threads to create separate connection \n" +
	// 	"e.g., $ ./echoclient --dst=198.162.52.217:3333 --freq=10 --inc=2 --total=10000 --threads=2 \n" +
	// 	"Tx 10 messages per second with acceleration rate of 2 messages each second for each thread, \n" +
	// 	"such that two threads send aggregate number of 10K messages. \n" +
	// 	"Note: frequency applies to aggregate message count, i.e., if N threads are running, \n" +
	// 	"each thread will Tx (freq / N) msg/s, to ensure number of messages sent matches \n" + 
	// 	"the imposed aggregate limit.\n")

	// fmt.Printf("This program requires 4 arguments. \n" + 
	// 	"\t --dst: destination server ip:port \n" +
	// 	"\t --freq: number of messages per second \n" +
	// 	"\t --inc: per second acceleration of messages for each thread \n" +
	// 	"\t --total: total number of messages to send \n" +		
	// 	"e.g., $ ./echoclient --dst=198.162.52.217:3333 --freq=10 --inc=2 --total=10000 \n" +
	// 	"Tx 10 messages per second with acceleration rate of 2 messages each second, \n" +
	// 	"Total number of messages (with increment) is equal to 10K. \n")
}


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


func main() {

	if len(os.Args) != 3 {
		usage()
		os.Exit(0)
	}

	dstServer := flag.String("dst", "0.0.0.0:0", "destination server ip:port")
	total := flag.Int("total", 100, "total number of messages to send")
	
	flag.Parse()

	if *total < 1 {
		fmt.Println("[ERROR] number of packets has to be >=1")
		os.Exit(1)
	}

	fmt.Printf("[INFO] running client to send total %d messages to %s server\n", *total, *dstServer)

	var conn *net.TCPConn
	tcpAddr, err := net.ResolveTCPAddr("tcp", *dstServer)
	handleError("[ERROR] Could not Resolve TCP address", err, true)
	
	// send each packet in different thread
	for i := 0; i < *total; i++ {
		conn, err = net.DialTCP("tcp", nil, tcpAddr)
		handleError("[ERROR] Could not dial to given TCP address", err, true)

		timestamp, err := time.Now().MarshalBinary()
		handleError("[ERROR] Could not get current time in binary", err, true)

		_, err = conn.Write([]byte(timestamp))
		handleError("[ERROR] Could not write to connection", err, true)

		conn.Close()

		fmt.Printf("[INFO] Number of packets sent: %d\n", i)
	}
	
	// for true {
	// 	time.Sleep(time.Second)
	// }
}
