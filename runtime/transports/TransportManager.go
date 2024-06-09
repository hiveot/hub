package transports

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/discotransport"
	"github.com/hiveot/hub/runtime/transports/embedded"
	"github.com/hiveot/hub/runtime/transports/httpstransport"
	"log/slog"
)

// TransportsManager aggregates multiple protocol bindings and manages the starting,
// stopping and routing of protocol messages.
// This implements the ITransportBinding interface like the protocols it manages.
type TransportsManager struct {
	// protocol transport bindings for events, actions and rpc requests
	// The embedded binding can be used directly with embedded services
	discoveryTransport *discotransport.DiscoveryTransport
	embeddedTransport  *embedded.EmbeddedTransport
	httpsTransport     api.ITransportBinding
	mqttTransport      api.ITransportBinding
	//natsTransport     api.ITransportBinding
	//grpcTransport     api.ITransportBinding

	// handler to pass incoming messages to
	handler func(tv *things.ThingMessage) hubclient.DeliveryStatus
}

// GetEmbedded returns the embedded transport protocol
// Intended to receive messages from, and send to, embedded services
func (svc *TransportsManager) GetEmbedded() *embedded.EmbeddedTransport {
	return svc.embeddedTransport
}

// GetConnectURL returns URL of the first protocol that has a baseurl
func (svc *TransportsManager) GetConnectURL() (baseURL string) {
	// right now only https has a baseurl
	if svc.httpsTransport != nil {
		baseURL = svc.httpsTransport.GetProtocolInfo().BaseURL
	}
	if baseURL == "" && svc.mqttTransport != nil {
		baseURL = svc.httpsTransport.GetProtocolInfo().BaseURL
	}
	return baseURL
}

// GetProtocolInfo returns information on the default protocol
func (svc *TransportsManager) GetProtocolInfo() (pi api.ProtocolInfo) {
	if svc.httpsTransport != nil {
		return svc.httpsTransport.GetProtocolInfo()
	}
	return
}

//// GetProtocols returns a list of active server protocol bindings
//func (svc *TransportsManager) GetProtocols() []api.ITransportBinding {
//	return svc.bindings
//}

// SendToClient sends a message to a connected agent or consumer client
// If an agent is connected through multiple protocols then this stops
// after the first successful delivery.
// TODO: optimize to use the most efficient protocol
// TODO: sending to multiple instances of the same client? (multiple browser tabs?)
func (svc *TransportsManager) SendToClient(
	clientID string, msg *things.ThingMessage) (stat hubclient.DeliveryStatus, found bool) {

	// for now simply send the action request to enabled protocol handlers
	if svc.embeddedTransport != nil {
		stat, found = svc.embeddedTransport.SendToClient(clientID, msg)
	}
	if !found && svc.httpsTransport != nil {
		stat, found = svc.httpsTransport.SendToClient(clientID, msg)
	}
	if !found && svc.mqttTransport != nil {
		stat, found = svc.mqttTransport.SendToClient(clientID, msg)
	}
	if !found {
		// if no subscribers exist then delivery fails
		err := fmt.Errorf("TransportsManager.SendToClient: Destination '%s' not found", clientID)
		stat.Failed(msg, err)
	}
	return stat, found
}

// SendEvent sends a event to all subscribers
// This returns an error if the event had no subscribers
func (svc *TransportsManager) SendEvent(
	msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {

	// delivery fails if there are no subscribers. Does this matter?
	stat.Failed(msg, errors.New("event has no subscribers"))

	// for now simply send the action request to enabled protocol handlers
	if svc.embeddedTransport != nil {
		stat1 := svc.embeddedTransport.SendEvent(msg)
		if stat1.Error == "" {
			stat = stat1
		}
	}
	if svc.httpsTransport != nil {
		stat2 := svc.httpsTransport.SendEvent(msg)
		if stat2.Error == "" {
			stat = stat2
		}
	}
	if svc.mqttTransport != nil {
		stat3 := svc.mqttTransport.SendEvent(msg)
		if stat3.Error == "" {
			stat = stat3
		}
	}
	return stat
}

// Start the protocol servers
func (svc *TransportsManager) Start(handler hubclient.MessageHandler) error {
	svc.handler = handler
	if svc.embeddedTransport != nil {
		err := svc.embeddedTransport.Start(handler)
		if err != nil {
			slog.Error("Embedded transport start error:", "err", err)
		}
	}
	if svc.httpsTransport != nil {
		err := svc.httpsTransport.Start(handler)
		if err != nil {
			slog.Error("HttpSSE transport start error:", "err", err)
		}
	}
	if svc.mqttTransport != nil {
		err := svc.mqttTransport.Start(handler)
		if err != nil {
			slog.Error("MQTT transport start error:", "err", err)
		}
	}
	if svc.discoveryTransport != nil {
		// TODO: support multiple protocols in the discovery record
		serverURL := svc.GetConnectURL()
		err := svc.discoveryTransport.Start(serverURL)
		if err != nil {
			slog.Error("Servuce discovery start error:", "err", err)
		}
	}
	return nil
}

// Stop the protocol servers
func (svc *TransportsManager) Stop() {
	if svc.discoveryTransport != nil {
		svc.discoveryTransport.Stop()
	}
	if svc.embeddedTransport != nil {
		svc.embeddedTransport.Stop()
	}
	if svc.httpsTransport != nil {
		svc.httpsTransport.Stop()
	}
	if svc.mqttTransport != nil {
		svc.mqttTransport.Stop()
	}
}

// NewTransportManager creates a new instance of the protocol manager.
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
func NewTransportManager(cfg *ProtocolsConfig,
	privKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	authenticator api.IAuthenticator) *TransportsManager {

	svc := TransportsManager{
		// the embedded transport protocol is required for the runtime
		// Embedded services are: authn, authz, directory, inbox, outbox services
		embeddedTransport: embedded.NewEmbeddedBinding(),
	}
	if cfg.EnableDiscovery {
		svc.discoveryTransport = discotransport.NewDiscoveryTransport(cfg.Discovery)
	}
	if cfg.EnableHTTPS {
		svc.httpsTransport = httpstransport.NewHttpSSETransport(
			&cfg.HttpsTransport,
			privKey, serverCert, caCert,
			authenticator)
	}
	if cfg.EnableMQTT {
		//svc.mqttTransport = mqtttransport.NewMqttTransport(
		//	&cfg.MqttTransport,
		//	privKey, serverCert, caCert,
		//	sessionAuth)
	}

	return &svc
}

// StartProtocolManager starts a new instance of the protocol manager.
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
func StartProtocolManager(cfg *ProtocolsConfig,
	privKey keys.IHiveKey,
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	authenticator api.IAuthenticator,
	handler hubclient.MessageHandler) (*TransportsManager, error) {

	svc := NewTransportManager(cfg, privKey, serverCert, caCert, authenticator)
	err := svc.Start(handler)

	return svc, err
}
