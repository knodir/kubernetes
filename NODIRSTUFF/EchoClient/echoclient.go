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
		"\t --inc: per second acceleration of messages for each thread \n" +
		"\t --total: total number of messages to send \n" +
		"\t --threads: number of threads to create separate connection \n" +
		"e.g., $ ./echoclient --dst=198.162.52.217:3333 --freq=10 --inc=2 --total=10000 --threads=2 \n" +
		"Tx 10 messages per second with acceleration rate of 2 messages each second for each thread, \n" +
		"such that two threads send aggregate number of 10K messages. \n" +
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
func sendMsg(conn *net.TCPConn, threadName, servAddr string, pktsPerSec, incRate, maxNumOfPkts int) {

	var sentAllPkts bool = false
	currNumOfPkts := 0
	maxPktsPerSec := pktsPerSec

	start_time := time.Now()
	for !sentAllPkts {
		// run msgs Tx until we send total required number of messages
		select {

		case <-time.NewTicker(time.Second).C:
			if currNumOfPkts > maxNumOfPkts {
				// done sending required number of packets, break the loop to exit the goroutine
				sentAllPkts = true
				break
			} else {
				for i := 0; i < maxPktsPerSec; i++ {
					timestamp, err := time.Now().MarshalBinary()
					handleError("[ERROR] Could not get current time in binary", err, true)

					_, err = conn.Write([]byte(timestamp))
					handleError("[ERROR] Could not write to connection", err, true)

					fmt.Printf("[INFO] %s sent %d messages\n", threadName, currNumOfPkts)
					currNumOfPkts++
				}
				maxPktsPerSec += incRate
			}
		}
	}

	end_time := time.Now()
	fmt.Printf("Average Throughput for %s is %f msgs/second\n", threadName, float64(currNumOfPkts)*1000000000/float64(end_time.Sub(start_time)))
}

func main() {

	if len(os.Args) != 6 {
		usage()
		os.Exit(0)
	}

	dstServer := flag.String("dst", "0.0.0.0:0", "destination server ip:port")
	msgsPerSec := flag.Int("freq", 10, "number of messages per second")
	acceleration := flag.Int("inc", 1, "per second acceleration of messages for each thread")
	total := flag.Int("total", 100, "total number of messages to send")
	threadNum := flag.Int("threads", 1, "number of threads to create separate connection")
	
	flag.Parse()

	if *threadNum < 1 {
		fmt.Println("[ERROR] number of threads should be >=1")
		os.Exit(0)
	}

	totalPerThread :=  (*total / *threadNum)

	fmt.Printf("[INFO] running client with %d threads, each with initial %d messages, with acceleration rate of %d per second, totaling %d messages to %s server\n", *threadNum, *msgsPerSec, *acceleration, totalPerThread, *dstServer)

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
		go sendMsg(tcpConn, thrdName, *dstServer, *msgsPerSec, *acceleration, totalPerThread)
	}
	
	for _, tcpConn := range threadToConnMap {
		defer tcpConn.Close()
	}

	for true {
		time.Sleep(time.Second)
	}
}
