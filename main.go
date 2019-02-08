package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

var localAddrStr string
var stopRunning chan int64

func CongestionStart(w http.ResponseWriter, r *http.Request) {
	var tg TrafficGenerator
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&tg)
	if err != nil {
		panic(err)
	}
	log.Println("Starting TrafficGenerator")
	tg.Initialize(localAddrStr)
	go tg.Run()
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
	rand.Seed(time.Now().UTC().UnixNano())
	http.HandleFunc("/congestion_start", CongestionStart)
	http.HandleFunc("/service_stop", ServiceStop)
	stopRunning = make(chan int64)
	go http.ListenAndServe(
		fmt.Sprintf("%s:%d", localAddrStr, localPort),
		nil)
	<-stopRunning
}
