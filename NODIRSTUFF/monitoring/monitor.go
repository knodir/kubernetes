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
	"flag"
	// "net/http"
	// "io/ioutil"

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
		cInfo, err := cadvisorClient.DockerContainer(containerName, &request)
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

// Returns percentage of the memory being used. Returns -1 if container is terminated.
func getContainerMemInUse(cadvisorClient *client.Client, 
	shortContName string, fullContName string) (usedPercentile int)  {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}

	// returns ContainerInfo struct
	cInfo, err := cadvisorClient.DockerContainer(fullContName, &request)
	if err != nil {
		// this means container was deleted (probably by replication controller).
		// we return -1 indicating termination of this container.
		fmt.Printf("[INFO][%s] this container got terminated\n", shortContName)
		return -1
		// handleError(fmt.Sprintf("[ERROR] Could not get container [%s] info", shortContName), err, true)
	}

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
	// fmt.Printf("[DEBUG] WorkingSet = %d, Usage = %d, Max = %d, Perc = %d\n", memoryStats.WorkingSet, memoryStats.Usage, memorySpecs.Limit, usedPercentile)
	
	return 
}

// get container's network stats
func getContainerNetInUse(cadvisorClient *client.Client, containerName string) {
	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}

	for {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.DockerContainer(containerName, &request)
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
func ramBasedScaling(cadvisorClient *client.Client, ctrlName string,
	shortContName string, fullContName string) {
	
	// var int usedPercentile 
	// threshold := 80
	counter := 0
	increment := false
	decrement := false

	for {
		fmt.Printf("[INFO][%s] counter = %d\n", shortContName, counter)
		increment = false
		decrement = false
		
		// gets percentage utilization of the container
		usedPercentile := getContainerMemInUse(cadvisorClient, shortContName, fullContName)
		// fmt.Printf("[DEBUG][%s] usedPercentile = %d", shortContName, usedPercentile)

		if usedPercentile == -1 {
			// monitoring module return -1 percentile when container gets terminated.
			// we need to terminate scaling mechanism as well; by exiting endless loop.
			fmt.Printf("[INFO][%s] Stopping container since monitoring stopped.\n", shortContName)
			break
		}

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
		if counter == 5 {
			increment = true
		}

		// trigger scale down
		if counter == 15 {
			decrement = true
		}

		if increment {
			// increment number of containers via replicationcontroller resize command			
			currReplicas := getCurrentReplica(ctrlName)
			fmt.Printf("[INFO][%s] currReplicas = %d\n", shortContName, currReplicas)

			newReplicas := currReplicas + 1
			if newReplicas == resizeRC(ctrlName, newReplicas) {
				fmt.Printf("[INFO][%s] resize is successful\n", shortContName)
			} else {
				fmt.Printf("[INFO][%s] resize is not successful\n", shortContName)
			}
		}

		if decrement {
			// decrement number of containers by replicationcontroller resize command			
			currReplicas := getCurrentReplica(ctrlName)
			fmt.Printf("[INFO][%s] currReplicas = %d\n", shortContName, currReplicas)

			newReplicas := currReplicas - 1
			if newReplicas == resizeRC(ctrlName, newReplicas) {
				fmt.Printf("[INFO][%s] resize is successful\n", shortContName)
			} else {
				fmt.Printf("[INFO][%s] resize is not successful\n", shortContName)
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


// returns name of the Docker image pod is running
func getPodImage(podName string) (dockerImage string) {

	// // kubectl exposes REST API, which will return all information in JSON.
	// // fixme: it would be more general solution if this JSON format is parsed 
	// // to retrieve Docker image name. However, as a quick & dirty solution, 
	// // I'll use exec(kubectl) and parse its string result. One also could consider
	// // writing go-kubectl client to expose convenient data structures to access JSON field,
	// // just like go-dockerclient parses JSON commands returned from docker CLI cmd. 
	// // Here is the short sample to retrieve kubectl REST result.
	// resp, err := http.Get("http://198.162.52.217:8080/api/v1beta1/pods")
	// if err != nil {
	// 	// handle error
	// }
	// defer resp.Body.Close()
	// body, err := ioutil.ReadAll(resp.Body)
	// responseJson := string(body)
	// fmt.Printf("responseJson = %s\n", responseJson)

	cmd := exec.Command("kubectl", "get", "pods", "--server=198.162.52.217:8080")
	// cmd.Stdin = strings.NewReader()
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("output is %s\n", out.String())

	// go through each line and find the line which starts with the given ctrlName	
	scanner := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	for scanner.Scan() {
		lineText := scanner.Text()
		if strings.HasPrefix(lineText, podName) {
			fields := strings.Fields(lineText)

			// for index := range fields {
			// 	fmt.Println(index, fields[index])
			// }

			// kubectl get pods returns pod information in following order
			// POD, IP, CONTAINER(S), IMAGE(S), HOST, LABELS, STATUS
			// we need to get field[3] to get the image pod running
			dockerImage = fields[3]

			// since pod always runs the same image, we can break the loop once we found pod
			break
		}
	}
	return
}

// get short and full ID of the containers running this image
func getContIDs(imageName string) (contIDs map[string]string) {
	contIDs = make(map[string]string)
	cmd := exec.Command("docker", "ps", "--no-trunc")
	// cmd.Stdin = strings.NewReader()
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("output is %s\n", out.String())

	// go through each line and find the line which has given imageName
	scanner := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	for scanner.Scan() {
		lineText := scanner.Text()
		// docker ps return information of running containers in the following order
		// CONTAINER ID, IMAGE, COMMAND, CREATED, STATUS, PORTS, NAMES
		// we get field[1] to check if this is the image we want. 
		// If so, field[0] is a container ID. We assign first 12 bytes of a container
		// as a short ID, and leave entire length for a full ID.
		fields := strings.Fields(lineText)
		dockerImage := fields[1]
		if strings.HasPrefix(dockerImage, imageName) {
			// fmt.Println(fields[0][0:12], fields[0])
			contIDs[fields[0][0:12]] = fields[0]
		}
	}
	return
}

func main() {

	// maps Docker short ID (12 bytes) to full ID (64 bytes)
	var prevContIDs map[string]string
	var currContIDs map[string]string
	var prevSize, currSize int


	controllerName := flag.String("pod", "None", "specify the name of the pod to monitor")
	flag.Parse()
	fmt.Println("controllerName =", *controllerName)

	// create cAdvisor client to monitor Docker instances
	cadvisorClient, err := client.NewClient("http://localhost:9090/")
	handleError("[ERROR] Could not create NewClient", err, true)

	// Max number of stats to return.
	numStats := 1
	request := info.ContainerInfoRequest{NumStats: numStats}


	// get name of the image pod is running
	dockerImage := getPodImage(*controllerName)
	fmt.Println("dockerImage =", dockerImage)		

	// get container ID running this image
	prevContIDs = getContIDs(dockerImage)
	fmt.Println("prevContIDs =", prevContIDs)

	// run monitoring mechanism for all newly created container
	for shortContName, fullContName := range prevContIDs {
		// returns ContainerInfo struct
		cInfo, err := cadvisorClient.DockerContainer(fullContName, &request)
		handleError("[ERROR] Could not get container info", err, true)
		fmt.Println("\ncInfo =", cInfo)

		// fmt.Printf("Name = %s, Aliases = %s, Namespace = %s", cInfo.Name, cInfo.Aliases, cInfo.Namespace)

		// run each Docker monitoring in a separate goroutine
		go ramBasedScaling(cadvisorClient, *controllerName, shortContName, fullContName)
		// cpuBasedScaling(cadvisorClient, fullContName)
		// netBasedScaling(cadvisorClient, fullContName)				
	}

	// continuously check Docker instances of this pod (i.e., running this image)
	// and monitor each instance for resource consumption.
	for {
		time.Sleep(READ_FREQ)
		newContainers := make(map[string]string)
		deletedContainers := make(map[string]string)

		// get currently running container ID with this image
		currContIDs = getContIDs(dockerImage)
		fmt.Println("[DEBUG] currContIDs =", currContIDs)

		// If number of previous and currently running containers does not match that means
		// number of pod was resized. We need to find whether it was increased 
		// or decreased, and which pod exactly got created or deleted. If they do match, 
		// we check if both sets have same items.
		prevSize = len(prevContIDs)
		currSize = len(currContIDs)
		if currSize != prevSize {
			fmt.Printf("[INFO] Pod was resized from [%d] to [%d]\n", prevSize, currSize)
			if (currSize > prevSize) {
				// Pod size was increased. Go over the current list of containers and 
				// find which pod(s) are added.				
				for key, val := range currContIDs {
					if prevContIDs[key] == "" {
						newContainers[key] = val
					}
				}
				fmt.Println("[DEBUG] newContainers =", newContainers)
			} else {
				// Pod size was decreased. Go over the previous list of containers and 
				// find which pod(s) were deleted.
				for key, val := range prevContIDs {
					if currContIDs[key] == "" {
						deletedContainers[key] = val
					}
				}
				fmt.Println("[DEBUG] deletedContainers =", deletedContainers)
			}
		} else {
			// This case rarely happens. It is most likely to happen when READ_FREQ duration is too large, 
			// i.e., one pod dies and replication controller creates another one to replace it.
			// Both of these actions should happen within the same READ_FREQ interval. 
			// In such case, monitoring module should go over each container's state in etcd,
			// remove stale values and update other ones accordingly.

			fmt.Println("numbers are the same")
		}
		prevContIDs = currContIDs

		if len(newContainers) != 0 {
			// run monitoring mechanism for all newly created container
			for shortContName, fullContName := range newContainers {
				// returns ContainerInfo struct
				cInfo, err := cadvisorClient.DockerContainer(fullContName, &request)
				handleError(fmt.Sprintf("[ERROR][%s] Could not get container info", shortContName), err, true)
				// fmt.Printf("[INFO][%s] cInfo = %s\n", shortContName, cInfo)

				go ramBasedScaling(cadvisorClient, *controllerName, shortContName, fullContName)
			}
		}
	}
}

