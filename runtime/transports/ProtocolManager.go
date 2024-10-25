package transports

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	sessions2 "github.com/hiveot/hub/runtime/authn/sessions"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/hubrouter"
	"github.com/hiveot/hub/runtime/transports/discotransport"
	"github.com/hiveot/hub/runtime/transports/httptransport"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// ProtocolManager aggregates multiple transport protocol bindings and manages
// the connection and session management.
//
// This implements the ITransportBinding interface like the protocols it manages.
// Incoming messages without an ID are assigned a new messageID
type ProtocolManager struct {
	// protocol transport bindings for events, actions and rpc requests
	// The embedded binding can be used directly with embedded services
	discoveryTransport *discotransport.DiscoveryTransport
	//embeddedTransport  *embedded.EmbeddedTransport
	httpTransport *httptransport.HttpBinding
	mqttTransport api.ITransportBinding
	//grpcTransport     api.ITransportBinding
	//dtwService *service.DigitwinService

	// handler to pass incoming messages to
	handler func(tv *hubclient.ThingMessage) hubclient.ActionProgress

	sm *sessions2.SessionManager
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

// GetEmbedded returns the embedded transport protocol
// Intended to receive messages from, and send to, embedded services
//func (svc *ProtocolManager) GetEmbedded() *embedded.EmbeddedTransport {
//	return svc.embeddedTransport
//}

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
func (svc *ProtocolManager) GetProtocolInfo() (pi api.ProtocolInfo) {
	if svc.httpTransport != nil {
		return svc.httpTransport.GetProtocolInfo()
	}
	return
}

//
//// GetConnectionByCID returns the client connection for sending messages to a client
//func (svc *ProtocolManager) GetConnectionByCID(cid string) sessions.IClientConnection {
//	return svc.cm.GetConnectionByCID(cid)
//}

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

//func (svc *ProtocolManager) WriteProperty(
//	agentID string, thingID string, name string, value any, messageID string, senderID string) (
//	found bool, status string, err error) {
//
//	// send the action to the sub-protocol bindings until there is a match
//	//if svc.embeddedTransport != nil {
//	//	 status,err = svc.embeddedTransport.WriteProperty(agentID, tThingID, name, value, messageID)
//	//}
//	if svc.httpTransport != nil {
//		found, status, err = svc.httpTransport.WriteProperty(
//			agentID, thingID, name, value, messageID, senderID)
//	}
//	if !found && svc.mqttTransport != nil {
//		//	status,err = svc.mqttTransport.WriteProperty(agentID, thingID, name, value, messageID)
//	}
//	return found, status, err
//}

// StartProtocolManager starts a new instance of the protocol manager.
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
//
// The transport manager implements the ITransportBinding API.
func StartProtocolManager(cfg *ProtocolsConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator api.IAuthenticator,
	hubRouter *hubrouter.HubRouter,
	dtwService *service.DigitwinService,
	cm *connections.ConnectionManager,
	sm *sessions2.SessionManager,
) (svc *ProtocolManager, err error) {

	svc = &ProtocolManager{
		//dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	if cfg.EnableHTTPS {
		svc.httpTransport, err = httptransport.StartHttpTransport(
			&cfg.HttpsTransport,
			serverCert, caCert,
			authenticator, hubRouter,
			cm)
	}
	if cfg.EnableMQTT {
		//svc.mqttTransport = mqtttransport.StartMqttTransport(
		//	&cfg.MqttTransport,
		//	privKey, serverCert, caCert,
		//	sessionAuth,cm)
	}
	if cfg.EnableDiscovery {
		serverURL := svc.GetConnectURL()
		svc.discoveryTransport = discotransport.StartDiscoveryTransport(
			cfg.Discovery, serverURL)
	}
	return svc, err
}
