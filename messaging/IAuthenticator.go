package messaging

// IAuthenticator is the interface of the authentication capability to obtain and
// validate session tokens.
type IAuthenticator interface {
	// CreateSessionToken creates a signed session token for a client and adds the session
	// sessionID is required. For persistent sessions use the clientID.
	CreateSessionToken(clientID, sessionID string, validitySec int) (token string)

	// DecodeSessionToken and return its claims
	DecodeSessionToken(sessionToken string, signedNonce string, nonce string) (
		clientID string, sessionID string, err error)

	// Login with a password and obtain a new session token
	Login(clientID string, password string) (token string, err error)

	// Logout removes the session
	Logout(clientID string)

	// RefreshToken the session token and retain its session ID
	//
	//	clientID Client whose token to refresh
	//	oldToken must be valid
	//	validitySec validity in seconds of the new token
	//
	// This returns a new token or an error if the old token isn't valid or doesn't match clientID
	RefreshToken(senderID string, oldToken string) (newToken string, err error)

	// ValidatePassword checks if the given password is valid for the client
	ValidatePassword(clientID string, password string) (err error)

	// ValidateToken validates the auth token and returns the token clientID.
	// If the token is invalid an error is returned
	ValidateToken(token string) (clientID string, sessionID string, err error)
}
