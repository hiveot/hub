package api

// IAuthenticator is the interface of the authentication capability to obtain and
// validate session tokens.
type IAuthenticator interface {
	// Login with the given password and create a session token for future login
	// This will verify the given password and if valid will generate an authentication
	// token containing a clientID and sessionID.
	// A new sessionID will be generated if none is provided.
	Login(clientID string, password string, sessionID string) (newToken string, sid string, err error)

	// CreateSessionToken creates a signed session token for a client.
	// sessionID is optional and only used if a matching session is required
	CreateSessionToken(clientID, sessionID string, validitySec int) (token string)

	// RefreshToken the session token and retain its session ID
	//
	//	clientID Client whose token to refresh
	//	oldToken must be valid
	//
	// This returns a new token or an error if the old token isn't valid or doesn't match clientID
	RefreshToken(clientID string, oldToken string) (newToken string, err error)

	// ValidateToken the session token and return the corresponding clientID and sessionID
	// If a sessionID is provided or the token contains a sessionID then the MUST match.
	ValidateToken(token string) (clientID string, sessionID string, err error)
}
