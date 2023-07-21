package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/core/config"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

// Default permissions for new users
var defaultPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"guest.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"guest.>", "_INBOX.>"}},
}

// Authn service permissions
var authnPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"things.authn.*.action.>"}},
}

// Authz service permissions
var authzPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"_INBOX.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"things.authz.*.action.>"}},
}

//var adminPermissions = &server.Permissions{
//	Publish:   &server.SubjectPermission{Allow: []string{">"}},
//	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
//}

// HubNatsServer runs an embedded NATS server using callout for authentication.
// This configures the server to use a separate callout account
// This configures the server for publishing  provides a static configuration for the server for authn, authz, directory, and history streaming
type HubNatsServer struct {
	// hub server configuration
	cfg *config.ServerConfig

	// The application account all generated tokens belong to.
	appAcctKey nkeys.KeyPair
	appAccount *server.Account

	// The account that the callout handler uses
	calloutAccountKey  nkeys.KeyPair
	calloutAccountName string
	calloutUserKey     nkeys.KeyPair

	//hubAccount *server.Account
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	serverOpts *server.Options
	ns         *server.Server
	calloutSub *nats.Subscription

	// the handler to verify authentication requests, or nil to accept any
	verifyAuthn func(req *jwt.AuthorizationRequestClaims) error
}

// AddServiceKey adds an nkey to the static server config and exclude it from the callout.
// Intended to work around having implement nkey auth ourselves as it aint pretty.
// No pubsub restrictions are set.
// This takes effect immediately.
func (srv *HubNatsServer) AddServiceKey(nkey nkeys.KeyPair) error {
	// add static service nkeys
	nkeyPub, _ := nkey.PublicKey()
	//allow := fmt.Sprintf("things.%s.>",name)
	//p := &server.Permissions{
	//			Publish: &server.SubjectPermission{
	//				Allow: []string{allow},
	//			},
	//			Subscribe: []string{">"},
	//		}
	srv.serverOpts.Nkeys = append(srv.serverOpts.Nkeys, &server.NkeyUser{
		Nkey:    nkeyPub,
		Account: srv.appAccount,
		//Permissions: p,
	})
	srv.serverOpts.AuthCallout.AuthUsers = append(srv.serverOpts.AuthCallout.AuthUsers, nkeyPub)
	err := srv.ns.ReloadOptions(srv.serverOpts)
	return err
}

// ConnectInProc connects to the server in-process using nkey. Intended for the core services.
// The client NKey must have been added using AddServiceKey.
func (srv *HubNatsServer) ConnectInProc(clientID string, clientKey nkeys.KeyPair) (*nats.Conn, error) {
	// The handler to sign the server issued challenge
	sigCB := func(nonce []byte) ([]byte, error) {
		return clientKey.Sign(nonce)
	}
	// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.caCert != nil {
		caCertPool.AddCert(srv.caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.caCert == nil,
	}
	clientKeyPub, _ := clientKey.PublicKey()
	cl, err := nats.Connect(srv.ns.ClientURL(), // don't need a URL for in-process connection
		nats.Name(clientID), // connection name for validation
		nats.Secure(tlsConfig),
		nats.Nkey(clientKeyPub, sigCB),
		nats.Timeout(time.Minute),
		nats.InProcessServer(srv.ns),
	)

	return cl, err
}

// create a new user jwt token
//
//	clientID is the client's login ID which is added as the token Name
//	clientPub is the client's public key
func (srv *HubNatsServer) createUserJWTToken(clientID string, clientPub string) (newToken string, err error) {
	validitySec := 3600 // the testing validity

	// build an jwt response; user_nkey (clientPub) is the subject
	uc := jwt.NewUserClaims(clientPub)

	uc.Name = clientID
	// Issuer is set to the account linked in the callout config.
	// Using IssuerAccount in server mode is unnecessary and fails with:
	//   "Error non operator mode account %q: attempted to use issuer_account"
	// not sure why this is an issue...
	//uc.IssuerAccount,_ = svr.calloutAcctKey.PublicKey()
	uc.IssuedAt = time.Now().Unix()
	// note: doc says aud should be public account key of the user
	// however, auth_callout.go does a lookup by name instead
	// see also: https://github.com/nats-io/nats-server/issues/4313
	//uc.Audience, _ = svr.appAccountKey.PublicKey()
	uc.Audience = srv.cfg.AppAccountName
	uc.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()

	//uc.UserPermissionLimits = *limits // todo

	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) != 0 {
		err = fmt.Errorf("validation error: %w", vr.Errors()[0])
	}
	newToken, err = uc.Encode(srv.calloutAccountKey)
	return newToken, err
}

// createSignedResponse generates a callout response
//
//	userPub is the public key of the user from the request
//	serverPub is the public key of the signing server (server.ID from the req)
//	userJWT is the generated user jwt token to include in the response
//	err set if the response indicates an auth error
func (srv *HubNatsServer) createSignedResponse(
	userPub string, serverPub string, userJWT string, rerr error) ([]byte, error) {

	calloutAcctPub, err := srv.calloutAccountKey.PublicKey()
	// create and send the response
	respClaims := jwt.NewAuthorizationResponseClaims(userPub)
	respClaims.Audience = serverPub
	respClaims.Issuer = calloutAcctPub
	respClaims.Jwt = userJWT
	if rerr != nil {
		respClaims.Error = rerr.Error()
	}

	respClaims.IssuedAt = time.Now().Unix()
	respClaims.Expires = time.Now().Add(time.Duration(100) * time.Second).Unix()
	// signingKey must be the issuer keys
	response, err := respClaims.Encode(srv.calloutAccountKey)
	return []byte(response), err
}

