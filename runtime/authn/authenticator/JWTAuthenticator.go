package authenticator

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/authn"
	"log/slog"
	"time"
)

// JWTAuthenticator for generating and validating session tokens.
// This implements the IAuthenticator interface
type JWTAuthenticator struct {
	// key used to create and verify session tokens
	signingKey keys.IHiveKey
	// authentication store for login verification
	authnStore authn.IAuthnStore
}

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//	sessionID of the new session or "" to generate a new session ID
//
// This returns a session token or an error if failed
func (svc *JWTAuthenticator) Login(clientID, password, sessionID string) (token string, err error) {

	if svc.authnStore == nil {
		return "", fmt.Errorf("Login: missing authnStore")
	}
	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	if err != nil {
		return "", err
	}
	validitySec := clientProfile.TokenValiditySec
	token, err = svc.CreateSessionToken(clientID, sessionID, validitySec)
	return token, err
}

// CreateSessionToken creates a new session token for the client
//
//	clientID is the account ID of a known client
//	sessionID for which this token is valid. "" to generate a new sessionID
//	validitySec is the token validity period or 0 for default based on client type
func (svc *JWTAuthenticator) CreateSessionToken(
	clientID string, sessionID string, validitySec int) (string, error) {

	// TODO: add support for nonce challenge with client pubkey

	// CreateSessionToken creates a signed JWT session token for a client.
	// The token is constructed with MapClaims containing "ID" as session ID and
	// "clientID" identifying the connectied client.
	// The token is signed with the given signing key-pair and valid for the given duration.

	validity := time.Second * time.Duration(validitySec)
	expiryTime := time.Now().Add(validity)
	if sessionID == "" {
		sid, _ := uuid.NewUUID()
		sessionID = sid.String()
	}
	signingKeyPub, _ := x509.MarshalPKIXPublicKey(svc.signingKey.PublicKey())
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	// Create the JWT claims, which includes the username, clientType and expiry time
	claims := jwt.MapClaims{
		//"alg": "ES256", // jwt.SigningMethodES256,
		"typ": "JWT",
		//"aud": authInfo.ClientID, // recipient of the jwt
		"sub": clientID,          // subject of the jwt, eg the client
		"iss": signingKeyPubStr,  // issuer of the jwt (public key)
		"exp": expiryTime.Unix(), // expiry time. Seconds since epoch
		"iat": time.Now().Unix(), // issued at. Seconds since epoch

		// custom claim fields
		"clientID":  clientID,
		"sessionID": sessionID,
	}

	// Declare the token with the algorithm used for signing, and the claims
	claimsToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	sessionToken, err := claimsToken.SignedString(svc.signingKey.PrivateKey())

	if err != nil {
		return "", err
	}

	return sessionToken, nil
}

// DecodeSessionToken verifies the given JWT token and returns its claims.
// optionally verify the signed nonce using the client's public key.
// This returns the auth info stored in the token.
//
// nonce based verification to prevent replay attacks is intended for future version.
//
// token is the jwt token string containing a session token
// This returns the client info reconstructed from the token or an error if invalid
func (svc *JWTAuthenticator) DecodeSessionToken(token string, signedNonce string, nonce string) (
	clientID string, sessionID string, err error) {

	signingKeyPub, _ := x509.MarshalPKIXPublicKey(svc.signingKey.PublicKey())
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	claims := jwt.MapClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, &claims,
		func(token *jwt.Token) (interface{}, error) {
			return svc.signingKey.PublicKey(), nil
		}, jwt.WithValidMethods([]string{
			jwt.SigningMethodES256.Name,
			jwt.SigningMethodES384.Name,
			jwt.SigningMethodES512.Name,
			"EdDSA",
		}),
		jwt.WithIssuer(signingKeyPubStr), // url encoded string
		jwt.WithExpirationRequired(),
	)
	cid := claims["clientID"]
	if cid != nil {
		clientID = cid.(string)
	}
	sid := claims["sessionID"]
	if sid != nil {
		sessionID = sid.(string)
	}
	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return clientID, sessionID,
			fmt.Errorf("ValidateToken: %w", err)
	}
	return clientID, sessionID, nil
}

// Refresh issues a new session token for the authenticated user.
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *JWTAuthenticator) RefreshToken(clientID string, oldToken string, validitySec int) (token string, err error) {
	// verify the token
	tokenClientID, sessionID, err := svc.DecodeSessionToken(oldToken, "", "")
	if tokenClientID != clientID {
		err = fmt.Errorf("RefreshToken:Token client '%s' differs from client '%s'", tokenClientID, clientID)
	}
	if err != nil {
		return "", fmt.Errorf("error validating oldToken of client %s: %w", clientID, err)
	}
	if sessionID == "" {
		id, _ := uuid.NewUUID()
		sessionID = id.String()
	}
	token, err = svc.CreateSessionToken(clientID, sessionID, validitySec)
	if err != nil {
		slog.Info("RefreshToken",
			"clientID", clientID, "err", err.Error())
	}
	return token, err
}

// Validate the session token
func (svc *JWTAuthenticator) ValidateToken(token string) (clientID string, sessionID string, err error) {
	slog.Info("ValidateToken", slog.String("clientID", clientID))
	cid, sid, err := svc.DecodeSessionToken(token, "", "")

	return cid, sid, err
}

// NewJWTAuthenticator returns a new instance of a JWT token authenticator
func NewJWTAuthenticator(signingKey keys.IHiveKey, authnStore authn.IAuthnStore) *JWTAuthenticator {
	svc := JWTAuthenticator{
		signingKey: signingKey,
		authnStore: authnStore,
	}
	return &svc
}
