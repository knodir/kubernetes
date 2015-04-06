// This program runs pod local etcd and accesses variables from it.

package main

import (
	"fmt"
	"os"

	"github.com/coreos/go-etcd/etcd"
)

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

	k8sMaster := []string{"http://198.162.52.217:4001"}
	client := etcd.NewClient(k8sMaster)

	ret, err := client.Set("/foo", "bar", 0)
	if (err == nil) {
		fmt.Println("Successfully set the entry")
	}
	
	ret, err = client.Get("/foo", false, false)
	fmt.Println("value =", ret.Node.Value)

	handleError("", err, true)

}