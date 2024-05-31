// Package hubclient with client for Hub gateway service discovery
package discovery

import (
	"fmt"
	"github.com/grandcat/zeroconf"
	"log/slog"
	"strings"
	"time"
)

const HIVEOT_DNSSD_TYPE = "_hiveot._tcp"

// DiscoverService searches for services with the given type and returns all its instances.
// This is a wrapper around various means of discovering services and supports the discovery of multiple
// instances of the same service (name). The serviceName must contain the simple name of the Hub service.
//
//	serviceType is the type of service to discover without the "_", eg "hiveot" in "_hiveot._tcp"
//	waitTime is the duration to wait for the result
//
// Returns the first instance address, port and discovery parameters, plus records of additional discoveries,
// or an error if nothing is found
func DiscoverService(serviceType string, waitTime time.Duration, firstResult bool) (
	address string, port int, params map[string]string,
	records []*zeroconf.ServiceEntry, err error) {

	params = make(map[string]string)

	serviceProtocol := "_" + serviceType + "._tcp"
	if serviceType == "" {
		serviceProtocol = HIVEOT_DNSSD_TYPE
	}
	records, err = DnsSDScan(serviceProtocol, waitTime, firstResult)
	if err != nil {
		return "", 0, nil, nil, err
	}
	if len(records) == 0 {
		err = fmt.Errorf("DiscoverService: no service of type '%s' found after %d seconds",
			serviceProtocol, int(waitTime/time.Second))
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
			slog.Info("DiscoverService: Ignoring non key-value in TXT record", "key", txtRecord)
		} else {
			params[kv[0]] = kv[1]
		}
	}
	return address, rec0.Port, params, records, nil
}

// LocateHub determines the nats URL to use.
// This first checks if a local connection can be made on the default port.
// Secondly, perform a DNS-SD search.
// If firstResult is set then return immediately after the first result or searchTime
func LocateHub(searchTime time.Duration, firstResult bool) (fullURL string) {
	if searchTime <= 0 {
		searchTime = time.Second * 3
	}

	// discover the service and determine the best matching record
	// yes, this seems like a bit of a pain
	// default is the hiveot service
	addr, port, params, records, err := DiscoverService("hiveot", searchTime, firstResult)
	if err != nil {
		// failed, nothing to be found
		slog.Warn("LocateHub: Hub not found")
		return ""
	}
	fullURL, found := params["rawurl"]
	if !found {
		fullURL = fmt.Sprintf("tcp://%s:%d%s", addr, port, params["path"])
	}
	slog.Info("LocateHub",
		slog.Int("Nr records", len(records)),
		slog.String("fullURL", fullURL),
	)
	return fullURL
}
