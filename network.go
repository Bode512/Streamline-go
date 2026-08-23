package main

import "net"

type networkInterface struct {
	Name     string   `json:"name"`
	IPs      []string `json:"ips"`
	Up       bool     `json:"up"`
	Loopback bool     `json:"loopback"`
}

func localInterfaces() []networkInterface {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	result := make([]networkInterface, 0, len(interfaces))
	for _, network := range interfaces {
		addresses, err := network.Addrs()
		if err != nil {
			continue
		}
		ips := make([]string, 0, len(addresses))
		for _, address := range addresses {
			ip, _, splitErr := net.ParseCIDR(address.String())
			if splitErr == nil {
				ips = append(ips, ip.String())
			}
		}
		result = append(result, networkInterface{
			Name: network.Name, IPs: ips, Up: network.Flags&net.FlagUp != 0,
			Loopback: network.Flags&net.FlagLoopback != 0,
		})
	}
	return result
}
