package transports

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/embedded"
	"github.com/hiveot/hub/runtime/transports/httpstransport"
	"log/slog"
)

// TransportsManager aggregates multiple protocol bindings and manages the starting,
// stopping and routing of protocol messages.
// This implements the ITransportBinding interface like the protocols it manages.
type TransportsManager struct {

	// handler for events, actions and rpc requests
	bindings []api.ITransportBinding

	// The embedded binding can be used directly with embedded services
	embeddedBinding *embedded.EmbeddedTransport

	// handler to pass incoming messages to
	handler func(tv *things.ThingMessage) api.DeliveryStatus
}

// AddProtocolBinding adds a protocol binding to the manager
// Protocols must be added before calling Start()
func (svc *TransportsManager) AddProtocolBinding(binding api.ITransportBinding) {
	svc.bindings = append(svc.bindings, binding)
}

// GetEmbedded returns the embedded protocol binding
// Intended to receive messages from, and send to, embedded services
func (svc *TransportsManager) GetEmbedded() *embedded.EmbeddedTransport {
	return svc.embeddedBinding
}

// GetConnectURL returns URL of the first protocol that has a baseurl
func (svc *TransportsManager) GetConnectURL() string {
	for _, b := range svc.bindings {
		pi := b.GetProtocolInfo()
		if pi.BaseURL != "" {
			return pi.BaseURL
		}
	}
	return ""
}

// GetProtocolInfo returns information on the preferred protocol
func (svc *TransportsManager) GetProtocolInfo() (pi api.ProtocolInfo) {
	if len(svc.bindings) > 0 {
		return svc.bindings[0].GetProtocolInfo()
	}
	return
}

// GetProtocols returns a list of active server protocol bindings
func (svc *TransportsManager) GetProtocols() []api.ITransportBinding {
	return svc.bindings
}

// SendToClient sends a message to a connected agent or consumer client
// If an agent is connected through multiple protocols then this stops
// after the first successful delivery.
// TODO: optimize to use the most efficient protocol
// TODO: sending to multiple instances of the same client?
func (svc *TransportsManager) SendToClient(
	clientID string, msg *things.ThingMessage) (stat api.DeliveryStatus, found bool) {

	// for now simply send the action request to all protocol handlers
	for _, protoHandler := range svc.bindings {
		stat, found = protoHandler.SendToClient(clientID, msg)
		// if delivery is not failed or pending then the remote client has received it
		if found {
			return stat, found
			//} else if stat.Status != api.DeliveryFailed &&
			//	stat.Status != api.DeliveryPending &&
			//	stat.Status != "" {
			//	return stat, found
		}
	}
	stat.Failed(msg, fmt.Errorf("SendToClient: Destination '%s' not found", clientID))
	return stat, false
}

// SendEvent sends a event to all subscribers
func (svc *TransportsManager) SendEvent(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	// delivery fails if there are no subscribers. Does this matter?
	stat.Status = api.DeliveryFailed
	stat.Error = "Destination not found"

	for _, protoHandler := range svc.bindings {
		// FIXME: only send to subscribers in the PB's
		stat2 := protoHandler.SendEvent(msg)
		if stat2.Status == api.DeliveryDelivered {
			stat.Status = api.DeliveryDelivered
			stat.Error = ""
		}
	}
	return stat
}

// Start the protocol servers
func (svc *TransportsManager) Start(handler api.MessageHandler) error {
	svc.handler = handler
	for _, pb := range svc.bindings {
		err := pb.Start(handler)
		if err != nil {
			slog.Error("Protocol binding error:", "err", err)
		}
	}
	return nil
}

// Stop the protocol servers
func (svc *TransportsManager) Stop() {
	for _, pb := range svc.bindings {
		pb.Stop()
	}
}

// NewProtocolManager creates a new instance of the protocol manager.
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
func NewProtocolManager(cfg *ProtocolsConfig,
	privKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	sessionAuth api.IAuthenticator) *TransportsManager {

	svc := TransportsManager{
		// the embedded protocol binding
		// 1. receives messages from embedded services to pass on to the middleware (source)
		// 2. sends messages to embedded services (sink)
		// Embedded services are: authn, authz, directory, inbox, outbox and history services
		embeddedBinding: embedded.NewEmbeddedBinding(),
	}
	svc.AddProtocolBinding(svc.embeddedBinding)
	svc.AddProtocolBinding(
		httpstransport.NewHttpsBinding(
			&cfg.HttpsBinding,
			privKey, serverCert, caCert,
			sessionAuth))

	return &svc
}

// StartProtocolManager starts a new instance of the protocol manager.
// This instantiates enabled protocol bindings, including the embedded binding
// to be used to register embedded services.
func StartProtocolManager(cfg *ProtocolsConfig,
	privKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	sessionAuth api.IAuthenticator, handler api.MessageHandler) (*TransportsManager, error) {

	svc := NewProtocolManager(cfg, privKey, serverCert, caCert, sessionAuth)
	err := svc.Start(handler)

	return svc, err
}