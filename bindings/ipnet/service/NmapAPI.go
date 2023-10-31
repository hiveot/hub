// Package service - NMap wrapper
// Used for scanning the local subnet and for port scanning an IP
// Insipred by https://github.com/lair-framework/go-nmap
package service

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log/slog"
	"os/exec"
	"time"
)

// DeviceInfo as discovered
type IPDeviceInfo struct {
	IP4      string             // Device IPv4 address
	IP6      string             // Device IPv6 address
	MAC      string             // Device MAC address
	Hostname string             //
	Latency  time.Duration      // device arping latency
	Ports    []IPDeviceInfoPort // Device connectable ports
}

// DeviceInfoPort containg discovered port number, port name, and protocol (tcp, udp, ...)
type IPDeviceInfoPort struct {
	Port     int
	Name     string
	Protocol string
}

// Structures to parse the NMAP XML output file
// Nmap wrapper
type NmapAPI struct {
	Version string        `xml:"version"`
	Args    string        `xml:"args,attr"`
	Start   int64         `xml:"start,attr"`
	Hosts   []NmapRunHost `xml:"host"`
}

// The NmapRunHost element that contains data of the discovered network hosts
type NmapRunHost struct {
	XMLName   xml.Name      `xml:"host"`
	Status    HostStatus    `xml:"status"`
	Ports     []HostPort    `xml:"ports>port"`
	Addresses []HostAddress `xml:"address"`
	Hostnames []Hostname    `xml:"hostnames>hostname"`
	Times     Times         `xml:"times"`
}

// HostStatus up, down, ...
type HostStatus struct {
	State     string  `xml:"state,attr"`
	Reason    string  `xml:"reason,attr"`
	ReasonTTL float32 `xml:"reason_ttl,attr"`
}

// HostPort Discovered ports
type HostPort struct {
	Protocol string          `xml:"protocol,attr"`
	Port     int             `xml:"portid,attr"`
	Service  HostPortService `xml:"service"`
}

// HostPortService Service names for the ports
type HostPortService struct {
	Name   string `xml:"name,attr"`
	Method string `xml:"method,attr"`
	Conf   int    `xml:"conf,attr"`
}

// HostAddress witth the IP adddress and ipv4 or ipv6 address type
type HostAddress struct {
	Address string `xml:"addr,attr"`
	Type    string `xml:"addrtype,attr"` // ipv4 | ipv6 | mac
}

// Hostname from local DNS lookup
type Hostname struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

// Times containing host latency
type Times struct {
	Latency string `xml:"srtt,attr"`
}

const nmapTmp = "/tmp/nmap-out.xml"

// ScanArgs with default nmap scanning arguments
// const ScanArgs = "" // Ping and Port scan
const ScanArgs = "-sn" // Ping scan without port scan    fastest way to discover hosts

// Parse the nmap XML output
// xmlFile: path to file containing the nmap scan output, obtained with option -oX
// Returns list of DeviceInfo objects and error info
func (nmapAPI *NmapAPI) ParseNmapXML(xmlFile string) ([]*IPDeviceInfo, error) {
	devices := make([]*IPDeviceInfo, 0)

	// Parse the XML output from nmap
	xmlResult, err := ioutil.ReadFile(xmlFile)
	if err != nil {
		slog.Error("Error reading tempfile with nmap result in xml", "err", err)
		return nil, err
	}

	// go-nmap is missing the Times struct, use our own structs for now
	// nmapData, err := gonmap.Parse(xmlResult)
	err = xml.Unmarshal(xmlResult, &nmapAPI)
	if err != nil {
		slog.Error("Error parsing nmap", "err", err)
		return nil, err
	}

	// Convert the list of hosts to DeviceInfo
	for _, host := range nmapAPI.Hosts {
		// Todo: how to get vendor?
		latency, _ := time.ParseDuration(host.Times.Latency + "us")
		portList := make([]IPDeviceInfoPort, 0)
		for _, portInfo := range host.Ports {
			portList = append(portList, IPDeviceInfoPort{
				Port:     portInfo.Port,
				Name:     portInfo.Service.Name,
				Protocol: portInfo.Protocol,
			})
		}
		hostName := ""
		if len(host.Hostnames) > 0 {
			hostName = host.Hostnames[0].Name
		}

		// device := &DeviceInfo{
		// 	Hostname: hostName,
		// 	latency:  latency,
		// 	lastSeen: time.Now(),
		// 	Ports:    portList,
		// }
		deviceInfo := new(IPDeviceInfo)
		deviceInfo.Hostname = hostName
		deviceInfo.Latency = latency
		deviceInfo.Ports = portList

		for _, addr := range host.Addresses {
			switch addr.Type {
			case "ipv4":
				deviceInfo.IP4 = addr.Address
			case "ipv6":
				deviceInfo.IP6 = addr.Address
			case "mac":
				deviceInfo.MAC = addr.Address
			}
		}
		devices = append(devices, deviceInfo)
	}
	return devices, nil
}

// ScanSubnet to scan a subnet or IP address on a network
//
//   - subnets: Subnet(s) to scan, x.y.z.0/24 or x.y.z.10-30 or multiple combinations
//   - scanPorts: Include a port scan in addition to a device scan
//   - useSudo: run nmap as root using sudo. The user must have sudo rights for this to work.
//     This provides the MAC address in the result
//     Note that the scan can last much longer as more data is gathered.
//
// @returns list of DeviceInfo of all discovered network nodes
func (nmapAPI *NmapAPI) ScanSubnet(subnets []string, scanPorts bool, useSudo bool) ([]*IPDeviceInfo, error) {
	// construct the nmap scan arguments
	slog.Info("Executing nmap", "subnets", subnets)
	scanArgList := make([]string, 0)
	if !scanPorts {
		scanArgList = append(scanArgList, ScanArgs) // default scan arguments
	}
	// scan output to file.
	scanArgList = append(scanArgList, []string{"-oX", nmapTmp}...)
	// last append the actual subnets
	scanArgList = append(scanArgList, subnets...)

	var cmd *exec.Cmd
	if useSudo {
		// TODO: location in config
		scanArgList = append([]string{"nmap"}, scanArgList...)
		cmd = exec.Command("sudo", scanArgList...)
	} else {
		cmd = exec.Command("nmap", scanArgList...)
	}
	cmd.Stderr = new(bytes.Buffer)
	out, err := cmd.Output()
	if err != nil {
		slog.Error("Error executing nmap", "stderr output", cmd.Stderr, "out", out, "err", err)
		return nil, err
	}
	deviceList, err := nmapAPI.ParseNmapXML(nmapTmp)
	if err != nil {
		return nil, err
	}

	return deviceList, nil
}

// NewNmapAPI constructor for NMAP wrapper
func NewNmapAPI() *NmapAPI {
	nmapAPI := new(NmapAPI)
	return nmapAPI
}
