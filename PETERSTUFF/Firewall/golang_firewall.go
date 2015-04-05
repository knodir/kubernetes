package main

import (
	//"code.google.com/p/gopacket"
	"code.google.com/p/gopacket/layers"
	"code.google.com/p/gopacket/pcap"
	"fmt"
	"github.com/openshift/geard/pkg/go-netfilter-queue"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

var ip_count map[string]int
var ip_conn map[string]*net.TCPConn

func main() {
	/* Firewall local state variables */
	servAddr := ""             //service address IP:PORT of ECHOFILTEREDSERVICE
	machineAddr := ""          //address of container
	path := ""                 //path to iptables
	var ifs []pcap.Interface   //List of interfaces
	var env_var []string       //List of environment variables
	var nfq *netfilter.NFQueue //Netfilter queue
	var err error
	const threshold int = 100

	ip_count = make(map[string]int)
	ip_conn = make(map[string]*net.TCPConn)

	/* Log information for debugging */
	fmt.Printf("Listing all devices\n")
	ifs, err = pcap.FindAllDevs()
	checkErr(err)

	for i := 0; i < len(ifs); i++ {
		fmt.Printf("Name: %s Desc: %s NumAddr: %d ", ifs[i].Name, ifs[i].Description, len(ifs[i].Addresses))
		if ifs[i].Name == "eth0" {
			machineAddr = ifs[i].Addresses[0].IP.String()
		}

		for j := 0; j < len(ifs[i].Addresses); j++ {
			fmt.Printf("Addr%d: %s ", j, ifs[i].Addresses[j].IP.String())
		}
		fmt.Printf("\n")
	}

	fmt.Printf("machine address is %s\n", machineAddr)

	env_var = os.Environ()
	for i := 1; i < len(env_var); i++ {
		fmt.Printf("%s\n", env_var[i])
		if strings.Contains(env_var[i], "ECHOFILTEREDSERVICE_PORT=") {
			result := strings.Split(env_var[i], "tcp://")
			servAddr = result[1]
		}
	}

	fmt.Printf("server address is %s\n", servAddr)

	path, err = exec.LookPath("iptables")
	checkErr(err)
	fmt.Printf("path: %s\n", path)

	/* Install IPTABLE rule to target NFQUEUE for all incoming traffic */
	cmd := append([]string{"-A"}, "INPUT", "-j", "NFQUEUE", "--queue-num", "0")
	err = exec.Command(path, cmd...).Run()
	checkErr(err)
	fmt.Printf("Added iptable rules\n")

	/* Process packets serially */
	nfq, err = netfilter.NewNFQueue(0, 100, netfilter.NF_DEFAULT_PACKET_SIZE)
	checkErr(err)
	defer nfq.Close()

	/* Start the Relay */
	go relay(machineAddr, servAddr)

	/* Perform firewall functions */
	packets := nfq.GetPackets()
	for true {
		fmt.Printf("Waiting for packets...\n")
		select {
		case p := <-packets:
			//fmt.Println(p.Packet)
			rcv_packet := p.Packet
			ipv4_layer := rcv_packet.Layer(layers.LayerTypeIPv4)
			ipv4, _ := ipv4_layer.(*layers.IPv4)
			//payload = ipv4.LayerContents()

			fmt.Printf("Received connection - srcAddr: %s\n", ipv4.SrcIP.String())

			if ipv4.SrcIP.String()+":3333" == servAddr {
				fmt.Printf("Reply received from echo server\n")
				p.SetVerdict(netfilter.NF_ACCEPT)
			} else if ip_count[ipv4.SrcIP.String()] < threshold {
				ip_count[ipv4.SrcIP.String()] = ip_count[ipv4.SrcIP.String()] + 1
				fmt.Printf("Packet # %d received from srcIP %s\n", ip_count[ipv4.SrcIP.String()], ipv4.SrcIP.String())
				p.SetVerdict(netfilter.NF_ACCEPT)

			} else {
				fmt.Printf("Number of Packets exceeded threshold\n")
				temp_conn := ip_conn[ipv4.SrcIP.String()]
				if temp_conn != nil {
					fmt.Printf("Closing Connection\n")
					temp_conn.Close()
					ip_conn[ipv4.SrcIP.String()] = nil
				}
				p.SetVerdict(netfilter.NF_DROP)
			}
		}
	}

}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func relay(listenAddr, servAddr string) {

	fmt.Printf("Starting Relay...\n")

	echo_listen_addr, err := net.ResolveTCPAddr("tcp", listenAddr+":3333")
	checkErr(err)
	echo_listener, err := net.ListenTCP("tcp", echo_listen_addr)
	checkErr(err)
	fmt.Printf("Resolved Listener...\n")
	for true {
		conn, _ := echo_listener.AcceptTCP()
		fmt.Printf("Accepted Connection: %s\n", conn.RemoteAddr().String())
		ip_conn[conn.RemoteAddr().String()] = conn
		go serve_conn(conn, servAddr)
	}
}

func serve_conn(client_conn *net.TCPConn, servAddr string) {
	var packet [1536]byte
	servTCPAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	checkErr(err)
	serv_conn, err := net.DialTCP("tcp", nil, servTCPAddr)
	checkErr(err)
	fmt.Printf("Started a relay connection....\n")
	/*read from client_conn*/
	for true {
		n, err := client_conn.Read(packet[0:])
		if err == nil {
			/*relay to serv_conn*/
			serv_conn.Write(packet[0:n])
			n, err := serv_conn.Read(packet[0:])
			if err == nil {
				client_conn.Write(packet[0:n])
			} else {
				break
			}
		} else {
			break
		}
	}
	serv_conn.Close()
	client_conn.Close()
}
