package testenv

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

// TestCalloutServer is an embedded NATS test messaging server using custom auth handler
// Issues this tries to solve:
//   - can't use password authentication when jwt is active
//   - can't use auth callout in operator mode
//   - is operator mode required?
type TestCalloutServer struct {
	account        *server.Account
	caCert         *x509.Certificate
	serverCert     *tls.Certificate
	ns             *server.Server
	calloutAcctKey nkeys.KeyPair
	calloutUserKey nkeys.KeyPair
	appAccountKey  nkeys.KeyPair
}

// create a new user authn token
//
//	clientPub is the client's public key
func (svr *TestCalloutServer) createNewAuthToken(clientPub string, clientID string) (newToken string, err error) {
	validitySec := 3600 // the testing validity

	// build an jwt response; user_nkey (clientPub) is the subject
	uc := jwt.NewUserClaims(clientPub)

	uc.Name = clientID
	//uc.IssuerAccount,_ = svr.calloutAcctKey.PublicKey()
	uc.IssuedAt = time.Now().Unix()
	// FIXME: doc says aud should be public account key of the user
	// however, auth_callout.go does a lookup by name instead
	//uc.Audience, _ = svr.appAccountKey.PublicKey()
	uc.Audience = "AppAccount"
	uc.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()

	//uc.UserPermissionLimits = *limits // todo

	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) != 0 {
		err = fmt.Errorf("validation error: %w", vr.Errors()[0])
	}
	newToken, err = uc.Encode(svr.calloutAcctKey)
	return newToken, err
}

// createSignedResponse generates a callout response
//
//	userPub is the public key of the user from the request
//	serverPub is the public key of the signing server (server.ID from the req)
//	userJWT is the generated user jwt token to include in the response
func (svr *TestCalloutServer) createSignedResponse(
	userPub string, serverPub string, userJWT string) (resp []byte, err error) {

	calloutAcctPub, err := svr.calloutAcctKey.PublicKey()
	// create and send the response
	respClaims := jwt.NewAuthorizationResponseClaims(userPub)
	respClaims.Audience = serverPub
	respClaims.Issuer = calloutAcctPub
	//respClaims.Error = "some error occurred"
	respClaims.Jwt = userJWT

	respClaims.IssuedAt = time.Now().Unix()
	respClaims.Expires = time.Now().Add(time.Duration(100) * time.Second).Unix()
	// signingKey must be the issuer keys
	response, err := respClaims.Encode(svr.calloutAcctKey)
	return []byte(response), err
}

// callout handler that issues an authorization token
func (svr *TestCalloutServer) handleCallOutReq(msg *nats.Msg) {

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

	//---
	// Verify the credentials in the request. As this is a test server we don't bother right now.
	//---
	clientID := connectOpts.Name // client identification
	newToken, err := svr.createNewAuthToken(userNKeyPub, clientID)
	resp, err := svr.createSignedResponse(userNKeyPub, server.ID, newToken)

	err = msg.Respond(resp)
	_ = err
}

// Create a client connection to this server for use by the callout handler
// TODO: Tom Anderson recommends using the in-process connection provider
// "That way you don't need to have an account for your server's non-nats part to do nats things."
//
//	https://pkg.go.dev/github.com/nats-io/nats.go#InProcessServer
//
// clientKey is the callout user NKey used to connect
func (svr *TestCalloutServer) connectToSelf(clientURL string, clientKey nkeys.KeyPair) (*nats.Conn, error) {
	// client connection
	caCertPool := x509.NewCertPool()
	if svr.caCert != nil {
		caCertPool.AddCert(svr.caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: svr.caCert == nil,
	}
	// The handler to sign the server issued challenge
	sigCB := func(nonce []byte) ([]byte, error) {
		return clientKey.Sign(nonce)
	}
	pubKey, _ := clientKey.PublicKey()
	nc, err := nats.Connect(clientURL,
		nats.Name("authn-service"), // connection name for logging
		nats.Secure(tlsConfig),
		nats.Nkey(pubKey, sigCB),
		nats.Timeout(time.Second*time.Duration(100)))
	return nc, err
}

// Start the server, listening on 127.0.0.1
// setup accounts provided with the given bundle
// this uses a single authnKey for singing the auth requests and logging in as the callout client
// not recommended and using an inproc connection is better
func (svr *TestCalloutServer) Start() (clientURL string, err error) {

	appAccountKeyPub, _ := svr.appAccountKey.PublicKey()
	calloutAcctKeyPub, _ := svr.calloutAcctKey.PublicKey()
	calloutUserKeyPub, _ := svr.calloutUserKey.PublicKey()

	appAccount := server.NewAccount("AppAccount")
	appAccount.Nkey = appAccountKeyPub

	calloutAccount := server.NewAccount("CalloutAccount")
	calloutAccount.Nkey = calloutAcctKeyPub

	opts := &server.Options{
		ServerName: "HiveOT Hub",
		Host:       "127.0.0.1", // must match the address on the generated cert
		Port:       9998,        // some random test port that doesn't interfere

		AuthTimeout: 10, // undocumented. what unit is this?
		Accounts:    []*server.Account{calloutAccount, appAccount},

		AuthCallout: &server.AuthCallout{
			Issuer: calloutAcctKeyPub,
			//Account:   "CalloutAccount",
			AuthUsers: []string{calloutUserKeyPub},
			//XKey:      "",
		},
		JetStream:          true,
		JetStreamMaxMemory: 10 * 1024 * 1024,
		// the callout user the calloutUser NKey to login
		Nkeys: []*server.NkeyUser{
			{
				Nkey:        calloutUserKeyPub,
				Permissions: adminPermissions,
				//Account:     calloutAccount,
			},
		},
	}
	if svr.caCert != nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(svr.caCert)
		clientCertList := []tls.Certificate{*svr.serverCert}
		tlsConfig := &tls.Config{
			ClientCAs:    caCertPool,
			RootCAs:      caCertPool,
			Certificates: clientCertList,
			ClientAuth:   tls.VerifyClientCertIfGiven,
		}
		opts.TLSConfig = tlsConfig
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return "", err
	}
	svr.ns = ns
	svr.account = svr.ns.GlobalAccount()
	go ns.Start()
	if !ns.ReadyForConnections(3 * time.Second) {
		panic("not ready for connection")
	}
	clientURL = ns.ClientURL()

	// install the callout handler
	// todo: use the in-process connection provider
	// should use a different client key from signing key. moot point after using inproc
	nc, err := svr.connectToSelf(clientURL, svr.calloutUserKey)
	if err != nil {
		return clientURL, err
	}
	sub, err := nc.Subscribe(server.AuthCalloutSubject, svr.handleCallOutReq)
	_ = sub
	return clientURL, err
}

func (svr *TestCalloutServer) Stop() {
	if svr.ns != nil {
		svr.ns.Shutdown()
	}
}

// NewTestCalloutServer create a new test server instance
//
//	serverCert optional cert for 127.0.0.1
//	calloutAcctKey key used as the callout account to sign auth request
func NewTestCalloutServer(
	appAccountKey nkeys.KeyPair,
	calloutAcctKey nkeys.KeyPair,
	calloutUserKey nkeys.KeyPair,
	serverCert *tls.Certificate, caCert *x509.Certificate) *TestCalloutServer {
	ts := &TestCalloutServer{
		serverCert:     serverCert,
		caCert:         caCert,
		calloutAcctKey: calloutAcctKey,
		calloutUserKey: calloutUserKey,
		appAccountKey:  appAccountKey,
	}
	return ts
}
