package natsnkeyserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natscoserver"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"time"
)

// EventsIntakeStreamName all group streams use this stream as their source
const EventsIntakeStreamName = "$events"

// NatsNKeyServer runs an embedded NATS server using nkeys for authentication.
// this implements the IMsgServer interface
type NatsNKeyServer struct {
	cfg      *NatsServerConfig
	natsOpts server.Options
	ns       *server.Server
	// enable callout authn with EnableCalloutHandler. nil to just use nkeys
	chook *natscoserver.NatsCalloutHook
	// the clients that can authenticate
	clients []authn.AuthnEntry
	// authorization of clients
	userGroupRoles map[string]authz.RoleMap
}

// ConnectInProcNC establishes a nats connection to the server for core services.
// This connects in-process using the service key.
// Intended for the core services to connect to the server.
//
//	serviceID of the connecting service
//	clientKey is optional alternate key or nil to use the built-in core service ID
func (srv *NatsNKeyServer) ConnectInProcNC(serviceID string, clientKey nkeys.KeyPair) (*nats.Conn, error) {

	if clientKey == nil {
		clientKey = srv.cfg.CoreServiceKP
	}
	// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.cfg.CaCert != nil {
		caCertPool.AddCert(srv.cfg.CaCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.cfg.CaCert == nil,
	}
	sigCB := func(nonce []byte) ([]byte, error) {
		sig, _ := clientKey.Sign(nonce)
		return sig, nil
	}
	serviceKeyPub, _ := clientKey.PublicKey()
	nc, err := nats.Connect(srv.ns.ClientURL(), // don't need a URL for in-process connection
		nats.Name(serviceID),
		nats.Secure(tlsConfig),
		nats.Nkey(serviceKeyPub, sigCB),
		nats.Timeout(time.Minute),
		nats.InProcessServer(srv.ns),
	)
	if err == nil {
		js, err2 := nc.JetStream()
		err = err2
		_ = js
	}
	return nc, err
}

// ConnectInProc establishes a connection to the server for core services.
// This connects in-process using the service key.
// Intended for the core services to connect to the server.
//
//	serviceID of the connecting service
func (srv *NatsNKeyServer) ConnectInProc(serviceID string) (hubclient.IHubClient, error) {

	nc, err := srv.ConnectInProcNC(serviceID, nil)
	if err != nil {
		return nil, err
	}
	hc, err := natshubclient.ConnectWithNC(nc, serviceID)
	return hc, err
}

// EnableCalloutHandler reconfigures the server for external callout authn
// The authn callout handler will issue tokens for the application account.
// Invoke this after successfully starting the server
func (srv *NatsNKeyServer) EnableCalloutHandler(
	authnVerifier func(request *jwt.AuthorizationRequestClaims) error) error {

	// Ideally the callout handler uses a separate callout account.
	// Apparently this isn't allowed so it runs in the application account.
	nc, err := srv.ConnectInProcNC("callout", nil)
	if err != nil {
		return fmt.Errorf("unable to connect callout handler: %w", err)
	}
	if err == nil {
		srv.chook, err = natscoserver.ConnectNatsCalloutHook(
			&srv.natsOpts,
			srv.cfg.AppAccountName, // issuerAcctName,
			srv.cfg.AppAccountKP,
			nc,
			authnVerifier)
	}
	return err
}

// Start the NATS server with the given configuration and create an event ingress stream
//
//	cfg.Setup must have been called first.
func (srv *NatsNKeyServer) Start() (clientURL string, err error) {

	srv.natsOpts, err = srv.cfg.CreateNatsNKeyOptions()
	if err != nil {
		return "", err
	}

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

	// the app account must have JS enabled
	ac, _ := srv.ns.LookupAccount(srv.cfg.AppAccountName)
	err = ac.EnableJetStream(nil) //use defaults
	if err != nil {
		return clientURL, fmt.Errorf("can't enable JS for app account: %w", err)
	}

	hasJS := ac.JetStreamEnabled()
	if !hasJS {
		return clientURL, fmt.Errorf("JS not enabled for app account '%s'", srv.cfg.AppAccountName)
	}

	// ensure the events intake stream exists
	nc, err := srv.ConnectInProcNC("jetsetup", nil)
	if err != nil {
		return clientURL, err
	}
	js, err := nc.JetStream()
	if err != nil {
		return clientURL, err
	}
	_, err = js.StreamInfo(EventsIntakeStreamName)
	if err != nil {
		// The intake stream receives events from all publishers and things
		subj := natshubclient.MakeSubject("", "", natshubclient.SubjectTypeEvent, ">")
		cfg := &nats.StreamConfig{
			Name:        EventsIntakeStreamName,
			Description: "HiveOT Events Intake Stream",
			Retention:   nats.LimitsPolicy,
			Subjects:    []string{subj},
			// since consumers are other streams, a short retention should suffice
			MaxAge: time.Hour,
		}
		_, err = js.AddStream(cfg)
	}

	return clientURL, err
}

// Stop the server
func (srv *NatsNKeyServer) Stop() {
	srv.ns.Shutdown()
}

// NewNatsNKeyServer creates a new instance of the Hub NATS server for NKey authn.
func NewNatsNKeyServer(cfg *NatsServerConfig) *NatsNKeyServer {

	srv := &NatsNKeyServer{cfg: cfg}
	return srv
}
