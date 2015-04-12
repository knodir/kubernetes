// Code adopted from PETERSTUFF/Firewall/test

package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
	"flag"
)

func usage() {
	fmt.Printf("This program requires 4 arguments. \n" + 
		"\t --dst: destination server ip:port \n" +
		"\t --freq: number of messages per second \n" +
		"\t --total: total number of messages to send \n" +
		"\t --threads: number of threads to create separate connection \n" +
		"e.g., $ ./echoclient --dst=198.162.52.217:3333 --freq=10 --total=10000 --threads=2 \n" +
		"Tx 10 message(s) per second, by two threads with aggregate message number 10K. \n" +
		"Note: frequency applies to aggregate message count, i.e., if N threads are running, \n" +
		"each thread will Tx (freq / N) msg/s, to ensure number of messages sent matches \n" + 
		"the imposed aggregate limit.\n")
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

// send message on given frequency
func sendMsg(conn *net.TCPConn, threadName, servAddr string, freq time.Duration, numPackets int) {	
	start_time := time.Now()
	for i := 0; i < numPackets; i++ {
		time.Sleep(freq)

		timestamp, err := time.Now().MarshalBinary()
		handleError("[ERROR] Could not get current time in binary", err, true)

		_, err = conn.Write([]byte(timestamp))
		handleError("[ERROR] Could not write to connection", err, true)

		fmt.Printf("[INFO] %s sent %d messages\n", threadName, i)
	}
	end_time := time.Now()
	fmt.Printf("Average Throughput for %s is %f msgs/second\n", threadName, float64(numPackets)*1000000000/float64(end_time.Sub(start_time)))
}

func main() {

	if len(os.Args) != 5 {
		usage()
		os.Exit(1)
	}

	dstServer := flag.String("dst", "0.0.0.0:0", "destination server ip:port")
	freq := flag.Int("freq", 10, "number of messages per second")
	total := flag.Int("total", 100, "total number of messages to send")
	threadNum := flag.Int("threads", 1, "number of threads to create separate connection")
	
	flag.Parse()

	if (*total == 0) {
		fmt.Println("[ERROR] number of threads can not be 0, it should be >=1")
		os.Exit(0)
	}

	eachThreadFreq := time.Microsecond * time.Duration(*threadNum * 1000000 / *freq)
	totalPerThread :=  (*total / *threadNum)

	fmt.Printf("[INFO] running client with %d threads, each with %s frequency, %d msgs per thread to %s server\n", *threadNum, eachThreadFreq, totalPerThread, *dstServer)

	var conn *net.TCPConn
	threadToConnMap := make(map[string]*net.TCPConn)
	threadName := "thread-"

	for i := 0; i < *threadNum; i++ {
		tcpAddr, err := net.ResolveTCPAddr("tcp", *dstServer)
		handleError("[ERROR] Could not Resolve TCP address", err, true)
		conn, err = net.DialTCP("tcp", nil, tcpAddr)
		handleError("[ERROR] Could not dial to given TCP address", err, true)

		threadToConnMap[threadName + strconv.Itoa(i)] = conn
	}

	for thrdName, tcpConn := range threadToConnMap {
		go sendMsg(tcpConn, thrdName, *dstServer, eachThreadFreq, totalPerThread)
	}
	
	for _, tcpConn := range threadToConnMap {
		defer tcpConn.Close()
	}

	for true {
		time.Sleep(time.Second)
	}
}
