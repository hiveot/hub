// Package hubclient with client for Hub gateway service discovery
package discovery

import (
	"fmt"
	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hub/transports"
	"log/slog"
	"strings"
	"time"
)

const HIVEOT_DNSSD_TYPE = "_hiveot._tcp"

//const HiveotSsescID = "ssesc"
//const HiveotWssID = "wss-hiveot"
//const WotHttpBasicID = "https"
//const WotWssID = "wss"

//const HiveotMqttWssID = "mqtt-wss"
//const HiveotMqttTcpID = "mqtt-tcps"

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

// LocateHub determines the available hub URLs.
// This first checks if a local connection can be made on the default port.
// Secondly, perform a DNS-SD search.
// If firstResult is set then return immediately after the first result or searchTime
func LocateHub(searchTime time.Duration, firstResult bool) (cfg DiscoveryConfig, err error) {
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
		return cfg, err
	}
	cfg = NewDiscoveryConfig(addr)
	cfg.ServerAddr = addr
	cfg.ServerPort = port
	cfg.ServiceID = "hiveot"
	cfg.HiveotWssURL = params[transports.ProtocolTypeHiveotWSS]
	cfg.HiveotSseURL = params[transports.ProtocolTypeHiveotSSE]
	cfg.WotHttpBasicURL = params[transports.ProtocolTypeWotHTTPBasic]
	cfg.WotWssURL = params[transports.ProtocolTypeWotWSS]
	//cfg.MqttTcpURL = params[HiveotMqttTcpID]
	//cfg.MqttWssURL = params[HiveotMqttWssID]

	// fallback to defaults
	if addr == "" {
		addr = "localhost"
		cfg.ServerAddr = addr
		//cfg.ServerPort = servers.DefaultHttpsPort
		//cfg.WssURL = fmt.Sprintf("wss://%s:%d%s", addr, servers.DefaultHttpsPort, "")
		//cfg.SsescURL = fmt.Sprintf("https://%s:%d%s", addr, servers.DefaultHttpsPort, "")
		//cfg.MqttTcpURL = fmt.Sprintf("https://%s:%d%s", addr, servers.DefaultMqttWssPort, "")
		//cfg.MqttWssURL = fmt.Sprintf("https://%s:%d%s", addr, servers.DefaultMqttTcpPort, "")
	}
	slog.Info("LocateHub",
		slog.Int("Nr records", len(records)),
		slog.String("addr", cfg.ServerAddr),
	)
	return cfg, err
}
