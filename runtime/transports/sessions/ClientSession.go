package sessions

import (
	"time"
)

// ClientSession of an authenticated client connected over http.
//
// Each client session has a session-id from the authentication token.
type ClientSession struct {
	// ID of this session
	sessionID string

	// ClientID is the login ID of the agent or consumer
	clientID string

	// remoteAddr contains the remote address this session belongs to
	remoteAddr string

	// track last used time to auto-close inactive sessions
	lastActivity time.Time
}

func (cs *ClientSession) GetClientID() string {
	return cs.clientID
}

// GetSessionID returns the ID of this client session.
// This ID is obtained from its authentication.
func (cs *ClientSession) GetSessionID() string {
	return cs.sessionID
}

// UpdateLastActivity sets the current time
func (cs *ClientSession) UpdateLastActivity() {
	cs.lastActivity = time.Now()
}

// NewClientSession creates a new client session
// Intended for use by the session manager.
// This subscribes to events for configured agents.
//
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(sessionID string, clientID string, remoteAddr string) *ClientSession {
	cs := ClientSession{
		sessionID:    sessionID,
		clientID:     clientID,
		remoteAddr:   remoteAddr,
		lastActivity: time.Now(),
	}

	return &cs
}
