package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"time"
)

type CrossTrafficComponent struct {
	Name      string  `json:"name"`
	Rate      float64 `json:"rate"`
	NextEvent int64
}

type CrossTrafficComponentArr []CrossTrafficComponent

type CrossTrafficGenerator struct {
	LocalAddr              string
	Duration               int64
	Targets                []string
	CrossTrafficComponents CrossTrafficComponentArr
	Start                  int64
	End                    int64
	CounterStart           int64
	CounterEnd             int64
	CounterBytes           int64
	// Done                   chan int64
}

func (ctg *CrossTrafficGenerator) NewCrossTrafficGenerator(localAddr string, duration int64, targets []string, ctc CrossTrafficComponentArr) {
	ctg.LocalAddr = localAddr
	ctg.Targets = targets
	ctg.Duration = duration
	ctg.CrossTrafficComponents = ctc
	// ctg.Done = done
}

func (ctg *CrossTrafficGenerator) Fetch(curr CrossTrafficComponent, fetchChan chan int64, eventCounter int64) {
	//fmt.Println(ctg.Target, curr.Name, curr.NextEvent, eventCounter)
	localAddr, err := net.ResolveIPAddr("ip", ctg.LocalAddr)
	if err != nil {
		panic(err)
	}
	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	webclient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				LocalAddr: &localTCPAddr,
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: false,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	tIndex := rand.Int31n(int32(len(ctg.Targets)))
	target := ctg.Targets[tIndex] + "/" + curr.Name
	ctg.CounterStart++
	log.Printf("CrossTrafficGenerator.Fetch %s\n", target)
	resp, err := webclient.Get(target)
	if err != nil {
		//fmt.Println("error", curr.Name, eventCounter)
		fetchChan <- eventCounter
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	ctg.CounterEnd++
	ctg.CounterBytes += int64(len(body))
	fetchChan <- eventCounter
}

func (ctg *CrossTrafficGenerator) GenerateNextEvent(rate float64) int64 {
	return int64((rand.ExpFloat64() / rate) * 1e9)
}

func (ctg *CrossTrafficGenerator) InitializeEventArr() {
	ctg.Start = time.Now().UTC().UnixNano()
	ctg.End = ctg.Start + ctg.Duration*1e9
	rand.Seed(ctg.Start)
	for i := range ctg.CrossTrafficComponents {
		if ctg.CrossTrafficComponents[i].NextEvent == 0 {
			ctg.CrossTrafficComponents[i].NextEvent = ctg.Start + ctg.GenerateNextEvent(ctg.CrossTrafficComponents[i].Rate)
		}
	}
	sort.Sort(ctg.CrossTrafficComponents)
}

func (ctg *CrossTrafficGenerator) Run() {
	fetchChan := make(chan int64, 1e6)
	var eventCounter int64 = 0
	//var eventDone int64

	log.Println("CrossTrafficGenerator.Run")

	ctg.InitializeEventArr()
	now := time.Now().UTC().UnixNano()

	for ctg.End > now {
		curr := ctg.CrossTrafficComponents[0]
		time.Sleep(time.Duration(curr.NextEvent-now) * time.Nanosecond)
		go ctg.Fetch(curr, fetchChan, eventCounter)
		eventCounter++
		ctg.CrossTrafficComponents[0].NextEvent = ctg.CrossTrafficComponents[0].NextEvent + ctg.GenerateNextEvent(ctg.CrossTrafficComponents[0].Rate)
		if ctg.CrossTrafficComponents[0].NextEvent > ctg.CrossTrafficComponents[1].NextEvent { //optimization assuming order of magnitude difference in arrival rates
			sort.Sort(ctg.CrossTrafficComponents)
		}
		now = time.Now().UTC().UnixNano()
	}
	//for eventDone < eventCounter {
	//	<-fetchChan
	//	eventDone++
	//}
	//ctg.Done <- 1
}
