package main

import ("fmt"
	"os"
	"time"
	"os/exec"
	"strings"
	"bytes"
	"bufio"
	"log"
	"strconv"

	"github.com/google/cadvisor/client"
	info "github.com/google/cadvisor/info/v1"
)

const READ_FREQ = 2 * time.Second

var hostRam int64 

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

// get CPU stats.
func getContainerCpuInUse(cadvisorClient *client.Client, containerName string) {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}

	for {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.ContainerInfo(containerName, &request)
		handleError("Could not get container info", err, true)

		// returns ContainerSpec
		spec := cInfo.Spec
		fmt.Println("cpuSpec =", spec.Cpu)
		fmt.Println("HasCpu =", spec.HasCpu)

		// returns CpuStats
		cpuStats := cInfo.Stats[0].Cpu
		fmt.Println("cpuStats =", cpuStats)
	
		time.Sleep(READ_FREQ)
	}
}

// Returns percentage of the memory being used. 
func getContainerMemInUse(cadvisorClient *client.Client, containerName string) (usedPercentile int)  {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}

	// returns ContainerInfo struct
	cInfo, err := cadvisorClient.ContainerInfo(containerName, &request)
	handleError("Could not get container info", err, true)

	// returns *ContainerSpec
	spec := cInfo.Spec

	// returns *ContainerStats
	stats := cInfo.Stats[0]

	// returns *MemorySpec
	memorySpecs := spec.Memory
	// fmt.Println("\nLimit =", memorySpecs.Limit)
	// fmt.Println("Reservation =", memorySpecs.Reservation)
	// fmt.Println("SwapLimit =", memorySpecs.SwapLimit)
	
	// returns MemoryStats
	memoryStats := stats.Memory
	usedPercentile = 100 * int(memoryStats.WorkingSet / memorySpecs.Limit)
	fmt.Printf("WorkingSet = %d, Usage = %d, Max = %d, Perc = %d\n", memoryStats.WorkingSet, memoryStats.Usage, memorySpecs.Limit, usedPercentile)
	
	return usedPercentile
}

// get container's network stats
func getContainerNetInUse(cadvisorClient *client.Client, containerName string) {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}

	for {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.ContainerInfo(containerName, &request)
		handleError("Could not get container info", err, true)

		// returns *ContainerStats
		stats := cInfo.Stats[0]

		// returns *NetworkStats
		networkStats := stats.Network
		fmt.Println("networkStats =", networkStats)
	
		time.Sleep(READ_FREQ)
	}

	// NetworkStats are:
	// type NetworkStats struct {
	// 	// Cumulative count of bytes received.
	// 	RxBytes uint64 `json:"rx_bytes"`
	// 	// Cumulative count of packets received.
	// 	RxPackets uint64 `json:"rx_packets"`
	// 	// Cumulative count of receive errors encountered.
	// 	RxErrors uint64 `json:"rx_errors"`
	// 	// Cumulative count of packets dropped while receiving.
	// 	RxDropped uint64 `json:"rx_dropped"`
	// 	// Cumulative count of bytes transmitted.
	// 	TxBytes uint64 `json:"tx_bytes"`
	// 	// Cumulative count of packets transmitted.
	// 	TxPackets uint64 `json:"tx_packets"`
	// 	// Cumulative count of transmit errors encountered.
	// 	TxErrors uint64 `json:"tx_errors"`
	// 	// Cumulative count of packets dropped while transmitting.
	// 	TxDropped uint64 `json:"tx_dropped"`
	// }
}


// Returns percentage of the memory being used in the host machine. 
func getHostMemInUse(cadvisorClient *client.Client) {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}
	rootPath := "/"

	for {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.ContainerInfo(rootPath, &request)
		handleError("Could not get the host info", err, true)

		// // returns *ContainerSpec
		// spec := cInfo.Spec
		// // returns *MemorySpec
		// memorySpecs := spec.Memory
		// fmt.Printf("\nLimit = %d\n", uint64(memorySpecs.Limit))
		// fmt.Printf("Reservation = %d\n", uint64(memorySpecs.Reservation))
		// fmt.Println("SwapLimit = %d\n", uint64(memorySpecs.SwapLimit))
		
		// returns *ContainerStats
		stats := cInfo.Stats[0]

		// returns MemoryStats
		memoryStats := stats.Memory
		fmt.Printf("WorkingSet = %d, Usage = %d\n", memoryStats.WorkingSet, memoryStats.Usage)
		usedPercentile := 100 * int64(memoryStats.Usage) / int64(hostRam)
		fmt.Printf("mem usage: %d\n", usedPercentile)
		time.Sleep(READ_FREQ)
	}
}


