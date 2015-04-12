package main

import (
	//	"./Netfilter"
	"code.google.com/p/gopacket"
	"code.google.com/p/gopacket/layers"
	"code.google.com/p/gopacket/pcap"
	"fmt"
	"github.com/openshift/geard/pkg/go-netfilter-queue"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

var ip_count map[string]int
var nat_map map[int]net.IP

func main() {
	/* Firewall local state variables */
	servAddr := "10.100.81.31:3333" //service address IP:PORT of ECHOFILTEREDSERVICE
	machineAddr := ""               //address of container
	path := ""                      //path to iptables
	var ifs []pcap.Interface        //List of interfaces
	var env_var []string            //List of environment variables
	var nfq *netfilter.NFQueue      //Netfilter queue
	var err error
	const THRESHOLD int = 100

	ip_count = make(map[string]int)
	nat_map = make(map[int]net.IP)

	/* Find machine IP and service IP */
	fmt.Printf("Listing all devices\n")
	ifs, err = pcap.FindAllDevs()
	checkErr(err)

	for i := 0; i < len(ifs); i++ {
		if ifs[i].Name == "eth0" {
			machineAddr = ifs[i].Addresses[0].IP.String()
		}
	}

	fmt.Printf("machine address is %s\n", machineAddr)

	env_var = os.Environ()
	for i := 1; i < len(env_var); i++ {
		if strings.Contains(env_var[i], "ECHOFILTEREDSERVICE_PORT=") {
			result := strings.Split(env_var[i], "tcp://")
			servAddr = result[1]
		}
	}

	//	servAddr = os.Args[1]
	fmt.Printf("server address is %s\n", servAddr)

	/* Install IPTABLE rule to bypass kernel network stack */
	path, err = exec.LookPath("iptables")
	checkErr(err)
	fmt.Printf("path: %s\n", path)

	cmd := append([]string{"-A"}, "INPUT", "-p", "tcp", "-j", "NFQUEUE", "--queue-num", "0")
	err = exec.Command(path, cmd...).Run()
	checkErr(err)
	fmt.Printf("Added iptable rules to bypass kernel network stack\n")

	/* Start netfilter to capture incoming packets*/
	nfq, err = netfilter.NewNFQueue(0, 100, netfilter.NF_DEFAULT_PACKET_SIZE)

	checkErr(err)
	defer nfq.Close()

	/* Create syscall raw socket for writing packets out*/
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	/* Listen for packets */
	packets := nfq.GetPackets()
	for true {
		select {
		case p := <-packets:
			fmt.Printf("MAIN Incoming packet before processing:\n")
			fmt.Println(p.Packet)
			IPlayer := p.Packet.Layer(layers.LayerTypeIPv4)
			ipv4, _ := IPlayer.(*layers.IPv4)
			payload := ipv4.LayerPayload()

			TCPlayer := p.Packet.Layer(layers.LayerTypeTCP)
			tcp, _ := TCPlayer.(*layers.TCP)

			//fmt.Printf("MAIN from src address: %s, dst address: %s\n", ipv4.SrcIP.String(), ipv4.DstIP.String())
			//fmt.Printf("MAIN from src port: %d, dst port: %d\n", tcp.SrcPort, tcp.DstPort)

			if ipv4.SrcIP.String()+":"+strconv.Itoa(int(tcp.SrcPort)) == servAddr {
				/* response packet
				 * if we have more than one srcIP, we will have to remap accordingly
				 * also if we are sending to more than one service, servAddr will have to iterate to find a matching one
				 */
				clientIPAddr := nat_map[int(tcp.DstPort)]
				sendRedirect(int(tcp.DstPort), fd, clientIPAddr, ipv4, payload)
				fmt.Printf("Processed response from servAddr\n")

			} else if ip_count[ipv4.SrcIP.String()] < THRESHOLD {
				/* keep mappings and statistics
				 * currently we can just use tcp.SrcPort, because we do not remap since we only receive one IP address
				 * from kube-proxy who can ensure it does not reuse SrcPorts that is already in use. If expanded to multi-IPs srcs
				 * in the future, we'll have to have a mapping of the srcPort from kube-proxy to a new Port
				 */
				nat_map[int(tcp.SrcPort)] = ipv4.SrcIP
				ip_count[ipv4.SrcIP.String()] = ip_count[ipv4.SrcIP.String()] + 1
				fmt.Printf("ip_count addr: %s, cnt: %d\n", ipv4.SrcIP.String(), ip_count[ipv4.SrcIP.String()])
				fmt.Printf("nat_map int: %d, val: %s\n", tcp.SrcPort, nat_map[int(tcp.SrcPort)])

				/* We will redirect this packet to servAddr */
				ipServ, port := getIPandPort(servAddr)
				sendRedirect(port, fd, ipServ, ipv4, payload)
				fmt.Printf("Processed Incoming Packet\n")
			} else {
				fmt.Printf("Number of packets exceeded threshold\n")
			}
			p.SetVerdict(netfilter.NF_DROP)
		}

	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func sendRedirect(port, fd int, addr net.IP, ipv4 *layers.IPv4, payload []byte) error {
	newIPv4 := &layers.IPv4{

		Version:    ipv4.Version,
		IHL:        ipv4.IHL,
		Id:         ipv4.Id,
		TOS:        ipv4.TOS,
		Flags:      ipv4.Flags,
		FragOffset: ipv4.FragOffset,
		TTL:        ipv4.TTL,
		Protocol:   ipv4.Protocol,
		Options:    ipv4.Options,
		Padding:    ipv4.Padding,
		SrcIP:      ipv4.DstIP,
		DstIP:      addr.To4(),
		Checksum:   0,
	}

	header := newIPv4.LayerContents()
	newIPv4.Checksum = csum(header)
	header = newIPv4.LayerContents()

	outbuf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(outbuf, opts, newIPv4, gopacket.Payload(payload))
	packetData := outbuf.Bytes()

	//newPacket := gopacket.NewPacket(packetData, layers.LayerTypeIPv4, gopacket.Default)
	//fmt.Println(newPacket)

	temp := addr.To4()
	dst_addr := syscall.SockaddrInet4{
		Port: port,
		Addr: [4]byte{temp[0], temp[1], temp[2], temp[3]},
	}
	err := syscall.Sendto(fd, packetData, 0, &dst_addr)

	return err
}

func getIPandPort(addr string) (servIP net.IP, port int) {
	result := strings.Split(addr, ":")
	servIP = net.ParseIP(result[0])
	port, _ = strconv.Atoi(result[1])
	return
}

func csum(b []byte) uint16 {
	var s uint32
	fmt.Printf("% X\n", b)
	for i := 0; i < len(b); i += 2 {
		s += uint32(b[i+1])<<8 | uint32(b[i])
		fmt.Println(s)
	}
	// add back the carry
	s = s>>16 + s&0xffff
	s = s + s>>16
	return uint16(^s)
}
