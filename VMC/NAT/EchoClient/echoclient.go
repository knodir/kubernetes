// Code adopted from PETERSTUFF/Firewall/test

package main

import (
	"fmt"
	"net"
	"os"
	"time"
	"flag"
	"strconv"
)

func usage() {
	fmt.Printf("This program requires 4 arguments. \n" + 
		"\t --dst: destination server ip:port \n" +
		"\t --total: total number of messages to send \n" +
		"\t --threads: number of threads to create separate connection \n" +
		"e.g., $ ./echoclient --dst=198.162.52.126:3333 --total=10000 --threads=5\n" +
		"to send 10K messages to 198.162.52.126:3333 concurrently by 5 threads" +
		"Note: each message is sent on a separate TCP connection which will have different port number. This models TCP connection from different clients.\n")
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

func sendMsg(thrdName, dstServer string, totalPerThread int) {
	var conn *net.TCPConn
	tcpAddr, err := net.ResolveTCPAddr("tcp", dstServer)
	handleError("[ERROR] Could not Resolve TCP address", err, true)
	
	// send each packet in different thread
	for i := 0; i < totalPerThread; i++ {
		conn, err = net.DialTCP("tcp", nil, tcpAddr)
		handleError("[ERROR] Could not dial to given TCP address", err, true)

		timestamp, err := time.Now().MarshalBinary()
		handleError("[ERROR] Could not get current time in binary", err, true)

		_, err = conn.Write([]byte(timestamp))
		handleError("[ERROR] Could not write to connection", err, true)

		conn.Close()

		fmt.Printf("[INFO] %s sent %d packets\n", thrdName, i)
		time.Sleep(time.Second)
	}	
}

func main() {
	var thrdName string
	var totalPerThread int

	if len(os.Args) != 4 {
		usage()
		os.Exit(0)
	}

	dstServer := flag.String("dst", "0.0.0.0:0", "destination server ip:port")
	total := flag.Int("total", 100, "total number of messages to send")
	threadNum := flag.Int("threads", 1, "number of threads to create separate connection")
	
	flag.Parse()

	if *total < 1 {
		fmt.Println("[ERROR] number of packets has to be >=1")
		os.Exit(1)
	}

	fmt.Printf("[INFO] running client to send total %d messages to %s server with %d threads\n", *total, *dstServer, *threadNum)

	totalPerThread = int(*total/(*threadNum))

	for index := 0; index < *threadNum; index++ {
		thrdName = "thread-" + strconv.Itoa(index)
		go sendMsg(thrdName, *dstServer, totalPerThread)
	}

	for true {
		time.Sleep(time.Second)
	}
}
