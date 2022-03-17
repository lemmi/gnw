package main

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

type ndp struct {
	c net.PacketConn
	p *ipv6.PacketConn
}

func newNDP() (ndp, error) {
	c, err := net.ListenPacket("ip6:58", "::")
	if err != nil {
		return ndp{}, err
	}
	p := ipv6.NewPacketConn(c)
	if err := p.SetMulticastHopLimit(255); err != nil {
		return ndp{}, err
	}

	return ndp{c: c, p: p}, nil
}

func (n ndp) Close() error {
	var result *multierror.Error
	if err := n.p.Close(); err != nil {
		result = multierror.Append(result, err)
	}
	if err := n.c.Close(); err != nil {
		result = multierror.Append(result, err)
	}
	return result.ErrorOrNil()
}

func ndpPayload(from net.HardwareAddr, to netip.Addr) ([]byte, error) {
	payload := make([]byte, 4)
	payload = append(payload, to.Unmap().AsSlice()...)
	payload = append(payload, 0x01, 0x01) // Option Source Link-Layer Adress
	payload = append(payload, from...)

	m := icmp.Message{
		Type: ipv6.ICMPTypeNeighborSolicitation,
		Body: &icmp.RawBody{Data: payload},
	}

	return m.Marshal(nil)
}

var IPv6MultiCastBase = [16]byte{0xff, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xff, 0x00, 0x00, 0x00}

func ndpMultiCastAddr(from net.Interface, to netip.Addr) *net.IPAddr {
	base := IPv6MultiCastBase
	tobytes := to.As16()
	copy(base[13:], tobytes[13:])
	return &net.IPAddr{
		IP:   base[:],
		Zone: from.Name,
	}
}

func ndpMessage(from net.Interface, to netip.Addr) (ipv6.Message, error) {
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

func (n ndp) solicit(timeout time.Duration, iface net.Interface, targets ...netip.Addr) ([]ipv6.Message, error) {
	var ms []ipv6.Message

	if len(targets) == 0 {
		return nil, nil
	}

	for _, target := range targets {
		if target.Is4() {
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

	if _, err := n.p.WriteBatch(ms, 0); err != nil {
		return nil, err
	}

	if err := n.p.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	rms := make([]ipv6.Message, len(targets))
	for i := range rms {
		rms[i].Buffers = [][]byte{make([]byte, iface.MTU)}
	}

	var nr int
loop:
	for nr < len(rms) {
		c, err := n.p.ReadBatch(rms[nr:], 0)
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
