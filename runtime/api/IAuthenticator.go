package api

// IAuthenticator is the interface of the authentication capability to obtain and
// validate session tokens.
type IAuthenticator interface {
	// CreateSessionToken creates a signed session token for a client.
	// sessionID is optional and only used if a matching session is required
	CreateSessionToken(clientID, sessionID string, validitySec int) (token string)

	// Login with a password and obtain a new session token
	Login(clientID string, password string) (token string, sessionID string, err error)

	// RefreshToken the session token and retain its session ID
	//
	//	clientID Client whose token to refresh
	//	oldToken must be valid
	//	validitySec validity in seconds of the new token
	//
	// This returns a new token or an error if the old token isn't valid or doesn't match clientID
	RefreshToken(clientID string, oldToken string) (newToken string, err error)

	// ValidatePassword checks if the given password is valid for the client
	ValidatePassword(clientID string, password string) (err error)

	// ValidateToken the session token and return the corresponding clientID and sessionID
	// If a sessionID is provided or the token contains a sessionID then the MUST match.
	ValidateToken(token string) (clientID string, sessionID string, err error)
}
