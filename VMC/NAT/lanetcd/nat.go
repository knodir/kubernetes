// NAT with etcd on another VirtualBox VM but the same physical host.

package main

import (
  "bytes"
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
  "github.com/coreos/go-etcd/etcd"
  "github.com/golang/glog"
  "github.com/openshift/geard/pkg/go-netfilter-queue"
)

var natMap map[int]net.IP // we use this for stat purposes, not the functionality
var etcdAddress []string = []string{"http://192.168.56.220:2379"}

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
  var servAddr string        // service address IP:PORT of ECHOFILTEREDSERVICE; retrived on runtime
  var machineAddr string     // address of container
  var ipTablesPath string    // path to iptables
  var ifs []pcap.Interface   // List of interfaces
  var env_var []string       // List of environment variables
  var nfq *netfilter.NFQueue // Netfilter queue
  var err error

  // we will store content of this map in etcd. Right now this map structure has
  // portNumber:IP_address mapping, e.g, 12345:10.0.0.1, 12346:10.0.0.2,
  // 12347:10.0.0.3 and so on. They will be stored at etcd in the following
  // format: /nat/ins1/port/IP_address. Examples /nat/ins1/12345/10.0.0.1,
  // /nat/ins1/12346/10.0.0.2, /nat/ins1/12347/10.0.0.3
  natMap = make(map[int]net.IP)

  // Find machine IP and service IP
  fmt.Printf("[DEBUG] Listing all devices\n")
  ifs, err = pcap.FindAllDevs()
  if err != nil {
    glog.Fatalf("failed to get network devices :: %v", err)
  } else {
    fmt.Printf("successfully got network devices :: %v\n", ifs)
  }

  for i := 0; i < len(ifs); i++ {
    if ifs[i].Name == "eth0" {
      machineAddr = ifs[i].Addresses[0].IP.String()
    }
  }

  fmt.Printf("machine address is :: %s\n", machineAddr)

  env_var = os.Environ()
  for i := 1; i < len(env_var); i++ {
    if strings.Contains(env_var[i], "ECHOFILTEREDSERVICE_PORT=") {
      result := strings.Split(env_var[i], "tcp://")
      servAddr = result[1]
    }
  }

  // servAddr = "198.162.52.126" // os.Args[1]
  fmt.Printf("server address is :: %s\n", servAddr)

  // Install IPTABLE rule to bypass kernel network stack
  ipTablesPath, err = exec.LookPath("iptables")
  if err != nil {
    glog.Fatalf("could not find iptables :: %v", err)
  } else {
    fmt.Printf("successfully found iptables at :: %s\n", ipTablesPath)
  }

  // block all connections, but 2379 which is used by etcd.
  cmd := append([]string{"-A"}, "INPUT", "-p", "tcp", "!", "--sport", "2379",
    "-j", "NFQUEUE", "--queue-num", "0")
  err = exec.Command(ipTablesPath, cmd...).Run()
  if err != nil {
    glog.Fatalf("could not add iptables rules :: %v", err)
  } else {
    fmt.Printf("%s %s\n", "successfully added iptables rule to capture",
      "all INPUT except port 2379 for etcd")
  }

  // Start netfilter to capture incoming packets
  nfq, err = netfilter.NewNFQueue(0, 10000, netfilter.NF_DEFAULT_PACKET_SIZE)
  if err != nil {
    glog.Fatalf("failed to create new netfilter queue :: %v", err)
  } else {
    fmt.Printf("successfully created new netfilter queue :: %v\n", err)
  }
  defer nfq.Close()

  /* Create syscall raw socket for writing packets out*/
  fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

  // Listen for packets
  packets := nfq.GetPackets()

  // open file to keep statistics
  statFile, err := os.Create("nat_map_capacity.txt")
  if err != nil {
    glog.Fatalf("could not create file to record stats :: %v", err)
  } else {
    fmt.Printf("successfully created file to record stats\n")
  }
  defer statFile.Close()

  // establish client connection to etcd server
  ectdClient := etcd.NewClient(etcdAddress)
  natEtcdPathRoot := "/nat/ins1/"
  natEtcdPath := ""

  for true {
    select {
    // process each incoming packet and record number of entries in the NAT map
    // (len(nat_map)) when each packet was received
    case p := <-packets:
      // fmt.Println("[DEBUG] MAIN Incoming packet before processing")
      // fmt.Println(p.Packet)
      IPlayer := p.Packet.Layer(layers.LayerTypeIPv4)
      ipv4, _ := IPlayer.(*layers.IPv4)
      payload := ipv4.LayerPayload()

      // appLayer := p.Packet.ApplicationLayer()
      // appPayload := appLayer.Payload()

      // appPayload := p.Packet.String()

      // get source time from the packet payload
      // var timestamp time.Time
      // err = (&timestamp).UnmarshalBinary(payload)
      // if err != nil {
      //   fmt.Println("failed to convert the payload to timestamp")
      // } else {
      //   fmt.Printf("successfully converted the payload to timestamp: %v\n",
      //     timestamp)
      // }

      TCPlayer := p.Packet.Layer(layers.LayerTypeTCP)
      tcp, _ := TCPlayer.(*layers.TCP)

      fmt.Printf("src %s:%s, dst %s:%s, payload: %v\n", ipv4.SrcIP.String(),
        tcp.SrcPort, ipv4.DstIP.String(), tcp.DstPort, payload)

      if fmt.Sprintf("%s:%d", ipv4.SrcIP.String(), int(tcp.SrcPort)) == servAddr {
        // This is response packet. If we have more than one srcIP, we will
        // have to remap each source port to IP. Also ,if we are sending to
        // more than one service, servAddr will have to iterate to find
        // the matching one

        natEtcdPath = natEtcdPathRoot + tcp.DstPort.String()
        ret, err := ectdClient.Get(natEtcdPath, false, false)
        if err != nil {
          glog.Fatalf("could not read value from the etcd :: %v", err)
        } else {
          fmt.Printf("successfully read value from etcd :: %s = %s\n", natEtcdPath, ret)
        }

        // clientIPAddr := nat_map[int(tcp.DstPort)]
        clientIPAddr := net.ParseIP(ret.Node.Value)
        fmt.Printf("clientIPAddr from etcd %s\n", clientIPAddr.String())

        sendRedirect(int(tcp.DstPort), fd, clientIPAddr, ipv4, payload)
        // fmt.Println("[DEBUG] Processed response from servAddr: ", servAddr)
        // fmt.Printf("src %s:%s, dst %s:%s, payload: %s\n", ipv4.SrcIP.String(),
        // tcp.SrcPort, ipv4.DstIP.String(), tcp.DstPort, payload)
      } else {
        // packet from the client
        // this is ugly hack to get timestamp from the payload. There should be
        // better way of getting application layer payload via
        // appLayer := p.Packet.ApplicationLayer()
        // appPayload := appLayer.Payload()
        // but somehow this proper way did not work for me.
        // TODO(knodir) do it in a proper way
        timestampPrefix := []byte{1, 0, 0, 0, 14, 205}
        if bytes.Compare(payload[len(payload)-15:len(payload)-9], timestampPrefix) == 0 {
          // fmt.Printf("\n\ntimestamp equal :: %v\n\n",
          //   payload[len(payload)-15:])
          var timestamp time.Time
          err = (&timestamp).UnmarshalBinary(payload[len(payload)-15:])
          if err != nil {
            fmt.Printf("could not UnmarshalBinary :: %v\n", err)
          } else {
            // fmt.Printf("timestamp :: %v\n", timestamp.UnixNano())
            // file write content: (src_packet_time, nat_time, len(nat_map))
            _, err = statFile.WriteString(fmt.Sprintf("%d %d %d\n",
              timestamp.UnixNano(), time.Now().UnixNano(), len(natMap)))
            if err != nil {
              glog.Fatalf("could not write to statFile :: %v", err)
            }
          }
        } else {
          // fmt.Printf("timestamp not equal:: %v", payload[len(payload)-15:])
        }
        // keep mappings and statistics. Currently we can just use tcp.SrcPort,
        // because we do not remap since we only receive one IP address from
        // kube-proxy who can ensure no SrcPorts are reused if it is already in use.
        // If expanded to multi-IPs srcs in the future, we'll have to have
        // a mapping of the srcPort from kube-proxy to a new Port

        natMap[int(tcp.SrcPort)] = ipv4.SrcIP

        // set port mapping in etcd
        natEtcdPath = fmt.Sprintf("%s%s", natEtcdPathRoot, tcp.SrcPort.String())
        _, err = ectdClient.Set(natEtcdPath, ipv4.SrcIP.String(), 0)
        if err != nil {
          glog.Fatalf("[ERROR] could not set natEtcdPath %s to %s :: %v",
            natEtcdPath, ipv4.SrcIP.String(), err)
        } else {
          fmt.Printf("successfully set natEtcdPath :: %s = %s\n",
            natEtcdPath, ipv4.SrcIP.String())
        }

        // ip_count[ipv4.SrcIP.String()] = ip_count[ipv4.SrcIP.String()] + 1
        // fmt.Printf("[DEBUG] ip_count addr: %s, cnt: %d\n",
        // ipv4.SrcIP.String(), ip_count[ipv4.SrcIP.String()])
        // fmt.Printf("[DEBUG] nat_map int: %d, val: %s\n",
        // tcp.SrcPort, nat_map[int(tcp.SrcPort)])

        // Redirect this packet to servAddr
        ipServ, port := getIPandPort(servAddr)
        sendRedirect(port, fd, ipServ, ipv4, payload)
        // fmt.Println("[DEBUG] Processed Incoming Packet")
        // fmt.Printf("src %s:%s, dst %s:%s, payload: %s\n", ipv4.SrcIP.String(),
        // tcp.SrcPort, ipv4.DstIP.String(), tcp.DstPort, payload)
      }
      p.SetVerdict(netfilter.NF_DROP)
    }
  }
}

func sendRedirect(port, fd int, addr net.IP, ipv4 *layers.IPv4,
  payload []byte) error {
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

  //newPacket := gopacket.NewPacket(packetData, layers.LayerTypeIPv4,
  // gopacket.Default)
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
