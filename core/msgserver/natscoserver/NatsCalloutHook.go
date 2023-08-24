package natscoserver

import (
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

// This is not ready for use. Callout is undocumented and integrating subject authorization
// is a still a question.
// Does callout expect permissions in a generated JWT? Does that mean that on each connection a JWT token
// must be generated? Does this effect performance compared to regular JWT which only needs to verify the token?

// NatsCalloutHook provides an easy-to-use hook to enable callout in the NATS server.
// It combines the various parts needed to function, such as account, nkeys, auth request
// handler and auth response creator.
// Intended for handling callouts in server mode. Not for use in operator mode.
// (that is a lot of work just to get your own authn handler called)
type NatsCalloutHook struct {
	// the server to configure
	serverOpts *server.Options

	// The account used to issue the tokens.
	// Due to a limitation of nats callout, this must be the same as the callout account itself.
	// See also https://github.com/nats-io/nats-server/issues/4335
	issuerAccountName string
	issuerAccountKey  nkeys.KeyPair
	// the callout account is the account used to connect the callout handler
	calloutAccountKey nkeys.KeyPair

	// the nats connection used to subscribe and receive callout requests
	nc *nats.Conn

	// the application handler to verify authentication requests
	// this is doing the actual work
	authnVerifier func(*jwt.AuthorizationRequestClaims) error
}

// createSignedResponse generates a callout response
//
//	userPub is the public key of the user from the request
//	serverPub is the public key of the signing server (server.ID from the req)
//	userJWT is the generated user jwt token to include in the response
//	err set if the response indicates an auth error
func (chook *NatsCalloutHook) createSignedResponse(
	userPub string, serverPub string, userJWT string, rerr error) ([]byte, error) {

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

// createUserJWTToken returns a new user jwt token using the issuer account
// This is invoked after the verification callback returns with success
//
// The token is signed by the issuer account. In server mode this must be the same
// account as the account the callout client belongs to.
//
//	clientID is the user's login/connect ID which is added as the token ID
//	clientPub is the users's public key which goes into the subject field of the jwt token
func (chook *NatsCalloutHook) createUserJWTToken(clientID string, clientPub string) (newToken string, err error) {
	validitySec := 3600 // the testing validity

	// build an jwt response; user_nkey (clientPub) is the subject
	uc := jwt.NewUserClaims(clientPub)

	uc.Name = clientID
	// Note: In server mode do not set issuer account. This is for operator mode only.
	// Using IssuerAccount in server mode is unnecessary and fails with:
	//   "Error non operator mode account %q: attempted to use issuer_account"
	// not sure why this is an issue...
	//uc.IssuerAccount,_ = svr.calloutAcctKey.PublicKey()
	//uc.Issuer, _ = chook.appAcctKey.PublicKey()

	uc.IssuedAt = time.Now().Unix()

	// Note: in server mode 'aud' should contain the account name. In operator mode it expects
	// the account key.
	// see also: https://github.com/nats-io/nats-server/issues/4313
	//uc.Audience, _ = chook.appAcctKey.PublicKey()
	uc.Audience = chook.issuerAccountName
	uc.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()

	//uc.UserPermissionLimits = *limits // todo

	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) != 0 {
		err = fmt.Errorf("validation error: %w", vr.Errors()[0])
	}
	// encode sets the issuer field to the public key
	newToken, err = uc.Encode(chook.issuerAccountKey)
	//newToken, err = uc.Encode(chook.calloutAccountKey)
	return newToken, err
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
	client := reqClaims.ClientInformation
	connectOpts := reqClaims.ConnectOptions
	tlsInfo := reqClaims.TLS

	_ = client
	_ = tlsInfo

	if chook.authnVerifier != nil {
		err = chook.authnVerifier(reqClaims)
	} else {
		err = fmt.Errorf("authcallout invoked without a verifier")
	}
	if err != nil {
		// note: if the client isn't know the caller will not receive this error
		slog.Warn("Invalid authn", "err", err,
			slog.String("userID", reqClaims.ConnectOptions.Name))
		resp, _ := chook.createSignedResponse(userNKeyPub, serverID.ID, "", err)
		_ = msg.Respond(resp)
		return
	}
	// on success, create a user JWT token, signed by the application account key,
	// and put the token in a ResponseClaim, signed by the callout account key.
	// Note that in server mode these keys must be the same.
	newToken := ""
	clientID := connectOpts.Name // client identification
	newToken, err = chook.createUserJWTToken(clientID, userNKeyPub)

	resp, err := chook.createSignedResponse(userNKeyPub, serverID.ID, newToken, err)
	if err != nil {
		slog.Error("error creating signed response", "err", err)
		err = msg.Respond(nil)
		return
	}

	err = msg.Respond(resp)
	if err != nil {
		slog.Error("error sending response", "err", err)
		return
	}
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
	chook.serverOpts.NoAuthUser = ""
	// adopt existing nkeys

	calloutSub, err := chook.nc.Subscribe(server.AuthCalloutSubject, chook.handleCallOutReq)

	_ = calloutSub
	return err
}

// ConnectNatsCalloutHook create a new instance of the NATS callout hook
// for use with NKey based configuration options.
// This configures the server to use callout hooks and subscribes to requests
// using the given connection.
//
// Reload the server options for it to take effect.
//
//   - serverOpts is the server mode options struct to update
//   - issuerAccountName is the name of the account used to issue the JWT tokens
//   - issuerAccountKey is the key-pair of the account used to issue the JWT tokens
//   - nc is the nats connection to use
//   - authnVerifier is the callback handler to verify an authn request
func ConnectNatsCalloutHook(
	serverOpts *server.Options,
	issuerAccountName string,
	issuerAccountKey nkeys.KeyPair,
	nc *nats.Conn,
	authnVerifier func(request *jwt.AuthorizationRequestClaims) error,
) (*NatsCalloutHook, error) {

	hook := &NatsCalloutHook{
		serverOpts:        serverOpts,
		issuerAccountName: issuerAccountName,
		issuerAccountKey:  issuerAccountKey,
		calloutAccountKey: issuerAccountKey, // currently must be the issuer for server mode
		nc:                nc,
		authnVerifier:     authnVerifier,
	}

	err := hook.start()

	return hook, err
}
