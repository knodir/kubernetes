package main

import (
	"./etcd_startup"
	"code.google.com/p/gopacket/pcap"
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

func main() {
	machine_ip := "127.0.0.1"
	machine_name := "random" + strconv.Itoa(rand.Int())
	machine_name = os.Args[1]
	ifs, _ := pcap.FindAllDevs()

	for i := 0; i < len(ifs); i++ {
		if ifs[i].Name == "eth0" {
			machine_ip = ifs[i].Addresses[0].IP.String()
		}
	}

	env_var := os.Environ()
	for i := 1; i < len(env_var); i++ {
		if strings.Contains(env_var[i], "HOSTNAME=") {
			result := strings.Split(env_var[i], "=")
			machine_name = result[1]
		}
	}

	etcd_startup.Etcd_start(machine_ip, "ECHOMASTERSERVICE", []string{"http://198.162.52.35:4001"}, machine_name)

	/*test code */
	etcd_local := []string{"http://" + machine_ip + ":4001"}
	client := etcd.NewClient(etcd_local)

	ret, err := client.Create("/glob", "10", 0)
	if err == nil {
		fmt.Println(ret.Node)
	} else {
		fmt.Println(err)
	}

	for true {
	}

}
