package servers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers/discoserver"
	"github.com/hiveot/hub/messaging/servers/hiveotsseserver"
	"github.com/hiveot/hub/messaging/servers/httpbasic"
	"github.com/hiveot/hub/messaging/servers/wssserver"
	"github.com/hiveot/hub/messaging/tputils/net"
	"github.com/hiveot/hub/messaging/tputils/tlsserver"
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

	// authenticator for validating incoming connections and to set the security scheme in TDs.
	// currently a single authenticator is used.
	// Maybe this should be dependent on the server?
	authenticator messaging.IAuthenticator

	//http server
	httpServer *tlsserver.TLSServer
	httpRouter *chi.Mux

	// http-basic server required for discovery and auth
	httpBasicServer *httpbasic.HttpBasicServer

	// transport protocol bindings in order of startup
	servers []messaging.ITransportServer
	// transport protocol bindings by protocol ID
	serversByProtocol map[string]messaging.ITransportServer

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

// AddTDForms adds forms to the given TD for all available transports
// This also adds the security scheme as supported by the authenticator.
func (svc *TransportManager) AddTDForms(tdoc *td.TD, includeAffordances bool) {

	svc.authenticator.AddSecurityScheme(tdoc)
	for _, srv := range svc.servers {
		srv.AddTDForms(tdoc, includeAffordances)
	}
	// MQTT
	//if svc.mqttTransport != nil {
	//	svc.mqttTransport.AddTDForms(tdoc,includeAffordances)
	//}

	// CoAP ?
}

// CloseAll closes all client connections
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

// GetConnectURL returns URL of the protocol.
// If protocolType is empty then the 'preferred' protocol type is used.
// This returns an empty URL if the protocol is not supported.
func (svc *TransportManager) GetConnectURL() (connectURL string) {
	protocolType := svc.PreferredProtocolType
	srv, found := svc.serversByProtocol[protocolType]
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
	s := svc.serversByProtocol[pref]
	if s == nil {
		return ""
	}
	return s.GetProtocolType()
}

// GetServer returns the server for the given protocol type
// This returns nil if the protocol is not active.
func (svc *TransportManager) GetServer(protocolType string) messaging.ITransportServer {
	s := svc.serversByProtocol[protocolType]
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
	slog.Error("Received request but no request handler is set in the TransportManager",
		"senderID", req.SenderID, "operation", req.Operation)
	err := errors.New("no request handler set")
	return req.CreateResponse(nil, err)
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
		svc.httpBasicServer, endpoints)
	return err
}

// Stop the protocol servers in reverse order
func (svc *TransportManager) Stop() {
	if svc.discoServer != nil {
		svc.discoServer.Stop()
		svc.discoServer = nil
	}
	for i := len(svc.servers) - 1; i >= 0; i-- {
		srv := svc.servers[i]
		srv.Stop()
	}
	svc.servers = nil
	svc.serversByProtocol = nil
	svc.httpServer.Stop()
	// mqtt
	//if svc.mqttTransport != nil {
	//	svc.mqttTransport.Stop()
	//}
	//if svc.embeddedTransport != nil {
	//	svc.embeddedTransport.Stop()
	//}
	slog.Info("Runtime transport stopped")

}

// Start starts the transport protocol manager and listens for incoming connections and messages.
//
// This returns an error if any server fails to start.
//
// Discovery has to be started separately with StartDiscovery, if desired, and must
// be provided with the Directory TD document to serve.
//
// The transport manager implements the ITransportBinding API.
func (svc *TransportManager) Start() (err error) {
	slog.Info("Start listening for requests")

	err = svc.httpServer.Start()
	if err != nil {
		return err
	}

	// TODO: on error stop all servers that were already started
	for _, srv := range svc.servers {
		err = srv.Start()
		if err != nil {
			return err
		}
	}

	return err
}

