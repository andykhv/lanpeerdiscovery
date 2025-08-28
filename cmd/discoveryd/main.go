package main

import (
	"context"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"time"

	"github.com/andykhv/lanpeerdiscovery/internal/netx"
	"github.com/andykhv/lanpeerdiscovery/internal/table"
	"github.com/andykhv/lanpeerdiscovery/internal/wire"
)

const (
	AnnounceInterval = 2 * time.Second
	AnnouncePort     = 39999 //broadcast/listen port
	EchoPort         = 40000 //probe port (UDP echo)
)

var (
	HostName   = "HOST_NAME"
	HostId     = "HOST_ID"
	StaleAfter = 5000 * time.Millisecond
	DownAfter  = 10000 * time.Millisecond
	EvictAfter = 20000 * time.Millisecond
	ProbeEvery = 1000 * time.Millisecond
)

func main() {
	HostName = os.Getenv("HOST_NAME")
	HostId = os.Getenv("HOST_ID")
	cfg := table.Config{
		StaleAfter: StaleAfter,
		DownAfter:  DownAfter,
		EvictAfter: EvictAfter,
		ProbeEvery: ProbeEvery,
	}
	t := &table.Table{
		Peers: map[string]*table.Peer{},
	}
	bus := &table.Bus{
		AnnounceCh:      make(chan table.Announce),
		ProbeRequestCh:  make(chan table.ProbeRequest),
		ProbeResponseCh: make(chan table.ProbeResponse),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	interfaceInfos, err := netx.Eligible()
	if err != nil {
		panic(err)
	}

	go t.Loop(ctx, bus, cfg, time.Now)

	conn := mustUDPListen(AnnouncePort)
	go listenLoop(ctx, conn, bus)

	go announceLoop(ctx, interfaceInfos)

	//go startEchoServer(EchoPort)

	<-ctx.Done()
	log.Println("exiting...")
}

func mustUDPListen(port int) *net.UDPConn {
	listenAddr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		panic(err)
	}
	return conn
}

func listenLoop(ctx context.Context, conn *net.UDPConn, bus *table.Bus) {
	buffer := make([]byte, 1024)
	for {
		log.Printf("listening\n")
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
			log.Printf("error: %v\n", err)
			continue
		}

		announce, err := wire.Decode(buffer[:n])
		if err != nil {
			log.Printf("error: %v\n", err)
			continue
		}

		if announce.ID == HostId {
			continue
		}

		log.Printf("announce from %s %s:%d (%s)\n", announce.ID, announce.Addr, announce.UDPPort, announce.Name)
		bus.AnnounceCh <- table.Announce{ID: announce.ID, Address: netip.AddrPortFrom(announce.Addr, uint16(announce.UDPPort))}
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
					log.Printf("err: %v\n", err)
					continue
				}

				log.Printf("announcing %v \n", remoteAddress)
				_, _ = conn.WriteToUDP(packet, remoteAddress)
				conn.Close()
			}
		}
	}
}
