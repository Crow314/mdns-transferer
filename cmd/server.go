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

	srv := new(conn.Server)
	wsSrv := &conn.WsSrv

	tc := make(chan *dns.Msg)
	mc.StartReceiver(tc)

	go proxy.LocalSender(mc, wsSrv.ReceiveChan())
	go proxy.RemoteTransferor(wsSrv, tc)

	srv.Up()
}
