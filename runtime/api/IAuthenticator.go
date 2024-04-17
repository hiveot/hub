package api

// IAuthenticator is the interface of the authentication capability to obtain and
// validate session tokens.
type IAuthenticator interface {
	// Login with the given password and create a session token for future login
	// This will verify the given password and if valid will generate an authentication
	// token containing a clientID and sessionID.
	// A new sessionID will be generated if none is provided.
	Login(clientID string, password string, sessionID string) (newToken string, err error)

	// CreateSessionToken creates a signed session token for a client.
	// If no sessionID is provided then one will be generated.
	CreateSessionToken(clientID, sessionID string, validitySec int) (token string)

	// RefreshToken the session token and retain its session ID
	RefreshToken(clientID string, oldToken string, validitySec int) (newToken string, err error)

	// ValidateToken the session token and return the corresponding clientID and sessionID
	ValidateToken(token string) (clientID string, sessionID string, err error)
}
