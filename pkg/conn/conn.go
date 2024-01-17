package conn

import (
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

const DefaultPort = 15353

type Connector struct {
	ipv4Conn *net.UDPConn
	ipv6Conn *net.UDPConn

	ipv4Peers map[*net.UDPAddr]struct{}
	ipv6Peers map[*net.UDPAddr]struct{}

	receiveChan chan *dns.Msg

	sendKeepAlive int32
	closed        int32

	sync.RWMutex
}

// NewConnectorSimply creates a new connector while proxies
func NewConnectorSimply(v4 bool, v6 bool) (*Connector, error) {
	var (
		v4Addr *net.UDPAddr
		v6Addr *net.UDPAddr
	)

	if !v4 && !v6 {
		return nil, fmt.Errorf("Must enable at least one of IPv4 and IPv6\n")
	}

	if v4 {
		v4Addr = &net.UDPAddr{IP: net.IPv4zero, Port: DefaultPort}
	}

	if v6 {
		v6Addr = &net.UDPAddr{IP: net.IPv6zero, Port: DefaultPort}
	}

	return NewConnector(v4Addr, v6Addr)
}

// NewConnector creates a new connector while proxies
func NewConnector(v4 *net.UDPAddr, v6 *net.UDPAddr) (*Connector, error) {
	var conn4 *net.UDPConn
	var conn6 *net.UDPConn
	var err error

	if v4 != nil {
		conn4, err = net.ListenUDP("udp4", v4)
		if err != nil {
			log.Printf("[ERROR] conn: Failed to bind to udp4 port: %v", err)
		}
	}

	if v6 != nil {
		conn6, err = net.ListenUDP("udp6", v6)
		if err != nil {
			log.Printf("[ERROR] conn: Failed to bind to udp6 port: %v", err)
		}
	}

	if conn4 == nil && conn6 == nil {
		return nil, fmt.Errorf("Failed to bind to any udp port\n")
	}

	c := &Connector{
		ipv4Conn: conn4,
		ipv6Conn: conn6,

		ipv4Peers: make(map[*net.UDPAddr]struct{}),
		ipv6Peers: make(map[*net.UDPAddr]struct{}),

		receiveChan: make(chan *dns.Msg),
	}
	return c, nil
}

// ReceiveChan returns receiveChan
func (c *Connector) ReceiveChan() <-chan *dns.Msg {
	return c.receiveChan
}

// Close closes all connections and channels
func (c *Connector) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		// something else already closed it
		return nil
	}

	atomic.CompareAndSwapInt32(&c.sendKeepAlive, 1, 0)

	c.RLock()
	log.Printf("[INFO] conn: Closing client %v", *c)
	c.RUnlock()

	c.Lock()
	if c.ipv4Conn != nil {
		_ = c.ipv4Conn.Close()
	}

	if c.ipv6Conn != nil {
		_ = c.ipv6Conn.Close()
	}

	if c.receiveChan != nil {
		close(c.receiveChan)
	}
	c.Unlock()

	return nil
}

// AddPeer adds new peer to which connector will send
func (c *Connector) AddPeer(peer *net.UDPAddr) error {
	if p4 := peer.IP.To4(); len(p4) == net.IPv4len {
		peer.IP = p4
	}

	switch len(peer.IP) {
	case net.IPv4len:
		c.Lock()
		c.ipv4Peers[peer] = struct{}{}
		c.Unlock()
	case net.IPv6len:
		c.Lock()
		c.ipv6Peers[peer] = struct{}{}
		c.Unlock()
	default:
		return fmt.Errorf("Illegal IP address\n")
	}

	log.Printf("[INFO] conn: Add peer: %v", peer)
	c.RLock()
	log.Printf("[DEBUG] conn: ipv4Peers: %v", c.ipv4Peers)
	log.Printf("[DEBUG] conn: ipv6Peers: %v", c.ipv6Peers)
	c.RUnlock()
	return nil
}

// SendPacket sends udp packet to peer proxy
func (c *Connector) SendPacket(data []byte) error {
	c.RLock()
	if c.ipv4Conn != nil {
		for peer := range c.ipv4Peers {
			_, err := c.ipv4Conn.WriteToUDP(data, peer)
			if err != nil {
				return err
			} else {
			}
		}
	}

	if c.ipv6Conn != nil {
		for peer := range c.ipv6Peers {
			_, err := c.ipv6Conn.WriteToUDP(data, peer)
			if err != nil {
				return err
			} else {
			}
		}
	}
	c.RUnlock()
	return nil
}

// SendMDNS sends mdns packet to peer proxy
func (c *Connector) SendMDNS(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		return err
	}

	err = c.SendPacket(buf)
	if err != nil {
		return err
	}

	return nil
}

// StartKeepAliveSender starts keepAliveSender goroutines
func (c *Connector) StartKeepAliveSender() {
	if !atomic.CompareAndSwapInt32(&c.sendKeepAlive, 0, 1) {
		return
	}

	// RFC9000
	// > ... experience shows that sending
	// > packets every 30 seconds is necessary to prevent the majority of
	// > middleboxes from losing state for UDP flows.
	go c.keepAliveSender(30 * time.Second)
}

// keepAliveSender sends Keep-Alive packets
func (c *Connector) keepAliveSender(d time.Duration) {
	for atomic.LoadInt32(&c.sendKeepAlive) == 1 {
		err := c.SendPacket([]byte{})
		if err != nil {
			log.Printf("[ERROR] conn: Failed to send keep-alive packet: %v", err)
		}

		time.Sleep(d)
	}
}

// StartReceiver starts receiver goroutines
func (c *Connector) StartReceiver() {
	c.RLock()
	if c.ipv4Conn != nil {
		go c.receiver(c.ipv4Conn)
	}

	if c.ipv6Conn != nil {
		go c.receiver(c.ipv6Conn)
	}
	c.RUnlock()

	log.Println("[INFO] conn: Started receiver goroutine")
}

// receiver is used to receive until we get a shutdown
func (c *Connector) receiver(l *net.UDPConn) {
	if l == nil {
		return
	}
	buf := make([]byte, 65536)
	for atomic.LoadInt32(&c.closed) == 0 {
		n, err := l.Read(buf)

		if atomic.LoadInt32(&c.closed) == 1 {
			return
		}

		if err != nil {
			log.Printf("[ERROR] conn: Failed to read packet: %v", err)
			continue
		}

		// Receive Keep-Alive Packet
		if n == 0 {
			continue
		}

		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			log.Printf("[ERROR] conn: Failed to unpack packet: %v", err)
			continue
		}

		c.RLock()
		c.receiveChan <- msg
		c.RUnlock()
	}
}
