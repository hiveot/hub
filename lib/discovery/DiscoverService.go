// Package hubclient with client for Hub gateway service discovery
package discovery

import (
	"fmt"
	"github.com/grandcat/zeroconf"
	"golang.org/x/exp/slog"
	"strings"
)

const HIVEOT_DNSSD_TYPE = "_hiveot._tcp"

// DiscoverService searches for services with the given type and returns all its instances.
// This is a wrapper around various means of discovering services and supports the discovery of multiple
// instances of the same service (name). The serviceName must contain the simple name of the Hub service.
// For example, use 'idprov' for the provisioning service which DNS-SD will publish as _idprov._tcp.
//
//	serviceType is the type of service to discover without the "_", eg "hiveot" in "_hiveot._tcp"
//	waitSec is the time to wait for the result
//
// Returns the first instance address, port and discovery parameters, plus records of additional discoveries,
// or an error if nothing is found
func DiscoverService(serviceType string, waitSec int) (
	address string, port int, params map[string]string,
	records []*zeroconf.ServiceEntry, err error) {
	params = make(map[string]string)

	serviceProtocol := "_" + serviceType + "._tcp"
	if serviceType == "" {
		serviceProtocol = HIVEOT_DNSSD_TYPE
	}
	records, err = DnsSDScan(serviceProtocol, waitSec)
	if err != nil {
		return "", 0, nil, nil, err
	}
	if len(records) == 0 {
		err = fmt.Errorf("no service of type '%s' found after %d seconds", serviceProtocol, waitSec)
		return "", 0, nil, nil, err
	}
	rec0 := records[0]

	// determine the address string
	// use the local IP if provided
	if len(rec0.AddrIPv4) > 0 {
		address = rec0.AddrIPv4[0].String()
	} else if len(rec0.AddrIPv6) > 0 {
		address = rec0.AddrIPv6[0].String()
	} else {
		// fall back to use host.domainname
		address = rec0.HostName
	}

	// reconstruct key-value parameters from TXT record
	for _, txtRecord := range rec0.Text {
		kv := strings.Split(txtRecord, "=")
		if len(kv) != 2 {
			slog.Info("Ignoring non key-value in TXT record", "key", txtRecord)
		} else {
			params[kv[0]] = kv[1]
		}
	}
	return address, rec0.Port, params, records, nil
}

// LocateHub determines the nats URL to use.
// This first checks if a local connection can be made on the default port.
// Secondly, perform a DNS-SD search.
func LocateHub(searchTime int) (fullURL string) {
	if searchTime <= 0 {
		searchTime = 1
	}

	// discover the service and determine the best matching record
	// yes, this seems like a bit of a pain
	// default is the hiveot service
	addr, port, params, records, err := DiscoverService("hiveot", searchTime)
	slog.Info("Found N records. Using the first record.", "N", len(records))
	if err != nil {
		// failed, nothing to be found
		return ""
	}
	fullURL = fmt.Sprintf("nats://%s:%s%s", addr, port, params["path"])
	return
}
