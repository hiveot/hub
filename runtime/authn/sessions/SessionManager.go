package sessions

import (
	"log/slog"
	"sync"
	"time"
)

// Default duration of a session
const DefaultSessionDuration = 30 * 24 * time.Hour

// ClientSession of an authenticated client connected over http.
//
// Each client session has a session-id from the authentication token.
type ClientSession struct {
	// ID of this session
	SessionID string

	// SenderID is the login ID of the agent or consumer
	ClientID string

	// Created time of the session.
	Created time.Time

	// Time the session expires. Tokens won't be refreshed after this time.
	Expiry time.Time
}

// SessionManager provides the ability to expire authentication tokens of a client
// through client sessions.
//
// Sessions are directly linked to the client's ID. When a client logs-in successfully,
// a new session is created with a new expiry. The provided session-id is included
// in client tokens.
//
// When the client performs a request, it must provide a valid token. If the token
// is valid, the sessionID in the token must refer to a valid session as a second
// validation factor.
// The session remains valid until it expires or is removed, like when the user logs out.
//
// A session is valid for a limited period of time. Eventually it will expire,
// unless a successful login extends the session. Once expired, token refresh
// will fail and the user is required to login again.
//
// If a client logs-in from multiple places, each will get a token that refers to the
// same session.  Therefore there is at most 1 session per client. If the session
// expires or is invalidated, all instances must login again.
// Similarly, logging out, logs the client out on all devices. This too is intentional.
//
// TODO:
//  1. add session expiry
//  2. Persist sessions between restart (to restore login) - config option?
type SessionManager struct {
	// existing sessions by sessionID
	sidSessions map[string]*ClientSession
	// existing sessions by clientID (1 session per client)
	clientSessions map[string]*ClientSession
	// mutex to access the sessions
	mux sync.RWMutex
	// default duration a session is valid for after successful login
	sessionDuration time.Duration
}

// NewSession creates a new session for the given clientID and sessionID.
// If a session for the client exists it will be removed.
// If a session for the client and sessionID exists, its expiry will be renewed.
//
// This returns the new session instance
func (sm *SessionManager) NewSession(clientID string, sessionID string) {

	slog.Debug("NewSession",
		slog.String("clientID", clientID),
		slog.String("sessionID", sessionID))
	if sessionID == "" || clientID == "" {
		slog.Error("NewSession: invalid clientID or sessionID")
		return
	}
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sess, found := sm.clientSessions[clientID]
	if !found {
		// new session for the client
		sess = &ClientSession{
			SessionID: sessionID,
			ClientID:  clientID,
			Created:   time.Now(),
			Expiry:    time.Now().Add(sm.sessionDuration),
		}
		sm.sidSessions[sessionID] = sess
		sm.clientSessions[clientID] = sess
	} else if sess.SessionID == sessionID {
		// renew expiry of the existing session
		sess.Expiry = time.Now().Add(sm.sessionDuration)
	} else {
		// a different session. Invalidate the existing session
		delete(sm.clientSessions, clientID)
		delete(sm.sidSessions, sess.SessionID)
		// new session for the client
		sess = &ClientSession{
			SessionID: sessionID,
			ClientID:  clientID,
			Created:   time.Now(),
			Expiry:    time.Now().Add(sm.sessionDuration),
		}
		sm.sidSessions[sessionID] = sess
		sm.clientSessions[clientID] = sess
	}
}

// GetSessionBySessionID returns the client session if available
// An error is returned if the sessionID is not an existing session
func (sm *SessionManager) GetSessionBySessionID(sessionID string) (sess ClientSession, found bool) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	session, found := sm.sidSessions[sessionID]
	if found {
		sess = *session
	}
	return sess, found
}

// GetSessionByClientID returns the latest client sessions by client's ID.
// An error is returned if the clientID does not have a session
// Intended for lookup of agents or consumers to send directed messages.
func (sm *SessionManager) GetSessionByClientID(clientID string) (sess ClientSession, found bool) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	session, found := sm.clientSessions[clientID]
	if found {
		sess = *session
	}
	return sess, found
}

// Remove the session
func (sm *SessionManager) Remove(sessionID string) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	si, found := sm.sidSessions[sessionID]
	if found {
		delete(sm.clientSessions, si.ClientID)
		delete(sm.sidSessions, sessionID)
	}
}

// RemoveAll closes all sessions
func (sm *SessionManager) RemoveAll() {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	slog.Info("RemoveAll. Closing remaining sessions", "count", len(sm.sidSessions))
	sm.sidSessions = make(map[string]*ClientSession)
	sm.clientSessions = make(map[string]*ClientSession)
}

func NewSessionmanager() *SessionManager {
	sm := &SessionManager{
		sidSessions:     make(map[string]*ClientSession),
		clientSessions:  make(map[string]*ClientSession),
		mux:             sync.RWMutex{},
		sessionDuration: DefaultSessionDuration,
	}
	return sm
}
