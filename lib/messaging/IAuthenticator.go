package messaging

import (
	"time"

	"github.com/hiveot/hivekit/go/wot/td"
)

// UserLoginArgs defines the arguments of the login function
// Login - Login with password
type UserLoginArgs struct {

	// ClientID with Login ID
	ClientID string `json:"clientID,omitempty"`

	// Password with Password
	Password string `json:"password,omitempty"`
}

// IAuthenticator is the interface of the authentication capability to obtain and
// validate session tokens.
type IAuthenticator interface {
	// AddSecurityScheme adds the wot securityscheme to the given TD
	AddSecurityScheme(tdoc *td.TD)

	// CreateSessionToken creates a signed session token for a client and adds the session
	// sessionID is required. For persistent sessions use the clientID.
	CreateSessionToken(clientID, sessionID string, validity time.Duration) (token string)

	// DecodeSessionToken and return its claims
	DecodeSessionToken(sessionToken string, signedNonce string, nonce string) (
		clientID string, sessionID string, err error)

	// GetAlg returns the supported security format and authentication algorithm.
	// This uses the vocabulary as defined in the TD.
	// JWT: "ES256", "ES512", "EdDSA"
	// paseto: "local" (symmetric), "public" (asymmetric)
	GetAlg() (string, string)

	// Login with a password and obtain a new session token with limited duration
	// This creates a new session. The token must be refreshed to keep the session alive.
	Login(login string, password string) (token string, err error)

	// Logout removes the session
	Logout(clientID string)

	// RefreshToken issues a new session token with an updated expiry time.
	// This extends the life of the session.
	//
	//	clientID Client whose token to refresh
	//	oldToken must be valid
	//	validitySec validity in seconds of the new token
	//
	// This returns a new token or an error if the old token isn't valid or doesn't match clientID
	RefreshToken(senderID string, oldToken string) (newToken string, err error)

	// Set the method to
	SetAuthServerURI(authServiceURI string)

	// ValidatePassword checks if the given password is valid for the client
	ValidatePassword(clientID string, password string) (err error)

	// ValidateToken validates the auth token and returns the token clientID.
	// If the token is invalid an error is returned
	ValidateToken(token string) (clientID string, sessionID string, err error)
}
