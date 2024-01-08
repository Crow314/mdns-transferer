package main

import (
	"github.com/Crow314/mdns-transferer/pkg/mdns"
	"github.com/Crow314/mdns-transferer/pkg/proxy"

	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestTransferReceivedPacketToConnector(t *testing.T) {
	var err error

	mc, err := mdns.NewClient(true, false)
	if err != nil {
		t.Fatalf("[ERROR] Failed to generate mdns client\n %v", err)
	}

	uconn4, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatalf("[ERROR] Failed to bind to udp4 port: %v", err)
	}

	mc.StartReceiver()
	txChan := make(chan *dns.Msg)

	defer func() {
		_ = mc.Close()
		close(txChan)
	}()

	f := func(msg *dns.Msg) error {
		txChan <- msg
		return nil
	}
	go proxy.RemoteTransferor(f, mc.ReceiveChan())

	msg := new(dns.Msg)
	msg.SetQuestion("_ipp-tls._tcp.local.", dns.TypePTR)
	msg.RecursionDesired = false

	buf, err := msg.Pack()
	if err != nil {
		t.Errorf("Error has occuered with converting DNS message: %v", err)
	}

	_, err = uconn4.WriteToUDP(buf, &net.UDPAddr{IP: net.ParseIP("224.0.0.251"), Port: 5353})
	if err != nil {
		t.Errorf("Error has occuered with WriteToUDP: %v", err)
	}

	select {
	case rcv := <-txChan:
		assert.Equal(t, *msg, *rcv)
	case <-time.After(3 * time.Second):
		t.Errorf("Server has not received message\n")
	}
}
