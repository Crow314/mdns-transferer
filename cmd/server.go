package main

import (
	"github.com/Crow314/mdns-transferer/pkg/conn"
	"github.com/Crow314/mdns-transferer/pkg/mdns"
	"github.com/Crow314/mdns-transferer/pkg/proxy"

	"log"

	"github.com/miekg/dns"
)

func main() {
	mc, err := mdns.NewClient(true, true)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate mdns client\n %v", err)
	}

	c, err := conn.NewConnector(nil, true, true)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate connector\n %v", err)
	}

	tc := make(chan *dns.Msg)
	mc.StartReceiver(tc)

	go proxy.LocalSender(mc.SendMessage, c.ReceiveChan())
	go proxy.RemoteTransferor(c.SendMDNS, tc)

	c.StartReceiver()
}
