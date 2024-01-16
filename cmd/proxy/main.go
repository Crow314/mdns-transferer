package main

import (
	"github.com/Crow314/mdns-transferer/pkg/conn"
	"github.com/Crow314/mdns-transferer/pkg/mdns"
	"github.com/Crow314/mdns-transferer/pkg/proxy"
	"sync"

	"log"
)

func main() {
	mc, err := mdns.NewClient(true, false)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate mdns client\n %v", err)
	}

	c, err := conn.NewConnectorSimply(true, false)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate connector\n %v", err)
	}

	go proxy.LocalSender(mc.SendMessage, c.ReceiveChan())
	go proxy.RemoteTransferor(c.SendMDNS, mc.ReceiveChan())

	mc.StartReceiver()
	c.StartReceiver()

	defer func() {
		_ = mc.Close()
		_ = c.Close()
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
