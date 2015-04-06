// Adopted from golang_firewall_v2. Changes are:
// - change threshold from incremental counter to msgs per second

package main

import (
	//	"./Netfilter"
	"fmt"
	"net"
	"os"
	"time"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"code.google.com/p/gopacket"
	"code.google.com/p/gopacket/layers"
	"code.google.com/p/gopacket/pcap"
	"github.com/openshift/geard/pkg/go-netfilter-queue"
	"github.com/coreos/go-etcd/etcd"
)

var ip_count map[string]int
var nat_map map[int]net.IP

var localThresPath string // etcd path from which we read the threshold of this firewall instance
var operThres int // operational threshold, continiously updated from etcd 


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

// run for the first time and determine operational value of the threshold
func registerAndPutThres(initThres int) {

	// Create client to connect to the etcd.
	k8sMaster := []string{"http://198.162.52.217:4001"}
	client := etcd.NewClient(k8sMaster)

	aggrThresPath := "/firewall/aggr"

	// check if aggregate path already exists
	ret, err := client.Get(aggrThresPath, false, false)
	if err != nil {
		// Path does not exist. This is the first instance of the Pod.
		// Insert aggregate threshold value at /firewall/aggr/
		ret, err = client.Set(aggrThresPath, strconv.Itoa(initThres), 0)
		handleError("[ERROR] Could not Set the aggr value to the etcd", err, true)
		fmt.Println("[INFO] Successfully set the aggregate threshold")

		// assign total aggregate value to this firewall instance.
		localThresPath = "/firewall/ins1"
		ret, err = client.Set(localThresPath, strconv.Itoa(initThres), 0)
		handleError("[ERROR] Could not set local value to the etcd", err, true)
		fmt.Printf("[INFO] Successfully set initThres: %d to [%s]\n", initThres, localThresPath)

	} else {
		// Aggregate path does exist. This is not the first instance of the Pod.
		// Count how many Pods exist and divide /firewall/aggr value evenly.
		// Update local threshold of the all instances with new value.
		ret, err = client.Get("/firewall/aggr", false, true)
		handleError("[ERROR] Could not read aggr value from etcd", err, true)
		aggrThresVal, _ := strconv.Atoi(ret.Node.Value)
		fmt.Println("[INFO] aggrThresVal =", aggrThresVal)

		ret, err = client.Get("/firewall", false, true)
		handleError("[ERROR] Could not read value from the etcd", err, true)

		totalInstances := ret.Node.Nodes.Len() - 1 // -1 for /firewall/aggr
		fmt.Println("[INFO] number of total instances =", totalInstances)

		newThresVal := aggrThresVal / (totalInstances + 1) // +1 for the instance being created
		fmt.Println("[INFO] newThresVal =", newThresVal)

		for _, node := range ret.Node.Nodes {
			// assign newThresVal to all instances, except /firewall/aggr
			if node.Key != aggrThresPath {
				ret, err = client.Set(node.Key, strconv.Itoa(newThresVal), 0)
				handleError(fmt.Sprintf("[ERROR] Could not set newThresVal to %s", node.Key), err, true)
				fmt.Printf("[INFO] Successfully set newThresVal: %d to [%s]\n", newThresVal, node.Key)
			}			
		}

		// assign newThresVal to the instance being created
		localThresPath = "/firewall/ins" + strconv.Itoa(totalInstances+1)
		ret, err = client.Set(localThresPath, strconv.Itoa(newThresVal), 0)
		handleError(fmt.Sprintf("[ERROR] Could not set newThresVal to %s", localThresPath), err, true)
		fmt.Printf("[INFO] Successfully set newThresVal: %d to [%s]\n", newThresVal, localThresPath)
	}	
}

