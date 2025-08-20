package wire

import (
	"encoding/json"
	"net/netip"
)

type Announce struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	UDPPort int        `json:"udpPort"`
	Addr    netip.Addr `json:"addr"`
	Version string     `json:"version"`
}

func Encode(a Announce) ([]byte, error) { return json.Marshal(a) }
func Decode(b []byte) (Announce, error) {
	var a Announce
	err := json.Unmarshal(b, &a)
	return a, err
}
