package servers

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/servers/discotransport"
	"github.com/hiveot/hub/transports/servers/hiveotserver"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/servers/mqttserver"
	"github.com/hiveot/hub/transports/servers/ssescserver"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// TransportManager aggregates multiple transport protocol servers and manages
// the connection and session management.
//
// This implements the ITransportBinding interface like the protocols it manages.
// Incoming messages without an ID are assigned a new correlationID
type TransportManager struct {
	// protocol transport bindings for events, actions and rpc requests
	// The embedded binding can be used directly with embedded services
	discoveryTransport *discotransport.DiscoveryTransport
	httpsTransport     *httpserver.HttpTransportServer
	ssescTransport     transports.ITransportServer
	wssTransport       transports.ITransportServer

	mqttsTransport transports.ITransportServer
	//dtwService *service.DigitwinService

	// handler to pass incoming messages to
	//handler func(tv *transports.IConsumer) hubclient.ActionStatus
	cm *connections.ConnectionManager
}

// AddTDForms adds forms for all active transports
func (svc *TransportManager) AddTDForms(td *td.TD) (err error) {
	if svc.httpsTransport != nil {
		err = svc.httpsTransport.AddTDForms(td)
	}
	//if svc.mqttsTransport != nil {
	//	svc.mqttsTransport.AddTDForms(td)
	//}
	return err
}

// GetForm returns the form for an operation using a transport protocol binding.
// If the protocol is not found this returns a nil and might cause a panic
func (svc *TransportManager) GetForm(op string, protocol string) (form td.Form) {
	switch protocol {
	case transports.ProtocolTypeHTTPS, transports.ProtocolTypeSSESC:
		form = svc.httpsTransport.GetForm(op, "", "")
	case transports.ProtocolTypeWSS:
		form = svc.wssTransport.GetForm(op, "", "")
		//case transports.ProtocolTypeMQTTCP:
		//	form = svc.mqttTcpTransport.GetForm(op, "", "")
		//case transports.ProtocolTypeMQTTWSS:
		//	form = svc.mqttWssTransport.GetForm(op, "", "")
	}
	return form
}

// GetConnectURL returns URL of the protocol
// If a protocol isn't available the default https url is returned
func (svc *TransportManager) GetConnectURL(protocolType string) (connectURL string) {
	if protocolType == transports.ProtocolTypeWSS && svc.wssTransport != nil {
		connectURL = svc.wssTransport.GetConnectURL()
	} else if protocolType == transports.ProtocolTypeSSESC && svc.ssescTransport != nil {
		connectURL = svc.ssescTransport.GetConnectURL()
	} else if protocolType == transports.ProtocolTypeMQTTCP && svc.mqttsTransport != nil {
		//connectURL = svc.mqttTcpTransport.GetConnectURL()
	} else if protocolType == transports.ProtocolTypeMQTTWSS && svc.mqttsTransport != nil {
		//connectURL = svc.mqttWssTransport.GetConnectURL()
	} else {
		connectURL = svc.httpsTransport.GetConnectURL()
	}
	return connectURL
}

// GetProtocolInfo returns information on the default protocol
//func (svc *TransportManager) GetProtocolInfo() (pi transports.ProtocolInfo) {
//	if svc.httpsTransport != nil {
//		return svc.httpsTransport.GetProtocolInfo()
//	}
//	return
//}

// Stop the protocol servers
func (svc *TransportManager) Stop() {
	if svc.discoveryTransport != nil {
		svc.discoveryTransport.Stop()
	}
	if svc.httpsTransport != nil {
		svc.httpsTransport.Stop()
	}
	//if svc.mqttsTransport != nil {
	//	svc.mqttsTransport.Stop()
	//}
	//if svc.embeddedTransport != nil {
	//	svc.embeddedTransport.Stop()
	//}
	slog.Info("Runtime transport stopped")

}

// StartProtocolManager starts a new instance of the protocol manager.
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
//
// The transport manager implements the ITransportBinding API.
func StartProtocolManager(cfg *ProtocolsConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator transports.IAuthenticator,
	handleNotification transports.ServerNotificationHandler,
	handleRequest transports.ServerRequestHandler,
	handleResponse transports.ServerResponseHandler,
	cm *connections.ConnectionManager,
) (svc *TransportManager, err error) {

	svc = &TransportManager{
		//dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	if cfg.EnableHTTPS {
		httpServer, err2 := httpserver.StartHttpTransportServer(
			cfg.HttpHost, cfg.HttpsPort,
			serverCert, caCert,
			authenticator,
			cm,
			handleNotification,
			handleRequest,
			handleResponse,
		)
		err = err2
		svc.httpsTransport = httpServer
		// http subprotocols
		if cfg.EnableSSESC {
			svc.ssescTransport = ssescserver.StartSseScTransportServer(
				"",
				cm, svc.httpsTransport)
			// support for hiveot protocol using http
			hiveotserver.StartHiveotProtocolServer(
				authenticator,
				cm,
				httpServer,
				handleNotification,
				handleRequest,
				handleResponse,
			)
		}
		if cfg.EnableWSS {
			svc.wssTransport = wssserver.StartWssTransportServer(
				"", cm,
				svc.httpsTransport,
				handleNotification,
				handleRequest,
				handleResponse,
			)
		}
	}
	if cfg.EnableMQTT {
		svc.mqttsTransport, err = mqttserver.StartMqttTransportServer(
			cfg.MqttHost, cfg.MqttTcpPort, cfg.MqttWssPort,
			serverCert, caCert,
			authenticator, cm,
			handleNotification,
			handleRequest,
			handleResponse,
		)
	}
	// FIXME: how to support multiple URLs in discovery. See the WoT discovery spec.
	if cfg.EnableDiscovery {
		cfg.Discovery.ServerAddr = cfg.HttpHost
		cfg.Discovery.ServerPort = cfg.HttpsPort
		cfg.Discovery.SsescURL = svc.GetConnectURL(transports.ProtocolTypeSSESC)
		cfg.Discovery.WssURL = svc.GetConnectURL(transports.ProtocolTypeWSS)
		cfg.Discovery.MqttWssURL = svc.GetConnectURL(transports.ProtocolTypeMQTTWSS)
		cfg.Discovery.MqttTcpURL = svc.GetConnectURL(transports.ProtocolTypeMQTTCP)

		svc.discoveryTransport, err = discotransport.StartDiscoveryTransport(cfg.Discovery)
	}
	return svc, err
}
