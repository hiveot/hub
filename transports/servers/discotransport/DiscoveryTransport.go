package discotransport

import (
	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/net"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// DiscoveryTransport is a thin wrapper around the discovery service
type DiscoveryTransport struct {
	cfg discovery.DiscoveryConfig
	// service discovery using mDNS
	disco *zeroconf.Server
}

// AddTDForms does not apply to the discovery service
func (svc *DiscoveryTransport) AddTDForms(*td.TD) {
	// nothing to do here
}

func (svc *DiscoveryTransport) Stop() {
	svc.disco.Shutdown()
}

func StartDiscoveryTransport(cfg discovery.DiscoveryConfig) (*DiscoveryTransport, error) {
	var err error
	svc := DiscoveryTransport{
		cfg: cfg,
	}
	// get the local address from outgoing address
	obip := net.GetOutboundIP("").String()
	svc.disco, err = discovery.ServeDiscovery(
		"hiveot", "hiveot", obip, cfg.ServerPort,
		map[string]string{
			transports.ProtocolTypeHiveotSSE:    cfg.HiveotSseURL,
			transports.ProtocolTypeHiveotWSS:    cfg.HiveotWssURL,
			transports.ProtocolTypeWotHTTPBasic: cfg.WotHttpBasicURL,
			transports.ProtocolTypeWotWSS:       cfg.WotWssURL,
			//discovery.HiveotMqttWssID: cfg.MqttWssURL,
			//discovery.HiveotMqttTcpID: cfg.MqttTcpURL,
		})
	if err != nil {
		slog.Error("Can't start discovery. Invalid server URL",
			"serverAddr", cfg.ServerAddr)
	}
	return &svc, err
}
