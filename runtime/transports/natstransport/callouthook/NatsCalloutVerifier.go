package callouthook

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/runtime/transports/natstransport/service"
	"github.com/nats-io/jwt/v2"
	"log/slog"
)

// NatsCalloutVerifier provides client verification using callout hooks.
//
// callout -> handler -> tokenizer -> verify ?
// auth -> server -> createToken -> tokenizer -> create ? needs authz ?
//
// This support authentication using password, nkey, certificate, and jwt token
// To use, provide 'VerifyAuthnReq' to EnableNatsCalloutHook(), which determines
// the authn method to use.
type NatsCalloutVerifier struct {
	msgServer *service.NatsMsgServer
	caCert    *x509.Certificate
}

// VerifyClientCert checks that a client certificate is used to connect and is
// that of the claimed clientID.
// There is no documentation on how to use this so this uses the
// claims.TLS.VerifiedChains field, as the claims.TLS.Cert field is empty.
func (v *NatsCalloutVerifier) VerifyClientCert(claims *jwt.AuthorizationRequestClaims) (string, error) {
	clientID := claims.ConnectOptions.Name
	if claims.TLS == nil || len(claims.TLS.VerifiedChains) == 0 {
		return clientID, fmt.Errorf("client doesn't have cert")
	}
	clientCertPEM := claims.TLS.VerifiedChains[0][0]

	// validate issuer, verify with CA
	certClientID, err := certs.VerifyCert(clientCertPEM, v.caCert)
	if err != nil {
		return clientID, fmt.Errorf("invalid client cert: %w", err)
	} else if certClientID != clientID {
		return clientID, fmt.Errorf("cert CN '%s' differs from clientID '%s'", certClientID, clientID)
	}
	return clientID, nil
}

// VerifyNKey claim
// Don't use this as it is incomplete. nats-server nkey change is
// embedded deep into its auth code and can't be easily separated.
// Workaround: let server handle NKeys through static config.
//
// See also nats-server auth.go:990 for dealing with nkeys. It aint pretty.
func (v *NatsCalloutVerifier) VerifyNKey(claims *jwt.AuthorizationRequestClaims) (string, error) {
	slog.Warn("use of nkey auth. Shouldn't the key be in the server opts nkeys?")

	host := claims.ClientInformation.Host
	nonce := claims.ClientInformation.Nonce
	userPub := claims.ClientInformation.User
	clientID := claims.ConnectOptions.Name
	pubKey := claims.ConnectOptions.Nkey
	signedNonce := claims.ConnectOptions.SignedNonce
	_ = nonce
	_ = signedNonce
	_ = host
	_ = userPub

	err := v.msgServer.ValidateNKey(clientID, pubKey, signedNonce, nonce)
	return clientID, err
}

// VerifyPassword checks the password claim
func (v *NatsCalloutVerifier) VerifyPassword(claims *jwt.AuthorizationRequestClaims) (string, error) {
	// verify password
	loginName := claims.ConnectOptions.Username
	passwd := claims.ConnectOptions.Password
	err := v.msgServer.ValidatePassword(loginName, passwd)
	return loginName, err
}

// VerifyToken verifies any JWT token
func (v *NatsCalloutVerifier) VerifyToken(claims *jwt.AuthorizationRequestClaims) (string, error) {
	token := claims.ConnectOptions.Token
	clientID := claims.ConnectOptions.Name
	//requestNonce := claims.RequestNonce
	requestNonce := claims.ClientInformation.Nonce
	signedNonce := claims.ConnectOptions.SignedNonce
	err := v.msgServer.ValidateToken(clientID, token, signedNonce, requestNonce)
	if err != nil {
		return clientID, fmt.Errorf("invalid token: %w", err)
	}
	return clientID, nil
}

// VerifyAuthnReq the authentication request
// For use with the callout hook to verify various means of authentication.
// claims contains various possible svc methods: password, nkey, jwt, certs
//
// Note that NATS server can already authenticate password, nkey, cert, and jwt tokens.
// However, NATS doesn't do multiple methods of password, nkey,cert and jwt. This verifier does them all.
// Since the server is updated with client auth info, actual verification goes back to the server.
// effectively a very roundabout way of doing what the server should have been able to.
func (v *NatsCalloutVerifier) VerifyAuthnReq(claims *jwt.AuthorizationRequestClaims) (clientID string, err error) {
	slog.Info("VerifyAuthnReq",
		slog.String("name", claims.ConnectOptions.Name),
		slog.String("host", claims.ClientInformation.Host))
	if claims.ConnectOptions.Nkey != "" {
		clientID, err = v.VerifyNKey(claims)
	} else if claims.ConnectOptions.Password != "" {
		clientID, err = v.VerifyPassword(claims)
	} else if claims.ConnectOptions.Token != "" {
		clientID, err = v.VerifyToken(claims)
	} else if claims.TLS != nil && len(claims.TLS.VerifiedChains) > 0 {
		clientID, err = v.VerifyClientCert(claims)
	} else {
		// unsupported
		err = fmt.Errorf("no auth credentials provided by user '%s' from host '%s'",
			claims.ClientInformation.Name, claims.ClientInformation.Host)
	}
	return clientID, err
}

func NewNatsCoVerifier(
	msgServer *service.NatsMsgServer, caCert *x509.Certificate) *NatsCalloutVerifier {

	v := &NatsCalloutVerifier{
		msgServer: msgServer,
		caCert:    caCert,
	}
	return v
}
