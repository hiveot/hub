// Package discovery to publish Hub services for discovery
package discovery

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/grandcat/zeroconf"
)

// ServeDiscovery publishes a service for discovery.
// See also 'DiscoverService' for discovery of this published service.
//
// WoST services use this to announce the service instance and how they can be reached on the local domain.
//
//	Instance = instance name of the service. Used to differentiate between services with the same name (type)
//	DiscoveryServiceName = name of the provided service, for example, ipp, idprov, hiveot
//
// This is a wrapper around one or more discovery methods. Internally this uses DNS-SD but can be
// expanded with additional protocols in the future.
//
//	DNS-SD will publish this as _<instance>._<serviceName>._tcp
//
//	instanceID is the unique ID of the service instance, usually the plugin-ID
//	serviceName is the discover name. For example "hiveot"
//	address service listening IP address
//	port service listing port
//	params is a map of key-value pairs to include in discovery, eg rawURL
//
// Returns the discovery service instance. Use Shutdown() when done.
func ServeDiscovery(instanceID string, serviceName string,
	address string, port int, params map[string]string) (*zeroconf.Server, error) {
	var ips []string

	slog.Info("ServeDiscovery",
		slog.String("instanceID", instanceID),
		slog.String("serviceName", serviceName),
		slog.String("address", address),
		slog.Int("port", port),
		"params", params)
	if serviceName == "" {
		err := fmt.Errorf("Empty serviceName")
		return nil, err
	}

	// only the local domain is supported
	domain := "local."
	hostname, _ := os.Hostname()

	// if the given address isn't a valid IP address. try to resolve it instead
	ips = []string{address}
	if net.ParseIP(address) == nil {
		// was a hostname provided instead IP?
		hostname = address
		parts := strings.Split(address, ":") // remove port
		actualIP, err := net.LookupIP(parts[0])
		if err != nil {
			// can't continue without a valid address
			slog.Error("Provided address is not an IP and cannot be resolved",
				"address", address, "err", err)
			return nil, err
		}
		ips = []string{actualIP[0].String()}
	}

	ifaces, err := utils.GetInterfaces(ips[0])
	if err != nil || len(ifaces) == 0 {
		slog.Warn("Address does not appear on any interface. Continuing anyways", "address", ips[0])
	}
	// add a text record with key=value pairs
	textRecord := []string{}
	for k, v := range params {
		textRecord = append(textRecord, fmt.Sprintf("%s=%s", k, v))
	}
	// I don't like this 'hiding' of the service type, but it is too DNS-SD specific
	serviceType := fmt.Sprintf("_%s._tcp", serviceName)
	server, err := zeroconf.RegisterProxy(
		instanceID, serviceType, domain, int(port), hostname, ips, textRecord, ifaces)
	if err != nil {
		slog.Error("Failed to start the zeroconf server", "err", err)
	}
	return server, err
}
