// Package service - arp wrapper
package service

import (
	"log/slog"
	"strings"

	mgarp "github.com/mostlygeek/arp"
)

// UpdateDeviceInfoFromArpCache will update the MAC address in the provided device information from the arp cache
// Intended to be used when sudo is not available for arp scanning
// Returns a map of all known IP to MAC addresses
func UpdateDeviceInfoFromArpCache(devices []*IPDeviceInfo) map[string]string {
	arpTable := mgarp.Table()
	for i := range devices {
		deviceInfo := devices[i]
		if mac, found := arpTable[deviceInfo.IP4]; found {
			// if a MAC is known, verify if it is still valid for the current IP
			if deviceInfo.MAC != "" {
				if mac != deviceInfo.MAC {
					// The ARP cache contains a different MAC for this IP address
					// Maybe the IP is updated.
					slog.Warn("ARP cache contains a different device MAC for IP",
						"ip", deviceInfo.IP4,
						"current MAC", deviceInfo.MAC,
						"ARP cache MAC", mac)
				}
			} else {
				deviceInfo.MAC = strings.ToUpper(mac)
			}
		}
	}
	return arpTable
}
