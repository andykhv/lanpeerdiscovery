package main

import (
	"errors"
	"fmt"
	"net"
)

func main() {
	interfaceInfos, err := broadcastInterfaces()
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	for _, interfaceInfo := range interfaceInfos {
		fmt.Printf("interface: %v ip: %v ipnet: %v broadcast: %v\n", interfaceInfo.Iface, interfaceInfo.IP, interfaceInfo.IPNet, interfaceInfo.Broadcast)
	}
}

type InterfaceInfo struct {
	Iface     net.Interface
	IP        net.IP
	IPNet     *net.IPNet
	Broadcast net.IP
}

// find interfaces that are up, non-loopback, and broadcastable
func broadcastInterfaces() ([]InterfaceInfo, error) {
	interfaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	var out []InterfaceInfo
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addresses, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, a := range addresses {
			ipnet, ok := a.(*net.IPNet)

			if !ok || ipnet == nil || ipnet.IP.To4() == nil {
				continue
			}

			mask := ipnet.Mask
			ip := ipnet.IP.To4()
			broadcast := make(net.IP, 4)
			for i := 0; i < 4; i++ {
				broadcast[i] = (ip[i] & mask[i]) | (^mask[i])
			}

			out = append(out, InterfaceInfo{
				Iface:     iface,
				IP:        ip,
				IPNet:     ipnet,
				Broadcast: broadcast,
			})
		}
	}

	if len(out) == 0 {
		return nil, errors.New("no available ipv4 lan broadcast interfaces found")
	}

	return out, nil
}
