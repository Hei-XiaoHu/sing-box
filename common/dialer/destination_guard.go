package dialer

import (
	"net"
	"net/netip"

	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
)

// destinationGuard enforces DialerOptions.BlockDestinations (xboard).
//
// It lives in DefaultDialer, below domain resolution, so every destination it
// sees is the final IP actually dialed: a domain that resolves to a blocked
// address (e.g. "localhost") is caught, and DNS rebinding cannot slip a
// different address in after a routing decision. UDP is checked per packet,
// because one UDP session may address many destinations after routing only
// looked at the first one.
type destinationGuard struct {
	prefixes []netip.Prefix
}

func newDestinationGuard(prefixes []netip.Prefix) *destinationGuard {
	if len(prefixes) == 0 {
		return nil
	}
	return &destinationGuard{prefixes: prefixes}
}

func (g *destinationGuard) blocked(addr netip.Addr) bool {
	if !addr.IsValid() {
		return true // not an IP at this layer: fail closed
	}
	addr = addr.Unmap()
	for _, prefix := range g.prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (g *destinationGuard) check(destination M.Socksaddr) error {
	if g == nil || !g.blocked(destination.Addr) {
		return nil
	}
	return E.New("blocked destination: ", destination)
}

func (g *destinationGuard) wrapPacketConn(conn net.PacketConn) net.PacketConn {
	if g == nil {
		return conn
	}
	return &guardPacketConn{PacketConn: conn, guard: g}
}

// guardPacketConn silently drops packets to blocked destinations, like a
// firewall would, and ignores packets coming back from them. Embedding the
// net.PacketConn interface (not the concrete type) hides fast paths such as
// batch writes, so every packet goes through WriteTo/ReadFrom.
type guardPacketConn struct {
	net.PacketConn
	guard *destinationGuard
}

func (c *guardPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	if c.guard.blocked(M.SocksaddrFromNet(addr).Addr) {
		return len(p), nil
	}
	return c.PacketConn.WriteTo(p, addr)
}

func (c *guardPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		n, addr, err := c.PacketConn.ReadFrom(p)
		if err != nil || !c.guard.blocked(M.SocksaddrFromNet(addr).Addr) {
			return n, addr, err
		}
	}
}
