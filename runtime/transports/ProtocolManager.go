package transports

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/discotransport"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/hiveot/hub/wot/transports/servers/httpserver"
	"github.com/hiveot/hub/wot/transports/servers/mqttserver"
	"github.com/hiveot/hub/wot/transports/servers/ssescserver"
	"github.com/hiveot/hub/wot/transports/servers/wssserver"
	"log/slog"
)

// ProtocolManager aggregates multiple transport protocol bindings and manages
// the connection and session management.
//
// This implements the ITransportBinding interface like the protocols it manages.
// Incoming messages without an ID are assigned a new requestID
type ProtocolManager struct {
	// protocol transport bindings for events, actions and rpc requests
	// The embedded binding can be used directly with embedded services
	discoveryTransport *discotransport.DiscoveryTransport
	httpTransport      *httpserver.HttpTransportServer
	ssescTransport     transports.ITransportServer
	wssTransport       transports.ITransportServer

	mqttTransport transports.ITransportServer
	//dtwService *service.DigitwinService

	// handler to pass incoming messages to
	//handler func(tv *transports.IConsumer) hubclient.ActionStatus
	cm *connections.ConnectionManager
}

// AddTDForms adds forms for all active transports
func (svc *ProtocolManager) AddTDForms(td *tdd.TD) (err error) {
	if svc.httpTransport != nil {
		err = svc.httpTransport.AddTDForms(td)
	}
	//if svc.mqttTransport != nil {
	//	svc.mqttTransport.AddTDForms(td)
	//}
	return err
}

// GetForm returns the form for an operation using a transport protocol binding.
// If the protocol is not found this returns a nil and might cause a panic
func (svc *ProtocolManager) GetForm(op string, protocol string) (form tdd.Form) {
	switch protocol {
	case transports.ProtocolTypeHTTP, transports.ProtocolTypeSSESC, transports.ProtocolTypeSSE:
		form = svc.httpTransport.GetForm(op)
	case transports.ProtocolTypeWSS:
		form = svc.wssTransport.GetForm(op)
	case transports.ProtocolTypeMQTT:
		form = svc.mqttTransport.GetForm(op)
	}
	return form
}

// GetConnectURL returns URL of the first protocol that has a baseurl
func (svc *ProtocolManager) GetConnectURL() (baseURL string) {
	// right now only https has a baseurl
	if svc.httpTransport != nil {
		baseURL = svc.httpTransport.GetProtocolInfo().BaseURL
	}
	//if baseURL == "" && svc.mqttTransport != nil {
	//	baseURL = svc.mqttTransport.GetProtocolInfo().BaseURL
	//}
	return baseURL
}

// GetProtocolInfo returns information on the default protocol
func (svc *ProtocolManager) GetProtocolInfo() (pi transports.ProtocolInfo) {
	if svc.httpTransport != nil {
		return svc.httpTransport.GetProtocolInfo()
	}
	return
}

// Stop the protocol servers
func (svc *ProtocolManager) Stop() {
	if svc.discoveryTransport != nil {
		svc.discoveryTransport.Stop()
	}
	if svc.httpTransport != nil {
		svc.httpTransport.Stop()
	}
	//if svc.mqttTransport != nil {
	//	svc.mqttTransport.Stop()
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
	digitwinRouter api.IDigitwinRouter,
	cm *connections.ConnectionManager,
) (svc *ProtocolManager, err error) {

	svc = &ProtocolManager{
		//dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	if cfg.EnableHTTPS {
		svc.httpTransport, err = httpserver.StartHttpTransportServer(
			cfg.HttpHost, cfg.HttpsPort,
			serverCert, caCert,
			authenticator, digitwinRouter.HandleMessage,
			cm)
		// http subprotocols
		if cfg.EnableSSESC {
			svc.ssescTransport = ssescserver.StartSseScTransportServer(
				cfg.HttpSSEPath,
				cm, svc.httpTransport)
		}
		if cfg.EnableWSS {
			svc.wssTransport = wssserver.StartWssTransportServer(
				cfg.HttpWSSPath,
				digitwinRouter.HandleMessage,
				cm, svc.httpTransport)
		}
	}
	if cfg.EnableMQTT {
		svc.mqttTransport, err = mqttserver.StartMqttTransportServer(
			cfg.MqttHost, cfg.MqttTcpPort, cfg.MqttWssPort,
			serverCert, caCert,
			authenticator, digitwinRouter.HandleMessage,
			cm)
	}
	if cfg.EnableDiscovery {
		serverURL := svc.GetConnectURL()
		svc.discoveryTransport = discotransport.StartDiscoveryTransport(
			cfg.Discovery, serverURL)
	}
	return svc, err
}
