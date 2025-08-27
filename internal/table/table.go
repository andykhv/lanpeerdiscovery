package table

import (
	"net/netip"
	"time"
)

type Status int

const (
	Unknown Status = iota
	Healthy
	Suspect
	Down
)

type Peer struct {
	ID        string
	Address   netip.AddrPort
	LastSeen  time.Time
	LastProbe time.Time
	RTTms     float64
	Status    Status
}

type Ann struct {
	ID      string
	Address netip.AddrPort
}

type ProbeRequest struct {
	ID      string
	Address netip.AddrPort
}

type ProbeResponse struct {
	ID   string
	OK   bool
	RTT  time.Duration
	When time.Time
}

type Bus struct {
	AnnounceCh      chan Ann
	ProbeRequestCh  chan ProbeRequest
	ProbeResponseCh chan ProbeResponse
}

type Table struct {
	Peers map[string]*Peer
}

type Config struct {
	StaleAfter time.Duration
	DownAfter  time.Duration
	EvictAfter time.Duration
	ProbeEvery time.Duration
}

func (t *Table) Loop(bus *Bus, cfg Config, now func() time.Time) {
	tickProbe := time.NewTicker(cfg.ProbeEvery)
	tickMaintenance := time.NewTicker((time.Second))
	defer tickProbe.Stop()
	defer tickMaintenance.Stop()

	for {
		select {
		case a := <-bus.AnnounceCh:
			peer := t.Peers[a.ID]
			if peer == nil {
				peer = &Peer{ID: a.ID}
				t.Peers[a.ID] = peer
			}
			peer.Address = a.Address
			peer.LastSeen = now()
		case r := <-bus.ProbeResponseCh:
			if peer := t.Peers[r.ID]; peer != nil {
				peer.LastProbe = r.When
				if r.OK {
					peer.Status = Healthy
					peer.RTTms = 0.8*peer.RTTms + 0.2*float64(r.RTT.Milliseconds()) //EWMA
				}
			}
		case <-tickProbe.C:
			for _, peer := range t.Peers {
				if now().Sub(peer.LastSeen) <= cfg.DownAfter {
					bus.ProbeRequestCh <- ProbeRequest{ID: peer.ID, Address: peer.Address}
				}
			}
		case <-tickMaintenance.C:
			for id, peer := range t.Peers {
				duration := now().Sub(peer.LastSeen)
				switch {
				case duration > cfg.EvictAfter:
					delete(t.Peers, id)
				case duration > cfg.DownAfter:
					peer.Status = Down
				case duration > cfg.StaleAfter && peer.Status == Healthy:
					peer.Status = Suspect
				}
			}
		}
	}
}
