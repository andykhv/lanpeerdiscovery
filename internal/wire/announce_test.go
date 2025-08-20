package wire

import (
	"net/netip"
	"testing"
)

func TestDecode(t *testing.T) {
	announce := `{
		"id": "123",
		"name": "test123",
		"udpPort": 8080,
		"addr": "1.1.1.1",
		"version": "0.1"
	}`

	expected := &Announce{
		ID:      "123",
		Name:    "test123",
		UDPPort: 8080,
		Addr:    netip.AddrFrom4([4]byte{1, 1, 1, 1}),
		Version: "0.1",
	}

	output, err := Decode([]byte(announce))

	if err != nil {
		t.Errorf("Decode(%v), want %v, error %v", announce, expected, err)
	}

	if output != *expected {
		t.Errorf("Decode(%v), want %v, error", announce, expected)
	}
}
