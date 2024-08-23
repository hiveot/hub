package discotransport

import (
	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hub/lib/discovery"
	"github.com/hiveot/hub/lib/net"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/url"
	"strconv"
)

// DiscoveryTransport is a thin wrapper around the discovery service
type DiscoveryTransport struct {
	cfg DiscoveryConfig
	// service discovery using mDNS
	disco *zeroconf.Server
}

// AddTDForms does not apply to the discovery service
func (svc *DiscoveryTransport) AddTDForms(td *tdd.TD) {
	// nothing to do here
}

// Start the discovery of the runtime service and include its transports
// TODO: support a record for each transport.
func (svc *DiscoveryTransport) Start(serverURL string) error {
	urlInfo, err := url.Parse(serverURL)
	if err == nil {
		port, _ := strconv.Atoi(urlInfo.Port())
		// get the local address from outgoing address
		obip := net.GetOutboundIP("").String()
		svc.disco, err = discovery.ServeDiscovery(
			"hiveot", "hiveot", obip, port,
			map[string]string{
				"rawurl": serverURL,
			})
	} else {
		slog.Error("Can't start discovery. Invalid server URL",
			"serverURL", serverURL)
	}
	return nil
}
func (svc *DiscoveryTransport) Stop() {
	svc.disco.Shutdown()
}

func NewDiscoveryTransport(cfg DiscoveryConfig) *DiscoveryTransport {
	svc := DiscoveryTransport{
		cfg: cfg,
	}
	return &svc
}
