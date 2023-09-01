package natsnkeyserver

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
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
	tokenizer *NatsJWTTokenizer
	msgServer msgserver.IMsgServer // use to get client list?
	caCert    *x509.Certificate
}

func (v *NatsCalloutVerifier) VerifyClientCert(claims *jwt.AuthorizationRequestClaims) error {
	if claims.TLS == nil || len(claims.TLS.Certs) == 0 {
		return fmt.Errorf("client doesn't have cert")
	}
	clientID := claims.Name
	clientCertPEM := claims.TLS.Certs[0]

	// validate issuer, verify with CA
	err := certs.VerifyCert(clientID, clientCertPEM, v.caCert)
	if err != nil {
		return fmt.Errorf("invalid client cert: %w", err)
	}
	return fmt.Errorf("client cert svc not yet supported")
}

// VerifyNKey claim
// Don't use this as it is incomplete. nats-server nkey change is
// embedded deep into its auth code and can't be easily separated.
// Workaround: let server handle NKeys through static config.
//
// FIXME: How to perform a nonce check with the remote client?
// See also nats-server auth.go:990 for dealing with nkeys. It aint pretty.
func (v *NatsCalloutVerifier) VerifyNKey(claims *jwt.AuthorizationRequestClaims) error {
	host := claims.ClientInformation.Host
	nonce := claims.ClientInformation.Nonce
	userPub := claims.ClientInformation.User
	userID := claims.ConnectOptions.Name
	nkey := claims.ConnectOptions.Nkey
	signedNonce := claims.ConnectOptions.SignedNonce
	_ = nonce
	_ = signedNonce
	_ = host
	_ = userID
	_ = userPub

	slog.Warn("use of nkey auth")

	//sig, err := base64.RawURLEncoding.DecodeString(c.opts.Sig)
	//if err != nil {
	//	return fmt.Errorf("signature not valid")
	//}
	pub, err := nkeys.FromPublicKey(nkey)
	_ = pub
	if err != nil {
		return fmt.Errorf("user nkey not valid: %v", err)
	}
	// FIXME: where does sig come from?
	sig := []byte("")
	if err := pub.Verify([]byte(nonce), sig); err != nil {
		return fmt.Errorf("signature not verified")
	}
	return fmt.Errorf("nkey svc not supported")
}

// VerifyPassword checks the password claim
func (v *NatsCalloutVerifier) VerifyPassword(claims *jwt.AuthorizationRequestClaims) error {
	// verify password
	loginName := claims.ConnectOptions.Username
	passwd := claims.ConnectOptions.Password
	err := v.msgServer.VerifyPassword(loginName, passwd)
	return err
}

// VerifyToken verifies any JWT token
func (v *NatsCalloutVerifier) VerifyToken(claims *jwt.AuthorizationRequestClaims) error {
	token := claims.ConnectOptions.Token
	clientID := claims.ConnectOptions.Name
	requestNonce := claims.RequestNonce
	signedNonce := claims.ConnectOptions.SignedNonce
	err := v.msgServer.VerifyToken(clientID, token, signedNonce, requestNonce)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	return nil
}

// VerifyAuthnReq the authentication request
// For use by the nats server
// claims contains various possible svc methods: password, nkey, jwt, certs
func (v *NatsCalloutVerifier) VerifyAuthnReq(claims *jwt.AuthorizationRequestClaims) (err error) {
	slog.Info("VerifyAuthnReq",
		slog.String("name", claims.ConnectOptions.Name),
		slog.String("host", claims.ClientInformation.Host))
	if claims.ConnectOptions.Nkey != "" {
		err = v.VerifyNKey(claims)
	} else if claims.ConnectOptions.Password != "" {
		err = v.VerifyPassword(claims)
	} else if claims.ConnectOptions.Token != "" {
		err = v.VerifyToken(claims)
	} else if claims.TLS != nil && claims.TLS.Certs != nil {
		err = v.VerifyClientCert(claims)
	} else {
		// unsupported
		err = fmt.Errorf("no auth credentials provided by user '%s' from host '%s'",
			claims.ClientInformation.Name, claims.ClientInformation.Host)
	}
	return err
}

func NewNatsCoVerifier(
	msgServer msgserver.IMsgServer, tokenizer *NatsJWTTokenizer, caCert *x509.Certificate) *NatsCalloutVerifier {

	v := &NatsCalloutVerifier{
		msgServer: msgServer,
		tokenizer: tokenizer,
		caCert:    caCert,
	}
	return v
}
