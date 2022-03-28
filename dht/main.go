package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ghr-dht/dht"
	"github.com/ghr-dht/store"
	"github.com/ghr-dht/tool"
)

func main() {
	d := new(dht.DHT)
	d.Host = "0.0.0.0:12121"
	d.RequestList = make(chan *dht.FindNodeReq, 10000)
	d.DataList = make(chan map[string]interface{}, 10000)
	d.ReponseList = make(chan *dht.Response, 10000)
	d.Id = tool.RandString(20)
	d.Start()
	for i := 0; i < 8; i++ {
		store.GetMeta()
	}
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-s
	fmt.Println("over")
}
