package remarkable

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

func HostsIP(addr string) bool {
	if strings.Contains(addr, ":") {
		addr = addr[:strings.Index(addr, ":")]
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}

	mask, err := getMask()
	if err != nil {
		return false
	}

	return mask.Contains(ip)
}

func getMask() (*net.IPNet, error) {
	iface, err := net.InterfaceByName("usb1")
	if err != nil {
		return nil, fmt.Errorf("failed to locate usb1 interface: %w", err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate ips for usb1 interface: %w", err)
	}

	for _, addr := range addrs {
		ip, mask, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}

		if strings.HasSuffix(ip.String(), ".1") {
			return mask, nil
		}
	}
	return nil, errors.New("no expected ip found on usb interface")
}
