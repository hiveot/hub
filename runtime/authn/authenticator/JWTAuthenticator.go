package authenticator

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"time"
)

// JWTAuthenticator for generating and validating session tokens.
// This implements the IAuthenticator interface
type JWTAuthenticator struct {
	// key used to create and verify session tokens
	signingKey keys.IHiveKey
	// authentication store for login verification
	authnStore api.IAuthnStore
}

// CreateSessionToken creates a new session token for the client
//
//	clientID is the account ID of a known client
//	sessionID for which this token is valid. "" to not apply a sessionID.
//	validitySec is the token validity period or 0 for default based on client type
//
// This returns the token
func (svc *JWTAuthenticator) CreateSessionToken(
	clientID string, sessionID string, validitySec int) string {

	// TODO: add support for nonce challenge with client pubkey

	// CreateSessionToken creates a signed JWT session token for a client.
	// The token is constructed with MapClaims containing "ID" as session ID and
	// "clientID" identifying the connectied client.
	// The token is signed with the given signing key-pair and valid for the given duration.

	validity := time.Second * time.Duration(validitySec)
	expiryTime := time.Now().Add(validity)
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
	sessionToken, _ := claimsToken.SignedString(svc.signingKey.PrivateKey())

	return sessionToken
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

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//	sessionID of the new session or "" to generate a new session ID
//
// This returns a session token, its session ID, or an error if failed
func (svc *JWTAuthenticator) Login(clientID, password, sessionID string) (token string, sid string, err error) {

	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	if err != nil {
		return "", "", err
	}
	if sessionID == "" {
		uid, _ := uuid.NewUUID()
		sessionID = uid.String()
	}
	validitySec := clientProfile.TokenValiditySec
	token = svc.CreateSessionToken(clientID, sessionID, validitySec)
	return token, sessionID, err
}

// RefreshToken issues a new authentication token for the authenticated user.
// This returns a refreshed token carrying the same session id as the old token.
// the old token must be a valid jwt token belonging to the clientID.
func (svc *JWTAuthenticator) RefreshToken(clientID string, oldToken string, validitySec int) (token string, err error) {
	// verify the token
	tokenClientID, sessionID, err := svc.DecodeSessionToken(oldToken, "", "")
	if err == nil && tokenClientID != clientID {
		err = fmt.Errorf("RefreshToken:Token client '%s' differs from client '%s'", tokenClientID, clientID)
	}
	if err != nil {
		return "", fmt.Errorf("RefreshToken: invalid oldToken of client %s: %w", clientID, err)
	}
	token = svc.CreateSessionToken(clientID, sessionID, validitySec)
	return token, err
}

// ValidateToken the session token
func (svc *JWTAuthenticator) ValidateToken(token string) (clientID string, sessionID string, err error) {
	cid, sid, err := svc.DecodeSessionToken(token, "", "")
	slog.Info("ValidateToken", slog.String("clientID", cid))

	return cid, sid, err
}

// NewJWTAuthenticator returns a new instance of a JWT token authenticator
func NewJWTAuthenticator(authnStore api.IAuthnStore, signingKey keys.IHiveKey) *JWTAuthenticator {
	svc := JWTAuthenticator{
		signingKey: signingKey,
		authnStore: authnStore,
	}
	return &svc
}

// NewJWTAuthenticatorFromFile returns a new instance of a JWT token authenticator
// loading a keypair from file or creating one if it doesn't exist.
// This returns nil if no signing key can be loaded or created
func NewJWTAuthenticatorFromFile(
	authnStore api.IAuthnStore,
	keysDir string, keyType keys.KeyType) *JWTAuthenticator {

	clientID := "authn"
	signingKey, err := keys.LoadCreateKeyPair(clientID, keysDir, keyType)
	if err != nil {
		slog.Error("NewJWTAuthenticatorFromFile failed creating key pair for client",
			"err", err.Error(), "clientID", clientID)
		return nil
	}
	_ = err
	svc := NewJWTAuthenticator(authnStore, signingKey)
	return svc
}
