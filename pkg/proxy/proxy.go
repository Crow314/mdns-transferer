package proxy

import (
	"log"

	"github.com/miekg/dns"
)

// LocalSender sends message received from remote to local with mdns
func LocalSender(f func(msg *dns.Msg) error, rc <-chan *dns.Msg) {
	for {
		msg := <-rc

		err := f(msg)
		if err != nil {
			log.Printf("[ERROR] proxy: Failed to send mdns message: %v", err)
		}
	}
}

// RemoteTransferor transfers message received from local with mdns
func RemoteTransferor(f func(msg *dns.Msg) error, rc <-chan *dns.Msg) {
	for {
		msg := <-rc
		err := f(msg)
		if err != nil {
			log.Printf("[ERROR] proxy: Failed to transefer mdns message: %v", err)
		}
	}
}
