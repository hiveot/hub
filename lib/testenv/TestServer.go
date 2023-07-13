package testenv

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"time"
)

// Default permissions for new users
var defaultPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"guest.>"}, Deny: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"guest.>", "_INBOX.>"}, Deny: []string{">"}},
}

var adminPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
}

// TestServer is an embedded NATS test messaging server
type TestServer struct {
	account    *server.Account
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	ns         *server.Server
}

// Start the server, listening on 127.0.0.1
// setup accounts provided with the given bundle
func (svr *TestServer) Start(bundle TestAuthBundle) (clientURL string, err error) {
	operatorClaim, _ := jwt.DecodeOperatorClaims(bundle.OperatorJWT)
	systemClaims, _ := jwt.DecodeAccountClaims(bundle.SystemAccountJWT)

	memoryResolver := &server.MemAccResolver{}
	err = memoryResolver.Store(systemClaims.Subject, bundle.SystemAccountJWT)
	if err != nil {
		panic(err)
	}
	appAccountClaims, _ := jwt.DecodeAccountClaims(bundle.AppAccountJWT)
	_ = memoryResolver.Store(appAccountClaims.Subject, bundle.AppAccountJWT)
	if err != nil {
		panic(err)
	}
	//appAccount := server.NewAccount("AppAccount")

	//servicePub, _ := bundle.ServiceNKey.PublicKey()
	opts := &server.Options{
		Host:            "127.0.0.1", // must match the address on the generated cert
		Port:            9998,        // some random test port that doesn't interfere
		AccountResolver: memoryResolver,
		SystemAccount:   systemClaims.Subject,

		JetStream:          true,
		TrustedOperators:   []*jwt.OperatorClaims{operatorClaim},
		JetStreamMaxMemory: 10 * 1024 * 1024,
		//NoAuthUser:         "unauthenticated",
		//Accounts: []*server.Account{appAccount},
		//Nkeys: []*server.NkeyUser{
		//	{
		//		Nkey:                   bundle.ServiceNKey,
		//		Permissions:            adminPermissions,
		//		Account:                hubAccount,
		//		SigningKey:             "",
		//		AllowedConnectionTypes: nil,
		//	},
		//},

		// can't include users in operator mode
		//Users: []*server.User{
		//	{
		//		Username: "unauthenticated",
		//		Password: "",
		//		//Permissions: defaultPermissions,
		//		//Account:     svr.account,
		//	},
		//},
	}

	if svr.caCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(svr.caCert)
		clientCertList := []tls.Certificate{*svr.serverCert}
		tlsConfig := &tls.Config{
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
		}
		opts.TLSConfig = tlsConfig
	}

	ns, err := server.NewServer(opts)

	if err != nil {
		return "", err
	}
	svr.ns = ns
	svr.account = svr.ns.GlobalAccount()
	go ns.Start()
	if !ns.ReadyForConnections(4 * time.Second) {
		panic("not ready for connection")
	}
	clientURL = ns.ClientURL()
	return clientURL, nil
}

func (svr *TestServer) Stop() {
	if svr.ns != nil {
		svr.ns.Shutdown()
	}
}

// NewTestServer create a new test server instance
//
//	serverCert optional cert for 127.0.0.1
func NewTestServer(serverCert *tls.Certificate, caCert *x509.Certificate) *TestServer {
	ts := &TestServer{
		serverCert: serverCert,
		caCert:     caCert,
	}
	return ts
}
