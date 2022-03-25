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
	d.RequestList = make(chan *dht.FindNodeReq, 100)
	d.DataList = make(chan map[string]interface{}, 100)
	d.ReponseList = make(chan *dht.Response, 100)
	d.Id = tool.RandString(20)
	d.Start()
	store.GetMeta()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-s
	fmt.Println("over")
}
