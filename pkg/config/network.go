package config

import (
	"fmt"
	"net"
)

// getIPFromInterface retrieves the IPv4 address from a network interface
func GetIPFromInterface(interfaceName string) (string, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", fmt.Errorf("interface %s not found: %w", interfaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for %s: %w", interfaceName, err)
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		// Check if it's an IPv4 address and not loopback
		if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no IPv4 address found on interface %s", interfaceName)
}

// listNetworkInterfaces returns all available network interfaces
func ListNetworkInterfaces() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, iface := range ifaces {
		names = append(names, iface.Name)
	}

	return names, nil
}
