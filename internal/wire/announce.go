package wire

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/netip"
)

type Announce struct {
	ID        string     `json:"id"` //hex(pub key)
	Name      string     `json:"name"`
	UDPPort   int        `json:"udpPort"`
	Addr      netip.Addr `json:"addr"`
	Version   string     `json:"version"`
	EpochMS   int64      `json:"epochMs"`
	Nonce     [12]byte   `json:"nonce"`
	PublicKey []byte     `json:"publicKey"`
	Signature []byte     `json:"signature"` //ed25519
}

func Encode(a Announce) ([]byte, error) { return json.Marshal(a) }
func Decode(b []byte) (Announce, error) {
	var a Announce
	err := json.Unmarshal(b, &a)
	return a, err
}

/* returns deterministic byte slice to sign and verify the announce.
 * independent of JSON order
 */
func (a *Announce) SignBytes() []byte {
	// Binary, fixed order: "ANN1" + ID + Addr + UDPPort + Version + EpochMS + Nonce
	// ID as bytes: if it's hex(pubkey), decode; else sign textual ID.
	var idbytes []byte
	if pubKey, err := hex.DecodeString(a.ID); err == nil && len(pubKey) == ed25519.PublicKeySize {
		idbytes = pubKey
	} else {
		idbytes = []byte(a.ID)
	}
	addr := a.Addr.AsSlice() // 4 or 16 bytes; we'll prefix a family byte
	bytes := make([]byte, 0, 4+1+len(idbytes)+1+len(addr)+8+1+len(a.Version)+8+12)
	bytes = append(bytes, 'a', 'n', 'n', '2') // version tag
	bytes = append(bytes, byte(len(idbytes))) // id len
	bytes = append(bytes, idbytes...)         // id
	if a.Addr.Is4() {
		bytes = append(bytes, 4)
	} else {
		bytes = append(bytes, 6)
	}
	bytes = append(bytes, addr...) // addr

	var u64 [8]byte
	binary.BigEndian.PutUint64(u64[:], uint64(a.UDPPort)) // udp port
	bytes = append(bytes, u64[:]...)
	bytes = append(bytes, byte(len(a.Version)))
	bytes = append(bytes, a.Version...)
	binary.BigEndian.PutUint64(u64[:], uint64(a.EpochMS)) // epoch ms
	bytes = append(bytes, u64[:]...)
	bytes = append(bytes, a.Nonce[:]...) // nonce
	return bytes
}

func (a *Announce) Sign(privateKey ed25519.PrivateKey) {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if len(a.PublicKey) == 0 {
		a.PublicKey = append([]byte(nil), publicKey...)
	}
	if a.ID == "" {
		a.ID = hex.EncodeToString(publicKey)
	}
	a.Signature = ed25519.Sign(privateKey, a.SignBytes())
}

func (a *Announce) Verify() bool {
	if len(a.PublicKey) != ed25519.PublicKeySize || len(a.Signature) != ed25519.SignatureSize {
		return false
	}
	if a.ID != hex.EncodeToString(a.PublicKey) {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(a.PublicKey), a.SignBytes(), a.Signature)
}
