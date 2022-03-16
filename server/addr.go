package server

import (
	"errors"
	"net"
)

var (
	privateBlocks []*net.IPNet
)

func init() {
	for _, b := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", "fd00::/8"} {
		if _, block, err := net.ParseCIDR(b); err == nil {
			privateBlocks = append(privateBlocks, block)
		}
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, privateBlock := range privateBlocks {
		if privateBlock.Contains(ip) {
			return true
		}
	}
	return false
}

func privateIP() (net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			switch addr := addr.(type) {
			case *net.IPAddr:
				if isPrivateIP(addr.IP) {
					return addr.IP, nil
				}
			case *net.IPNet:
				if isPrivateIP(addr.IP) {
					return addr.IP, nil
				}
			}
		}
	}
	return nil, errors.New("no private IP")
}
