package mdns

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestTransferReceivedPacketToConnector(t *testing.T) {
	var err error

	mc, err := NewClient(true, false)
	if err != nil {
		t.Fatalf("[ERROR] Failed to generate mdns client\n %v", err)
	}

	uconn4, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatalf("[ERROR] Failed to bind to udp4 port: %v", err)
	}

	mc.receiveRejectFilter = []net.IP{}

	mc.StartReceiver()

	defer func() {
		_ = mc.Close()
	}()

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
	case rcv := <-mc.ReceiveChan():
		assert.Equal(t, *msg, *rcv)
	case <-time.After(3 * time.Second):
		t.Errorf("Server has not received message\n")
	}
}

func TestRejectPacketSentMyself(t *testing.T) {
	var err error

	mc, err := NewClient(true, false)
	if err != nil {
		t.Fatalf("[ERROR] Failed to generate mdns client\n %v", err)
	}

	uconn4, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatalf("[ERROR] Failed to bind to udp4 port: %v", err)
	}

	mc.StartReceiver()

	defer func() {
		_ = mc.Close()
	}()

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
	case _ = <-mc.ReceiveChan():
		t.Errorf("Server received message sent oneself\n")
	case <-time.After(3 * time.Second):
	}
}
