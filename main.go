package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

type InputConf struct {
	Targets                []string                `json:"targets"`
	Duration               int64                   `json:"duration"`
	ConcurrentFetches      int64                   `json:"concurrent_fetches"`
	CrossTrafficComponents []CrossTrafficComponent `json:"cross_traffic_components"`
}

var localAddrStr string
var stopRunning chan int64

func CongestionStart(w http.ResponseWriter, r *http.Request) {
	var conf InputConf
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&conf)
	if err != nil {
		panic(err)
	}
	log.Println("Starting CrossTrafficGenerator")
	ctg := new(CrossTrafficGenerator)
	ctg.NewCrossTrafficGenerator(localAddrStr, conf.Duration,
		conf.ConcurrentFetches, conf.Targets,
		conf.CrossTrafficComponents)
	go ctg.Run()
}

func ServiceStop(w http.ResponseWriter, r *http.Request) {
	stopRunning <- 1
}

func main() {
	localAddrStr = os.Args[1]
	localPort, err := strconv.ParseUint(os.Args[2], 10, 16)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/congestion_start", CongestionStart)
	http.HandleFunc("/service_stop", ServiceStop)
	stopRunning = make(chan int64)
	go http.ListenAndServe(
		fmt.Sprintf("%s:%d", localAddrStr, localPort),
		nil)
	<-stopRunning
}
