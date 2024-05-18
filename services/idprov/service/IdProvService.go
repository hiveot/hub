package service

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
)

const DefaultIoTCertValidityDays = 14
const ApprovedSecret = "approved"
const DefaultRetrySec = 12 * 3600

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
	hc hubclient.IHubClient
	// the manage service
	mng *ManageIdProvService

	// server listening port
	port uint
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
func (svc *IdProvService) Start(hc hubclient.IHubClient) (err error) {
	slog.Warn("Starting the provisioning service", "clientID", hc.ClientID())
	svc.hc = hc
	//svc.Stop()
	svc.mng = StartManageIdProvService(svc.hc)
	if err != nil {
		return err
	}
	// FIXME: FIX THE SERVICE AUTHORIZATION
	// Set the required permissions for using this service
	// any user roles can view the directory
	//myProfile := authnclient.NewAuthnUserClient(svc.hc)
	//err = myProfile.SetServicePermissions(idprovapi.ManageProvisioningCap, []string{
	//	authapi.ClientRoleManager,
	//	authapi.ClientRoleAdmin,
	//	authapi.ClientRoleService})
	if err != nil {
		return err
	}

	// Start the HTTP server
	svc.httpServer, err = StartIdProvHttpServer(svc.port, svc.serverCert, svc.caCert, svc.mng)
	return err
}

// Stop the provisioning service
func (svc *IdProvService) Stop() {
	slog.Warn("Stopping the provisioning service")
	if svc.httpServer != nil {
		svc.httpServer.Stop()
		svc.httpServer = nil
	}
	if svc.mng != nil {
		svc.mng.Stop()
		svc.mng = nil
	}
}

// NewIdProvService creates a new provisioning service instance
func NewIdProvService(port uint, serverCert *tls.Certificate, caCert *x509.Certificate) *IdProvService {
	svc := &IdProvService{
		port:       port,
		serverCert: serverCert,
		caCert:     caCert,
	}

	return svc
}
