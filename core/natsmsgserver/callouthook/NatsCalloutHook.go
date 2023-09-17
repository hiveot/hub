package callouthook

import (
	"fmt"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"sync/atomic"
	"time"
)

// This is not ready for use. Callout is undocumented and not released for production.
// TODO: when applying fixed NKEY and regular users, add them to the callout exclude list.
// Ugh, 750 lines of code total to deal with NATS auth. All I want for xmas is jwt working with password auth.

// NatsCalloutHook provides an easy-to-use hook to enable callout in the NATS server.
// It combines the various parts needed to function, such as account, nkeys, auth request
// handler and auth response creator.
// Intended for handling callouts in server mode. Not for use in operator mode.
// (that is a lot of work just to get your own authn handler called)
type NatsCalloutHook struct {
	// the server to configure
	serverOpts *server.Options

	successCount atomic.Int32 // nr of successful callout requests
	failCount    atomic.Int32 // nr of failed callout requests

	// The account used to issue the tokens.
	// Due to a limitation of nats callout, this must be the same as the callout account itself.
	// See also https://github.com/nats-io/nats-server/issues/4335
	issuerAccountName string
	issuerAccountKey  nkeys.KeyPair
	// the callout account is the account used to connect the callout handler
	calloutAccountKey nkeys.KeyPair

	// the nats connection used to subscribe and receive callout requests
	nc *nats.Conn

	msgServer *natsmsgserver.NatsMsgServer
	// token factory for a known client with the given public key
	//createJWTToken func(clientID string, pubKey string) (newToken string, err error)

	// the application handler to verify authentication requests
	// this returns the client that is connecting
	authnVerifier func(*jwt.AuthorizationRequestClaims) (clientID string, err error)
}

// createSignedResponse generates a callout response
//
//	userPub is the public key of the user from the request (not the signin user)
//	serverPub is the public key of the signing server (server.ID from the req)
//	userJWT is the generated user jwt token to include in the response
//	err set if the response indicates an auth error
func (chook *NatsCalloutHook) createSignedResponse(
	userPub string, serverPub string, userJWT string, rerr error) ([]byte, error) {

	//slog.Info("createSignedResponse", "pub", userPub)
	// create and send the response
	respClaims := jwt.NewAuthorizationResponseClaims(userPub)
	respClaims.Audience = serverPub
	//respClaims.Audience = chook.cfg.AppAccountName
	respClaims.Jwt = userJWT
	if rerr != nil {
		respClaims.Error = rerr.Error()
	}

	// TODO: add token validity config
	respClaims.IssuedAt = time.Now().Unix()
	respClaims.Expires = time.Now().Add(time.Duration(100) * time.Second).Unix()
	response, err := respClaims.Encode(chook.calloutAccountKey)
	return []byte(response), err
}

// GetCounters return the number of success and failed callout requests
func (chook *NatsCalloutHook) GetCounters() (int, int) {
	successCount := int(chook.successCount.Load())
	failCount := int(chook.failCount.Load())
	return successCount, failCount
}

// callout handler invoked by the callout subscription.
// This invokes the custom authentication callback to verify the users' authenticity,
// then on success it creates a JWT token and submits an AuthorizationResultClaim response
// that contains the user JWT.
func (chook *NatsCalloutHook) handleCallOutReq(msg *nats.Msg) {

	reqClaims, err := jwt.DecodeAuthorizationRequestClaims(string(msg.Data))
	if err != nil {
		return
	}

	slog.Info("handleCallOutReq", slog.String("userID", reqClaims.ConnectOptions.Name))
	userNKeyPub := reqClaims.UserNkey
	serverID := reqClaims.Server
	clientID := ""

	if chook.authnVerifier != nil {
		clientID, err = chook.authnVerifier(reqClaims)
	} else {
		err = fmt.Errorf("handleCallOutReq: invoked without a verifier")
	}
	if err != nil || clientID == "" {
		chook.failCount.Add(1)
		// note: if the client isn't know the caller will not receive this error
		slog.Warn("handleCallOutReq: Invalid authn", "err", err,
			slog.String("clientID", clientID),
			slog.String("reqClaims.Name", reqClaims.Name))
		resp, _ := chook.createSignedResponse(userNKeyPub, serverID.ID, "", err)
		_ = msg.Respond(resp)
		return
	}
	// on success, create a user JWT token, signed by the application account key,
	// and put the token in a ResponseClaim, signed by the callout account key.
	// Note that in server mode these keys must be the same.
	newToken := ""

	// FIXME: where is ClientInformation documented?
	//clientType := reqClaims.Tags.ClientInformation.Tags["clientType"]

	// note that callouts generates a new key on the fly and expects the token
	// to use this key. Why is unknown...
	authInfo, err := chook.msgServer.GetClientAuth(clientID)
	authInfo.PubKey = userNKeyPub
	if err == nil {
		newToken, err = chook.msgServer.CreateJWTToken(authInfo)
	}
	resp, err := chook.createSignedResponse(userNKeyPub, serverID.ID, newToken, err)
	if err != nil {
		chook.failCount.Add(1)
		slog.Error("error creating signed response", "err", err)
		err = msg.Respond(nil)
		return
	}

	err = msg.Respond(resp)
	if err != nil {
		chook.failCount.Add(1)
		slog.Error("error sending response", "err", err)
		return
	}
	chook.successCount.Add(1)
	_ = err
}

