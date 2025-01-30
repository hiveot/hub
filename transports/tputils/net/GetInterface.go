// Package utils with functions to get the outbound network interface
package net

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
)

// GetInterfaces returns a list of active network interfaces excluding the loopback interface
//
//	address to only return the interface that serves the given IP address
func GetInterfaces(address string) ([]net.Interface, error) {
	result := make([]net.Interface, 0)
	ip := net.ParseIP(address)

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		// ignore interfaces without address
		if err == nil {
			for _, a := range addrs {
				switch v := a.(type) {
				case *net.IPAddr:
					result = append(result, iface)
					slog.Info("Found: Interface" + v.String())

				case *net.IPNet:
					ifNet := a.(*net.IPNet)
					hasIP := ifNet.Contains(ip)

					// ignore loopback interface
					if hasIP && !a.(*net.IPNet).IP.IsLoopback() {
						result = append(result, iface)
						slog.Info(fmt.Sprintf("Found network %v : %s [%v/%v]\n", iface.Name, v, v.IP, v.Mask))
					}
				}
			}
		}
	}
	return result, nil
}

// GetOutboundInterface Get preferred outbound network interface of this machine
// Credits: https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
// and https://qiita.com/shaching/items/4c2ee8fd2914cce8687c
func GetOutboundInterface(address string) (interfaceName string, macAddress string, ipAddr net.IP) {
	if address == "" {
		address = "1.1.1.1"
	}

	// This dial command doesn't actually create a connection
	conn, err := net.Dial("udp", address+":9999")
	if err != nil {
		slog.Error("GetOutboundInterface",
			slog.String("address", address),
			"err", err)
		return "", "", nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ipAddr = localAddr.IP

	// find the first interface for this address
	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {

		if addrs, err := interf.Addrs(); err == nil {
			for index, addr := range addrs {
				slog.Debug(fmt.Sprintf("[%d]%s > %s", index, interf.Name, addr))

				// only interested in the name with current IP address
				if strings.Contains(addr.String(), ipAddr.String()) {
					interfaceName = interf.Name
					macAddress = fmt.Sprint(interf.HardwareAddr)
					break
				}
			}
		}
	}
	return
}
