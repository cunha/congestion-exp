package main

import (
	"container/heap"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

type TrafficComponent struct {
	Name      string  `json:"name"`
	Rate      float64 `json:"rate"`
	NextEvent int64
}

func (tc *TrafficComponent) InitNextEvent(tstamp int64) {
	if tc.NextEvent != 0 {
		panic("in InitNextEvent() but NextEvent != 0")
	}
	tc.NextEvent = tstamp
	tc.UpdateNextEvent()
}
func (tc *TrafficComponent) UpdateNextEvent() {
	tc.NextEvent += int64((rand.ExpFloat64() / tc.Rate) * 1e9)
}

type TCHeap []*TrafficComponent

func (tch TCHeap) Len() int            { return len(tch) }
func (tch TCHeap) Less(i, j int) bool  { return tch[i].NextEvent < tch[j].NextEvent }
func (tch TCHeap) Swap(i, j int)       { tch[i], tch[j] = tch[j], tch[i] }
func (tch *TCHeap) Push(x interface{}) { *tch = append(*tch, x.(*TrafficComponent)) }
func (tch TCHeap) Pop() interface{}    { panic("no pop for efficiency") }

type TrafficGenerator struct {
	Targets              []string            `json:"targets"`
	Duration             int64               `json:"duration"`
	MaxConcurrentFetches int64               `json:"concurrent_fetches"`
	TrafficComponents    []*TrafficComponent `json:"cross_traffic_components"`
	LocalAddr            string
	OngoingFetches       int64
	Start                int64
	End                  int64
	Heap                 TCHeap
}

func (tg *TrafficGenerator) Initialize(localAddr string) {
	tg.LocalAddr = localAddr
	atomic.StoreInt64(&tg.OngoingFetches, 0)
	tg.OngoingFetches = 0
	tg.Start = time.Now().UTC().UnixNano()
	tg.End = tg.Start + tg.Duration*1e9
	for _, tc := range tg.TrafficComponents {
		tc.InitNextEvent(tg.Start)
		heap.Push(&tg.Heap, tc)
	}
	if len(tg.Heap) != len(tg.TrafficComponents) {
		panic("wrong number of entries in tg.Heap")
	}
}

func (tg *TrafficGenerator) Fetch(curr *TrafficComponent) {
	defer atomic.AddInt64(&tg.OngoingFetches, -1)
	localAddr, err := net.ResolveIPAddr("ip", tg.LocalAddr)
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
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
				DualStack: false,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	tIndex := rand.Int31n(int32(len(tg.Targets)))
	target := tg.Targets[tIndex] + "/" + curr.Name
	log.Printf("%d TrafficGenerator.Fetch %s\n",
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
	fmt.Printf("%d downloaded %d bytes in %s (%d B/s) with %d concurrent\n",
		time.Now().UTC().UnixNano(),
		int(len(body)),
		duration,
		int(float64(len(body))/duration.Seconds()),
		atomic.LoadInt64(&tg.OngoingFetches))
}

func (tg *TrafficGenerator) Run() {
	log.Println("TrafficGenerator.Run")
	now := time.Now().UTC().UnixNano()
	for now < tg.End {
		curr := tg.Heap[0]
		time.Sleep(time.Duration(curr.NextEvent-now) * time.Nanosecond)
		atomic.AddInt64(&tg.OngoingFetches, 1)
		if atomic.LoadInt64(&tg.OngoingFetches) < tg.MaxConcurrentFetches {
			go tg.Fetch(curr)
		} else {
			fmt.Println("Skipping fetch, to many ongoing fetches")
			atomic.AddInt64(&tg.OngoingFetches, -1)
		}
		curr.UpdateNextEvent()
		heap.Fix(&tg.Heap, 0)
		now = time.Now().UTC().UnixNano()
	}
}
