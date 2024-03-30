package jwtauth

// SessionInfo defines client authentication and authorization information
type ClientAuthInfo struct {
	// UserID, ServiceID or AgentID of the client
	ClientID string

	// ClientType identifies the client as a ClientTypeDevice, ClientTypeService or ClientTypeUser
	ClientType string

	// The PEM encoded client's public key, if any
	PubKey string

	// password encrypted with argon2id or bcrypt
	PasswordHash string

	// The client's role
	Role string // Name of user's role
}
