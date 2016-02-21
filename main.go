package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type InputConf struct {
	Target                 string                  `json:"target"`
	Duration               int64                   `json:"duration"`
	CrossTrafficComponents []CrossTrafficComponent `json:"cross_traffic_components"`
}

func main() {
	var conf InputConf

	confFile, err := ioutil.ReadFile("input.conf")
	if err != nil {
		log.Fatal("opening conf file: ", err.Error())
	}
	err = json.Unmarshal(confFile, &conf)
	ctg := new(CrossTrafficGenerator)
	done := make(chan int64)
	ctg.NewCrossTrafficGenerator(conf.Duration, conf.Target, conf.CrossTrafficComponents, done)
	go ctg.Run()
	<-done
}
