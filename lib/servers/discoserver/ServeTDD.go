// Package discoserver to publish Hub services for discovery
package discoserver

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/utils"
)

// Parameter names in the discovery record
// const AuthParam = "auth"       // authentication server
// const ConnectParam = "connect" // hub connection server
const DefaultServiceName = "wot"

// hiveot endpoint identifiers
const AuthEndpoint = "login"
const WSSEndpoint = "wss"
const SSEEndpoint = "sse"

// DefaultHttpGetDirectoryTDPath contains the path to the digital twin directory
// TD document uses the 'well-known' path
const DefaultHttpGetDirectoryTDPath = "/.well-known/wot"

// ServeTDDiscovery publishes a Thing Document Directory discovery record.
// This sets type to 'Directory' and scheme to https.
//
// WoT defines this as service name: _directory._sub._{serviceName}._tcp with the
// TXT record containing the fields 'td', 'type', and 'scheme'
// See also: https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
//
// endpoints is a hiveot addon that provides connection addresses for transport
// protocols without the need to use forms. This is intended for connection
// oriented protocols such as websocket, sse-sc, mqtt, and others. The schema
// identifies the protocol.
// Intended for simplify use when all affordances use the same protocol.
//
//	instanceName is the name of the server instance. "hiveot" for the hub.
//	serviceName is the discover name. Default is 'wot'. This can be changed for testing.
//	tddURL is the URL the directory TD is served at.
//	endpoints contains a map of {scheme:connection} URLs
//
// Returns the discovery service instance. Use Shutdown() when done.
func ServeTDDiscovery(
	instanceName string, serviceName string,
	tddURL string, endpoints map[string]string,
) (
	*zeroconf.Server, error) {

	parts, err := url.Parse(tddURL)
	if err != nil {
		return nil, err
	}

	// setup the introduction mechanism
	if instanceName == "" {
		instanceName, _ = os.Hostname()
	}
	if serviceName == "" {
		serviceName = DefaultServiceName
	}
	tdPath := parts.Path
	if tdPath == "" {
		tdPath = DefaultHttpGetDirectoryTDPath
	}
	portString := parts.Port()
	portNr, err := strconv.Atoi(portString)
	if err != nil {
		return nil, err
	}
	scheme := parts.Scheme
	// FIXME: DNS might not work for the local network. Use an IP instead.
	// Does this support IPv6?
	outIP := utils.GetOutboundIP("")
	address := outIP.String()
	//address := parts.Hostname()

	subType := "_directory._sub"
	// add WoT discovery parameters
	params := map[string]string{
		"td":     tdPath,
		"scheme": scheme,
		"type":   "Directory",
	}
	// add connection endpoints as parameters
	for ep, epURL := range endpoints {
		params[ep] = epURL
	}
	slog.Info("Serving discovery for address",
		slog.String("address", address),
		slog.Int("port", portNr),
	)
	discoServer, err := ServeDnsSD(
		instanceName, serviceName, subType,
		address, portNr, params)

	return discoServer, err
}

// ServeDnsSD publishes a service discovery record.
//
//	DNS-SD will publish this as _<instance>._<serviceName>._tcp
//
//	instanceID is the unique ID of the service instance, usually the plugin-ID
//	serviceName is the discover name. For example "wot"
//	address service listening IP address
//	port service listing port
//	params is a map of key-value pairs to include in discovery, eg td, type and scheme in wot
//
// Returns the discovery service instance. Use Shutdown() when done.
func ServeDnsSD(instanceID string, serviceName string, subType string,
	address string, port int, params map[string]string) (*zeroconf.Server, error) {
	var ips []string

	slog.Info("ServeDnsSD",
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
	if subType != "" {
		//serviceType = serviceType + "," + subType
		serviceType = serviceType + ",test1"
	}
	server, err := zeroconf.RegisterProxy(
		instanceID, serviceType, domain, int(port), hostname, ips, textRecord, ifaces)
	if err != nil {
		slog.Error("Failed to start the zeroconf server", "err", err)
	}
	return server, err
}
