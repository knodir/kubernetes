// Code adopted from PETERSTUFF/Firewall/test

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func usage() {
	fmt.Printf("run as -> ./echoclient 198.162.52.217:3333 100 <- to send message every 100 microseconds. 1000000 <- number of packets to send\n")
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)

	}
}

func main() {

	if len(os.Args) != 4 {
		usage()
		os.Exit(1)
	}

	servAddr := os.Args[1]
	fmt.Printf("ADDRESS OF SERVICE IS: %s\n", servAddr)
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	checkErr(err)

	freqInMicrosec, _ := strconv.Atoi(os.Args[2])
	numPackets, _ := strconv.Atoi(os.Args[3])

	start_time := time.Now()
	for i := 0; i < numPackets; i++ {
		<-time.After(time.Microsecond * time.Duration(freqInMicrosec))
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		checkErr(err)

		timestamp, err := time.Now().MarshalBinary()
		checkErr(err)

		_, err = conn.Write([]byte(timestamp))
		checkErr(err)

		/*fmt.Printf("write to server = %s\n", strEcho)

		reply := make([]byte, 1024)

		_, err = conn.Read(reply)
		if err != nil {
			fmt.Printf("Write to server failed\n")
			os.Exit(1)
		}

		fmt.Printf("reply from server=%s\n", string(reply))
		*/
		conn.Close()
		fmt.Println(i)
	}
	end_time := time.Now()
	fmt.Printf("Average Throughput is %f packets/second\n", float64(numPackets)*1000000000/float64(end_time.Sub(start_time)))
}
