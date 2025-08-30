package probe

import (
	"net"
	"net/netip"
	"time"
)

func StartEchoServer(port int) error {
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
	conn, err := net.ListenUDP("udp4", addr)

	if err != nil {
		return err
	}

	go func() {
		buf := make([]byte, 2048)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			conn.WriteToUDP(buf[:n], addr)
		}
	}()

	return nil
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
