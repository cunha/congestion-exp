package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type InputConf struct {
	Targets                []string                `json:"targets"`
	Duration               int64                   `json:"duration"`
	CrossTrafficComponents []CrossTrafficComponent `json:"cross_traffic_components"`
}

var crossTrafficOn bool
var done chan int64
var stopRunning chan int64

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
		log.Fatal("error parsing json: ", err.Error())
	}
	log.Println("Starting capture")
	ctg := new(CrossTrafficGenerator)
	done = make(chan int64)
	ctg.NewCrossTrafficGenerator(conf.Duration, conf.Targets, conf.CrossTrafficComponents, done)
	go ctg.Run()
	<-done
	crossTrafficOn = false
	log.Println(ctg.CounterStart, " flows started, ", ctg.CounterEnd, " completed, ", ctg.CounterBytes, " bytes downloaded")
}

func CongestionStop(w http.ResponseWriter, r *http.Request) {
	if crossTrafficOn {
		done <- 1
	}
}

func ServiceStop(w http.ResponseWriter, r *http.Request) {
	CongestionStop(w, r)
	stopRunning <- 1
}

func main() {
	http.HandleFunc("/congestion_start", CongestionStart)
	http.HandleFunc("/congestion_stop", CongestionStop)
	http.HandleFunc("/service_stop", ServiceStop)
	crossTrafficOn = false
	stopRunning = make(chan int64)
	go http.ListenAndServe(":9001", nil)
	<-stopRunning
}
