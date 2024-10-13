package transports

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/hubrouter"
	"github.com/hiveot/hub/runtime/transports/discotransport"
	"github.com/hiveot/hub/runtime/transports/httptransport"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// TransportManager aggregates multiple transport protocol bindings and manages the starting,
// stopping and routing of protocol messages.
// This implements the ITransportBinding interface like the protocols it manages.
// Incoming messages without an ID are assigned a new messageID
type TransportManager struct {
	// protocol transport bindings for events, actions and rpc requests
	// The embedded binding can be used directly with embedded services
	discoveryTransport *discotransport.DiscoveryTransport
	//embeddedTransport  *embedded.EmbeddedTransport
	httpTransport *httptransport.HttpTransport
	mqttTransport api.ITransportBinding
	//natsTransport     api.ITransportBinding
	//grpcTransport     api.ITransportBinding
	dtwService *service.DigitwinService

	// handler to pass incoming messages to
	handler func(tv *hubclient.ThingMessage) hubclient.DeliveryStatus
}

// AddTDForms adds forms for all active transports
func (svc *TransportManager) AddTDForms(td *tdd.TD) (err error) {
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
//func (svc *TransportManager) GetEmbedded() *embedded.EmbeddedTransport {
//	return svc.embeddedTransport
//}

// GetConnectURL returns URL of the first protocol that has a baseurl
func (svc *TransportManager) GetConnectURL() (baseURL string) {
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
func (svc *TransportManager) GetProtocolInfo() (pi api.ProtocolInfo) {
	if svc.httpTransport != nil {
		return svc.httpTransport.GetProtocolInfo()
	}
	return
}

// receive a message and ensure it has a message ID
//func (svc *TransportManager) handleMessage(msg *hubclient.ThingMessage) hubclient.DeliveryStatus {
//	if msg.MessageID == "" {
//		msg.MessageID = shortid.MustGenerate()
//	}
//	stat := svc.handler(msg)
//	// help detect problems with message ID mismatch
//	if stat.MessageID != msg.MessageID {
//		slog.Error("Delivery status has missing messageID",
//			"thingID", msg.ThingID,
//			"messageType", msg.MessageType,
//			"key", msg.Name,
//			"request messageID", msg.MessageID,
//			"status messageID", stat.MessageID,
//			"senderID", msg.SenderID,
//		)
//	}
//	return stat
//}

//// GetProtocols returns a list of active server protocol bindings
//func (svc *TransportManager) GetProtocols() []api.ITransportBinding {
//	return svc.bindings
//}

// SendToClient sends a message to a connected agent or consumer client.
// If an agent is connected through multiple protocols then this stops after the first
// successful delivery. ?
//
// TODO: can the sessionID be used instead of the clientID in case a client has multiple connections?
//
//	Maybe support both sending to clientID and sessionID. Notifications can go to all sessions
//	of a client while response of API requests are go the the session that sent it.
//func (svc *TransportManager) SendToClient(
//	clientID string, msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus, found bool) {
//
//	// for now simply send the action request to enabled protocol handlers
//	if svc.embeddedTransport != nil {
//		stat, found = svc.embeddedTransport.SendToClient(clientID, msg)
//	}
//	if !found && svc.httpTransport != nil {
//		stat, found = svc.httpTransport.SendToClient(clientID, msg)
//	}
//	if !found && svc.mqttTransport != nil {
//		stat, found = svc.mqttTransport.SendToClient(clientID, msg)
//	}
//	if !found {
//		// if no subscribers exist then delivery fails
//		err := fmt.Errorf("TransportManager.SendToClient: Destination '%s' not found", clientID)
//		stat.Failed(msg, err)
//	}
//	return stat, found
//}

// InvokeAction invokes an action on an agent's Thing
func (svc *TransportManager) InvokeAction(
	agentID, thingID string, name string, value any, messageID string, senderID string) (
	found bool, status string, output any, err error) {

	// send the action to the sub-protocol bindings until there is a match
	//if svc.embeddedTransport != nil {
	//	return svc.embeddedTransport.InvokeAction(agentID, tThingID, name, value, messageID)
	//}
	if svc.httpTransport != nil {
		found, status, output, err = svc.httpTransport.InvokeAction(
			agentID, thingID, name, value, messageID, senderID)
	}
	if err != nil && svc.mqttTransport != nil {
		//	svc.mqttTransport.InvokeAction(agentID, thingID, name, value, messageID)
		//}
	}
	return found, status, output, err
}

// PublishEvent sends a event to all subscribers
func (svc *TransportManager) PublishEvent(
	dThingID string, name string, value any, messageID string, agentID string) {

	// simply send the event request to the protocol handlers
	//if svc.embeddedTransport != nil {
	//	svc.embeddedTransport.PublishEvent(dThingID, name, value, messageID)
	//}
	if svc.httpTransport != nil {
		svc.httpTransport.PublishEvent(dThingID, name, value, messageID, agentID)
	}
	//if svc.mqttTransport != nil {
	//	svc.mqttTransport.PublishEvent(dThingID, name, value, messageID)
	//}
}

// PublishProgressUpdate send the action status update to the client.
// This fails if no binding has a connection with this client
func (svc *TransportManager) PublishProgressUpdate(
	clientID string, stat hubclient.DeliveryStatus, agentID string) (found bool, err error) {
	if svc.httpTransport != nil {
		found, err = svc.httpTransport.PublishProgressUpdate(clientID, stat, agentID)
	} else {
		err = fmt.Errorf("PublishProgressUpdate: No connection with consumer '%s'", clientID)
		found = false
	}
	return found, err
}

// PublishProperty passes it on to all property observers
func (svc *TransportManager) PublishProperty(dThingID string, name string, value any, messageID string, agentID string) {
	if svc.httpTransport != nil {
		svc.httpTransport.PublishProperty(dThingID, name, value, messageID, agentID)
	}
}

// Stop the protocol servers
func (svc *TransportManager) Stop() {
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

func (svc *TransportManager) WriteProperty(
	agentID string, thingID string, name string, value any, messageID string, senderID string) (
	found bool, status string, err error) {

	// send the action to the sub-protocol bindings until there is a match
	//if svc.embeddedTransport != nil {
	//	 status,err = svc.embeddedTransport.WriteProperty(agentID, tThingID, name, value, messageID)
	//}
	if svc.httpTransport != nil {
		found, status, err = svc.httpTransport.WriteProperty(
			agentID, thingID, name, value, messageID, senderID)
	}
	if !found && svc.mqttTransport != nil {
		//	status,err = svc.mqttTransport.WriteProperty(agentID, thingID, name, value, messageID)
	}
	return found, status, err
}

// StartTransportManager starts a new instance of the transport manager.
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
//
// The transport manager implements the ITransportBinding API.
func StartTransportManager(cfg *ProtocolsConfig,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator api.IAuthenticator,
	hubRouter *hubrouter.HubRouter,
	dtwService *service.DigitwinService,
) (svc *TransportManager, err error) {

	svc = &TransportManager{
		dtwService: dtwService,
	}
	// the embedded transport protocol is required for the runtime
	// Embedded services are: authn, authz, directory, inbox, outbox services
	//svc.embeddedTransport = embedded.StartEmbeddedBinding()

	if cfg.EnableHTTPS {
		svc.httpTransport, err = httptransport.StartHttpTransport(
			&cfg.HttpsTransport,
			serverCert, caCert,
			authenticator, hubRouter,
			dtwService)
	}
	if cfg.EnableMQTT {
		//svc.mqttTransport = mqtttransport.StartMqttTransport(
		//	&cfg.MqttTransport,
		//	privKey, serverCert, caCert,
		//	sessionAuth)
	}
	if cfg.EnableDiscovery {
		serverURL := svc.GetConnectURL()
		svc.discoveryTransport = discotransport.StartDiscoveryTransport(
			cfg.Discovery, serverURL)
	}
	return svc, err
}