// continously called to update operational threshold value from the etcd
func updateOperThres() {
	k8sMaster := []string{"http://198.162.52.217:4001"}
	client := etcd.NewClient(k8sMaster)

	ret, err := client.Get(localThresPath, false, false)
	handleError("[ERROR] Could not read value from the etcd", err, true)

	operThres, _ = strconv.Atoi(ret.Node.Value)
	fmt.Println("[INFO] Updated operThres to:", ret.Node.Value)
}

func main() {
	/* Firewall local state variables */
	servAddr := "10.100.81.31:3333" //service address IP:PORT of ECHOFILTEREDSERVICE
	machineAddr := ""               //address of container
	path := ""                      //path to iptables
	var ifs []pcap.Interface        //List of interfaces
	var env_var []string            //List of environment variables
	var nfq *netfilter.NFQueue      //Netfilter queue
	var err error
	const THRESHOLD int = 100 // indicates number of messages per second

	registerAndPutThres(THRESHOLD)
	
	// ticker to clear msg counter every second
	tickChan := time.NewTicker(time.Second).C

	ip_count = make(map[string]int)
	nat_map = make(map[int]net.IP)

	/* Find machine IP and service IP */
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

	//	servAddr = os.Args[1]
	fmt.Printf("[DEBUG] server address is %s\n", servAddr)

	/* Install IPTABLE rule to bypass kernel network stack */
	path, err = exec.LookPath("iptables")
	handleError("[ERROR] can not find iptables", err, true)
	fmt.Printf("[DEBUG] iptables is located at: %s\n", path)

	// cmd := append([]string{"-A"}, "INPUT", "-p", "tcp", "-j", "NFQUEUE", "--queue-num", "0")
	// err = exec.Command(path, cmd...).Run()
	// handleError("Could not add iptables rules", err, true)
	// fmt.Printf("Added iptable rules to bypass kernel network stack\n")

	/* Start netfilter to capture incoming packets*/
	nfq, err = netfilter.NewNFQueue(0, 100, netfilter.NF_DEFAULT_PACKET_SIZE)
	handleError("[ERROR] could not create new netfilter queue", err, true)
	defer nfq.Close()

	/* Create syscall raw socket for writing packets out*/
	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	/* Listen for packets */
	packets := nfq.GetPackets()


	for true {
		select {

		case <- tickChan:
			for fwAddr, counterVal := range ip_count {
				ip_count[fwAddr] = 0
				fmt.Printf("[INFO] cleared message counter for [%s] counter was: %d\n", fwAddr, counterVal)
			}
			// ip_count[ipv4.SrcIP.String()] = 0
			// update operational threshold value from the etcd
			updateOperThres()

		case p := <-packets:
			fmt.Println("[INFO] MAIN Incoming packet before processing:\n", p.Packet)
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
				fmt.Println("[INFO] Processed response from servAddr")

			} else if ip_count[ipv4.SrcIP.String()] < operThres {
				/* keep mappings and statistics
				 * currently we can just use tcp.SrcPort, because we do not remap since we only receive one IP address
				 * from kube-proxy who can ensure it does not reuse SrcPorts that is already in use. If expanded to multi-IPs srcs
				 * in the future, we'll have to have a mapping of the srcPort from kube-proxy to a new Port
				 */
				nat_map[int(tcp.SrcPort)] = ipv4.SrcIP
				ip_count[ipv4.SrcIP.String()] = ip_count[ipv4.SrcIP.String()] + 1
				fmt.Printf("[INFO] ip_count addr: %s, cnt: %d\n", ipv4.SrcIP.String(), ip_count[ipv4.SrcIP.String()])
				fmt.Printf("[INFO] nat_map int: %d, val: %s\n", tcp.SrcPort, nat_map[int(tcp.SrcPort)])

				/* We will redirect this packet to servAddr */
				ipServ, port := getIPandPort(servAddr)
				sendRedirect(port, fd, ipServ, ipv4, payload)
				fmt.Println("[INFO] Processed Incoming Packet")
			} else {
				fmt.Printf("[INFO] Number of packets exceeded threshold\n")
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
