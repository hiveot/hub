package discoclient

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hiveot/hub/lib/servers/discoserver"
)

const HIVEOT_TDD_DNSSD_TYPE = "_hiveot._wot._tcp"
const WOT_DNSSD_TYPE = "_wot._tcp"
const WOT_TDD_DNSSD_TYPE = "_directory._sub._wot._tcp"

//const WOT_UDP_DNSSD_TYPE = "_wot._udp"

type DiscoveryResult struct {
	Addr        string // IP or hostname of the server
	Port        int    // port the server listens on
	IsDirectory bool   // URL is that of a Thing Directory
	IsThing     bool   // URL is of a Thing
	Instance    string
	// predefined WoT discovery parameters
	Scheme string // Scheme part of the URL
	Type   string // Thing or Directory
	TD     string // absolute pathname of the TD or TDD
	// hiveot connection endpoints
	AuthEndpoint string            // authentication service endpoint
	SSEEndpoint  string            // Http/SSE-SC transport protocol
	WSSEndpoint  string            // Websocket transport
	Params       map[string]string // optional parameters
}

// DiscoverTDD supports introduction mechanisms to bootstrap the WoT discovery
// process and returns a list of discovered directory TD URLs.
//
// This supports multiple discovery methods and merges the result.
//
// This supports an optional filter on the instance. Intended to filter for a
// specific type of directory, for example a "hiveot" hub directory.
//
//	serviceName of the service to discover. Default is "wot"
//	waitTime is the duration to wait for the results
//	firstResult returns immediately with the first result instead of waiting the full waitTime
func DiscoverTDD(serviceName string, waitTime time.Duration, firstResult bool) []*DiscoveryResult {

	drList := make([]*DiscoveryResult, 0)

	if serviceName == "" {
		serviceName = discoserver.DefaultServiceName
	}

	// query well-known thing TD or TDD
	drList2, err := DiscoverWithDnsSD(serviceName, waitTime, firstResult)
	if err == nil && len(drList2) > 0 {
		drList = append(drList, drList2...)
	}

	// remove duplicates
	return drList
}

// DiscoverWithDnsSD searches for services with the given instance or service name.
//
//	[_{instanceName}.]_{serviceName}._tcp service type
//
// instanceName is optional to search for a particular implementation such as hiveot
func DiscoverWithDnsSD(
	serviceName string, waitTime time.Duration, firstResult bool) ([]*DiscoveryResult, error) {

	drList := make([]*DiscoveryResult, 0)

	serviceType := fmt.Sprintf("_%s._tcp", serviceName)
	if waitTime == 0 {
		waitTime = time.Second * 3
	}
	records, err := DnsSDScan(serviceType, waitTime, firstResult)
	if err != nil {
		return drList, err
	}
	if len(records) == 0 {
		err = fmt.Errorf("DiscoverService: no service of type '%s' found after %d seconds",
			serviceType, int(waitTime/time.Second))
		return drList, err
	}
	for _, rec := range records {
		discoResult := DiscoveryResult{
			Params:   make(map[string]string),
			Instance: rec.Instance,
			Port:     rec.Port,
		}

		// determine the address string
		// use the local IP if provided
		if len(rec.AddrIPv4) > 0 {
			discoResult.Addr = rec.AddrIPv4[0].String()
		} else if len(rec.AddrIPv6) > 0 {
			discoResult.Addr = rec.AddrIPv6[0].String()
		} else {
			// fall back to use host.domainname
			discoResult.Addr = rec.HostName
		}
		// https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
		if rec.ServiceName() == WOT_TDD_DNSSD_TYPE || rec.ServiceName() == HIVEOT_TDD_DNSSD_TYPE {
			discoResult.IsDirectory = true
		} else if rec.ServiceName() == WOT_DNSSD_TYPE {
			//this is a thing
			discoResult.IsThing = true
		} else {
			// not sure what this is
		}

		// For TCP-based services, the following information MUST be included in the
		// TXT record that is pointed to by the Service Instance Name:
		for _, txtRecord := range rec.Text {
			kv := strings.Split(txtRecord, "=")
			if len(kv) != 2 {
				slog.Info("DiscoverService: Ignoring non key-value in TXT record", "key", txtRecord)
				continue
			}
			key := kv[0]
			val := kv[1]
			if key == "td" {
				discoResult.TD = val // Absolute pathname of the TD/TDD
			} else if key == "type" {
				discoResult.Type = val // Type of TD, "Thing" or "Directory" or "Hiveot"
				discoResult.IsDirectory = val == "Directory"
				discoResult.IsThing = val == "Thing"
			} else if key == "scheme" {
				// http (default), https, coap+tcp, coaps+tcp
				discoResult.Scheme = val // Scheme part of URL
			} else if key == discoserver.WSSEndpoint {
				// 'base' is specific to hiveot to provide a default connection URL
				discoResult.WSSEndpoint = val
			} else if key == discoserver.SSEEndpoint {
				// 'base' is specific to hiveot to provide a default connection URL
				discoResult.SSEEndpoint = val
			} else if key == discoserver.AuthEndpoint {
				discoResult.AuthEndpoint = val
			}
			discoResult.Params[key] = val
		}
		drList = append(drList, &discoResult)
	}
	return drList, nil
}
