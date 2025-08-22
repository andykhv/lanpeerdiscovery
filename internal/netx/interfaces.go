package netx

import (
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
)

type InterfaceInfo struct {
	Interface net.Interface
	IP        netip.Addr
	Prefix    netip.Prefix
	Broadcast netip.Addr
}

func Eligible() ([]InterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var out []InterfaceInfo

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addresses, _ := iface.Addrs()
		for _, address := range addresses {
			ipnet, ok := address.(*net.IPNet)
			if !ok {
				continue
			}

			netIpAddr, ok := netip.AddrFromSlice(ipnet.IP.To4())
			if !ok || !netIpAddr.Is4() {
				continue
			}

			ones, _ := ipnet.Mask.Size()
			prefix := netip.PrefixFrom(netIpAddr, ones)
			ones = prefix.Bits()
			mask := uint32(0xffffffff << (32 - ones))
			ip4 := prefix.Addr().As4()
			ipInt := binary.BigEndian.Uint32(ip4[:])
			broadcastInt := ipInt | ^mask
			var broadcastBytes [4]byte
			binary.BigEndian.PutUint32(broadcastBytes[:], broadcastInt)
			broadcast := netip.AddrFrom4(broadcastBytes)

			out = append(out, InterfaceInfo{
				Interface: iface,
				IP:        netIpAddr,
				Prefix:    prefix,
				Broadcast: broadcast,
			})
		}
	}

	if len(out) == 0 {
		return nil, errors.New("no eligible IPv4 interfaces")
	}

	return out, nil
}
