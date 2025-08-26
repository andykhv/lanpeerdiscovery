package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"time"

	"github.com/andykhv/lanpeerdiscovery/internal/netx"
	"github.com/andykhv/lanpeerdiscovery/internal/wire"
)

const (
	AnnounceInterval = 2 * time.Second
	AnnouncePort     = 39999 //broadcast/listen port
	EchoPort         = 40000 //probe port (UDP echo)
)

var (
	HostName = "HOST_NAME"
	HostId   = "HOST_ID"
)

func main() {
	HostName = os.Getenv("HOST_NAME")
	HostId = os.Getenv("HOST_ID")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	interfaceInfos, err := netx.Eligible()
	if err != nil {
		panic(err)
	}

	for _, ifc := range interfaceInfos {
		conn := mustUDPListen(ifc.IP, AnnouncePort)
		go listenLoop(ctx, conn)
	}

	go announceLoop(ctx, interfaceInfos)

	//go startEchoServer(EchoPort)

	<-ctx.Done()
	fmt.Println("exiting...")
}

func mustUDPListen(ip netip.Addr, port int) *net.UDPConn {
	listenAddr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		panic(err)
	}
	return conn
}

func listenLoop(ctx context.Context, conn *net.UDPConn) {
	buffer := make([]byte, 1024)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _, err := conn.ReadFromUDP((buffer))
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}

		announce, err := wire.Decode(buffer[:n])
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}

		if announce.ID == HostId {
			fmt.Printf("error: announce.Id == HostId\n")
			continue
		}

		fmt.Printf("announce from %s %s:%d (%s)\n", announce.ID, announce.Addr, announce.UDPPort, announce.Name)
	}
}

func announceLoop(ctx context.Context, interfaces []netx.InterfaceInfo) {
	t := time.NewTicker(AnnounceInterval)
	defer t.Stop()
	announce := wire.Announce{
		ID:      HostId,
		Name:    HostName,
		UDPPort: EchoPort,
		Version: "0.1",
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, iface := range interfaces {
				a := announce
				a.Addr = iface.IP
				packet, _ := wire.Encode(a)
				remoteAddress := &net.UDPAddr{IP: iface.Broadcast.AsSlice(), Port: AnnouncePort}
				listenAddress := &net.UDPAddr{IP: iface.IP.AsSlice(), Port: 0}
				conn, err := net.ListenUDP("udp4", listenAddress)

				if err != nil {
					fmt.Printf("err: %v\n", err)
					continue
				}

				_, _ = conn.WriteToUDP(packet, remoteAddress)
				conn.Close()
			}
		}
	}
}
