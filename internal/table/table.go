package table

import (
	"context"
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

type Announce struct {
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

type ListPeersRequest struct {
}

type ListPeersResponse struct {
	When  time.Time
	Peers []Peer
}

type Bus struct {
	AnnounceCh          chan Announce
	ProbeRequestCh      chan ProbeRequest
	ProbeResponseCh     chan ProbeResponse
	ListPeersRequestCh  chan ListPeersRequest
	ListPeersResponseCh chan ListPeersResponse
}

type Table struct {
	Peers map[string]*Peer
	// Seen  map[string]map[[12]byte]time.Time // ID -> nonce -> time
}

type Config struct {
	StaleAfter time.Duration
	DownAfter  time.Duration
	EvictAfter time.Duration
	ProbeEvery time.Duration
}

func (t *Table) Loop(ctx context.Context, bus *Bus, cfg Config, now func() time.Time) {
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
		case <-bus.ListPeersRequestCh:
			peers := make([]Peer, 0, len(t.Peers))
			for _, p := range t.Peers {
				peers = append(peers, *p)
			}

			response := ListPeersResponse{
				When:  time.Now(),
				Peers: peers,
			}
			bus.ListPeersResponseCh <- response
		case <-ctx.Done():
			return
		}
	}
}
