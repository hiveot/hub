package service

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/core/msgserver/natsmsgserver"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports/natstransport"
	"github.com/hiveot/hub/lib/net"
	"github.com/hiveot/hub/lib/vocab"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"log/slog"
	"time"
)

// EventsIntakeStreamName all group streams use this stream as their source
const EventsIntakeStreamName = "$events"

// NatsMsgServer runs an embedded NATS server using nkeys for authentication.
// this implements the IMsgServer interface
// See also the callouthook addon for adding JWT token support using nats callouts.
type NatsMsgServer struct {
	Config   *natsmsgserver.NatsServerConfig
	NatsOpts server.Options
	ns       *server.Server

	// map of known clients by ID for quick lookup during auth
	authClients map[string]msgserver.ClientAuthInfo

	// map of permissions for each role
	rolePermissions map[string][]msgserver.RolePermission
	// map of permissions for each service
	servicePermissions map[string][]msgserver.RolePermission

	// connection urls the server is listening on
	tlsURL string
	wssURL string
	udsURL string
}

// ConnectInProcNC establishes a nats connection to the server for core services.
// This connects in-process using the service key.
// Intended for the core services to connect to the server.
//
//	serviceID of the connecting service
//	clientKey is optional alternate key or nil to use the built-in core service ID
func (srv *NatsMsgServer) ConnectInProcNC(serviceID string, clientKP nkeys.KeyPair) (*nats.Conn, error) {

	if clientKP == nil {
		clientKP = srv.Config.CoreServiceKP
	}
	// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.Config.CaCert != nil {
		caCertPool.AddCert(srv.Config.CaCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.Config.CaCert == nil,
	}
	sigCB := func(nonce []byte) ([]byte, error) {
		sig, _ := clientKP.Sign(nonce)
		return sig, nil
	}
	serviceKeyPub, _ := clientKP.PublicKey()
	nc, err := nats.Connect(srv.ns.ClientURL(), // don't need a URL for in-process connection
		nats.Name(serviceID),
		nats.Secure(tlsConfig),
		nats.Nkey(serviceKeyPub, sigCB),
		nats.Timeout(time.Minute),
		nats.InProcessServer(srv.ns),
	)
	slog.Info("ConnectInProc", "serviceID", serviceID, "nkeyPub", serviceKeyPub)
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
func (srv *NatsMsgServer) ConnectInProc(serviceID string) (*hubclient.HubClient, error) {

	nc, err := srv.ConnectInProcNC(serviceID, nil)
	if err != nil {
		return nil, err
	}
	tp := natstransport.NewNatsTransport("", serviceID, nil)
	err = tp.ConnectWithConn(nc)
	hc := hubclient.NewHubClientFromTransport(tp, serviceID)
	return hc, err
}

func (srv *NatsMsgServer) Core() string {
	return "nats"
}

// GetServerURLs is the URL used to connect to this server. This is set on Start
func (srv *NatsMsgServer) GetServerURLs() (tlsURL string, wssURL string, udsURL string) {
	return srv.tlsURL, srv.wssURL, srv.udsURL
}

// Start the NATS server with the given configuration and create an event ingress stream
//
//	config.Setup must have been called first.
func (srv *NatsMsgServer) Start() (err error) {

	srv.NatsOpts, err = srv.Config.CreateNatsNKeyOptions()
	if err != nil {
		return err
	}

	// start nats
	srv.ns, err = server.NewServer(&srv.NatsOpts)
	if err != nil {
		return err
	}

	srv.ns.ConfigureLogger()

	// startup
	go srv.ns.Start()
	if !srv.ns.ReadyForConnections(30 * time.Second) {
		err = errors.New("nats: not ready for connection")
		return err
	}
	outboundIP := net.GetOutboundIP("")
	srv.tlsURL = srv.ns.ClientURL()
	srv.wssURL = fmt.Sprintf("wss://%s:%d", outboundIP.String(), srv.Config.WSPort)
	srv.udsURL = "" // not supported?

	// the app account must have JS enabled
	ac, _ := srv.ns.LookupAccount(srv.Config.AppAccountName)
	err = ac.EnableJetStream(nil) //use defaults
	if err != nil {
		return fmt.Errorf("can't enable JS for app account: %w", err)
	}

	hasJS := ac.JetStreamEnabled()
	if !hasJS {
		return fmt.Errorf("JS not enabled for app account '%s'", srv.Config.AppAccountName)
	}

	// tokenizer
	//srv.tokenizer = NewNatsJWTTokenizer(
	//	srv.config.AppAccountName, srv.config.AppAccountKP)

	// ensure the events intake stream exists
	nc, err := srv.ConnectInProcNC("jetsetup", nil)
	if err != nil {
		return err
	}
	js, err := nc.JetStream()
	if err != nil {
		return err
	}
	_, err = js.StreamInfo(EventsIntakeStreamName)
	if err != nil {
		// The intake stream receives events from all publishers and things
		// FIXME: the format is already defined in the client.
		subj := natstransport.MakeSubject(vocab.MessageTypeEvent, "", "", "", "")

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

	return err
}

// Stop the server
func (srv *NatsMsgServer) Stop() {
	srv.ns.Shutdown()
}

// NewNatsMsgServer creates a new instance of the Hub NATS server for NKey authn.
func NewNatsMsgServer(
	cfg *natsmsgserver.NatsServerConfig, rolePermissions map[string][]msgserver.RolePermission) *NatsMsgServer {

	srv := &NatsMsgServer{Config: cfg,
		rolePermissions:    rolePermissions,
		servicePermissions: make(map[string][]msgserver.RolePermission, 0),
	}
	return srv
}
