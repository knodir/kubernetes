// Base version of NAT without any etcd to store the state.

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code.google.com/p/gopacket"
	"code.google.com/p/gopacket/layers"
	"code.google.com/p/gopacket/pcap"
	"github.com/openshift/geard/pkg/go-netfilter-queue"
)

var nat_map map[int]net.IP

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
	// Firewall local state variables
	var servAddr string // service address IP:PORT of ECHOFILTEREDSERVICE; retrived on runtime
	var  machineAddr string         // address of container
	var ipTablesPath string         // path to iptables
	var ifs []pcap.Interface        // List of interfaces
	var env_var []string            // List of environment variables
	var nfq *netfilter.NFQueue      // Netfilter queue
	var err error

	nat_map = make(map[int]net.IP)

	// Find machine IP and service IP 
	fmt.Printf("[DEBUG] Listing all devices\n")
	ifs, err = pcap.FindAllDevs()
	handleError("[ERROR] Could not get all network devices", err, true)

	for i := 0; i < len(ifs); i++ {
		if ifs[i].Name == "eth0" {
			machineAddr = ifs[i].Addresses[0].IP.String()
		}
	}

	fmt.Printf("[DEBUG] machine address is %s\n", machineAddr)

	env_var = os.Environ()
	for i := 1; i < len(env_var); i++ {
		if strings.Contains(env_var[i], "ECHOFILTEREDSERVICE_PORT=") {
			result := strings.Split(env_var[i], "tcp://")
			servAddr = result[1]
		}
	}

	// servAddr = "198.162.52.126" // os.Args[1]
	fmt.Printf("[DEBUG] server address is %s\n", servAddr)

	// Install IPTABLE rule to bypass kernel network stack 
	ipTablesPath, err = exec.LookPath("iptables")
	handleError("[ERROR] can not find iptables", err, true)
	fmt.Printf("[DEBUG] iptables is located at: %s\n", ipTablesPath)

	// block all connections, but 4001 which is used by etcd. We don't use etcd here, but we still need to push one rule to iptables so it blocks everything. So, just leaving this rule as it is.
	cmd := append([]string{"-A"}, "INPUT", "-p", "tcp", "!", "--sport", "4001", "-j", "NFQUEUE", "--queue-num", "0")	
	err = exec.Command(ipTablesPath, cmd...).Run()
	handleError("[ERROR] Could not add iptables rules", err, true)
	fmt.Println("[DEBUG] Added iptables rule to capture all INPUT but port 22")


	// Start netfilter to capture incoming packets
	nfq, err = netfilter.NewNFQueue(0, 10000, netfilter.NF_DEFAULT_PACKET_SIZE)
	handleError("[ERROR] could not create new netfilter queue", err, true)
	defer nfq.Close()

	/* Create syscall raw socket for writing packets out*/
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	// Listen for packets 
	packets := nfq.GetPackets()

	// open file to keep statistics
	statFile, err := os.Create("nat_map_capacity.txt")
	handleError("[ERROR] Could not create file to record stats", err, true)
	defer statFile.Close()

	// _, err = statFile.WriteString("nat_time, len(nat_map)\n")
	// handleError("[ERROR] Could not write to stats file", err, true)
	
	for true {
		select {

		// process each incoming packet and record number of entries in the NAT map (len(nat_map)) when each packet was received
		case p := <-packets:
			// file entry format (nat_time, len(nat_map))
			_, err = statFile.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10) + " " + strconv.Itoa(len(nat_map)) + "\n")
			handleError("[ERROR] Could not write to statFile", err, true)
			
			// fmt.Println("[DEBUG] MAIN Incoming packet before processing")
			// fmt.Println(p.Packet)
			IPlayer := p.Packet.Layer(layers.LayerTypeIPv4)
			ipv4, _ := IPlayer.(*layers.IPv4)
			payload := ipv4.LayerPayload()

			TCPlayer := p.Packet.Layer(layers.LayerTypeTCP)
			tcp, _ := TCPlayer.(*layers.TCP)

			// fmt.Printf("src %s:%s, dst %s:%s, payload: %s\n", ipv4.SrcIP.String(), tcp.SrcPort, ipv4.DstIP.String(), tcp.DstPort, payload)

			if ipv4.SrcIP.String()+":"+strconv.Itoa(int(tcp.SrcPort)) == servAddr {
				/* response packet
				 * if we have more than one srcIP, we will have to remap accordingly
				 * also if we are sending to more than one service, servAddr will have to iterate to find a matching one
				 */
				clientIPAddr := nat_map[int(tcp.DstPort)]
				sendRedirect(int(tcp.DstPort), fd, clientIPAddr, ipv4, payload)
				// fmt.Println("[DEBUG] Processed response from servAddr: ", servAddr)
				// fmt.Printf("src %s:%s, dst %s:%s, payload: %s\n", ipv4.SrcIP.String(), tcp.SrcPort, ipv4.DstIP.String(), tcp.DstPort, payload)
			} else {
				/* keep mappings and statistics
				 * currently we can just use tcp.SrcPort, because we do not remap since we only receive one IP address
				 * from kube-proxy who can ensure it does not reuse SrcPorts that is already in use. If expanded to multi-IPs srcs
				 * in the future, we'll have to have a mapping of the srcPort from kube-proxy to a new Port
				 */
				nat_map[int(tcp.SrcPort)] = ipv4.SrcIP
				// ip_count[ipv4.SrcIP.String()] = ip_count[ipv4.SrcIP.String()] + 1
				// fmt.Printf("[DEBUG] ip_count addr: %s, cnt: %d\n", ipv4.SrcIP.String(), ip_count[ipv4.SrcIP.String()])
				// fmt.Printf("[DEBUG] nat_map int: %d, val: %s\n", tcp.SrcPort, nat_map[int(tcp.SrcPort)])

				// We will redirect this packet to servAddr 
				ipServ, port := getIPandPort(servAddr)
				sendRedirect(port, fd, ipServ, ipv4, payload)
				// fmt.Println("[DEBUG] Processed Incoming Packet")
				// fmt.Printf("src %s:%s, dst %s:%s, payload: %s\n", ipv4.SrcIP.String(), tcp.SrcPort, ipv4.DstIP.String(), tcp.DstPort, payload)
			}
			p.SetVerdict(netfilter.NF_DROP)
		}

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

// extract IP address and port number from string
func getIPandPort(addr string) (servIP net.IP, port int) {
	result := strings.Split(addr, ":")
	servIP = net.ParseIP(result[0])
	port, _ = strconv.Atoi(result[1])
	return
}

// compute the checksum
func csum(b []byte) uint16 {
	var s uint32
	// fmt.Printf("% X\n", b)
	for i := 0; i < len(b); i += 2 {
		s += uint32(b[i+1])<<8 | uint32(b[i])
		// fmt.Println(s)
	}
	// add back the carry
	s = s>>16 + s&0xffff
	s = s + s>>16
	return uint16(^s)
}
