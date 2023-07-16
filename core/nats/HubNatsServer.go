package nats

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

// Default permissions for new users
var defaultPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"guest.>"}, Deny: []string{">"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"guest.>", "_INBOX.>"}, Deny: []string{">"}},
}

//var adminPermissions = &server.Permissions{
//	Publish:   &server.SubjectPermission{Allow: []string{">"}},
//	Subscribe: &server.SubjectPermission{Allow: []string{">"}},
//}

// HubNatsServer runs an embedded NATS server using callout for authentication.
// This configures the server to use a separate callout account
// This configures the server for publishing  provides a static configuration for the server for authn, authz, directory, and history streaming
type HubNatsServer struct {
	// hostname must match that of the server certificate
	hostName   string
	port       int
	caCert     *x509.Certificate
	serverCert *tls.Certificate
	// message storage directory
	storeDir string

	// The application account all generated tokens belong to.
	appAccountKey  nkeys.KeyPair
	appAccountName string
	// The account that the callout handler belongs to
	calloutAccountKey  nkeys.KeyPair
	calloutAccountName string
	// The callout handler client key
	calloutUserKey nkeys.KeyPair

	//hubAccount *server.Account
	serverOpts *server.Options
	ns         *server.Server
	calloutSub *nats.Subscription

	// the handler to verify authentication requests, or nil to accept any
	verifyAuthn func(req *jwt.AuthorizationRequestClaims) error
}

// Connect to the server in-process. Intended for the callout client.
func (srv HubNatsServer) connectToServer(clientKey nkeys.KeyPair) (*nats.Conn, error) {
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
	cl, err := nats.Connect("", // don't need a URL for in-process connection
		nats.Name("callout-client"), // connection name for logging
		nats.Secure(tlsConfig),
		nats.Nkey(clientKeyPub, sigCB),
		nats.InProcessServer(srv.ns),
	)

	return cl, err
}

// create a new user authn token
//
//	clientID is the client's login ID which is added as the token Name
//	clientPub is the client's public key
func (srv *HubNatsServer) createNewAuthToken(clientID string, clientPub string) (newToken string, err error) {
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
	// FIXME: doc says aud should be public account key of the user
	// however, auth_callout.go does a lookup by name instead
	// see also: https://github.com/nats-io/nats-server/issues/4313
	//uc.Audience, _ = svr.appAccountKey.PublicKey()
	uc.Audience = srv.appAccountName
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
	server := reqClaims.Server
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
		newToken, err = srv.createNewAuthToken(clientID, userNKeyPub)
	}
	resp, err := srv.createSignedResponse(userNKeyPub, server.ID, newToken, err)

	err = msg.Respond(resp)
	_ = err
}

// SetAuthnVerifier sets a new authentication verifier method
// Intended for testing
func (srv *HubNatsServer) SetAuthnVerifier(authnVerifier func(request *jwt.AuthorizationRequestClaims) error) {
	srv.verifyAuthn = authnVerifier
}

// Start the NATS server using
//
// This configures an account and admin user
//
//	host to listen on or "" for all interfaces
//	port to listen or 0 for default port
//	serverCert is the CA signed server certificate for the given host
//	caCert is the certificate of the CA that signed the server crt
func (srv *HubNatsServer) Start() (clientURL string, err error) {

	// Configure and Start the server
	appAccountPub, _ := srv.appAccountKey.PublicKey()
	appAccount := server.NewAccount(srv.appAccountName)
	appAccount.Nkey = appAccountPub

	// the callout account handler runs in a separate internal account
	srv.calloutAccountKey, _ = nkeys.CreateAccount()
	srv.calloutAccountName = "CalloutAccount"
	calloutAccountPub, _ := srv.calloutAccountKey.PublicKey()
	calloutAccount := server.NewAccount(srv.calloutAccountName)
	calloutAccount.Nkey = calloutAccountPub

	srv.calloutUserKey, _ = nkeys.CreateUser()
	calloutUserPub, _ := srv.calloutUserKey.PublicKey()

	// run the server with
	srv.serverOpts = &server.Options{
		Host:      srv.hostName,
		Port:      srv.port,
		JetStream: true,
		//JetStreamMaxMemory: 1*1024*1024*1024,

		Accounts: []*server.Account{calloutAccount, appAccount},
		AuthCallout: &server.AuthCallout{
			Issuer:    calloutAccountPub,
			Account:   srv.calloutAccountName,
			AuthUsers: []string{calloutUserPub},
		},

		// Predefined internal users for internal the callout user is the only predefined user
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
		srv.serverOpts.TLSTimeout = 10000 // for debugging auth
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
	//srv.ns.Reload()
	// start JetStream
	//if err == nil {
	//	jsConfig := server.JetStreamConfig{
	//		MaxMemory:  1 * 1024 * 1024 * 1024,
	//		MaxStore:   0,
	//		StoreDir:   "/tmp/hiveot/store",
	//		Domain:     "",
	//		CompressOK: false,
	//	}
	//	err = srv.ns.EnableJetStream(&jsConfig)
	//}
	if err == nil {
		// start discovery
	}

	// install the callout handler
	// todo: use the in-process connection provider
	// should use a different client key from signing key. moot point after using inproc
	nc, err := srv.connectToServer(srv.calloutUserKey)
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
//
// appAcctName and key are optional. A default will be generated on start.
// If not provided then a new account key will be generated on restart.
// While great for testing, it also invalidates all issued keys.
//
//		appAcctName is the name of the hub's account. Default is 'hiveot-{hostname}'
//	 appAcctKey is the key to protect the application account with.
//		storeDir directory to store jetstream data
//		hostName to match the server certificate
//		port to listen on. 0 for default (4222)
//		serverCert TLS certificate to present to clients
//		caCert the server is signed by
//	 verifyer function that verifies the credentials
//
// storeDir is the location for message storage.
func NewHubNatsServer(
	appAcctName string,
	appAcctKey nkeys.KeyPair,
	storeDir string,
	hostName string,
	port int,
	serverCert *tls.Certificate, caCert *x509.Certificate,
	verifyAuthn func(authReq *jwt.AuthorizationRequestClaims) error,
) *HubNatsServer {

	if appAcctName == "" {
		appAcctName = "hiveot-" + hostName
	}
	if appAcctKey == nil {
		appAcctKey, _ = nkeys.CreateAccount()
	}
	if storeDir == "" {
		storeDir = "/tmp/nats/jetstream"
	}
	srv := &HubNatsServer{
		hostName:       hostName,
		port:           port,
		caCert:         caCert,
		serverCert:     serverCert,
		appAccountKey:  appAcctKey,
		appAccountName: appAcctName,
		storeDir:       storeDir,
		verifyAuthn:    verifyAuthn,
	}
	return srv
}
