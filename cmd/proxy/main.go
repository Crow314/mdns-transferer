package main

import (
	"github.com/Crow314/mdns-transferer/pkg/conn"
	"github.com/Crow314/mdns-transferer/pkg/mdns"
	"github.com/Crow314/mdns-transferer/pkg/proxy"

	"log"
)

func main() {
	mc, err := mdns.NewClient(true, false)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate mdns client\n %v", err)
	}

	c, err := conn.NewConnector(nil, true, false)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate connector\n %v", err)
	}

	go proxy.LocalSender(mc.SendMessage, c.ReceiveChan())
	go proxy.RemoteTransferor(c.SendMDNS, mc.ReceiveChan())

	c.StartReceiver()
}
