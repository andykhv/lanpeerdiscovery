package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func main() {
	interfaceInfos, err := broadcastInterfaces()
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	fmt.Print("=== INTERFACES ===\n")
	for _, interfaceInfo := range interfaceInfos {
		fmt.Printf("%v ip: %v ipnet: %v broadcast: %v\n", interfaceInfo.Iface, interfaceInfo.IP, interfaceInfo.IPNet, interfaceInfo.Broadcast)

		msg := []byte("discovery: hello from " + interfaceInfo.IP.String())
		fmt.Printf("broadcasting... %v\n", time.Now())
		if err := broadcast(interfaceInfo, 9999, msg); err != nil {
			fmt.Printf("broadcast error on interface: %v\n", err)
		} else {
			fmt.Printf("broadcast success!\n")
		}
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

func broadcast(interfaceInfo InterfaceInfo, port int, payload []byte) error {
	listenAddr := &net.UDPAddr{IP: interfaceInfo.IP, Port: 0} //port is set to 0, so it can be automatically chosen when creating a Listen UDPConn
	relayAddr := &net.UDPAddr{IP: interfaceInfo.Broadcast, Port: port}

	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		return fmt.Errorf("ListenUDP on %s: %w", interfaceInfo.IP, err)
	}
	defer conn.Close()

	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.WriteToUDP(payload, relayAddr)
	return err
}
