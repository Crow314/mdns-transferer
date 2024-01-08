package conn

import (
	"log"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestCommunicateWhileProxies(t *testing.T) {
	var err error

	clConn, err := NewConnectorSimply(true, false)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate connector\n %v", err)
	}

	srvConn, err := NewConnector(
		&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 25353}, nil)
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate connector\n %v", err)
	}

	srvConn.StartReceiver()

	defer func() {
		_ = clConn.Close()
		_ = srvConn.Close()
	}()

	err = clConn.AddPeer(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 25353})
	if err != nil {
		log.Printf("[ERROR] Invalid peer address\n %v", err)
	}

	msg := new(dns.Msg)
	msg.SetQuestion("_ipp-tls._tcp.local.", dns.TypePTR)
	msg.RecursionDesired = false

	err = clConn.SendMDNS(msg)
	if err != nil {
		t.Errorf("Error has occuered with SendMDNS: %v", err)
	}

	select {
	case rcv := <-srvConn.ReceiveChan():
		assert.Equal(t, *msg, *rcv)
	case <-time.After(3 * time.Second):
		t.Errorf("Server has not received message\n")
	}
}
