package main

import (
	"fmt"

	"github.com/andykhv/lanpeerdiscovery/internal/netx"
)

func main() {
	ifaces, err := netx.Eligible()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	for _, iface := range ifaces {
		fmt.Printf("interface: %v - IP:%v - Broadcast: %v", iface.Interface.Name, iface.IP, iface.Broadcast)
	}
}
