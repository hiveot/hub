package servers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers/discoserver"
	"github.com/hiveot/hub/messaging/servers/hiveotsseserver"
	"github.com/hiveot/hub/messaging/servers/httpserver"
	"github.com/hiveot/hub/messaging/servers/wssserver"
	"github.com/hiveot/hub/wot/td"
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
	servers map[string]messaging.ITransportServer

	// Registered handler for processing received notifications
	notificationHandler messaging.NotificationHandler
	// Registered handler for processing received requests
	requestHandler messaging.RequestHandler
	// Registered handler for processing received responses
	responseHandler messaging.ResponseHandler
	// serve TDD discovery, if enabled. This needs a handler for TDD requests
	discoServer *discoserver.DiscoveryServer

	// PreferredProtocolType to publish in discovery
	PreferredProtocolType string
}

// AddTDForms adds forms for all active transports
func (svc *TransportManager) AddTDForms(tdoc *td.TD, includeAffordances bool) {

	for _, srv := range svc.servers {
		srv.AddTDForms(tdoc, includeAffordances)
	}
	// MQTT
	//if svc.mqttTransport != nil {
	//	svc.mqttTransport.AddTDForms(tdoc,includeAffordances)
	//}

	// CoAP ?

	// http protocol server including scheme includes subprotocols
	if svc.httpsTransport != nil {
		svc.httpsTransport.AddTDForms(tdoc, includeAffordances)
		svc.httpsTransport.AddSecurityScheme(tdoc)
	}
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
func (svc *TransportManager) GetConnectURL() (connectURL string) {
	protocolType := svc.PreferredProtocolType
	srv, found := svc.servers[protocolType]
	if found {
		return srv.GetConnectURL()
	}
	return ""
}

// GetConnectionByClientID returns the first connection belonging to the given clientID.
// Intended to send requests to an agent which only have a single connection.
// If a protocol isn't available the default https url is returned
func (svc *TransportManager) GetConnectionByClientID(clientID string) messaging.IConnection {
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
func (svc *TransportManager) GetConnectionByConnectionID(clientID, cid string) messaging.IConnection {
	for _, srv := range svc.servers {
		c := srv.GetConnectionByConnectionID(clientID, cid)
		if c != nil {
			return c
		}
	}
	return nil
}

// GetHiveotEndpoints return available endpoints
func (svc *TransportManager) GetHiveotEndpoints() map[string]string {
	endpoints := make(map[string]string)
	for _, s := range svc.servers {
		connectURL := s.GetConnectURL()
		parts, _ := url.Parse(connectURL)
		endpoints[parts.Scheme] = connectURL
	}
	return endpoints
}

// GetProtocolType returns the preferred protocol
func (svc *TransportManager) GetProtocolType() string {
	pref := svc.PreferredProtocolType
	s := svc.servers[pref]
	if s == nil {
		return ""
	}
	return s.GetProtocolType()
}

// GetServer returns the server for the given protocol type
// This returns nil if the protocol is not active.
func (svc *TransportManager) GetServer(protocolType string) messaging.ITransportServer {
	s := svc.servers[protocolType]
	return s
}

// Pass incoming notifications from any of the transport protocols to the registered handler
func (svc *TransportManager) handleNotification(notif *messaging.NotificationMessage) {
	if svc.notificationHandler != nil {
		svc.notificationHandler(notif)
	}
}

// Pass incoming requests from any of the transport protocols to the registered handler
func (svc *TransportManager) handleRequest(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
	if svc.requestHandler != nil {
		return svc.requestHandler(req, c)
	}
	return req.CreateResponse(nil, errors.New("No request handler set"))
}

// Pass incoming responses from any of the transport protocols to the registered handler
func (svc *TransportManager) handleResponse(resp *messaging.ResponseMessage) error {
	if svc.responseHandler != nil {
		return svc.responseHandler(resp)
	}
	return errors.New("No response handler set")
}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *TransportManager) SendNotification(notification *messaging.NotificationMessage) {
	// pass it to protocol servers to use their way of sending messages to subscribers
	// CloseAllClientConnections close all connections from the given client.
	// Intended to close connections after a logout.
	for _, srv := range svc.servers {
		srv.SendNotification(notification)
	}
}

func (svc *TransportManager) SetNotificationHandler(h messaging.NotificationHandler) {
	svc.notificationHandler = h
}
func (svc *TransportManager) SetRequestHandler(h messaging.RequestHandler) {
	svc.requestHandler = h
}
func (svc *TransportManager) SetResponseHandler(h messaging.ResponseHandler) {
	svc.responseHandler = h
}

// StartDiscovery starts the introduction and exploration discovery of the directory
// TD document.
//
//	instanceName alternate service instance, "" is hiveot. Intended for testing.
//
// This serves two discovery mechanisms.
// 1. The WoT discovery through the directory TD and Thing forms
// 2. The HiveOT discovery through the use of connection URLs and Request/Response
// message envelopes without the need for forms.
func (svc *TransportManager) StartDiscovery(instanceName string, tdPath string, dirTD string) (err error) {
	if svc.discoServer != nil {
		err = fmt.Errorf("StartDiscovery: already running")
		slog.Error(err.Error())
		return err
	}
	// Get a list of hiveot endpoints for secondary discovery
	endpoints := svc.GetHiveotEndpoints()

	// start directory introduction and exploration discovery server
	svc.discoServer, err = discoserver.StartDiscoveryServer(
		instanceName, "", dirTD, tdPath,
		svc.httpsTransport, endpoints)
	return err
}

// Stop the protocol servers
func (svc *TransportManager) Stop() {
	if svc.discoServer != nil {
		svc.discoServer.Stop()
		svc.discoServer = nil
	}
	for _, srv := range svc.servers {
		srv.Stop()
	}
	svc.servers = make(map[string]messaging.ITransportServer)

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
// The http-basic binding provides the services for authentication and discovery.
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
	authenticator messaging.IAuthenticator,
) (svc *TransportManager, err error) {

	svc = &TransportManager{
		servers: make(map[string]messaging.ITransportServer),
		//PreferredProtocolType: transports.ProtocolTypeWotHTTPBasic,
		//dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	// Http is needed for all subprotocols and for the auth endpoint
	if cfg.EnableHiveotAuth || cfg.EnableWSS || cfg.EnableHiveotSSE {

		svc.PreferredProtocolType = messaging.ProtocolTypeWotHTTPBasic

		// 1. HTTP server supports authentication
		httpServer, err2 := httpserver.StartHttpTransportServer(
			cfg.HttpHost, cfg.HttpsPort,
			serverCert, caCert,
			authenticator,
		)
		err = err2
		svc.httpsTransport = httpServer

		// 2. HiveOT HTTP/SSE-SC sub-protocol
		if cfg.EnableHiveotSSE {
			ssePath := hiveotsseserver.DefaultHiveotSsePath
			hiveotSseServer := hiveotsseserver.StartHiveotSseServer(
				ssePath,
				svc.httpsTransport,
				nil,
				svc.handleNotification,
				svc.handleRequest,
				svc.handleResponse)
			svc.servers[messaging.ProtocolTypeHiveotSSE] = hiveotSseServer
			// sse is better than http-basic
			svc.PreferredProtocolType = messaging.ProtocolTypeHiveotSSE
		}

		// 3. WSS protocol
		if cfg.EnableWSS {
			converter := &wssserver.HiveotMessageConverter{}
			wssPath := wssserver.DefaultWssPath
			hiveotWssServer, err := wssserver.StartWssServer(
				wssPath, converter, messaging.ProtocolTypeWSS,
				svc.httpsTransport,
				nil,
				svc.handleNotification,
				svc.handleRequest,
				svc.handleResponse,
			)
			if err == nil {
				svc.servers[messaging.ProtocolTypeWSS] = hiveotWssServer
			}
			// wss is better than http, sse or wot wss
			svc.PreferredProtocolType = messaging.ProtocolTypeWSS
		}

		//// 4. WoT WSS Protocol
		//if cfg.EnableWotWSS42 {
		//	// WoT WSS uses the same wss socket server as hiveot but with a
		//	// different message converter.
		//	converter := &wssserver.WoTWss42MessageConverter{}
		//	wssPath := wssserver.DefaultWotWss42Path
		//	wotWssServer, err := wssserver.StartWssServer(
		//		wssPath, converter, messaging.ProtocolTypeWotWSS,
		//		svc.httpsTransport,
		//		nil,
		//		svc.handleNotification,
		//		svc.handleRequest,
		//		svc.handleResponse,
		//	)
		//	if err == nil {
		//		svc.servers[messaging.ProtocolTypeWotWSS] = wotWssServer
		//	}
		//	// WoT wss is better than http or sse
		//	svc.PreferredProtocolType = messaging.ProtocolTypeWotWSS
		//}

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
	return svc, err
}
