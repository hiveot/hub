package discoserver

import (
	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"log/slog"
	"net/http"
	"net/url"
)

// DiscoveryServer supports the WoT discovery introduction and exploration
// mechanisms for obtaining TD's for use by consumers.
type DiscoveryServer struct {

	//connectURL to publish with the default connection endpoint
	connectURL string

	// tdPath holds the path the directory TD can be reached at (well known)
	tdPath string

	// The directory TD document in JSON for serving on the path publish in exploration.
	dirTD string
	// service discovery using mDNS
	dnssdServer *zeroconf.Server
	// the http server that servers the exploration endpoint
	httpTransport *httpserver.HttpTransportServer
}

// AddTDForms does not apply to the discovery service
//func (svc *DiscoveryServer) AddTDForms(*td.TD) {
//	// nothing to do here
//}

// HandleGetDirTD returns the requested the TD Directory
// This provides a WoT discovery exploration mechanism
func (svc *DiscoveryServer) HandleGetDirTD(w http.ResponseWriter, r *http.Request) {
	svc.httpTransport.WriteReply(w, svc.dirTD, transports.StatusCompleted, nil)
}

func (svc *DiscoveryServer) Stop() {
	if svc.dnssdServer != nil {
		svc.dnssdServer.Shutdown()
	}
}

// ServeDirectoryTD registers the handler for serving the Directory TD
func (svc *DiscoveryServer) ServeDirectoryTD(tdPath string, tdJSON string) {

	svc.dirTD = tdJSON
	svc.httpTransport.AddOps(nil, []string{}, http.MethodGet,
		tdPath, svc.HandleGetDirTD)

}

// StartDiscoveryServer starts a DNS-SD server for serving a Thing Directory over https.
//
// connectURL is a hiveot addon that provides the default connection address
// without the need to use forms. This is intended for connection oriented protocols
// such as websocket, sse-sc, mqtt, and others. The schema identifies the protocol.
// Intended for simplify use when all affordances use the same protocol.
// This is included in the 'base' param of the discovery record.
//
// This sets the authURL to that of the http transport
//
//	instanceName to use with DNS-SD. "" is default "hiveot"
//	serviceName to use with DNS-SD . "" is default of "wot"
//	dirTD is the directory TD in json (required)
//	tdPath is the path on which to server the directory TD (Default is /.well-known/wot)
//	httpServer is the server to use for serving the TD
//	connectURL is used to determine the server default connection address
func StartDiscoveryServer(instanceName string, serviceName string,
	dirTD string, tdPath string, httpTransport *httpserver.HttpTransportServer,
	connectURL string) (
	*DiscoveryServer, error) {

	svc := &DiscoveryServer{
		dirTD:         dirTD,
		tdPath:        tdPath,
		connectURL:    connectURL,
		dnssdServer:   nil,
		httpTransport: httpTransport,
	}

	// serve the Directory TD exploration endpoint
	svc.ServeDirectoryTD(tdPath, dirTD)

	// serve the introduction mechanism
	tddURL, _ := url.JoinPath(httpTransport.GetConnectURL(), tdPath)
	authURL := httpTransport.GetAuthURL()
	dnssdServer, err := ServeTDDiscovery(instanceName, serviceName, tddURL, connectURL, authURL)
	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"ServerURL", tddURL,
			"connectURL", connectURL,
			"err", err.Error())
	}
	svc.dnssdServer = dnssdServer

	return svc, err
}
