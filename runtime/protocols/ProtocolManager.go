package protocols

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/keys"
	thing "github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/protocolbindings/api"
	"github.com/hiveot/hub/runtime/protocolbindings/httpsbinding"
	"log/slog"
)

// ProtocolManager aggregates multiple protocol bindings and manages the starting,
// stopping and routing of protocol messages.
// This implements the IProtocolCallback interface.
type ProtocolManager struct {

	// handler for events, actions and rpc requests
	bindings []api.IProtocolBinding
}

// AddProtocolBinding adds a protocol binding to the manager
// Protocols must be added before calling Start()
func (svc *ProtocolManager) AddProtocolBinding(binding api.IProtocolBinding) {
	svc.bindings = append(svc.bindings, binding)
}

// Start the protocol servers
func (svc *ProtocolManager) Start() error {
	for _, pb := range svc.bindings {
		err := pb.Start()
		if err != nil {
			slog.Error("Protocol binding error:", "err", err)
		}
	}
	return nil
}

// Stop the protocol servers
func (svc *ProtocolManager) Stop() {
	for _, pb := range svc.bindings {
		pb.Stop()
	}
}

// NewProtocolManager creates a new instance of the protocol manager
func NewProtocolManager(
	privKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	msgHandler func(tv *thing.ThingValue) ([]byte, error)) *ProtocolManager {
	httpsPort := uint(9000)
	svc := ProtocolManager{}
	svc.AddProtocolBinding(
		httpsbinding.NewHttpsBinding(httpsPort, privKey, serverCert, caCert, msgHandler))

	return &svc
}
