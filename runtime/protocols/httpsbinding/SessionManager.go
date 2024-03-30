package httpsbinding

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"github.com/google/uuid"
	"github.com/hiveot/hub/lib/keys"
	"log/slog"
	"sync"
	"time"
)

// SessionManager tracks client sessions using their authentication token.
//
// TODO:
//  1. close session after not being used for X seconds
type SessionManager struct {
	// existing sessions by sessionID (remoteAddr)
	sessions map[string]*ClientSession
	// mutex to access the sessions
	mux sync.RWMutex
}

// NewSession creates a new session for the given clientID and remote address.
// The clientID must already have been authorized first.
//
// This returns the new session instance or an error if the client has too many sessions.
func (sm *SessionManager) NewSession(clientID string, remoteAddr string) (*ClientSession, error) {
	var cs *ClientSession
	var sessionID string

	slog.Info("NewSession")
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	cs = NewClientSession(sessionID, clientID, remoteAddr)
	sm.mux.Lock()
	sm.sessions[sessionID] = cs
	sm.mux.Unlock()
	return cs, nil
}

// Close closes the hub connection and event channel, removes the session
func (sm *SessionManager) Close(sessionID string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	si, found := sm.sessions[sessionID]
	if !found {
		slog.Info("Close. Session was already closed.", "sessionID", sessionID)
		return errors.New("Session not found")
	}
	si.Close()
	delete(sm.sessions, sessionID)
	return nil
}

// GetSession returns the client session if available
// An error is returned if the sessionID is not an existing session
func (sm *SessionManager) GetSession(sessionID string) (*ClientSession, error) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	if sessionID == "" {
		return nil, errors.New("missing sessionID")
	}
	session, found := sm.sessions[sessionID]
	if !found {
		return nil, errors.New("sessionID '" + sessionID + "' not found")
	}
	session.lastActivity = time.Now()
	return session, nil
}

// Init initializes the session manager
//
//	hubURL with address of the hub message bus
//	messaging core to use or "" for auto-detection
//	signingKey for cookies
//	caCert of the messaging server
//	tokenKP optional keys to use for refreshing tokens of authenticated users
func (sm *SessionManager) Init(hubURL string, core string,
	signingKey *ecdsa.PrivateKey, caCert *x509.Certificate,
	tokenKP keys.IHiveKey) {
}

// The global session manager instance.
// Init must be called before use.
var sessionmanager = func() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*ClientSession),
	}
	return sm
}()

// GetSessionManager returns the sessionManager singleton
func GetSessionManager() *SessionManager {
	return sessionmanager
}
