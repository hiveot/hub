package testenv

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"time"
)

// TestJWTServer is an embedded NATS test messaging server using JWT in operator mode.
// Issues:
//   - can't use password authentication when jwt is active
//   - can't use auth callout in operator mode
//   - is operator mode required
type TestJWTServer struct {
	account    *server.Account
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	ns         *server.Server
	//
}

// Start the server, listening on 127.0.0.1
// setup accounts provided with the given bundle
func (svr *TestJWTServer) Start(bundle TestAuthBundle) (clientURL string, err error) {
	operatorClaim, _ := jwt.DecodeOperatorClaims(bundle.OperatorJWT)
	_ = operatorClaim
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
	systemAccountPub, err := bundle.SystemAccountNKey.PublicKey()
	//systemAccount := server.NewAccount("SYS")
	//_ = systemAccount
	//appAccount := server.NewAccount("AppAccount")

	//servicePub, _ := bundle.ServiceNKey.PublicKey()
	opts := &server.Options{
		Host:            "127.0.0.1", // must match the address on the generated cert
		Port:            9998,        // some random test port that doesn't interfere
		AccountResolver: memoryResolver,
		SystemAccount:   systemAccountPub, //"SYS",
		//Accounts:        []*server.Account{systemAccount, appAccount},

		JetStream:          true,
		TrustedOperators:   []*jwt.OperatorClaims{operatorClaim},
		JetStreamMaxMemory: 10 * 1024 * 1024,
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

func (svr *TestJWTServer) Stop() {
	if svr.ns != nil {
		svr.ns.Shutdown()
	}
}

// NewTestJWTServer create a new test server instance
//
//	serverCert optional cert for 127.0.0.1
func NewTestJWTServer(serverCert *tls.Certificate, caCert *x509.Certificate) *TestJWTServer {
	ts := &TestJWTServer{
		serverCert: serverCert,
		caCert:     caCert,
	}
	return ts
}
