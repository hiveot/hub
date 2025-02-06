package servers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/discotransport"
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
	discoveryTransport *discotransport.DiscoveryTransport
	// http transport for subprotocols
	httpsTransport *httpserver.HttpTransportServer

	// all transport protocol bindings by protocol ID
	servers map[string]transports.ITransportServer

	// Registered handler for processing received requests
	requestHandler transports.RequestHandler
	// Registered handler for processing received responses
	responseHandler transports.ResponseHandler
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
// This returns an empty URL if the protocol is not supported.
func (svc *TransportManager) GetConnectURL(protocolType string) (connectURL string) {
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

// Stop the protocol servers
func (svc *TransportManager) Stop() {
	if svc.discoveryTransport != nil {
		svc.discoveryTransport.Stop()
	}
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
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
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
		//dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	// Http is needed for all subprotocols because of their auth endpoints
	if cfg.EnableHiveotWSS || cfg.EnableHiveotSSE ||
		cfg.EnableWotHTTPBasic || cfg.EnableWotWSS {

		// 1. HTTP server with login
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
		}

		// 3. HTTP HiveOT WSS protocol
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

		if cfg.EnableWotWSS {
			//	svc.wotWssServer = wssserver_old.StartWotWssServer(
			//		"", cm,
			//		svc.httpsTransport,
			//		handleRequest,
			//		handleResponse,
			//	)
			//svc.servers = append(svc.servers, svc.wotWssServer)
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
	// FIXME: how to support multiple URLs in discovery. See the WoT discovery spec.
	if cfg.EnableDiscovery {
		cfg.Discovery.ServerAddr = cfg.HttpHost
		cfg.Discovery.ServerPort = cfg.HttpsPort
		cfg.Discovery.HiveotSseURL = svc.GetConnectURL(transports.ProtocolTypeHiveotSSE)
		cfg.Discovery.HiveotWssURL = svc.GetConnectURL(transports.ProtocolTypeHiveotWSS)
		cfg.Discovery.WotHttpBasicURL = svc.GetConnectURL(transports.ProtocolTypeWotHTTPBasic)
		cfg.Discovery.WotWssURL = svc.GetConnectURL(transports.ProtocolTypeWotWSS)
		//cfg.Discovery.MqttWssURL = svc.GetConnectURL(transports.ProtocolTypeMQTTWSS)
		//cfg.Discovery.MqttTcpURL = svc.GetConnectURL(transports.ProtocolTypeMQTTCP)

		svc.discoveryTransport, err = discotransport.StartDiscoveryTransport(cfg.Discovery)
	}
	return svc, err
}
