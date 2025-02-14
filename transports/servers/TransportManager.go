package servers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/discoserver"
	"github.com/hiveot/hub/transports/servers/hiveotsseserver"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// TransportManager aggregates multiple transport protocol servers and manages
// the connection and session management.
//
// This implements the ITransportBinding interface like the protocols it manages.
//
// Incoming requests and response are passed to the provided handlers.
// To send an asynchronous request or a response use the SendRequest/SendResponse
// methods, or SendNotification to broadcast using the binding's way of handling
// subscriptions.
type TransportManager struct {

	// protocol transport bindings for events, actions and rpc requests
	// The embedded binding can be used directly with embedded services
	//discoveryTransport *discotransport.DiscoveryTransport

	// http transport for subprotocols
	httpsTransport *httpserver.HttpTransportServer

	// all transport protocol bindings by protocol ID
	servers map[string]transports.ITransportServer

	// Registered handler for processing received requests
	requestHandler transports.RequestHandler
	// Registered handler for processing received responses
	responseHandler transports.ResponseHandler
	// serve TDD discovery, if enabled. This needs a handler for TDD requests
	discoServer *discoserver.DiscoveryServer

	// PreferredProtocolType to publish in discovery
	PreferredProtocolType string
}

// AddTDForms adds forms for all active transports
func (svc *TransportManager) AddTDForms(td *td.TD) (err error) {

	for _, srv := range svc.servers {
		err = srv.AddTDForms(td)
	}
	// MQTT
	//if svc.mqttTransport != nil {
	//	err = svc.mqttTransport.AddTDForms(td)
	//}

	// CoAP ?
	return err
}

// CloseAll closes all incoming connections
func (svc *TransportManager) CloseAll() {
	for _, srv := range svc.servers {
		srv.CloseAll()
	}
}

// CloseAllClientConnections close all connections from the given client.
// Intended to close connections after a logout.
func (svc *TransportManager) CloseAllClientConnections(clientID string) {
	for _, srv := range svc.servers {
		srv.CloseAllClientConnections(clientID)
	}
}

// GetForm returns the form for an operation using a transport (sub)protocol binding.
// If the protocol is not found this returns a nil and might cause a panic
func (svc *TransportManager) GetForm(op string, thingID string, name string) (form *td.Form) {
	for _, srv := range svc.servers {
		form = srv.GetForm(op, thingID, name)
		if form != nil {
			return form
		}
	}
	return nil
}

// GetConnectURL returns URL of the protocol.
// If protocolType is empty then the 'preferred' protocol type is used.
// This returns an empty URL if the protocol is not supported.
func (svc *TransportManager) GetConnectURL(protocolType string) (connectURL string) {
	if protocolType == "" {
		protocolType = svc.PreferredProtocolType
	}

	srv, found := svc.servers[protocolType]
	if found {
		return srv.GetConnectURL(protocolType)
	}
	return ""
}

// GetConnectionByClientID returns the first connection belonging to the given clientID.
// Intended to send requests to an agent which only have a single connection.
// If a protocol isn't available the default https url is returned
func (svc *TransportManager) GetConnectionByClientID(clientID string) transports.IConnection {
	for _, srv := range svc.servers {
		c := srv.GetConnectionByClientID(clientID)
		if c != nil {
			return c
		}
	}
	return nil
}

// GetConnectionByConnectionID returns the connection of the given ID
// If a protocol isn't available the default https url is returned
func (svc *TransportManager) GetConnectionByConnectionID(clientID, cid string) transports.IConnection {
	for _, srv := range svc.servers {
		c := srv.GetConnectionByConnectionID(clientID, cid)
		if c != nil {
			return c
		}
	}
	return nil
}

// GetServer returns the server for the given protocol type
// This returns nil if the protocol is not active.
func (svc *TransportManager) GetServer(protocolType string) transports.ITransportServer {
	s := svc.servers[protocolType]
	return s
}

