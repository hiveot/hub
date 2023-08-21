package natsserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
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

// construct a permissions object for a client and its group memberships
func (srv *NatsNKeyServer) makePermissions(clientProf *authn.ClientProfile, groupsRole authz.RoleMap) *server.Permissions {
	subPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  nil,
	}
	pubPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  nil,
	}
	perm := &server.Permissions{
		Publish:   nil,
		Subscribe: nil,
		Response:  nil,
	}
	// all clients can use their inbox, using inbox prefix
	subInbox := "_INBOX." + clientProf.ClientID + ".>"
	subPerm.Allow = append(subPerm.Allow, subInbox)

	// services can pub actions and subscribe
	if clientProf.ClientType == authn.ClientTypeService {
		pubService := natshubclient.MakeSubject("", "", "action", ">")
		pubPerm.Allow = append(subPerm.Allow, pubService)
		subService := natshubclient.MakeSubject("", "", "event", ">")
		subPerm.Allow = append(subPerm.Allow, subService)
	}
	// users and services can subscribe to streams (groups)
	if clientProf.ClientType == authn.ClientTypeUser || clientProf.ClientType == authn.ClientTypeService {
		for groupName, role := range groupsRole {
			// group members can read from the stream
			readSubj := "$JS.API.CONSUMER.CREATE." + groupName
			subPerm.Allow = append(subPerm.Allow, readSubj)

			// TODO: operators and managers can publish actions for all things in the group
			// Can we use a stream publish that mapped back to the thing?
			// eg: {groupName}.{publisher}.{thing}.action.>
			// maps to things.{publisher}.{thing}.action.>
			// where the stream has a filter on all things added to the stream?
			if role == authz.GroupRoleOperator || role == authz.GroupRoleManager {
				actionSubj := groupName + ".*.*.action.>"
				pubPerm.Allow = append(pubPerm.Allow, actionSubj)
			}
		}
	}
	if clientProf.ClientType == authn.ClientTypeDevice {
		// devices can pub/sub on their own address and their inbox
		pubDevice := natshubclient.MakeSubject(clientProf.ClientID, "", "event", ">")
		pubPerm.Allow = append(subPerm.Allow, pubDevice)
		subDevice := natshubclient.MakeSubject(clientProf.ClientID, "", "action", ">")
		subPerm.Allow = append(subPerm.Allow, subDevice)
	}
	return perm
}

// ReloadClients loads the authn and authz from the stores and applies
// then to the static nats server config.
//
//	clients is a list of user, device and service identities
func (srv *NatsNKeyServer) ReloadClients(
	clients []authn.AuthnEntry, userGroupRoles map[string]authz.RoleMap) error {

	pwUsers := []*server.User{}
	nkeyUsers := []*server.NkeyUser{}

	// keep the core service that was added on server start
	coreServicePub, _ := srv.cfg.CoreServiceKP.PublicKey()
	nkeyUsers = append(nkeyUsers, &server.NkeyUser{
		Nkey:        coreServicePub,
		Permissions: nil, // unlimited access
		Account:     srv.cfg.appAcct,
	})
	// keep the 'unauthenticated' user
	// TODO: make this optional. provisioning should use a special provisioning user
	//pwUsers = append(pwUsers, &server.User{
	//	Username:    NoAuthUserID,
	//	Password:    "",
	//	Permissions: noAuthPermissions,
	//	Account:     srv.cfg.appAcct,
	//})

	// apply all clients
	for _, entry := range clients {
		clientRoles := userGroupRoles[entry.ClientID]
		userPermissions := srv.makePermissions(&entry.ClientProfile, clientRoles)

		if entry.PasswordHash != "" {
			pwUsers = append(pwUsers, &server.User{
				Username:    entry.ClientID,
				Password:    entry.PasswordHash,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
			})
		}

		if entry.PubKey != "" {
			// add an nkey entry
			nkeyUsers = append(nkeyUsers, &server.NkeyUser{
				Nkey:        entry.PubKey,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
			})
		}
	}
	srv.natsOpts.Users = pwUsers
	srv.natsOpts.Nkeys = nkeyUsers
	err := srv.ns.ReloadOptions(&srv.natsOpts)
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
