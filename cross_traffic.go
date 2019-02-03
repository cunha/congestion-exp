package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"sync/atomic"
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
	MaxConcurrentFetches   int64
	OngoingFetches         int64
	Targets                []string
	CrossTrafficComponents CrossTrafficComponentArr
	Start                  int64
	End                    int64
	CounterStart           int64
	CounterEnd             int64
	CounterBytes           int64
}

func (ctg *CrossTrafficGenerator) NewCrossTrafficGenerator(localAddr string, duration int64, maxConcurrentFetches int64, targets []string, ctc CrossTrafficComponentArr) {
	ctg.LocalAddr = localAddr
	ctg.Targets = targets
	ctg.Duration = duration
	ctg.MaxConcurrentFetches = maxConcurrentFetches
	ctg.OngoingFetches = 0
	ctg.CrossTrafficComponents = ctc
}

func (ctg *CrossTrafficGenerator) Fetch(curr CrossTrafficComponent) {
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
	log.Printf("%d CrossTrafficGenerator.Fetch %s\n",
		time.Now().UTC().UnixNano(), target)
	start := time.Now()
	resp, err := webclient.Get(target)
	if err != nil {
		fmt.Println("error", curr.Name)
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	duration := time.Since(start)
	fmt.Printf("downloaded %d bytes in %s (%d B/s) with %d concurrent\n",
		int(len(body)),
		duration,
		int(float64(len(body))/duration.Seconds()),
		atomic.LoadInt64(&ctg.OngoingFetches))
	atomic.AddInt64(&ctg.OngoingFetches, -1)
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
	log.Println("CrossTrafficGenerator.Run")

	atomic.StoreInt64(&ctg.OngoingFetches, 0)
	ctg.InitializeEventArr()
	now := time.Now().UTC().UnixNano()

	for ctg.End > now {
		curr := ctg.CrossTrafficComponents[0]
		time.Sleep(time.Duration(curr.NextEvent-now) * time.Nanosecond)
		atomic.AddInt64(&ctg.OngoingFetches, 1)
		if atomic.LoadInt64(&ctg.OngoingFetches) < ctg.MaxConcurrentFetches {
			go ctg.Fetch(curr)
		} else {
			fmt.Println("Skipping fetch, to many ongoing fetches")
			atomic.AddInt64(&ctg.OngoingFetches, -1)
		}
		fmt.Printf("%s evtime %d ", ctg.CrossTrafficComponents[0].Name, ctg.CrossTrafficComponents[0].NextEvent)
		ctg.CrossTrafficComponents[0].NextEvent = ctg.CrossTrafficComponents[0].NextEvent + ctg.GenerateNextEvent(ctg.CrossTrafficComponents[0].Rate)
		fmt.Printf("nextev %d\n", ctg.CrossTrafficComponents[0].NextEvent)
		// if ctg.CrossTrafficComponents[0].NextEvent > ctg.CrossTrafficComponents[1].NextEvent { //optimization assuming order of magnitude difference in arrival rates
		sort.Sort(ctg.CrossTrafficComponents)
		// }
		now = time.Now().UTC().UnixNano()
	}
}
