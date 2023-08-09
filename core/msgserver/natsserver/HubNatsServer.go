package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"time"
)

// HubNatsServer runs an embedded NATS server using callout for authentication.
// This configures the server to use a separate callout account
// This configures the server for publishing  provides a static configuration for the server for authn, authz, directory, and history streaming
type HubNatsServer struct {
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	//serverOpts *server.Options
	// predefined authn key for connecting core services
	//serviceKey nkeys.KeyPair
	//serviceJWT string
	ns *server.Server

	// the handler to verify authentication requests, or nil to accept any
	//verifyAuthn func(req *jwt.AuthorizationRequestClaims) error
}

// ConnectInProc connects to the server in-process using the service key.
// Intended for the core services to connect to the server.
// func (srv *HubNatsServer) ConnectInProc(clientID string) (*nats.Conn, error) {
func (srv *HubNatsServer) ConnectInProc(clientJWT string, clientKP nkeys.KeyPair) (*nats.Conn, error) {
	//func (srv *HubNatsServer) ConnectInProc(clientCreds []byte) (*nats.Conn, error) {
	// The handler to sign the server issued challenge
	//sigCB := func(nonce []byte) ([]byte, error) {
	//	return srv.serviceKey.Sign(nonce)
	//}
	// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.caCert != nil {
		caCertPool.AddCert(srv.caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.caCert == nil,
	}

	//clientJWT, err := jwt.ParseDecoratedJWT(clientCreds)
	claims, err := jwt.DecodeUserClaims(clientJWT)
	if err != nil {
		return nil, err
	}
	//clientKP, err := jwt.ParseDecoratedUserNKey(clientCreds)
	//if err != nil {
	//	return nil, err
	//}
	clientSeed, _ := clientKP.Seed()
	cl, err := nats.Connect(srv.ns.ClientURL(), // don't need a URL for in-process connection
		nats.Name(claims.Name), // connection name for validation
		nats.Secure(tlsConfig),
		//nats.Nkey(clientKeyPub, sigCB),
		nats.UserJWTAndSeed(clientJWT, string(clientSeed)),
		nats.Timeout(time.Minute),
		nats.InProcessServer(srv.ns),
	)

	return cl, err
}

// CreateServerConfig create the for managing the NATS server configuration
// This returns the nats server configuration based on the parameters.
func (srv *HubNatsServer) CreateServerConfig(
	hubcfg *msgserver.MsgServerConfig,
	operatorJWT string,
	systemAccountJWT string,
	appAccountJWT string,
	serviceKey nkeys.KeyPair,
) *server.Options {
	var err error
	// JWT needs an account resolver.
	// Use the simple in-memory resolver as there is only one account.
	memoryResolver := &server.MemAccResolver{}
	operatorClaim, err := jwt.DecodeOperatorClaims(operatorJWT)
	_ = operatorClaim
	if err != nil {
		panic(err)
	}

	systemClaims, _ := jwt.DecodeAccountClaims(systemAccountJWT)
	systemAccountPub := systemClaims.Subject
	err = memoryResolver.Store(systemClaims.Subject, systemAccountJWT)
	if err != nil {
		panic(err)
	}
	appAccountClaims, _ := jwt.DecodeAccountClaims(appAccountJWT)
	_ = memoryResolver.Store(appAccountClaims.Subject, appAccountJWT)
	if err != nil {
		panic(err)
	}
	serviceKeyPub, _ := serviceKey.PublicKey()
	_ = serviceKeyPub
	//noAuthKey, _ := nkeys.FromSeed([]byte(natshubclient.PublicUnauthenticatedNKey))

	//noAuthKeyPub, _ := noAuthKey.PublicKey()
	natsOpts := &server.Options{
		Host:             hubcfg.Host,
		Port:             hubcfg.Port,
		AccountResolver:  memoryResolver,
		SystemAccount:    systemAccountPub, //"SYS",
		TrustedOperators: []*jwt.OperatorClaims{operatorClaim},

		//TrustedKeys: []string{operatorClaim.Subject},
		//Nkeys: []*server.NkeyUser{
		//	{Nkey: serviceKeyPub},
		//	{Nkey: noAuthKeyPub}}, //Permissions:            noAuthPermissions,
		//Account:                appAccountClaims.Subject,
		//SigningKey:             "",
		//AllowedConnectionTypes: nil,

		JetStream:          true,
		JetStreamMaxMemory: int64(hubcfg.MaxDataMemoryMB) * 1024 * 1024,
		StoreDir:           hubcfg.DataDir,

		// logging
		Debug:   true,
		Logtime: true,
	}

	if srv.caCert != nil && srv.serverCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(srv.caCert)
		clientCertList := []tls.Certificate{*srv.serverCert}
		tlsConfig := &tls.Config{
			ServerName:   "HiveOT Hub",
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MinVersion:   tls.VersionTLS13,
		}
		natsOpts.TLSTimeout = 1000 // for debugging auth
		natsOpts.TLSConfig = tlsConfig
	}
	return natsOpts
}

// Start the NATS server
// This creates a auth callout and application account.
func (srv *HubNatsServer) Start(opts *server.Options) (clientURL string, err error) {

	// start nats
	srv.ns, err = server.NewServer(opts)
	if err != nil {
		return "", err
	}

	srv.ns.ConfigureLogger()

	// startup
	go srv.ns.Start()
	if !srv.ns.ReadyForConnections(30 * time.Second) {
		err = errors.New("nats: not ready for connection")
		return "", err
	}
	clientURL = srv.ns.ClientURL()

	return clientURL, err
}

// Stop the server
func (srv *HubNatsServer) Stop() {
	srv.ns.Shutdown()
}

// NewHubNatsServer creates a new instance of the Hub NATS server
// The given configuration is optional. The server will run with production settings out of the box.
//
// Use SetAuthnVerifier function to install the callout authn handler.
//
//	cfg contains an initialized server configuration for use as hiveot hub
//	serverCert is the TLS certificate of the server signed by the CA
//	caCert is the CA certificate
//	serviceNKey is the services nkey
func NewHubNatsServer(
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	// serviceKey nkeys.KeyPair,
	// serviceJWT string,
) *HubNatsServer {

	srv := &HubNatsServer{
		caCert:     caCert,
		serverCert: serverCert,
		//serviceKey: serviceKey,
		//serviceJWT: serviceJWT,
	}
	return srv
}
