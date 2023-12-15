package session

import (
	"crypto/x509"
	"errors"
	"github.com/hiveot/hub/lib/hubclient"
	"sync"
	"time"
)

// SessionInfo with hub connection of connected clients
type SessionInfo struct {
	// The user's session
	SessionID string
	// User ID for login
	LoginID string
	// Auth token is obtained from the secure cookie and used to re-connect to the hub
	AuthToken string
	// Expiry of the session
	Expiry time.Time
	// The associated hub client for pub/sub
	HC *hubclient.HubClient
}

type SessionManager struct {
	// existing sessions by sessionID
	sessions map[string]*SessionInfo
	mux      sync.RWMutex
	// Hub connection info
	hubURL string
	// Hub CA certificate
	caCert *x509.Certificate
	// Hub core if known (mqtt or nats)
	core string
}

// Open returns a new sessions for the given client
// If a session already exists, it is returned
// The session contains a hub client for publishing and subscribing messages
func (sm *SessionManager) Open(sessionID string, loginID string) (*SessionInfo, error) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	si, exists := sm.sessions[sessionID]
	if !exists {
		si = &SessionInfo{
			SessionID: sessionID,
			LoginID:   loginID,
			HC:        hubclient.NewHubClient(sm.hubURL, loginID, sm.caCert, sm.core),
		}
		sm.sessions[sessionID] = si
	}
	return si, nil
}

// ConnectWithToken to the Hub using an existing token.
// This returns the session info.
//func ConnectWithToken(clientID string, token string) (*SessionInfo, error) {
//}

// GetSession returns the client session info if available
// Returns the session or an error if it session doesn't exist
func (sm *SessionManager) GetSession(sessionID string) (*SessionInfo, error) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	if sessionID == "" {
		return nil, errors.New("missing sessionID")
	}
	session, found := sm.sessions[sessionID]
	if !found {
		return nil, errors.New("sessionID'" + sessionID + "' not found")
	}
	return session, nil
}

// Close closes a session and disconnect if connected
func (sm *SessionManager) Close(sessionID string) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	si, found := sm.sessions[sessionID]
	if !found {
		return
	}
	si.HC.Disconnect()
	delete(sm.sessions, sessionID)
}

// NewSessionManager returns a new instance of client sessions containing
// the hub message bus connection.
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*SessionInfo),
	}
	return sm
}