// NewTransportManager creates a new instance of the transport protocol manager.
// The transport manager implements the ITransportBinding API.
//
// The authenticator is used to validate connections and set security scheme in AddTDForms.
func NewTransportManager(cfg *ProtocolsConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator messaging.IAuthenticator,
	notifHandler messaging.NotificationHandler,
	reqHandler messaging.RequestHandler,
	respHandler messaging.ResponseHandler,
) (svc *TransportManager) {

	svc = &TransportManager{
		//serverCert:          serverCert,
		//caCert:              caCert,
		authenticator:       authenticator,
		servers:             make([]messaging.ITransportServer, 0),
		serversByProtocol:   make(map[string]messaging.ITransportServer),
		notificationHandler: notifHandler,
		requestHandler:      reqHandler,
		responseHandler:     respHandler,
		//PreferredProtocolType: transports.ProtocolTypeHTTPBasic,
		//dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	// Http-basic is needed for all subprotocols and for the auth endpoint
	if cfg.EnableHiveotAuth || cfg.EnableWSS || cfg.EnableHiveotSSE {

		svc.PreferredProtocolType = messaging.ProtocolTypeHTTPBasic

		// 1.A HTTP server required for http-basic
		// if host is empty then listen on all interfaces
		svc.httpServer, svc.httpRouter = tlsserver.NewTLSServer(
			cfg.HttpHost, cfg.HttpsPort, serverCert, caCert)

		httpAddr := fmt.Sprintf("%s:%d", cfg.HttpHost, cfg.HttpsPort)
		if cfg.HttpHost == "" {
			connectIP := net.GetOutboundIP("")
			httpAddr = fmt.Sprintf("%s:%d", connectIP.String(), cfg.HttpsPort)
		}

		svc.httpBasicServer = httpbasic.NewHttpBasicServer(
			httpAddr, svc.httpRouter,
			authenticator,
			svc.handleNotification,
			svc.handleRequest,
			svc.handleResponse)
		if cfg.EnableHttpStatic {
			svc.httpBasicServer.EnableStatic(cfg.HttpStaticBase, cfg.HttpStaticDirectory)
		}

		// http-basic provides the login method
		svc.authenticator.SetAuthServerURI(svc.httpBasicServer.GetAuthServerURI())

		// FIXME: routes only available after start
		protectedRouter := svc.httpBasicServer.GetProtectedRouter()

		//err = err2
		svc.serversByProtocol[messaging.ProtocolTypeHTTPBasic] = svc.httpBasicServer
		svc.servers = append(svc.servers, svc.httpBasicServer)

		// 2. HiveOT HTTP/SSE-SC sub-protocol
		if cfg.EnableHiveotSSE {
			ssePath := hiveotsseserver.DefaultHiveotSsePath
			if cfg.HiveotSSEPath != "" {
				ssePath = cfg.HiveotSSEPath
			}
			hiveotSseServer := hiveotsseserver.NewHiveotSseServer(
				httpAddr, ssePath, protectedRouter,
				nil,
				svc.handleNotification,
				svc.handleRequest,
				svc.handleResponse)
			svc.serversByProtocol[messaging.ProtocolTypeHiveotSSE] = hiveotSseServer
			svc.servers = append(svc.servers, hiveotSseServer)
			// sse is better than http-basic
			svc.PreferredProtocolType = messaging.ProtocolTypeHiveotSSE
		}

		// 3. WSS protocol
		if cfg.EnableWSS {
			converter := &wssserver.HiveotMessageConverter{}
			wssPath := wssserver.DefaultWssPath
			if cfg.WSSPath != "" {
				wssPath = cfg.WSSPath
			}
			hiveotWssServer := wssserver.NewWssServer(
				httpAddr, wssPath, protectedRouter, converter,
				//svc.httpBasicServer,
				nil,
				svc.handleNotification,
				svc.handleRequest,
				svc.handleResponse,
			)

			svc.serversByProtocol[messaging.ProtocolTypeWSS] = hiveotWssServer
			svc.servers = append(svc.servers, hiveotWssServer)

			// wss is better than http, sse or wot wss
			svc.PreferredProtocolType = messaging.ProtocolTypeWSS
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
	return svc
}
