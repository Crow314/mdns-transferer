package main

import (
	"github.com/Crow314/mdns-transferer/pkg/conn"
	"github.com/Crow314/mdns-transferer/pkg/mdns"
	"github.com/Crow314/mdns-transferer/pkg/proxy"

	"github.com/miekg/dns"
)

func main() {
	mc, err := mdns.NewClient(true, true)
	if err != nil {

	}

	c, err := conn.NewConnector(nil, true, true)
	if err != nil {

	}

	tc := make(chan *dns.Msg)
	mc.StartReceiver(tc)

	go proxy.LocalSender(mc.SendMessage, c.ReceiveChan())
	go proxy.RemoteTransferor(c.SendMDNS, tc)

	c.StartReceiver()
}
