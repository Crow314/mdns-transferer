package proxy

import (
	"github.com/Crow314/mdns-transferer/pkg/conn"
	"github.com/Crow314/mdns-transferer/pkg/mdns"
	"github.com/miekg/dns"
	"log"
)

// LocalSender sends message received from remote to local with mdns
func LocalSender(c *mdns.Client, rc <-chan *dns.Msg) {
	for {
		msg := <-rc

		err := c.SendMessage(msg)
		if err != nil {
			log.Printf("[ERROR] proxy: Failed to send mdns message: %v", err)
		}
	}
}

// RemoteTransferor transfers message received from local with mdns
func RemoteTransferor(wsSrv *conn.WsServer, rc <-chan *dns.Msg) {
	for {
		msg := <-rc
		wsSrv.SendMessage(msg)
	}
}
