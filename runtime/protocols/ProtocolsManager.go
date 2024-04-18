package protocols

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding"
	"github.com/hiveot/hub/runtime/router"
	"log/slog"
)

// ProtocolsManager aggregates multiple protocol bindings and manages the starting,
// stopping and routing of protocol messages.
// This implements the IProtocolCallback interface.
type ProtocolsManager struct {

	// handler for events, actions and rpc requests
	bindings []api.IProtocolBinding

	// handler to pass incoming messages to
	handler func(tv *things.ThingMessage) ([]byte, error)
}

// AddProtocolBinding adds a protocol binding to the manager
// Protocols must be added before calling Start()
func (svc *ProtocolsManager) AddProtocolBinding(binding api.IProtocolBinding) {
	svc.bindings = append(svc.bindings, binding)
}

// SendActionToAgent sends an action request to the agent.
func (svc *ProtocolsManager) SendActionToAgent(
	agentID string, msg *things.ThingMessage) (reply []byte, err error) {
	// for now simply send the action request to all protocol handlers
	for _, protoHandler := range svc.bindings {
		reply, err = protoHandler.SendActionToAgent(agentID, msg)
		if err == nil {
			// avoid double delivery
			break
		}
	}
	return reply, err
}

// SendEvent sends a event to all subscribers
func (svc *ProtocolsManager) SendEvent(msg *things.ThingMessage) {
	for _, protoHandler := range svc.bindings {
		protoHandler.SendEvent(msg)
	}
}

// Start the protocol servers
func (svc *ProtocolsManager) Start(handler router.MessageHandler) error {
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
func (svc *ProtocolsManager) Stop() {
	for _, pb := range svc.bindings {
		pb.Stop()
	}
}

// NewProtocolManager creates a new instance of the protocol manager
func NewProtocolManager(cfg *ProtocolsConfig,
	privKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	sessionAuth api.IAuthenticator) *ProtocolsManager {

	svc := ProtocolsManager{}
	svc.AddProtocolBinding(
		httpsbinding.NewHttpsBinding(
			&cfg.HttpsBinding,
			privKey, serverCert, caCert,
			sessionAuth))

	return &svc
}
