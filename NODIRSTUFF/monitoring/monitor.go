package main

import ("fmt"
	"os"
	"time"

	"github.com/google/cadvisor/client"
	"github.com/google/cadvisor/info"
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
	fullContName := "/docker/"+containerName

	for {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.ContainerInfo(fullContName, &request)
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
func getContainerMemInUse(cadvisorClient *client.Client, containerName string) {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}
	fullContName := "/docker/"+containerName

	for {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.ContainerInfo(fullContName, &request)
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
		usedPercentile := memoryStats.WorkingSet / memorySpecs.Limit
		fmt.Printf("WorkingSet = %d, Usage = %d, Percentage = %d\n", memoryStats.WorkingSet, memoryStats.Usage, usedPercentile*100)
		time.Sleep(READ_FREQ)
	}
}

// get container's network stats
func getContainerNetInUse(cadvisorClient *client.Client, containerName string) {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}
	fullContName := "/docker/"+containerName

	for {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.ContainerInfo(fullContName, &request)
		handleError("Could not get container info", err, true)

		// returns *ContainerStats
		stats := cInfo.Stats[0]

		// returns *NetworkStats
		networkStats := stats.Network
		fmt.Println("networkStats =", networkStats)
	
		time.Sleep(READ_FREQ)
	}
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

func main() {
	// cadvisorClient, err := client.NewClient("http://localhost:8080/")
	// checkFail("Could not create NewClient", err, true)	

	// // returns MachineInfo
	// mInfo, err := cadvisorClient.MachineInfo()
	// checkFail("Could not get MachineInfo", err, false)
	// // fmt.Println("\nmachineInfo = ", mInfo)
	// fmt.Printf("\nMemoryCapacity = %d\n", int64(mInfo.MemoryCapacity))
	// hostRam = mInfo.MemoryCapacity


	// fmt.Println("\nTopology = ", mInfo.Topology)
	// fmt.Println("\nFilesystems = ", mInfo.Filesystems)

	// containerName := "fea8bfdc36b33d032a9dbc5c5d62ee39335d54b877302851a1cee03e1ecf5f81"

	// // Max number of stats to return.
	// numStats := 1
	// request := info.ContainerInfoRequest{NumStats: numStats}

	// // returns ContainerInfo struct
	// fullContName := "/docker/"+containerName
	// cInfo, err := cadvisorClient.ContainerInfo(fullContName, &request)
	// checkFail("Could not get container info", err, true)
	// fmt.Println("\ncInfo =", cInfo)

	// fmt.Println("Name =", cInfo.Name)
	// fmt.Println("Aliases =", cInfo.Aliases)
	// // fmt.Println("Namespace = ", cInfo.Namespace)

	// getContainerMemInUse(cadvisorClient, containerName)
	// getContainerNetInUse(cadvisorClient, containerName)
	// getContainerCpuInUse(cadvisorClient, containerName)


    // endpoint := "unix:///var/run/docker.sock"
    endpoint := "http://localhost:8080"

	// getHostMemInUse(cadvisorClient)
	ProvisionCAdvisor(endpoint)
	// ProvisionContainer()
}

