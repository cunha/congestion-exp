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
	CrossTrafficComponents []CrossTrafficComponent `json:"cross_traffic_components"`
}

var localAddrStr string
var crossTrafficOn bool
var stopRunning chan int64

// var done chan int64

func CongestionStart(w http.ResponseWriter, r *http.Request) {
	if crossTrafficOn {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	crossTrafficOn = true
	var conf InputConf
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&conf)
	if err != nil {
		panic(err)
	}
	log.Println("Starting capture")
	ctg := new(CrossTrafficGenerator)
	// done = make(chan int64)
	// ctg.NewCrossTrafficGenerator(localAddrStr, conf.Duration, conf.Targets, conf.CrossTrafficComponents, done)
	ctg.NewCrossTrafficGenerator(localAddrStr, conf.Duration, conf.Targets, conf.CrossTrafficComponents)
	go ctg.Run()
	// <-done
	// crossTrafficOn = false
	// log.Println(ctg.CounterStart, " flows started, ", ctg.CounterEnd, " completed, ", ctg.CounterBytes, " bytes downloaded")
}

func CongestionStop(w http.ResponseWriter, r *http.Request) {
	// if crossTrafficOn {
	// 	done <- 1
	// }
	return
}

func ServiceStop(w http.ResponseWriter, r *http.Request) {
	CongestionStop(w, r)
	stopRunning <- 1
}

func main() {
	localAddrStr = os.Args[1]
	localPort, err := strconv.ParseUint(os.Args[2], 10, 16)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/congestion_start", CongestionStart)
	http.HandleFunc("/congestion_stop", CongestionStop)
	http.HandleFunc("/service_stop", ServiceStop)
	crossTrafficOn = false
	stopRunning = make(chan int64)
	go http.ListenAndServe(
		fmt.Sprintf("%s:%d", localAddrStr, localPort),
		nil)
	<-stopRunning
}
