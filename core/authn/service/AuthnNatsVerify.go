package service

import (
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
)

// AuthnNatsVerify handles nats authentication verification
// Intended for use by nats callout
type AuthnNatsVerify struct {
	svc *AuthnService
}

func (v *AuthnNatsVerify) VerifyClientCert(claims *jwt.AuthorizationRequestClaims) error {
	if claims.TLS == nil || len(claims.TLS.Certs) == 0 {
		return fmt.Errorf("client doesn't have cert")
	}
	clientID := claims.Name
	clientCertPEM := claims.TLS.Certs[0]

	// validate issuer, verify with CA
	err := v.svc.ValidateCert(clientID, clientCertPEM)
	if err != nil {
		return fmt.Errorf("invalid client cert: %w", err)
	}
	return fmt.Errorf("client cert svc not yet supported")
}

// VerifyNatsJWT verifies the NATS JWT token that is signed by the account
func (v *AuthnNatsVerify) VerifyNatsJWT(claims *jwt.AuthorizationRequestClaims) error {

	err := v.svc.ValidateNatsJWT(
		claims.ClientInformation.Name,
		claims.ConnectOptions.Token,
		claims.ConnectOptions.SignedNonce,
		claims.ClientInformation.Nonce)
	//
	return err
}

// VerifyNKey claim
// Don't use this as it is incomplete. nats-server nkey change is
// embedded deep into its auth code and can't be easily separated.
// Workaround: let server handle NKeys through static config.
//
// FIXME: How to perform a nonce check with the remote client?
// See also nats-server auth.go:990 for dealing with nkeys. It aint pretty.
func (v *AuthnNatsVerify) VerifyNKey(claims *jwt.AuthorizationRequestClaims) error {
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

func (v *AuthnNatsVerify) VerifyPassword(claims *jwt.AuthorizationRequestClaims) error {
	// verify password
	loginName := claims.ConnectOptions.Username
	passwd := claims.ConnectOptions.Password
	err := v.svc.ValidatePassword(loginName, passwd)
	return err
}

// VerifyToken verifies a standard JWT token
func (v *AuthnNatsVerify) VerifyToken(claims *jwt.AuthorizationRequestClaims) error {
	token := claims.ConnectOptions.Token
	clientID := claims.ConnectOptions.Name
	_, tokenClaims, err := v.svc.ValidateToken(clientID, token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	_ = tokenClaims
	return nil
}

// VerifyAuthnReq the authentication request
// For use by the nats server
// claims contains various possible svc methods: password, nkey, jwt, certs
func (v *AuthnNatsVerify) VerifyAuthnReq(claims *jwt.AuthorizationRequestClaims) (err error) {
	slog.Info("VerifyAuthnReq",
		slog.String("name", claims.ConnectOptions.Name),
		slog.String("host", claims.ClientInformation.Host))
	if claims.ConnectOptions.Nkey != "" {
		err = v.VerifyNKey(claims)
	} else if claims.ConnectOptions.SignedNonce != "" {
		// JWT field is empty so we're using the Token field and expect a signed nonce
		err = v.VerifyNatsJWT(claims)
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

func NewAuthnNatsVerify(svc *AuthnService) *AuthnNatsVerify {
	v := &AuthnNatsVerify{svc: svc}
	return v
}
