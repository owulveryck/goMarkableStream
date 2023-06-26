package main

import (
	"fmt"
	"net"
)

func ifaces() {
	// Get the list of network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("Failed to retrieve network interfaces:", err)
		return
	}

	// Iterate through the interfaces
	for _, iface := range interfaces {
		// Filter out loopback and non-up interfaces
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				fmt.Println("Failed to retrieve addresses for interface", iface.Name, ":", err)
				continue
			}

			// Iterate through the addresses
			for _, addr := range addrs {
				// Check if the address is an IP address
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						fmt.Println("Local IP address:", ipnet.IP.String())
					}
				}
			}
		}
	}
}
