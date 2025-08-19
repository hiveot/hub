package discoserver

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hub/messaging/servers/httpbasic"
)

// DiscoveryServer supports the WoT discovery introduction and exploration
// mechanisms for obtaining TD's for use by consumers.
type DiscoveryServer struct {

	//connectURL to publish with the default connection endpoint
	//endpoints map[string]string

	// tdPath holds the path the directory TD can be reached at (well known)
	tdPath string

	// The directory TD document in JSON for serving on the path publish in exploration.
	dirTDJSON string
	// service discovery using mDNS
	dnssdServer *zeroconf.Server
	// the http server that servers the exploration endpoint
	httpTransport *httpbasic.HttpBasicServer
}

// AddTDForms does not apply to the discovery service
//func (svc *DiscoveryServer) AddTDForms(*td.TD, includeAffordances bool) {
//	// nothing to do here
//}

// HandleGetDirTD returns the requested the TD Directory
// This provides a WoT discovery exploration mechanism
func (svc *DiscoveryServer) HandleGetDirTD(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(svc.dirTDJSON))
}

func (svc *DiscoveryServer) Stop() {
	if svc.dnssdServer != nil {
		svc.dnssdServer.Shutdown()
	}
}

// ServeDirectoryTD registers the handler for serving the Directory TD
func (svc *DiscoveryServer) ServeDirectoryTD(tdPath string, tdJSON string) {

	svc.dirTDJSON = tdJSON
	//svc.httpTransport.AddOps(nil, []string{}, http.MethodGet,
	//	tdPath, svc.HandleGetDirTD)
	r := svc.httpTransport.GetPublicRouter()
	r.Get(tdPath, svc.HandleGetDirTD)
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
//	endpoints with hiveot protocol connection scheme:URL endpoints
func StartDiscoveryServer(instanceName string, serviceName string,
	dirTDJSON string, tdPath string, httpTransport *httpbasic.HttpBasicServer,
	endpoints map[string]string) (
	*DiscoveryServer, error) {

	svc := &DiscoveryServer{
		dirTDJSON:     dirTDJSON,
		tdPath:        tdPath,
		dnssdServer:   nil,
		httpTransport: httpTransport,
	}

	// serve the Directory TD exploration endpoint
	svc.ServeDirectoryTD(tdPath, dirTDJSON)

	// serve the introduction mechanism
	tddURL, _ := url.JoinPath(httpTransport.GetConnectURL(), tdPath)
	dnssdServer, err := ServeTDDiscovery(
		instanceName, serviceName, tddURL, endpoints)
	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"HubURL", tddURL,
			"tdPath", tdPath,
			"err", err.Error())
	}
	svc.dnssdServer = dnssdServer

	return svc, err
}
