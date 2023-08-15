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

// NatsJWTServer runs an embedded NATS server using JWT for authentication.
type NatsJWTServer struct {
	cfg      NatsServerConfig
	natsOpts server.Options
	ns       *server.Server
}

// ConnectInProc connects to the server in-process using the service key.
// Intended for the core services to connect to the server.
func (srv *NatsJWTServer) ConnectInProc(clientID string) (*nats.Conn, error) {
	//func (srv *NatsJWTServer) ConnectInProc(clientCreds []byte) (*nats.Conn, error) {
	// The handler to sign the server issued challenge
	//sigCB := func(nonce []byte) ([]byte, error) {
	//	return srv.serviceKey.Sign(nonce)
	//}
	clientJWT := srv.cfg.CoreServiceJWT
	clientKP := srv.cfg.CoreServiceKP

	// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.cfg.CaCert != nil {
		caCertPool.AddCert(srv.cfg.CaCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.cfg.CaCert == nil,
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

// Start the NATS server with the given configuration
//
//	cfg.Setup() must have been called first.
func (srv *NatsJWTServer) Start(cfg NatsServerConfig) (clientURL string, err error) {
	srv.cfg = cfg
	srv.natsOpts = cfg.CreateNatsJWTOptions()

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

// NewNatsJWTServer creates a new instance of the Hub NATS server using JWT authentication
func NewNatsJWTServer() *NatsJWTServer {

	srv := &NatsJWTServer{}
	return srv
}
