package service

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/core/nats"
)

// HubServer contains the embedded NATS JetStream server
// * custom authentication
// * custom authorization
type HubServer struct {
	serverCert *tls.Certificate
	caCert     *x509.Certificate
	hubImpl    nats.HubNatsServer
	authn      authn.IManageAuthn
	authz      authz.IAuthz
}

// Start the Hub messaging server
func (hs *HubServer) Start(host string, port int, serverCert *tls.Certificate, caCert *x509.Certificate) (clientURL string, err error) {
	return hs.hubImpl.Start(host, port, serverCert, caCert)
}

// Stop the server
func (hs *HubServer) Stop() {
	hs.hubImpl.Stop()
}

// NewHubServer creates a server instance using a certificate
func NewHubServer() *HubServer {
	hs := &HubServer{
		hubImpl: nats.HubNatsServer{},
		//authn:   service.NewAuthnService(),
		//authz:   service2.NewAuthzService(),
	}
	return hs
}
