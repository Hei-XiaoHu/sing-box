package dialer

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"

	"github.com/stretchr/testify/require"
)

func newGuardTestDialer(t *testing.T, blocked ...string) *DefaultDialer {
	t.Helper()
	var options option.DialerOptions
	for _, prefix := range blocked {
		options.BlockDestinations = append(options.BlockDestinations, netip.MustParsePrefix(prefix))
	}
	d, err := NewDefault(context.Background(), options)
	require.NoError(t, err)
	return d
}

func udpEcho(t *testing.T) *net.UDPAddr {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { pc.Close() })
	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = pc.WriteTo(buf[:n], addr)
		}
	}()
	return pc.LocalAddr().(*net.UDPAddr)
}

func udpRoundTrip(t *testing.T, d *DefaultDialer, target *net.UDPAddr) bool {
	t.Helper()
	pc, err := d.ListenPacket(context.Background(), M.SocksaddrFromNet(target))
	require.NoError(t, err)
	defer pc.Close()
	_, err = pc.WriteTo([]byte("ping"), target)
	require.NoError(t, err, "blocked packets are dropped silently")
	_ = pc.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 16)
	_, _, err = pc.ReadFrom(buf)
	return err == nil
}

func TestDestinationGuard(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	tcpPort := uint16(listener.Addr().(*net.TCPAddr).Port)
	echo := udpEcho(t)

	t.Run("blocked", func(t *testing.T) {
		d := newGuardTestDialer(t, "127.0.0.0/8", "::/127")
		for _, addr := range []string{"127.0.0.1", "::ffff:127.0.0.1"} {
			_, err := d.DialContext(context.Background(), "tcp", M.ParseSocksaddrHostPort(addr, tcpPort))
			require.ErrorContains(t, err, "blocked destination", addr)
			_, err = d.DialContext(context.Background(), "udp", M.ParseSocksaddrHostPort(addr, uint16(echo.Port)))
			require.ErrorContains(t, err, "blocked destination", addr)
		}
		require.False(t, udpRoundTrip(t, d, echo), "UDP packet to blocked destination must be dropped")
	})

	t.Run("other prefixes unaffected", func(t *testing.T) {
		d := newGuardTestDialer(t, "10.0.0.0/8")
		conn, err := d.DialContext(context.Background(), "tcp", M.ParseSocksaddrHostPort("127.0.0.1", tcpPort))
		require.NoError(t, err)
		conn.Close()
		require.True(t, udpRoundTrip(t, d, echo))
	})

	t.Run("disabled by default", func(t *testing.T) {
		d := newGuardTestDialer(t)
		require.Nil(t, d.guard)
		require.True(t, udpRoundTrip(t, d, echo))
	})
}
