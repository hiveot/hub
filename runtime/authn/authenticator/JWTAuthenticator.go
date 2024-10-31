package authenticator

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/config"
	"github.com/hiveot/hub/runtime/authn/sessions"
	"github.com/teris-io/shortid"
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
	//
	AgentTokenValiditySec    int
	ConsumerTokenValiditySec int
	ServiceTokenValiditySec  int

	// sessionmanager tracks session IDs
	sm *sessions.SessionManager
}

// CreateSessionToken creates a new session token for the client
//
//	clientID is the account ID of a known client
//	sessionID for which this token is valid. Use clientID to allow no session (agents)
//	validitySec is the token validity period or 0 for default based on client type
//
// This returns the token
func (svc *JWTAuthenticator) CreateSessionToken(
	clientID string, sessionID string, validitySec int) (token string) {

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
		//"aud": authInfo.SenderID, // recipient of the jwt
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
	claimsClientID := claims["clientID"]
	if claimsClientID != nil {
		clientID = claimsClientID.(string)
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
func (svc *JWTAuthenticator) Login(clientID string, password string) (token string, err error) {
	var sessionID string
	// a user login always creates a session token
	err = svc.ValidatePassword(clientID, password)
	if err != nil {
		return "", err
	}

	// check if this user has an existing session. Generate the token using its
	// existing sessionID.
	cs, found := svc.sm.GetSessionByClientID(clientID)
	if found {
		// use the existing session id and renew the session expiry
		sessionID = cs.SessionID
	} else {
		// password login always uses the consumer token validity
		sessionID = shortid.MustGenerate()
	}
	// create the session to allow token refresh
	svc.sm.NewSession(clientID, sessionID)
	token = svc.CreateSessionToken(clientID, sessionID, svc.ConsumerTokenValiditySec)

	return token, err
}

// Logout removes the client session
func (svc *JWTAuthenticator) Logout(clientID string) {
	cs, found := svc.sm.GetSessionByClientID(clientID)
	if found {
		svc.sm.Remove(cs.SessionID)
	}
}

// RefreshToken requests a new token based on the old token
// This requires that the existing session is still valid
func (svc *JWTAuthenticator) RefreshToken(
	senderID string, clientID string, oldToken string) (newToken string, err error) {

	// validation only succeeds if there is an active session
	tokenClientID, sessionID, err := svc.ValidateToken(oldToken)
	if err != nil || clientID != senderID || clientID != tokenClientID {
		return newToken, fmt.Errorf("SenderID mismatch")
	}
	// must still be a valid client
	prof, err := svc.authnStore.GetProfile(senderID)
	_ = prof
	if err != nil || prof.Disabled {
		return newToken, fmt.Errorf("Profile for '%s' is disabled", clientID)
	}
	validitySec := svc.ConsumerTokenValiditySec
	if prof.ClientType == authn.ClientTypeAgent {
		validitySec = svc.AgentTokenValiditySec
	} else if prof.ClientType == authn.ClientTypeService {
		validitySec = svc.ServiceTokenValiditySec
	}
	newToken = svc.CreateSessionToken(clientID, sessionID, validitySec)
	return newToken, err
}

func (svc *JWTAuthenticator) ValidatePassword(clientID, password string) (err error) {
	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	return err
}

// ValidateToken the session token
// For agents, the sessionID equals the clientID and no session check will take place. (sessions are for consumers only)
func (svc *JWTAuthenticator) ValidateToken(token string) (clientID string, sessionID string, err error) {
	clientID, sid, err := svc.DecodeSessionToken(token, "", "")
	if err != nil {
		return "", "", err
	}
	// agents don't require a session
	// TBD: if agents do need sessions then the sessions need to be persisted and restored.
	// This is a bit of a pain to manage so a future consideration.
	if clientID == sid {
		return clientID, sid, nil
	}
	cs, found := svc.sm.GetSessionBySessionID(sid)
	if !found {
		slog.Warn("ValidateToken. No session found for client", "clientID", clientID)
		return "", "", fmt.Errorf("Session is no longer valid")
	}
	// if the session has expired, remove it
	if cs.Expiry.Before(time.Now()) {
		svc.sm.Remove(sid)
		slog.Warn("ValidateToken. Session has expired", "clientID", clientID)
		return "", "", fmt.Errorf("Session has expired")
	}
	return clientID, sid, nil
}

// NewJWTAuthenticator returns a new instance of a JWT token authenticator
func NewJWTAuthenticator(authnStore api.IAuthnStore, signingKey keys.IHiveKey) *JWTAuthenticator {
	svc := JWTAuthenticator{
		signingKey: signingKey,
		authnStore: authnStore,
		// validity can be changed by user of this service
		AgentTokenValiditySec:    config.DefaultAgentTokenValiditySec,
		ConsumerTokenValiditySec: config.DefaultConsumerTokenValiditySec,
		ServiceTokenValiditySec:  config.DefaultServiceTokenValiditySec,
		sm:                       sessions.NewSessionmanager(),
	}
	return &svc
}

// NewJWTAuthenticatorFromFile returns a new instance of a JWT token authenticator
// loading a keypair from file or creating one if it doesn't exist.
// This returns nil if no signing key can be loaded or created
//func NewJWTAuthenticatorFromFile(
//	authnStore api.IAuthnStore,
//	keysDir string, keyType keys.KeyType) *JWTAuthenticator {
//
//	clientID := "authn"
//	signingKey, err := keys.LoadCreateKeyPair(clientID, keysDir, keyType)
//	if err != nil {
//		slog.Error("NewJWTAuthenticatorFromFile failed creating key pair for client",
//			"err", err.Error(), "clientID", clientID)
//		return nil
//	}
//	_ = err
//	svc := NewJWTAuthenticator(authnStore, signingKey)
//	return svc
//}
