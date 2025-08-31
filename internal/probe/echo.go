package probe

import (
	"context"
	"net"
	"net/netip"
	"time"
)

func StartEchoServer(ctx context.Context, port int) error {
	laddr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return err
	}

	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.SetReadDeadline(time.Now())
		conn.Close()
	}()

	buf := make([]byte, 2048)
	for {
		n, raddr, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if _, err := conn.WriteToUDPAddrPort(buf[:n], raddr); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
	}
}

func Probe(addr netip.AddrPort) (time.Duration, bool) {
	c, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: addr.Addr().AsSlice(), Port: int(addr.Port())})
	if err != nil {
		return 0, false
	}

	defer c.Close()

	payload := []byte{0xCA, 0xFE, 0xBA, 0xBE}
	start := time.Now()
	c.SetDeadline(start.Add(1 * time.Second))

	if _, err := c.Write(payload); err != nil {
		return 0, false
	}

	var buffer [16]byte
	if _, _, err := c.ReadFromUDP(buffer[:]); err != nil {
		return 0, false
	}

	return time.Since(start), true
}
