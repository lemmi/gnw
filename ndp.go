package main

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

type ndp struct {
	*ipv6.PacketConn
}

func newNDP() (ndp, error) {
	c, err := net.ListenPacket("ip6:58", "::")
	if err != nil {
		return ndp{}, err
	}
	p := ipv6.NewPacketConn(c)
	p.SetMulticastHopLimit(255)

	return ndp{p}, nil
}

func ndpPayload(from net.HardwareAddr, to net.IP) ([]byte, error) {
	payload := make([]byte, 4)
	payload = append(payload, to...)
	payload = append(payload, 0x01, 0x01) // Option Source Link-Layer Adress
	payload = append(payload, from...)

	m := icmp.Message{
		Type: ipv6.ICMPTypeNeighborSolicitation,
		Body: &icmp.RawBody{payload},
	}

	return m.Marshal(nil)
}

var IPv6MultiCastBase = net.IP{0xff, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xff, 0x00, 0x00, 0x00}

func ndpMultiCastAddr(from net.Interface, to net.IP) *net.IPAddr {
	return &net.IPAddr{
		IP:   append(IPv6MultiCastBase[:13:13], to[13:]...),
		Zone: from.Name,
	}
}

func ndpMessage(from net.Interface, to net.IP) (ipv6.Message, error) {
	payload, err := ndpPayload(from.HardwareAddr, to)
	if err != nil {
		return ipv6.Message{}, err
	}

	return ipv6.Message{
		Buffers: [][]byte{payload},
		OOB:     nil,
		Addr:    ndpMultiCastAddr(from, to),
		N:       0,
		NN:      0,
		Flags:   0,
	}, nil
}

func (n ndp) solicit(timeout time.Duration, iface net.Interface, targets ...net.IP) ([]ipv6.Message, error) {
	var ms []ipv6.Message

	if len(targets) == 0 {
		return nil, nil
	}

	for _, target := range targets {
		if t4 := target.To4(); t4 != nil {
			// skip v4 addresses
			continue
		}
		m, err := ndpMessage(iface, target)
		if err != nil {
			return nil, err
		}

		fmt.Printf("Sending Neighbor Solicitation request to %s\n", m.Addr)
		ms = append(ms, m)
	}

	if _, err := n.WriteBatch(ms, 0); err != nil {
		return nil, err
	}

	if err := n.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	rms := make([]ipv6.Message, len(targets))
	for i := range rms {
		rms[i].Buffers = [][]byte{make([]byte, iface.MTU)}
	}

	var nr int
loop:
	for nr < len(rms) {
		c, err := n.ReadBatch(rms[nr:], 0)
		if err != nil {
			if e, ok := err.(*net.OpError); ok && e.Timeout() {
				break loop
			}
			return nil, err
		}
		nr += c
		fmt.Println(c)
	}

	return rms[:nr], nil
}
