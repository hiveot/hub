package service

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/services/idprov/idprovapi"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/messaging"
	"log/slog"
)

//
//const DefaultIoTCertValidityDays = 14
//const ApprovedSecret = "approved"
//const DefaultRetrySec = 12 * 3600

// IdProvService handles provisioning requests from devices and services.
// This starts listening on the provisioning port using a server certificate signed by the Hub CA.
// If enabled, publish the DNS-SD discovery record with the server address and port.
//
// Connecting clients can request an auth token, providing their ID and public key.
// The server responds with a token or wait-for-approval.
// In case of wait for approval the client must repeat the request, in 1 minute intervals until approval is received
// or rejected.
//
// The server connects to the Hub to obtain auth tokens from the auth service. Tokens issued will be very short lived
// and require the device to refresh with the auth service directly, after connecting to the hub.
//
// The provisioning server can be started and stopped on the fly so it only runs when needed.
type IdProvService struct {

	// Hub connection
	ag *messaging.Agent
	// the manage service
	ManageIdProv *ManageIdProvService

	// server listening port
	port int
	// server TLS certificate
	serverCert *tls.Certificate
	// hiveot CA that signed the server cert
	caCert *x509.Certificate
	// the http server that received provisioning requests
	httpServer *IdProvHttpServer
}

// Start the provisioning service
// 1. start the management service
// 2. set allowed roles for RPC calls to this service
// 3. Start the http request server
// 4. start the security check for rogue DNS-SD records
// 5. start DNS-SD discovery server
//
// cc is the connection with the hub to receive requests
func (svc *IdProvService) Start(cc transports.IConnection) (err error) {

	ag := messaging.NewAgent(cc, nil, nil, nil, 0)
	slog.Info("Starting the provisioning service", "clientID", ag.GetClientID())
	svc.ag = ag
	svc.ManageIdProv, err = StartManageIdProvService(svc.ag)
	if err != nil {
		return err
	}

	// Set the required permissions for using this service
	err = authz.UserSetPermissions(&ag.Consumer, authz.ThingPermissions{
		AgentID: ag.GetClientID(),
		ThingID: idprovapi.ManageServiceID,
		Allow: []authz.ClientRole{
			authz.ClientRoleManager,
			authz.ClientRoleAdmin,
			authz.ClientRoleService},
	})
	if err != nil {
		return err
	}
	// the agent maps incoming action requests to the management service methods
	StartIdProvAgent(svc.ManageIdProv, ag)

	// Start the HTTP server
	svc.httpServer, err = StartIdProvHttpServer(
		svc.port, svc.serverCert, svc.caCert, svc.ManageIdProv)
	return err
}

// Stop the provisioning service
func (svc *IdProvService) Stop() {
	slog.Info("Stopping the provisioning service")
	if svc.httpServer != nil {
		svc.httpServer.Stop()
		svc.httpServer = nil
	}
	if svc.ManageIdProv != nil {
		svc.ManageIdProv.Stop()
		svc.ManageIdProv = nil
	}
}

// NewIdProvService creates a new provisioning service instance
func NewIdProvService(port int, serverCert *tls.Certificate, caCert *x509.Certificate) *IdProvService {
	svc := &IdProvService{
		port:       port,
		serverCert: serverCert,
		caCert:     caCert,
	}

	return svc
}
