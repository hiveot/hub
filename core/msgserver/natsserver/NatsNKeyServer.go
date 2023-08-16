package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"time"
)

// NatsNKeyServer runs an embedded NATS server using nkeys for authentication.
type NatsNKeyServer struct {
	cfg      *NatsServerConfig
	natsOpts server.Options
	ns       *server.Server
	// enable callout authn with EnableCalloutHandler. nil to just use nkeys
	chook *NatsCalloutHook
}

// ApplyAuthn applies a new list of users to the static configuration
func (srv *NatsNKeyServer) ApplyAuthn(clients []authn.IAuthnUser) error {
	return fmt.Errorf("todo")
}

// AddDevice adds a IoT device authn key to the app account and reloads the options.
// Devices can pub/sub on their own subject, eg: things.{deviceID}.>
func (srv *NatsNKeyServer) AddDevice(deviceID string, deviceKeyPub string) error {
	// if callout has been activated then exclude the key from invoking callout
	if srv.natsOpts.AuthCallout != nil {
		srv.natsOpts.AuthCallout.AuthUsers = append(srv.natsOpts.AuthCallout.AuthUsers, deviceKeyPub)
	}

	appAcct, err := srv.ns.LookupAccount(srv.cfg.AppAccountName)
	if err != nil {
		return fmt.Errorf("missing app account: %w", err)
	}
	srv.natsOpts.Nkeys = append(srv.natsOpts.Nkeys, &server.NkeyUser{
		Nkey:    deviceKeyPub,
		Account: appAcct,
	})
	err = srv.ns.ReloadOptions(&srv.natsOpts)
	return err
}

// AddService adds a core service authn key to the app account and reloads the options.
// Services can pub/sub to all things subjects
func (srv *NatsNKeyServer) AddService(serviceID string, serviceKeyPub string) error {
	// if callout has been activated then exclude the key from invoking callout
	if srv.natsOpts.AuthCallout != nil {
		srv.natsOpts.AuthCallout.AuthUsers = append(srv.natsOpts.AuthCallout.AuthUsers, serviceKeyPub)
	}

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

// AddUser adds a user login/pw to the app account and reloads the options.
// Users can pub to inboxes
func (srv *NatsNKeyServer) AddUser(userID string, password string, userKeyPub string) error {
	// if callout has been activated then exclude the key from callout
	if srv.natsOpts.AuthCallout != nil && userKeyPub != "" {
		srv.natsOpts.AuthCallout.AuthUsers = append(srv.natsOpts.AuthCallout.AuthUsers, userKeyPub)
	}

	appAcct, err := srv.ns.LookupAccount(srv.cfg.AppAccountName)
	if err != nil {
		return fmt.Errorf("missing app account: %w", err)
	}
	if userKeyPub != "" {
		srv.natsOpts.Nkeys = append(srv.natsOpts.Nkeys, &server.NkeyUser{
			Nkey:    userKeyPub,
			Account: appAcct,
			// todo: permissions
		})
	}
	if password != "" {
		srv.natsOpts.Users = append(srv.natsOpts.Users, &server.User{
			Username:    userID,
			Password:    password,
			Permissions: nil, // TODO
			Account:     appAcct,
		})
	}
	err = srv.ns.ReloadOptions(&srv.natsOpts)
	return err
}

// UpdateKey changes the public key of a user login and reload options
// This fails if oldKey doesn't exist. The caller should have added it first to ensure proper permissions
// This returns an error if the old key is not found
//
// WARNING: changing the public key of the connected account can cause an authentication failure when
// sending the reply. The workaround is to delay the reload.
func (srv *NatsNKeyServer) UpdateKey(oldKey string, newKey string) error {
	for _, n := range srv.natsOpts.Nkeys {
		if n.Nkey == oldKey {
			n.Nkey = newKey
			// sending a reply after changing the key of the caller causes an authentication error
			// therefore apply after returning
			go func() {
				_ = srv.ns.ReloadOptions(&srv.natsOpts)
			}()
			return nil
		}
	}
	return fmt.Errorf("can't update key  '%s' is not found", oldKey)
}

// UpdatePassword changes the password of a user login and reload options
// This returns an error if the user is not found
func (srv *NatsNKeyServer) UpdatePassword(userID string, password string) error {
	for _, u := range srv.natsOpts.Users {
		if u.Username == userID {
			u.Password = password
			err := srv.ns.ReloadOptions(&srv.natsOpts)
			return err
		}
	}
	return fmt.Errorf("can't update password as user '%s' is not found", userID)
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
	if err == nil {
		js, err2 := cl.JetStream()
		err = err2
		_ = js
	}
	return cl, err
}

// EnableCalloutHandler reconfigures the server for external callout authn
// The authn callout handler will issue tokens for the application account.
// Invoke this after successfully starting the server
func (srv *NatsNKeyServer) EnableCalloutHandler(
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

// Start the NATS server with the given configuration
//
//	cfg.Setup must have been called first.
func (srv *NatsNKeyServer) Start(cfg *NatsServerConfig) (clientURL string, err error) {

	srv.cfg = cfg
	srv.natsOpts, err = cfg.CreateNatsNKeyOptions()
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

	// how to enable jetstream for account?

	// add the core service account
	coreServicePub, _ := srv.cfg.CoreServiceKP.PublicKey()
	err = srv.AddService("core-service", coreServicePub)
	if err != nil {
		return clientURL, err
	}
	// app account must have JS enabled
	ac, _ := srv.ns.LookupAccount(srv.cfg.AppAccountName)
	err = ac.EnableJetStream(nil) //use defaults
	if err != nil {
		return clientURL, fmt.Errorf("can't enable JS for app account: %w", err)
	}

	hasJS := ac.JetStreamEnabled()
	if !hasJS {
		return clientURL, fmt.Errorf("JS not enabled for app account '%s'", srv.cfg.AppAccountName)
	}

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
