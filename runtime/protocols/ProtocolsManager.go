package protocols

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/protocols/api"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding"
	"log/slog"
)

// ProtocolsManager aggregates multiple protocol bindings and manages the starting,
// stopping and routing of protocol messages.
// This implements the IProtocolCallback interface.
type ProtocolsManager struct {

	// handler for events, actions and rpc requests
	bindings []api.IProtocolBinding
}

// AddProtocolBinding adds a protocol binding to the manager
// Protocols must be added before calling Start()
func (svc *ProtocolsManager) AddProtocolBinding(binding api.IProtocolBinding) {
	svc.bindings = append(svc.bindings, binding)
}

// Start the protocol servers
func (svc *ProtocolsManager) Start() error {
	for _, pb := range svc.bindings {
		err := pb.Start()
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
	msgHandler func(tv *things.ThingValue) ([]byte, error)) *ProtocolsManager {
	svc := ProtocolsManager{}
	svc.AddProtocolBinding(
		httpsbinding.NewHttpsBinding(&cfg.HttpsBinding, privKey, serverCert, caCert, msgHandler))

	return &svc
}