// callout handler that issues an authorization token
func (srv *HubNatsServer) handleCallOutReq(msg *nats.Msg) {

	reqClaims, err := jwt.DecodeAuthorizationRequestClaims(string(msg.Data))
	if err != nil {
		return
	}

	slog.Info("received authcallout", slog.String("userID", reqClaims.ConnectOptions.Name))
	userNKeyPub := reqClaims.UserNkey
	serverID := reqClaims.Server
	client := reqClaims.ClientInformation
	connectOpts := reqClaims.ConnectOptions
	tlsInfo := reqClaims.TLS

	_ = client
	_ = tlsInfo

	err = nil
	if srv.verifyAuthn != nil {
		err = srv.verifyAuthn(reqClaims)
	} else {
		err = fmt.Errorf("authcallout invoked without a verifier")
	}
	newToken := ""
	if err == nil {
		clientID := connectOpts.Name // client identification
		newToken, err = srv.createUserJWTToken(clientID, userNKeyPub)
	}
	resp, err := srv.createSignedResponse(userNKeyPub, serverID.ID, newToken, err)

	err = msg.Respond(resp)
	_ = err
}

// SetAuthnVerifier sets a new authentication verifier method
// This will be invoked by the callout auth handler.
// Install before starting the server.
func (srv *HubNatsServer) SetAuthnVerifier(authnVerifier func(request *jwt.AuthorizationRequestClaims) error) {
	srv.verifyAuthn = authnVerifier
}

// Start the NATS server
// This creates a auth callout and application account.
func (srv *HubNatsServer) Start() (clientURL string, err error) {

	// Configure and Start the server
	// Two accounts and several static NKeys for core clients
	appAccountPub, _ := srv.appAcctKey.PublicKey()
	srv.appAccount = server.NewAccount(srv.cfg.AppAccountName)
	srv.appAccount.Nkey = appAccountPub

	// the callout authentication handler runs in a separate internal account
	srv.calloutAccountKey, _ = nkeys.CreateAccount()
	srv.calloutAccountName = "CalloutAccount"
	calloutAccountPub, _ := srv.calloutAccountKey.PublicKey()
	calloutAccount := server.NewAccount(srv.calloutAccountName)
	calloutAccount.Nkey = calloutAccountPub
	srv.calloutUserKey, _ = nkeys.CreateUser()
	calloutUserPub, _ := srv.calloutUserKey.PublicKey()

	// run the server with
	srv.serverOpts = &server.Options{
		Host:      srv.cfg.Host,
		Port:      srv.cfg.Port,
		JetStream: true,
		//JetStreamMaxMemory: 1*1024*1024*1024,

		Accounts: []*server.Account{calloutAccount, srv.appAccount},
		AuthCallout: &server.AuthCallout{
			Issuer:    calloutAccountPub,
			Account:   srv.calloutAccountName,
			AuthUsers: []string{calloutUserPub},
		},

		// Predefined internal users for
		Nkeys: []*server.NkeyUser{
			// callout user in its own account for use by callout handler, authn and authz
			{
				Nkey:    calloutUserPub,
				Account: calloutAccount,
			},
		},
		Users: []*server.User{},
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
		srv.serverOpts.TLSTimeout = 1000 // for debugging auth
		srv.serverOpts.TLSConfig = tlsConfig
	}
	// start nats
	srv.ns, err = server.NewServer(srv.serverOpts)
	srv.ns.ConfigureLogger()

	//if err == nil {
	//	srv.hubAccount = srv.ns.GlobalAccount()
	//}
	if err == nil {
		go srv.ns.Start()
		if !srv.ns.ReadyForConnections(3 * time.Second) {
			err = errors.New("nats: not ready for connection")
		} else {
			clientURL = srv.ns.ClientURL()
		}
	}

	if err == nil {
		// start discovery
	}

	// install the callout handler
	// todo: use the in-process connection provider
	// should use a different client key from signing key. moot point after using inproc
	nc, err := srv.ConnectInProc("authn-callout", srv.calloutUserKey)
	if err != nil {
		return clientURL, err
	}
	srv.calloutSub, err = nc.Subscribe(server.AuthCalloutSubject, srv.handleCallOutReq)
	return clientURL, err
}

// Stop the server
func (srv *HubNatsServer) Stop() {
	_ = srv.calloutSub.Unsubscribe()
	srv.ns.Shutdown()
}

// NewHubNatsServer creates a new instance of the Hub NATS server
// The given configuration is optional. The server will run with production settings out of the box.
//
// Use SetAuthnVerifier function to install the callout authn handler.
//
//	cfg contains an initialized server configuration for use as hiveot hub
//	appAcctKey is the application account nkey
//	serverCert tls certificate to run the server with
//	caCert CA the server runs with
func NewHubNatsServer(cfg *config.ServerConfig, appAcctKey nkeys.KeyPair, serverCert *tls.Certificate, caCert *x509.Certificate) *HubNatsServer {

	srv := &HubNatsServer{
		cfg:        cfg,
		appAcctKey: appAcctKey,
		caCert:     caCert,
		serverCert: serverCert,
	}
	return srv
}
