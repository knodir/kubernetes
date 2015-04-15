// This program puts stress to CPU by constantly running loop without any operaton.
// Specify frequency of the loop as a paramater (in Nanoseconds)
// Usage: go run helloworld.go -freq=5 or ./helloworld-freq=5

package main

import (
	"fmt"
	"flag"
	"time"
)

func main() {


	loopFreq := flag.Int("freq", 1000000000, "operation frequency in nanoseconds.")

	flag.Parse()

	fmt.Printf("Running loop with %d nanoseconds frequency. \nRun with -freq=nanoValue to specify the frequency. \nPress Ctrl+C to terminate.\n", *loopFreq)

	sleepDur := time.Nanosecond * time.Duration(*loopFreq)

	for {
		time.Sleep(sleepDur)
	}
}
