package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/nats-io/jwt/v2"
)

// NatsCalloutServer runs an embedded NATS server using callout for authentication.
// This uses the nkeys config
type NatsCalloutServer struct {
	NatsNKeyServer
	chook *NatsCalloutHook
}

// AddService adds a core service authn key to the app account and reloads the options.
// Services can pub/sub to all things subjects
// Use of the given nkey is excluded from invoking the callout handler.
func (srv *NatsCalloutServer) AddService(serviceID string, serviceKeyPub string) error {
	// if callout has been activated then exclude the key from invoking callout
	if srv.natsOpts.AuthCallout != nil {
		srv.natsOpts.AuthCallout.AuthUsers = append(srv.natsOpts.AuthCallout.AuthUsers, serviceKeyPub)
	}
	// NatsNKeyServer.AddService reloads the options
	err := srv.NatsNKeyServer.AddService(serviceID, serviceKeyPub)
	return err
}

// SetCalloutHandler reconfigures the server for external callout authn
// The authn callout handler will issue tokens for the application account.
// Invoke this after successfully starting the server
func (srv *NatsCalloutServer) SetCalloutHandler(
	authnVerifier func(request *jwt.AuthorizationRequestClaims) error) error {

	// Ideally the callout handler uses a separate callout account.
	// Apparently this isn't allowed so it runs in the application account.
	nc, err := srv.ConnectInProc("callout", nil)
	if err != nil {
		return fmt.Errorf("unable to connect callout handler: %w", err)
	}
	if err == nil {
		srv.chook, err = ConnectNatsCalloutHook(
			&srv.natsOpts,
			srv.cfg.AppAccountName, // issuerAcctName,
			srv.cfg.AppAccountKP,
			nc,
			authnVerifier)
	}
	return err
}

// NewNatsCalloutServer creates a new instance of the Hub NATS server for
// external callout authn.
//
// Use SetAuthnVerifier function to install the callout authn handler.
//
//	serverCert is the TLS certificate of the server signed by the CA
//	caCert is the CA certificate
func NewNatsCalloutServer(
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
) *NatsCalloutServer {

	srv := &NatsCalloutServer{
		NatsNKeyServer: NatsNKeyServer{
			caCert:     caCert,
			serverCert: serverCert,
		},
		chook: nil,
	}
	return srv
}
