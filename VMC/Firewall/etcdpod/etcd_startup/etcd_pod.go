// This program runs pod local etcd and accesses variables from it.

package etcd_startup

import (
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"log"
	"os"
	"os/exec"
	"time"
)

func Etcd_start(ip string, service string, k8sMaster []string, hostname string) {
	var err error
	var ret *etcd.Response
	var buf []byte
	var discovery_key string
	var curl_path string

	/*Set up curl*/
	curl_path, err = exec.LookPath("curl")
	checkErr(err)

	/*Set up etcd client*/
	fmt.Println("ip:", ip, "k8sMaster:", k8sMaster, "hostname:", hostname, "service:", service)
	client := etcd.NewClient(k8sMaster)

	/*Make Reads all quorum*/
	client.SetConsistency(etcd.STRONG_CONSISTENCY)

	/*Try and get the discovery key for this service*/
	ret, err = client.Get("/registry/etcd/"+service+"/discovery", false, false)
	if err != nil {
		if getErrorCode(err) == 100 {
			/*key not found, so fetch a new discovery key*/
			curl_cmd := append([]string{"https://discovery.etcd.io/new?size=3"})
			buf, err = exec.Command(curl_path, curl_cmd...).Output()
			checkErr(err)

			/*create the discovery key, only succeeds if it does not yet exist*/
			ret, err = client.Create("/registry/etcd/"+service+"/discovery", string(buf), 0)
			if err != nil {
				if getErrorCode(err) == 105 {
					/*key already exists, so grab it now*/
					ret, err = client.Get("/registry/etcd/"+service+"/discovery", false, false)
				}
			}
		}
	}
	checkErr(err)
	discovery_key = ret.Node.Value
	fmt.Printf("Discovery: %s\n", discovery_key)

	/*start local etcd*/
	etcdPath, err := exec.LookPath("etcd")
	checkErr(err)
	etcdCmd := append([]string{"-discovery=" + discovery_key}, "-name="+hostname, "-data-dir=/home/etcd", "-addr="+ip+":4001", "-peer-addr="+ip+":7001")
	fmt.Println(etcdCmd)
	cmd := exec.Command(etcdPath, etcdCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	checkErr(err)

	fmt.Printf("etcd started...pid is %d\n", cmd.Process.Pid)

	/*wait till etcd starts*/
	//etcd_local := []string{"http://" + ip + ":4001"}
	//local_client := etcd.NewClient(etcd_local)

	<-time.After(time.Second * 5)
	/*
		for true {
			_, err = local_client.Create("/nonce", "10", 0)
			if err == nil {
				break
			} else {
				//fmt.Println(err)
			}
		}
		_, err = local_client.Delete("/nonce", false)
		checkErr(err)*/
}

func getErrorCode(err error) int {
	err_etcd, ok := err.(*etcd.EtcdError)
	if ok {
		return err_etcd.ErrorCode
	} else {
		return -1
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
