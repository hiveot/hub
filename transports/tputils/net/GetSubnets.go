package net

import (
	"net"
	"strconv"
)

// GetIP4Subnets of the valid IPv4 interfaces
// Returns list of one or more ip/subnet strings
// This is often a single subnet unless there is wifi, multiple cards or vlans
func GetIP4Subnets() ([]string, error) {
	subnets := []string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		ifAddrList, err := iface.Addrs()
		if err == nil {
			// For each address, get its IP address and netmask
			for _, addr := range ifAddrList {
				// addr can be different types, one of which can by net.IPAddr type
				switch v := addr.(type) {
				case *net.IPNet:
					// only take a valid IPv4 address
					if !v.IP.IsLoopback() && (v.IP.To4() != nil) {
						ip := v.IP.String()
						subnetSize, bits := v.Mask.Size()
						_ = bits
						//log.Debugf("GetIP4Subnets - IP: %v, subnet size:%v bits:%v", ip, subnetSize, bits)
						subnet := ip + "/" + strconv.Itoa(subnetSize)
						subnets = append(subnets, subnet)
					}
				}
			}
		}
	}

	return subnets, nil
}