// start configures the server to use callout based authentication.
// The server config must be reloaded by the caller for this to take effect.
//
// This adds a user key for the callout handler to connect to the server.
//
//	accountName is the account to run under
//	accountKey is the key used to connect to the server and sign tokens
//	authnVerifier verifies authentication requests.
func (chook *NatsCalloutHook) start() error {

	// create an internal nkey for the callout handler and add it to the config
	issuerAccountPub, _ := chook.calloutAccountKey.PublicKey()

	// exclude the existing nkeys (which includes the callout handler nkey)
	ignoreKeys := []string{}
	for _, nk := range chook.serverOpts.Nkeys {
		ignoreKeys = append(ignoreKeys, nk.Nkey)
	}
	chook.serverOpts.AuthCallout = &server.AuthCallout{
		Issuer:    issuerAccountPub,
		Account:   chook.issuerAccountName,
		AuthUsers: ignoreKeys,
		XKey:      "",
	}
	// remove users as password will be handled by callout.
	// Also, callout errors when handler has a nkey connection which doesn't
	// exist in the users section. (probably a bug) auth.go:288
	chook.serverOpts.Users = []*server.User{}
	//chook.serverOpts.NoAuthUser = ""
	// adopt existing nkeys

	calloutSub, err := chook.nc.Subscribe(server.AuthCalloutSubject, chook.handleCallOutReq)

	_ = calloutSub
	return err
}

// EnableNatsCalloutHook create an instance of the NATS callout hook
// for use with NKey based configuration options.
// This configures the server to use callout hooks and subscribes to requests
// using the given connection.
//
// Reload the server options for it to take effect.
//
// NOTE: If password users are defined, server will report an error "Authorization callout user %q not valid".
// This is because auth.go:291 which does a check but skips nkeys if any user is defined in natsOpts.
// As this is a check only, it can be ignored. However it would be better to let the callout validator to handle
// the password auth and not apply these in nats users options.
//
//   - serverOpts is the server mode options struct to update
//   - issuerAccountName is the name of the account used to issue the JWT tokens
//   - issuerAccountKey is the key-pair of the account used to issue the JWT tokens
//   - nc is the nats connection to use
//   - authnVerifier is the callback handler to verify an authn request
func EnableNatsCalloutHook(
	srv *natsmsgserver.NatsMsgServer,
	// authnVerifier func(request *jwt.AuthorizationRequestClaims) (clientID string, err error),
) (*NatsCalloutHook, error) {

	authnVerifier := NewNatsCoVerifier(srv, srv.Config.CaCert)

	// Ideally the callout handler uses a separate callout account.
	// Apparently this isn't allowed so it runs in the application account.
	nc, err := srv.ConnectInProcNC("callout", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to connect callout handler: %w", err)
	}
	// tokenizer is needed to create a JWT auth token after verification succeeds
	//tokenizer := NewNatsJWTTokenizer(srv.cfg.AppAccountName, srv.cfg.AppAccountKP)

	hook := &NatsCalloutHook{
		serverOpts:        &srv.NatsOpts,
		issuerAccountName: srv.Config.AppAccountName,
		issuerAccountKey:  srv.Config.AppAccountKP,
		calloutAccountKey: srv.Config.AppAccountKP, // must be the issuer for server mode
		nc:                nc,
		msgServer:         srv,
		//createJWTToken:    srv.CreateJWTToken,
		authnVerifier: authnVerifier.VerifyAuthnReq,
	}

	err = hook.start()

	return hook, err
}
