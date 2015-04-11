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
	"github.com/coreos/go-etcd/etcd"
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
	if memorySpecs.Limit != 0 {
		usedPercentile = 100 * int(memoryStats.WorkingSet / memorySpecs.Limit)
	} else {
		usedPercentile = 0
	}
	fmt.Printf("[DEBUG][%s] WorkingSet = %d, Usage = %d, Max = %d, Perc = %d\n", shortContName, memoryStats.WorkingSet, memoryStats.Usage, memorySpecs.Limit, usedPercentile)
	
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
func ramBasedScaling(cadvisorClient *client.Client, ctrlName string, fullContName string) {
	
	// var int usedPercentile 
	// threshold := 80
	counter := 0
	increment := false
	decrement := false
	shortContName := fullContName[:12]

	for {
		// fmt.Printf("[INFO][%s] counter = %d\n", shortContName, counter)
		increment = false
		decrement = false
		
		// gets percentage utilization of the container
		usedPercentile := getContainerMemInUse(cadvisorClient, shortContName, fullContName)
		// fmt.Printf("[DEBUG][%s] usedPercentile = %d", shortContName, usedPercentile)

		if usedPercentile == -1 {
			// monitoring module return -1 percentile when container gets terminated.
			// we need to terminate scaling mechanism as well; by exiting endless loop.
			fmt.Printf("[INFO][%s] Stopping container since monitoring has failed.\n", shortContName)
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

		// // trigger scale up
		// if counter == 4 {
		// 	increment = true
		// }
		// // trigger scale down
		// if counter == 17 {
		// 	decrement = true
		// }

		// trigger scale up
		if usedPercentile > 4 {
			increment = true
		}
		// trigger scale down
		if usedPercentile < 0 {
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

// Get container image this RC is running
func getPodImage(ctrlName string) (podImage string) {
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
	handleError("[ERROR] Could not execute command $ kubectl get pods", err, true)
	// fmt.Printf("output is %s\n", out.String())

	// go through each line and find the line which starts with the given ctrlName	
	scanner := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	for scanner.Scan() {
		lineText := scanner.Text()
		if strings.HasPrefix(lineText, ctrlName) {
			fields := strings.Fields(lineText)

			// for index := range fields {
			// 	fmt.Println(index, fields[index])
			// }

			// kubectl get pods returns pod information in following order
			// POD, IP, CONTAINER(S), IMAGE(S), HOST, LABELS, STATUS
			// we need to get field[3] for pod image.
			podImage = fields[3]
		}
	}
	return
}

// Get podname and to container fullname mapping, e.g., [echoservercontroller-02qp4: 9b2531...2e]
func getPodToContMap(ctrlName, imageName string) (podnameToContMap map[string]string) {
	podnameToContMap = make(map[string]string)
	var pods []string // list of currently running pods with this image

	// 
	cmd := exec.Command("kubectl", "get", "pods", "--server=198.162.52.217:8080")
	// cmd.Stdin = strings.NewReader()
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	handleError("[ERROR] Could not execute command $ kubectl get pods", err, true)
	// fmt.Printf("output is %s\n", out.String())

	// go through each line and find the line which starts with the given ctrlName to get podname
	scanner := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	for scanner.Scan() {
		lineText := scanner.Text()
		if strings.HasPrefix(lineText, ctrlName) {
			fields := strings.Fields(lineText)

			// for index := range fields {
			// 	fmt.Println(index, fields[index])
			// }

			// kubectl get pods returns pod information in following order
			// POD, IP, CONTAINER(S), IMAGE(S), HOST, LABELS, STATUS
			// we need to get fields[0] for podname.
			pods = append(pods, fields[0])
		}
	}

	// get Docker IDs corresponding to these pods.
	cmd = exec.Command("docker", "ps", "--no-trunc")
	// cmd.Stdin = strings.NewReader()
	cmd.Stdout = &out
	err = cmd.Run()
	handleError("[ERROR] Could not execute command $ docker ps", err, true)
	// fmt.Printf("output is %s\n", out.String())

	scanner = bufio.NewScanner(bytes.NewReader(out.Bytes()))
	for scanner.Scan() {
		lineText := scanner.Text()
		fields := strings.Fields(lineText)
		// for index := range fields {
		// 	fmt.Println(index, fields[index])
		// }
		// fmt.Println(fields[1], fields[len(fields)-1])

		// docker ps return information of running containers in the following order
		// CONTAINER ID, IMAGE, COMMAND, CREATED, STATUS, PORTS, NAMES
		// fields[0] has full container ID, fields[1] has image name and last element (NAMES) 
		// contains controller name. We identify matching pod/docker by this combination.
		if strings.HasPrefix(fields[1], imageName) && strings.Contains(fields[len(fields)-1], ctrlName) {
			for index := range pods {
				if strings.Contains(fields[len(fields)-1], pods[index]) {
					podnameToContMap[pods[index]] = fields[0]
				}
			}
		}
	}
	return
}

// This function removes the etcd state associated with the deleted container, and 
// rewrites all state associated with other running containers. 
// We clean state by simply removing /firewall/<docker-id> key from etcd. 
// The new state of the running containers are computed from an aggregate value and 
// assigned to each running container's etcd path /firewall/<podname> 
func cleanContState(deletedConts, runningConts map[string]string) {
	k8sMaster := []string{"http://198.162.52.217:4001"}
	client := etcd.NewClient(k8sMaster)

	aggrThresPath := "/firewall/aggr"
	mboxName := "/firewall/"
	var oldThres int

	for podName, _ := range deletedConts {
		// rm shared state for this container
		statePath := mboxName + podName // e.g., /firewall/firewallcontroller-t3w5u
		_, err := client.Delete(statePath, true)
		if err != nil {
			handleError(fmt.Sprintf("[ERROR] Could not clean %s from etcd", statePath), err, false)
		} else {
			fmt.Printf("[INFO] Successfully cleaned %s\n", statePath)
		}
	}

	// check if aggregate path already exists
	ret, err := client.Get(aggrThresPath, false, false)	
	if err != nil {
		// Path does not exist.
		handleError("[WARN] aggr path does not exist", err, false)

		// // assign total aggregate value to this firewall instance.
		// localThresPath = "/firewall/ins1"
		// ret, err = client.Set(localThresPath, strconv.Itoa(initThres), 0)
		// handleError("[ERROR] Could not set local value to the etcd", err, true)
		// fmt.Printf("[INFO] Successfully set initThres: %d to [%s]\n", initThres, localThresPath)

	} else {
		// Aggregate path does exist. This is not the first instance of the Pod.
		// Count how many Pods exist and divide /firewall/aggr value evenly.
		// Update local threshold of the all instances with new value.
		aggrThresVal, _ := strconv.Atoi(ret.Node.Value)
		fmt.Println("[INFO] aggrThresVal =", aggrThresVal)

		ret, err = client.Get("/firewall", false, true)
		handleError("[ERROR] Could not read value from the etcd", err, true)

		totalInstances := ret.Node.Nodes.Len() - 1 // -1 for /firewall/aggr
		fmt.Println("[INFO] number of total instances =", totalInstances)

		newThresVal := aggrThresVal // compute new threshold for new pods

		// if there is more than one instance, divide counter evenly
		if (totalInstances > 1) {
			newThresVal = aggrThresVal / totalInstances
		}

		fmt.Println("[INFO] newThresVal =", newThresVal)

		for _, node := range ret.Node.Nodes {

			// get old threshold value to debugging
			statePath := node.Key
			retVal, err := client.Get(statePath, false, false)
			
			if err != nil {
				handleError(fmt.Sprintf("[WARN] Could not read old threshold value for [%s]", statePath), err, false)	
			} else {
				oldThres, _ = strconv.Atoi(retVal.Node.Value)
				// fmt.Println("[INFO] oldThres =", oldThres)
			}

			// assign newThresVal to all instances, except /firewall/aggr
			if node.Key != aggrThresPath {
				_, err = client.Set(node.Key, strconv.Itoa(newThresVal), 0)
				handleError(fmt.Sprintf("[ERROR] Could not set newThresVal to %s", node.Key), err, true)
				fmt.Printf("[INFO] Successfully changed old threshold %d to newThresVal %d for [%s]\n", oldThres, newThresVal, node.Key)
			}
		}
	}	
	
}

func main() {

	// maps Docker short ID (12 bytes) to full ID (64 bytes)
	var prevPodToContMap map[string]string
	var currPodToContMap map[string]string
	var prevSize, currSize int


	controllerName := flag.String("controller", "None", "specify the name of the controller to monitor")
	flag.Parse()

	if *controllerName == "None" {
		fmt.Println("Usage: ./monitor --controller=<k8s-replication_controller-name>")
		os.Exit(0)
	}

	// create cAdvisor client to monitor Docker instances
	cadvisorClient, err := client.NewClient("http://localhost:9090/")
	handleError("[ERROR] Could not create NewClient", err, true)

	podImage := getPodImage(*controllerName)
	fmt.Println("[INFO] podImage =", podImage)

	prevPodToContMap = getPodToContMap(*controllerName, podImage)
	fmt.Println("[INFO] prevPodToContMap =", prevPodToContMap)

	// run monitoring mechanism for already running containers
	for _, contName := range prevPodToContMap {
		// run each Docker monitoring in a separate goroutine
		go ramBasedScaling(cadvisorClient, *controllerName, contName)
		// cpuBasedScaling(cadvisorClient, contName)
		// netBasedScaling(cadvisorClient, contName)				
	}

	// continuously check Docker instances of this pod (i.e., running this image)
	// and monitor each instance for resource consumption.
	for {
		time.Sleep(READ_FREQ)
		newContainers := make(map[string]string)
		deletedContainers := make(map[string]string)

		currPodToContMap = getPodToContMap(*controllerName, podImage)

		// If number of previously and currently running containers does not match that means
		// pod replicas was resized. We need to find whether it was increased 
		// or decreased, and which pod exactly got created or deleted. If they do match, 
		// we check if both sets have same items.
		prevSize = len(prevPodToContMap)
		currSize = len(currPodToContMap)
		if currSize != prevSize {
			fmt.Printf("[INFO] Pod was resized from [%d] to [%d]\n", prevSize, currSize)
			if (currSize > prevSize) {
				// Pod size was increased. Go over the current list of containers and 
				// find which pod(s) are added.				
				for podName, contName := range currPodToContMap {
					if prevPodToContMap[podName] == "" {
						newContainers[podName] = contName
					}
				}
				fmt.Println("[DEBUG] newContainers =", newContainers)
			} else {
				// Pod size was decreased. Go over the previous list of containers and 
				// find which pod(s) were deleted.
				for podName, contName := range prevPodToContMap {
					if currPodToContMap[podName] == "" {
						deletedContainers[podName] = contName
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

			// fmt.Println("numbers are the same")
		}
		prevPodToContMap = currPodToContMap

		if len(newContainers) != 0 {
			// run monitoring mechanism for all newly created container
			for _, contName := range newContainers {
				go ramBasedScaling(cadvisorClient, *controllerName, contName)
			}
		}

		if len(deletedContainers) != 0 {
			// clean state of the deleted container and fix for all other running containers
			cleanContState(deletedContainers, currPodToContMap)
		}

	}
}