// get current number of replicas of the given replicationcontroller 
func getCurrentReplica(ctrlName string) (replicas int) {
	// kubectl get rc --server=198.162.52.217:8080

	// right now finding number of replicas of the given contoller is done manually;
	// by string parsing. Fixme: automate this via kubectl golang client.
	cmd := exec.Command("kubectl", "get", "rc", "--server=198.162.52.217:8080")
	// cmd.Stdin = strings.NewReader()
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("output is %q\n", out.String())

	// go through each line and find the line which starts with the given ctrlName	
	scanner := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	for scanner.Scan() {
		lineText := scanner.Text()
		if strings.HasPrefix(lineText, ctrlName) {
			// fmt.Println(lineText)
			// get the last digit of this line and return it as number of replicas
			// fixme: this is really naive way, e.g. does not work when number of
			// replicas are two or more digit number.
			replicas, _ = strconv.Atoi(lineText[len(lineText)-1:])
		}
		
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("reading standard input:", err)
	}
	// fmt.Println(ctrlName, "has", replicas, "replicas")
	return
}

// Get current number of replicas of the given replicationcontroller.
// Return new size of the RC, i.e., returns value equal to newSize if success,
// -1 otherwise.
func resizeRC(ctrlName string, newSize int) (finalSize int) {
	// we execute following command to resize number of replicas: 
	// kubectl resize --replicas=1 rc firewallcontroller --server=198.162.52.217:8080
	// Fixme: use kubectl Go client to do this in more general way

	replicaSize := fmt.Sprintf("--replicas=%d", newSize)
	// fmt.Println(replicaSize)
	cmd := exec.Command("kubectl", "resize", replicaSize, "rc", ctrlName, "--server=198.162.52.217:8080")
	// cmd.Stdin = strings.NewReader()
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("output is %s\n", out.String())

	// successful execution returns string "resized\n", anything else is failure.
	if out.String() == "resized\n" {
		finalSize = newSize
	} else {
		finalSize = -1
	}

	return
}

// provision additional container when current container reaches specific RAM threshold.
func ramBasedScaling(cadvisorClient *client.Client, containerName string) {
	
	// var int usedPercentile 
	// threshold := 80
	counter := 0
	increment := false
	decrement := false

	for {
		fmt.Println("counter = ", counter)
		increment = false
		decrement = false
		
		// gets percentage utilization of the container
		// usedPercentile = getContainerMemInUse(cadvisorClient, containerName)

		// Decide if container reached the predefined threshold.
		// Since k8s does not support resource limitation yet, 
		// we will manually trigger scaling up and scaling down. 
		// fixme: adopt real monitoring when it k8s supports it.
		// Refer to "Kubernetes resource monitoring" section of VMC notes for more info.
		// if usedPercentile > threshold {
		// 	// do scale up
		// }
		// if usedPercentile < threshold {
		// 	// do scale down
		// }


		// trigger scale up
		if counter == 1 {
			increment = true
		}

		// trigger scale down
		if counter == 10 {
			decrement = true
		}

		ctrlName := "firewallcontroller" // "echoservercontroller"
		if increment {
			// increment number of containers by replicationcontroller resize command			
			currReplicas := getCurrentReplica(ctrlName)
			fmt.Println("currReplicas =", currReplicas)

			newReplicas := currReplicas + 1
			if newReplicas == resizeRC(ctrlName, newReplicas) {
				fmt.Println("resize is successful")
			} else {
				fmt.Println("resize is not successful")
			}
		}

		if decrement {
			// decrement number of containers by replicationcontroller resize command			
			currReplicas := getCurrentReplica(ctrlName)
			fmt.Println("currReplicas =", currReplicas)

			newReplicas := currReplicas - 1
			if newReplicas == resizeRC(ctrlName, newReplicas) {
				fmt.Println("resize is successful")
			} else {
				fmt.Println("resize is not successful")
			}
		}

		time.Sleep(READ_FREQ)
		counter++
	}
}

// provision additional container when current container reaches specific CPU threshold.
func cpuBasedScaling(cadvisorClient *client.Client, containerName string) {
	
	// gets percentage utilization of the container
	getContainerCpuInUse(cadvisorClient, containerName)
}

// provision additional container when current container reaches specific network threshold.
func netBasedScaling(cadvisorClient *client.Client, containerName string) {
	getContainerNetInUse(cadvisorClient, containerName)
}

func main() {
	cadvisorClient, err := client.NewClient("http://localhost:9090/")
	handleError("Could not create NewClient", err, true)	

	contName := "2da04ca875d0c08c49c4820cee74f3e37a5f64c7d5363b06d7580d7d8661d9bc"

	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}

	// returns ContainerInfo struct
	cInfo, err := cadvisorClient.DockerContainer(contName, &request)
	handleError("Could not get container info", err, true)
	fmt.Println("\ncInfo =", cInfo)

	// fmt.Println("Name =", cInfo.Name)
	// fmt.Println("Aliases =", cInfo.Aliases)
	// fmt.Println("Namespace = ", cInfo.Namespace)

	ramBasedScaling(cadvisorClient, contName)
	// cpuBasedScaling(cadvisorClient, fullContName)
	// netBasedScaling(cadvisorClient, fullContName)
}

