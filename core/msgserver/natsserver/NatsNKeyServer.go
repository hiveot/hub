package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"time"
)

// NatsNKeyServer runs an embedded NATS server using nkeys for authentication.
type NatsNKeyServer struct {
	cfg      NatsServerConfig
	natsOpts server.Options
	ns       *server.Server
}

// AddService adds a core service authn key to the app account and reloads the options.
// Services can pub/sub to all things subjects
func (srv *NatsNKeyServer) AddService(serviceID string, serviceKeyPub string) error {
	appAcct, err := srv.ns.LookupAccount(srv.cfg.AppAccountName)
	if err != nil {
		return fmt.Errorf("missing app account: %w", err)
	}
	srv.natsOpts.Nkeys = append(srv.natsOpts.Nkeys, &server.NkeyUser{
		Nkey:    serviceKeyPub,
		Account: appAcct,
	})
	err = srv.ns.ReloadOptions(&srv.natsOpts)
	return err
}

// ConnectInProc connects to the server in-process using the service key.
// Intended for the core services to connect to the server.
// A custom clientKey can be used to authn which must have been added first with AddClient
//
//	serviceID of the connecting service
//	clientKey is optional alternate key or nil to use the built-in core service ID
func (srv *NatsNKeyServer) ConnectInProc(serviceID string, clientKey nkeys.KeyPair) (*nats.Conn, error) {

	// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.cfg.CaCert != nil {
		caCertPool.AddCert(srv.cfg.CaCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.cfg.CaCert == nil,
	}
	if clientKey == nil {
		clientKey = srv.cfg.CoreServiceKP
	}
	sigCB := func(nonce []byte) ([]byte, error) {
		sig, _ := clientKey.Sign(nonce)
		return sig, nil
	}
	serviceKeyPub, _ := clientKey.PublicKey()
	cl, err := nats.Connect(srv.ns.ClientURL(), // don't need a URL for in-process connection
		nats.Name(serviceID),
		nats.Secure(tlsConfig),
		nats.Nkey(serviceKeyPub, sigCB),
		nats.Timeout(time.Minute),
		nats.InProcessServer(srv.ns),
	)
	return cl, err
}

// Start the NATS server with the given configuration
//
//	cfg.Setup must have been called first.
func (srv *NatsNKeyServer) Start(cfg NatsServerConfig) (clientURL string, err error) {

	srv.cfg = cfg
	srv.natsOpts = cfg.CreateNatsNKeyOptions()
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

	// how to enable jetstream for account?

	// add the core service account
	coreServicePub, _ := srv.cfg.CoreServiceKP.PublicKey()
	err = srv.AddService("core-service", coreServicePub)

	return clientURL, err
}

// Stop the server
func (srv *NatsNKeyServer) Stop() {
	srv.ns.Shutdown()
}

// NewNatsNKeyServer creates a new instance of the Hub NATS server for NKey authn.
func NewNatsNKeyServer() *NatsNKeyServer {

	srv := &NatsNKeyServer{}
	return srv
}
