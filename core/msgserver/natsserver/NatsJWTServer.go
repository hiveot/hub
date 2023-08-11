package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"time"
)

// NatsJWTServer runs an embedded NATS server using callout for authentication.
// This configures the server to use a separate callout account
// This configures the server for publishing  provides a static configuration for the server for authn, authz, directory, and history streaming
type NatsJWTServer struct {
	cfg        NatsJWTConfig
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	natsOpts   server.Options
	// predefined authn key for connecting core services
	//serviceKey nkeys.KeyPair
	//serviceJWT string
	ns *server.Server

	// the handler to verify authentication requests, or nil to accept any
	//verifyAuthn func(req *jwt.AuthorizationRequestClaims) error
}

// ConnectInProc connects to the server in-process using the service key.
// Intended for the core services to connect to the server.
func (srv *NatsJWTServer) ConnectInProc(clientID string) (*nats.Conn, error) {
	//func (srv *NatsJWTServer) ConnectInProc(clientCreds []byte) (*nats.Conn, error) {
	// The handler to sign the server issued challenge
	//sigCB := func(nonce []byte) ([]byte, error) {
	//	return srv.serviceKey.Sign(nonce)
	//}
	clientJWT := srv.cfg.AppServiceJWT
	clientKP := srv.cfg.AppServiceKP

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

// Start the NATS server
func (srv *NatsJWTServer) Start(cfg NatsJWTConfig) (clientURL string, err error) {

	srv.natsOpts, srv.cfg = CreateNatsJWTOptions(cfg)

	// start nats
	srv.ns, err = server.NewServer(&srv.natsOpts)
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
func (srv *NatsJWTServer) Stop() {
	srv.ns.Shutdown()
}

// NewNatsJWTServer creates a new instance of the Hub NATS server
// The given configuration is optional. The server will run with production settings out of the box.
//
// Use SetAuthnVerifier function to install the callout authn handler.
//
//	cfg contains an initialized server configuration for use as hiveot hub
//	serverCert is the TLS certificate of the server signed by the CA
//	caCert is the CA certificate
//	serviceNKey is the services nkey
func NewNatsJWTServer(
	serverCert *tls.Certificate,
	caCert *x509.Certificate,
	// serviceKey nkeys.KeyPair,
	// serviceJWT string,
) *NatsJWTServer {

	srv := &NatsJWTServer{
		caCert:     caCert,
		serverCert: serverCert,
		//serviceKey: serviceKey,
		//serviceJWT: serviceJWT,
	}
	return srv
}