func (svc *TransportManager) handleRequest(
	req *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {
	if svc.requestHandler != nil {
		return svc.requestHandler(req, c)
	}
	return req.CreateResponse(nil, errors.New("No request handler set"))
}

func (svc *TransportManager) handleResponse(resp *transports.ResponseMessage) error {
	if svc.responseHandler != nil {
		return svc.responseHandler(resp)
	}
	return errors.New("No response handler set")
}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *TransportManager) SendNotification(notification *transports.ResponseMessage) {
	// pass it to protocol servers to use their way of sending messages to subscribers
	// CloseAllClientConnections close all connections from the given client.
	// Intended to close connections after a logout.
	for _, srv := range svc.servers {
		srv.SendNotification(notification)
	}
}
func (svc *TransportManager) SetRequestHandler(h transports.RequestHandler) {
	svc.requestHandler = h
}
func (svc *TransportManager) SetResponseHandler(h transports.ResponseHandler) {
	svc.responseHandler = h
}

// StartDiscovery starts the introduction and exploration discovery of the directory
// TD document.
// This is started separately as the TD to be served is created after the transport
// manager has started.
func (svc *TransportManager) StartDiscovery(tdPath string, dirTD string) (err error) {
	if svc.discoServer != nil {
		err = fmt.Errorf("StartDiscovery: already running")
		slog.Error(err.Error())
		return err
	}
	// Use the server preferred protocol type for hiveot clients
	connectURL := svc.GetConnectURL(svc.PreferredProtocolType)
	// start directory introduction and exploration discovery server
	svc.discoServer, err = discoserver.StartDiscoveryServer(
		"", "", dirTD, tdPath, svc.httpsTransport, connectURL)
	return err
}

// Stop the protocol servers
func (svc *TransportManager) Stop() {
	//if svc.discoveryTransport != nil {
	//	svc.discoveryTransport.Stop()
	//}
	for _, srv := range svc.servers {
		srv.Stop()
	}
	svc.servers = make(map[string]transports.ITransportServer)

	// http transport used by all subprotocols
	if svc.httpsTransport != nil {
		svc.httpsTransport.Stop()
	}
	// mqtt
	//if svc.mqttTransport != nil {
	//	svc.mqttTransport.Stop()
	//}
	//if svc.embeddedTransport != nil {
	//	svc.embeddedTransport.Stop()
	//}
	slog.Info("Runtime transport stopped")

}

