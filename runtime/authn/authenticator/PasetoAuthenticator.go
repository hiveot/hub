package authenticator

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/hiveot/hivekit/go/keys"
	"github.com/hiveot/hivekit/go/wot/td"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/config"
	"github.com/hiveot/hub/runtime/authn/sessions"
	"github.com/teris-io/shortid"
)

// PasetoAuthenticator for generating and validating session tokens.
// This implements the IAuthenticator interface
type PasetoAuthenticator struct {
	// key used to create and verify session tokens
	signingKey ed25519.PrivateKey
	// authentication store for login verification
	authnStore authnstore.IAuthnStore
	// The URI of the authentication service that provides paseto tokens
	authServerURI             string
	AgentTokenValidityDays    int
	ConsumerTokenValidityDays int
	ServiceTokenValidityDays  int

	// sessionmanager tracks session IDs
	sm *sessions.SessionManager
}

// AddSecurityScheme adds this authenticator's security scheme to the given TD.
// This authenticator uses paseto tokens as bearer tokens that can be obtained from
// the login authentication service.
func (srv *PasetoAuthenticator) AddSecurityScheme(tdoc *td.TD) {

	// bearer security scheme for authenticating http and subprotocol connections
	format, alg := srv.GetAlg()

	tdoc.AddSecurityScheme("bearer_paseto", td.SecurityScheme{
		//AtType:        nil,
		Description: "Bearer token authentication",
		//Descriptions:  nil,
		//Proxy:         "",
		Scheme:        "bearer",          // nosec, basic, digest, bearer, psk, oauth2, apikey or auto
		Authorization: srv.authServerURI, // service to obtain a token
		Name:          "authorization",
		Alg:           alg,
		Format:        format,   // jwe, cwt, jws, jwt, paseto
		In:            "header", // query, body, cookie, uri, auto
	})
}

// CreateSessionToken creates a new session token for the client
//
//	clientID is the account ID of a known client
//	sessionID for which this token is valid. Use clientID to allow no session (agents)
//	validitySec is the token validity period or 0 for default based on client type
//
// This returns the token
func (svc *PasetoAuthenticator) CreateSessionToken(
	clientID string, sessionID string, validity time.Duration) (token string) {

	// TODO: add support for nonce challenge with client pubkey

	// CreateSessionToken creates a signed Paseto session token for a client.
	// The token is signed with the given signing key-pair and valid for the given duration.
	expiryTime := time.Now().Add(validity)

	pToken := paseto.NewToken()
	pToken.SetIssuer("hiveot")
	pToken.SetSubject(clientID)
	pToken.SetExpiration(expiryTime)
	pToken.SetIssuedAt(time.Now())
	pToken.SetNotBefore(time.Now())
	// custom claims
	pToken.SetString("sessionID", sessionID)
	pToken.SetString("clientID", clientID)

	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromEd25519(svc.signingKey)
	if err != nil {
		slog.Error("failed making paseto secret key from ED25519")
		secretKey = paseto.NewV4AsymmetricSecretKey()
	}
	signedToken := pToken.V4Sign(secretKey, nil)

	return signedToken
}

// DecodeSessionToken verifies the given token and returns its claims.
// optionally verify the signed nonce using the client's public key. (todo)
// This returns the auth info stored in the token.
//
// nonce based verification to prevent replay attacks is intended for future version.
//
// token is the token string containing a session token
// This returns the client info reconstructed from the token or an error if invalid
func (svc *PasetoAuthenticator) DecodeSessionToken(sessionKey string, signedNonce string, nonce string) (
	clientID string, sessionID string, err error) {
	var pToken *paseto.Token

	pasetoParser := paseto.NewParserForValidNow()
	pubKey := svc.signingKey.Public().(ed25519.PublicKey)
	v4PubKey, err := paseto.NewV4AsymmetricPublicKeyFromEd25519(pubKey)
	if err == nil {
		pToken, err = pasetoParser.ParseV4Public(v4PubKey, sessionKey, nil)
	}
	if err == nil {
		clientID, err = pToken.GetString("clientID")
	}
	if err == nil {
		sessionID, err = pToken.GetString("sessionID")
	}
	if err != nil {
		slog.Warn("DecodeSessionToken: the given session token is no longer valid: ", "err", err.Error())
	}
	return clientID, sessionID, err
}

// GetAlg returns the authentication scheme and algorithm
func (svc *PasetoAuthenticator) GetAlg() (string, string) {
	return "paseto", "public"
}

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//	sessionID of the new session or "" to generate a new session ID
//
// This returns a session token, its session ID, or an error if failed
func (svc *PasetoAuthenticator) Login(clientID string, password string) (token string, err error) {
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
	validity := time.Hour * time.Duration(24*svc.ConsumerTokenValidityDays)
	token = svc.CreateSessionToken(clientID, sessionID, validity)

	return token, err
}

