package service

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/utils/net"
)

// determineSubnetsToScan returns a list of subnets and IP addresses to scan, obtained from configuration
// If no subnets are configured, determine the subnets of the interfaces
func (svc *IPNetBinding) determineSubnetsToScan() []string {
	// auto determine subnets
	subnets, _ := net.GetIP4Subnets(true)

	return subnets
}

// onDiscoveredDevice adds the discovered network device
// If the device was already discovered it will be updated
// This will add the device by its MAC and IP to the device lookup table
func (svc *IPNetBinding) onDiscoveredDevice(deviceInfo *IPDeviceInfo, publish bool) {
	// Use the MAC then hostname, then IP as device address
	address := deviceInfo.IP4
	if address == "" {
		address = deviceInfo.IP6
	}
	if address == "" {
		slog.Error("onDiscoveredDevice: Ignored device with no IP address", "deviceInfo", deviceInfo)
		return
	}

	// Check if the discovered device is known (MAC or IP)
	dev := svc.devicesMap[address]
	if dev == nil && deviceInfo.MAC != "" {
		// this device might be pre-configured with a MAC address. Perform a lookup
		dev = svc.devicesMap[deviceInfo.MAC]
	}
	if dev != nil {
		// Existing node
		slog.Info("onDiscoveredDevice: Updating existing device info",
			"MAC", deviceInfo.MAC, "IP4", deviceInfo.IP4, "IP6", deviceInfo.IP6, "hostname", deviceInfo.Hostname)
		// Update the existing device info with discovered data
	} else {
		// new node
		slog.Info("onDiscoveredDevice: New device found",
			"MAC", deviceInfo.MAC, "IP4", deviceInfo.IP4, "IP6", deviceInfo.IP6, "hostname", deviceInfo.Hostname)
		// Nodes are Identified by IP address
		svc.devicesMap[deviceInfo.IP4] = deviceInfo
		svc.devicesMap[deviceInfo.IP6] = deviceInfo
		svc.devicesMap[deviceInfo.MAC] = deviceInfo
	}

	// TODO: Store and publish ports
	//portsProp := node.NewProperty("ports", device.DataTypeString, device.UnitNameNone, false)
	// Only update ports if it has results
	// if deviceInfo.Ports != nil && len(deviceInfo.Ports) > 0 {
	//portsProp.Value = deviceInfo.Ports
	// }
	//// Update discovered nodes
	//if publish {
	//  //if !found {
	//    svc.myzoneMqtt.PublishNode(node, true)
	//    svc.myzoneMqtt.PublishPropertyValues(node)
	//  //}
	//}
}

// Poll nodes on the network. Must be called after Start()
// This uses nmap to scan the network and arpcache to determine the mac address
// publish controls if the results are published
// The 'device' parameters in discovery are defined in the config file.
func (svc *IPNetBinding) Poll() {
	slog.Info("Starting polling")
	nmapAPI := new(NmapAPI)
	subNets := svc.determineSubnetsToScan()

	// Perform a port scan on all IPs which will load up the arp cache and get the MAC from arp.
	// If sudo us available the use sudo instead to get the MAC addresses
	scanAsRoot := svc.config.ScanAsRoot

	deviceInfoList, _ := nmapAPI.ScanSubnet(subNets, svc.config.PortScan, scanAsRoot)

	// If not using root then use arp cache to discover mac addresses
	if !scanAsRoot {
		UpdateDeviceInfoFromArpCache(deviceInfoList)
	}
	// Add discovered nodes and update existing nodes with discovery result
	for _, deviceInfo := range deviceInfoList {
		svc.onDiscoveredDevice(deviceInfo, true)
	}

	//svc.myzoneMqtt.PublishNode(svc.config.Discovery.Topic, svc.config.Nodes)
	slog.Info("Poll: Completed",
		"nr result (might have dups)", len(deviceInfoList),
		"nr known nodes", len(svc.devicesMap))

}