// StartTransportManager starts a new instance of the transport protocol manager.
// This instantiates and starts enabled protocol bindings.
//
// The http-basic binding is provides the services for authentication and discovery.
//
// Discovery has to be started separately with StartDiscovery, if desired, and must
// be provided with the Directory TD document to serve.
//
// The transport manager implements the ITransportBinding API.
// Use SetRequestHandler and SetResponseHandler to setup the receivers of incoming
// requests and responses
func StartTransportManager(cfg *ProtocolsConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator transports.IAuthenticator,
	// handleRequest transports.RequestHandler,
	// handleResponse transports.ResponseHandler,
) (svc *TransportManager, err error) {

	svc = &TransportManager{
		servers: make(map[string]transports.ITransportServer),
		//PreferredProtocolType: transports.ProtocolTypeWotHTTPBasic,
		//dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	// Http is needed for all subprotocols because of their auth endpoints
	if cfg.EnableHiveotWSS || cfg.EnableHiveotSSE ||
		cfg.EnableWotHTTPBasic || cfg.EnableWotWSS {

		svc.PreferredProtocolType = transports.ProtocolTypeWotHTTPBasic

		// 1. HTTP server supports authentication and discovery
		httpServer, err2 := httpserver.StartHttpTransportServer(
			cfg.HttpHost, cfg.HttpsPort,
			serverCert, caCert,
			authenticator,
		)
		err = err2
		svc.httpsTransport = httpServer

		// 2. HTTP HiveOT SSE-SC sub-protocol
		if cfg.EnableHiveotSSE {
			ssePath := hiveotsseserver.DefaultHiveotSsePath
			hiveotSseServer := hiveotsseserver.StartHiveotSseServer(ssePath,
				svc.httpsTransport, nil, svc.handleRequest, svc.handleResponse)
			svc.servers[transports.ProtocolTypeHiveotSSE] = hiveotSseServer
			// sse is better than http-basic
			svc.PreferredProtocolType = transports.ProtocolTypeHiveotSSE
		}

		// 3. WoT WSS Protocol
		if cfg.EnableWotWSS {
			// WoT WSS uses the same wss socket server as hiveot but with a
			// different message converter.
			converter := &wssserver.WotWssMessageConverter{}
			wssPath := wssserver.DefaultWotWssPath
			wotWssServer, err := wssserver.StartHiveotWssServer(
				wssPath, converter, transports.ProtocolTypeWotWSS,
				svc.httpsTransport,
				nil,
				svc.handleRequest,
				svc.handleResponse,
			)
			if err == nil {
				svc.servers[transports.ProtocolTypeWotWSS] = wotWssServer
			}
			// WoT wss is better than http or sse
			svc.PreferredProtocolType = transports.ProtocolTypeWotWSS
		}

		// 4. HiveOT WSS protocol
		if cfg.EnableHiveotWSS {
			converter := &wssserver.HiveotMessageConverter{}
			wssPath := wssserver.DefaultHiveotWssPath
			hiveotWssServer, err := wssserver.StartHiveotWssServer(
				wssPath, converter, transports.ProtocolTypeHiveotWSS,
				svc.httpsTransport,
				nil,
				svc.handleRequest,
				svc.handleResponse,
			)
			if err == nil {
				svc.servers[transports.ProtocolTypeHiveotWSS] = hiveotWssServer
			}
			// wss is better than http, sse or wot wss
			svc.PreferredProtocolType = transports.ProtocolTypeHiveotWSS
		}
		if cfg.EnableWotHTTPBasic {
			//	svc.wotHttpBasicServer = StartWotHttpBasicServer(
			//		"", cm,
			//		svc.httpsTransport,
			//		handleRequest,
			//		handleResponse,
			//	)
			//svc.servers = append(svc.servers, svc.wotHttpBasicServer)
		}
	}
	//if cfg.EnableMQTT {
	//svc.mqttsTransport, err = mqttserver.StartMqttTransportServer(
	//	cfg.MqttHost, cfg.MqttTcpPort, cfg.MqttWssPort,
	//	serverCert, caCert,
	//	authenticator, cm,
	//	handleRequest,
	//	handleResponse,
	//)
	//svc.servers = append(svc.servers, svc.mqttsTransport)
	//}
	if cfg.EnableDiscovery {
		//dirTD := r.digitwin.GetTD()

		// start directory introduction and exploration discovery server
		//svc.discoServer, err = discoserver.StartDiscoveryServer(
		//	"", "", "", dirTD, svc.httpsTransport)

		// start discovery with the TDD endpoint
		//cfg.Discovery.HiveotSseURL = svc.GetConnectURL(transports.ProtocolTypeHiveotSSE)
		//cfg.Discovery.HiveotWssURL = svc.GetConnectURL(transports.ProtocolTypeHiveotWSS)
		//cfg.Discovery.WotHttpBasicURL = svc.GetConnectURL(transports.ProtocolTypeWotHTTPBasic)
		//cfg.Discovery.WotWssURL = svc.GetConnectURL(transports.ProtocolTypeWotWSS)
		//cfg.Discovery.MqttWssURL = svc.GetConnectURL(transports.ProtocolTypeMQTTWSS)
		//cfg.Discovery.MqttTcpURL = svc.GetConnectURL(transports.ProtocolTypeMQTTCP)

		//svc.discoveryTransport, err = discotransport.StartDiscoveryTransport(cfg.Discovery)
	}
	return svc, err
}