// Logout removes the client session
func (svc *PasetoAuthenticator) Logout(clientID string) {
	cs, found := svc.sm.GetSessionByClientID(clientID)
	if found {
		svc.sm.Remove(cs.SessionID)
	}
}

// RefreshToken requests a new token based on the old token
// This requires that the existing session is still valid
func (svc *PasetoAuthenticator) RefreshToken(
	senderID string, oldToken string) (newToken string, err error) {

	// validation only succeeds if there is an active session
	tokenClientID, sessionID, err := svc.ValidateToken(oldToken)
	if err != nil || senderID != tokenClientID {
		return newToken, fmt.Errorf("SenderID mismatch")
	}
	// must still be a valid client
	prof, err := svc.authnStore.GetProfile(senderID)
	_ = prof
	if err != nil || prof.Disabled {
		return newToken, fmt.Errorf("Profile for '%s' is disabled", senderID)
	}
	validityDays := svc.ConsumerTokenValidityDays
	if prof.ClientType == authn.ClientTypeAgent {
		validityDays = svc.AgentTokenValidityDays
	} else if prof.ClientType == authn.ClientTypeService {
		validityDays = svc.ServiceTokenValidityDays
	}
	validity := time.Duration(validityDays) * 24 * time.Hour
	newToken = svc.CreateSessionToken(senderID, sessionID, validity)
	return newToken, err
}

// SetAuthServerURI this sets the server endpoint starting the authorization flow.
// This is included when adding the TD security scheme in AddSecurityScheme()
func (svc *PasetoAuthenticator) SetAuthServerURI(serverURI string) {
	svc.authServerURI = serverURI
}

func (svc *PasetoAuthenticator) ValidatePassword(clientID, password string) (err error) {
	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	return err
}

// ValidateToken the session token
// For agents, the sessionID equals the clientID and no session check will take place. (sessions are for consumers only)
func (svc *PasetoAuthenticator) ValidateToken(token string) (clientID string, sessionID string, err error) {
	clientID, sid, err := svc.DecodeSessionToken(token, "", "")
	if err != nil {
		return clientID, sid, err
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
		return clientID, sid, fmt.Errorf("Session is no longer valid")
	}
	// if the session has expired, remove it
	if cs.Expiry.Before(time.Now()) {
		svc.sm.Remove(sid)
		slog.Warn("ValidateToken. Session has expired", "clientID", clientID)
		return clientID, sid, fmt.Errorf("Session has expired")
	}
	return clientID, sid, nil
}

// NewPasetoAuthenticator returns a new instance of a Paseto token authenticator using the given signing key
// the session manager is used
func NewPasetoAuthenticator(
	authnStore authnstore.IAuthnStore,
	signingKey ed25519.PrivateKey,
	sm *sessions.SessionManager) *PasetoAuthenticator {

	paseto.NewV4AsymmetricSecretKey()

	svc := PasetoAuthenticator{
		signingKey: signingKey,
		authnStore: authnStore,
		//authServerURI: authServerURI, use SetAuthServerURI
		// validity can be changed by user of this service
		AgentTokenValidityDays:    config.DefaultAgentTokenValidityDays,
		ConsumerTokenValidityDays: config.DefaultConsumerTokenValidityDays,
		ServiceTokenValidityDays:  config.DefaultServiceTokenValidityDays,
		sm:                        sm,
	}
	return &svc
}

// NewPasetoAuthenticatorFromFile returns a new instance of a Paseto token authenticator
// loading a keypair from file or creating one if it doesn't exist.
// This returns nil if no signing key can be loaded or created
//
// The authServerURI is included the TD security scheme to point consumers to the
// endpoint to obtain tokens for this authenticator.
func NewPasetoAuthenticatorFromFile(
	authnStore authnstore.IAuthnStore,
	keysDir string,
	sm *sessions.SessionManager) *PasetoAuthenticator {

	clientID := "authn"
	authKey, err := keys.LoadCreateKeyPair(clientID, keysDir, keys.KeyTypeEd25519)

	if err != nil {
		slog.Error("NewPasetoAuthenticatorFromFile failed loading or creating a Paseto key pair",
			"err", err.Error(), "clientID", clientID)
		panic("failed loading or creating Paseto key pair")
	}
	signingKey := authKey.PrivateKey().(ed25519.PrivateKey)
	_ = err
	svc := NewPasetoAuthenticator(authnStore, signingKey, sm)
	return svc
}
