package mdns

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	ipv4mdns = "224.0.0.251"
	ipv6mdns = "ff02::fb"
	mdnsPort = 5353
)

var (
	ipv4Addr = &net.UDPAddr{
		IP:   net.ParseIP(ipv4mdns),
		Port: mdnsPort,
	}
	ipv6Addr = &net.UDPAddr{
		IP:   net.ParseIP(ipv6mdns),
		Port: mdnsPort,
	}
)

// QueryParam is used to customize how a Lookup is performed
type QueryParam struct {
	Timeout             time.Duration // Lookup timeout, default 1 second
	WantUnicastResponse bool          // Unicast response desired, as per 5.4 in RFC
	DisableIPv4         bool          // Whether to disable usage of IPv4 for MDNS operations. Does not affect discovered addresses.
	DisableIPv6         bool          // Whether to disable usage of IPv6 for MDNS operations. Does not affect discovered addresses.
}

type Client struct {
	use_ipv4 bool
	use_ipv6 bool

	ipv4UnicastConn *net.UDPConn
	ipv6UnicastConn *net.UDPConn

	ipv4MulticastConn *net.UDPConn
	ipv6MulticastConn *net.UDPConn

	closed   int32
	closedCh chan struct{} // TODO(reddaly): This doesn't appear to be used.
}

// NewClient creates a new mdns Client that can be used to query
// for records
func NewClient(v4 bool, v6 bool) (*Client, error) {
	if !v4 && !v6 {
		return nil, fmt.Errorf("Must enable at least one of IPv4 and IPv6 querying")
	}

	// TODO(reddaly): At least attempt to bind to the port required in the spec.
	// Create a IPv4 listener
	var uconn4 *net.UDPConn
	var uconn6 *net.UDPConn
	var mconn4 *net.UDPConn
	var mconn6 *net.UDPConn
	var err error

	if v4 {
		uconn4, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			log.Printf("[ERROR] mdns: Failed to bind to udp4 port: %v", err)
		}
	}

	if v6 {
		uconn6, err = net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6zero, Port: 0})
		if err != nil {
			log.Printf("[ERROR] mdns: Failed to bind to udp6 port: %v", err)
		}
	}

	if uconn4 == nil && uconn6 == nil {
		return nil, fmt.Errorf("failed to bind to any unicast udp port")
	}

	if v4 {
		mconn4, err = net.ListenMulticastUDP("udp4", nil, ipv4Addr)
		if err != nil {
			log.Printf("[ERROR] mdns: Failed to bind to udp4 port: %v", err)
		}
	}
	if v6 {
		mconn6, err = net.ListenMulticastUDP("udp6", nil, ipv6Addr)
		if err != nil {
			log.Printf("[ERROR] mdns: Failed to bind to udp6 port: %v", err)
		}
	}

	if mconn4 == nil && mconn6 == nil {
		return nil, fmt.Errorf("failed to bind to any multicast udp port")
	}

	c := &Client{
		use_ipv4:          v4,
		use_ipv6:          v6,
		ipv4MulticastConn: mconn4,
		ipv6MulticastConn: mconn6,
		ipv4UnicastConn:   uconn4,
		ipv6UnicastConn:   uconn6,
		closedCh:          make(chan struct{}),
	}
	return c, nil
}

// Close is used to cleanup the Client
func (c *Client) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		// something else already closed it
		return nil
	}

	log.Printf("[INFO] mdns: Closing Client %v", *c)
	close(c.closedCh)

	if c.ipv4UnicastConn != nil {
		c.ipv4UnicastConn.Close()
	}
	if c.ipv6UnicastConn != nil {
		c.ipv6UnicastConn.Close()
	}
	if c.ipv4MulticastConn != nil {
		c.ipv4MulticastConn.Close()
	}
	if c.ipv6MulticastConn != nil {
		c.ipv6MulticastConn.Close()
	}

	return nil
}

// setInterface is used to set the query interface, uses system
// default if not provided
func (c *Client) setInterface(iface *net.Interface) error {
	if c.use_ipv4 {
		p := ipv4.NewPacketConn(c.ipv4UnicastConn)
		if err := p.SetMulticastInterface(iface); err != nil {
			return err
		}
		p = ipv4.NewPacketConn(c.ipv4MulticastConn)
		if err := p.SetMulticastInterface(iface); err != nil {
			return err
		}
	}
	if c.use_ipv6 {
		p2 := ipv6.NewPacketConn(c.ipv6UnicastConn)
		if err := p2.SetMulticastInterface(iface); err != nil {
			return err
		}
		p2 = ipv6.NewPacketConn(c.ipv6MulticastConn)
		if err := p2.SetMulticastInterface(iface); err != nil {
			return err
		}
	}
	return nil
}

// SendMessage is used to multicast a query out
func (c *Client) SendMessage(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		return err
	}
	if c.ipv4UnicastConn != nil {
		_, err = c.ipv4UnicastConn.WriteToUDP(buf, ipv4Addr)
		if err != nil {
			return err
		}
	}
	if c.ipv6UnicastConn != nil {
		_, err = c.ipv6UnicastConn.WriteToUDP(buf, ipv6Addr)
		if err != nil {
			return err
		}
	}
	return nil
}

// StartReceiver starts receiver goroutines
func (c *Client) StartReceiver(msgCh chan<- *dns.Msg) {
	if c.ipv4UnicastConn != nil {
		go c.receiver(c.ipv4UnicastConn, msgCh)
	}

	if c.ipv6UnicastConn != nil {
		go c.receiver(c.ipv6UnicastConn, msgCh)
	}

	if c.ipv4MulticastConn != nil {
		go c.receiver(c.ipv4MulticastConn, msgCh)
	}

	if c.ipv6MulticastConn != nil {
		go c.receiver(c.ipv6MulticastConn, msgCh)
	}
}

// receiver is used to receive until we get a shutdown
func (c *Client) receiver(l *net.UDPConn, msgCh chan<- *dns.Msg) {
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
			log.Printf("[ERROR] mdns: Failed to read packet: %v", err)
			continue
		}
		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			log.Printf("[ERROR] mdns: Failed to unpack packet: %v", err)
			continue
		}
		select {
		case msgCh <- msg:
		case <-c.closedCh:
			return
		}
	}
}