package main

import (
  "fmt"
  "os"
  "bufio"
  "strings"
  "strconv"
)

func prepareData(expName string) {
  natFilename := fmt.Sprintf("%s/%s", expName, "NATMapCap.txt")
  serverFilename := fmt.Sprintf("%s/%s", expName, "latency.txt")
  plotDatafile := fmt.Sprintf("%s/%s", expName, "plotdata.txt")
  var jumpStep uint64 = 40
  var aggrLatency, jumpIndex uint64

  natFile, err := os.Open(natFilename)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  defer natFile.Close()
  natReader := bufio.NewReader(natFile)
  natScanner := bufio.NewScanner(natReader)

  serverFile, err := os.Open(serverFilename)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  serverReader := bufio.NewReader(serverFile)
  serverScanner := bufio.NewScanner(serverReader)
  defer serverFile.Close()
  var split []string

  // open a file to write plotting data
  plotFile, err := os.Create(plotDatafile)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  defer plotFile.Close()

  // load server data into memory since we do a lot of traversal on it
  serverData := make(map[uint64]uint64)
  for serverScanner.Scan() {
    split = strings.Split(serverScanner.Text(), " ")
    sTime, err := strconv.ParseUint(split[1], 10, 64) //replace to split[0] since new docker image puts client time first
    if err != nil {
      fmt.Printf("err :: %v\n", err)
    }

    sLatency, err := strconv.ParseUint(split[2], 10, 64)
    if err != nil {
      fmt.Printf("err :: %v\n", err)
    }
    serverData[sTime] = sLatency
  }
  // fmt.Println(serverData)
  fmt.Println("len =", len(serverData))
  _, err = plotFile.WriteString(fmt.Sprintf("%s %s\n", 
    "natCapacity", "msgLatency"))

  msInNs := uint64(1000000)
  // read nat capacity and corresponding latency value
  for natScanner.Scan() {
    split = strings.Split(natScanner.Text(), " ")
    clientTime, err := strconv.ParseUint(split[0], 10, 64)
    if err != nil {
      fmt.Printf("err :: %v\n", err)
    }

    natCap, err := strconv.ParseUint(split[2], 10, 64)
    if err != nil {
      fmt.Printf("err :: %v\n", err)
    }

    aggrLatency += uint64(serverData[clientTime]/msInNs)
    if jumpIndex % jumpStep == 0 {
      aggrLatency = uint64(aggrLatency / jumpStep)
      fmt.Printf("%d: %d\n", natCap, aggrLatency)
      _, err = plotFile.WriteString(fmt.Sprintf("%d %d\n", 
        natCap, aggrLatency))
      aggrLatency = 0
      jumpIndex = 0
    } 
    jumpIndex++
  }
}

func main() {
  prepareData("coloc")
  prepareData("native")
  prepareData("local")
}
