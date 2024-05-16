package sessions

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"sync"
	"time"
)

// SessionManager tracks client sessions using their authentication token.
//
// TODO:
//  1. close session after not being used for X seconds
//  2. Send notification if a client connects and disconnects
//     needed to send push notifications to clients (primarily agents and services)
type SessionManager struct {
	// existing sessions by sessionID (remoteAddr)
	sidSessions map[string]*ClientSession
	// existing sessions by clientID
	// FIXME: what if a client connects multiple times? Should this be an array?
	clientSessions map[string]*ClientSession
	// mutex to access the sessions
	mux sync.RWMutex
}

// NewSession creates a new session for the given clientID and remote address.
// The clientID must already have been authorized first.
// sessionID is optional sessionID if one exists.
//
// This returns the new session instance or an error if the client has too many sessions.
func (sm *SessionManager) NewSession(clientID string, remoteAddr string, sessionID string) (*ClientSession, error) {
	var cs *ClientSession

	slog.Info("NewSession")
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	cs = NewClientSession(sessionID, clientID, remoteAddr)
	sm.mux.Lock()
	sm.sidSessions[sessionID] = cs
	sm.clientSessions[clientID] = cs
	sm.mux.Unlock()
	return cs, nil
}

// Close closes the hub connection and event channel, removes the session
func (sm *SessionManager) Close(sessionID string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	si, found := sm.sidSessions[sessionID]
	if !found {
		slog.Info("Close. Session was already closed.", "sessionID", sessionID)
		return errors.New("Session not found")
	}
	si.Close()
	delete(sm.sidSessions, sessionID)
	delete(sm.clientSessions, si.clientID)
	return nil
}

// CloseAll closes all sessions
func (sm *SessionManager) CloseAll() {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	slog.Info("CloseAll. Closing remaining sessions", "count", len(sm.sidSessions))
	for sid, session := range sm.sidSessions {
		_ = sid
		session.Close()
	}
	sm.sidSessions = make(map[string]*ClientSession)
	sm.clientSessions = make(map[string]*ClientSession)
}

// GetSession returns the client session if available
// An error is returned if the sessionID is not an existing session
func (sm *SessionManager) GetSession(sessionID string) (*ClientSession, error) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	if sessionID == "" {
		return nil, errors.New("missing sessionID")
	}
	session, found := sm.sidSessions[sessionID]
	if !found {
		return nil, errors.New("sessionID '" + sessionID + "' not found")
	}
	session.lastActivity = time.Now()
	return session, nil
}

// GetSessionByClientID returns the latest client session by the client's ID
// An error is returned if the clientID does not have a session
// Intended for lookup of agents.
// TODO: if a client has multiple sessions and one closes then return one of the others.
// This currently forgets the client if one of its sessions closes. This is fine for
// agents as they typically only have a single session.
func (sm *SessionManager) GetSessionByClientID(clientID string) (*ClientSession, error) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	if clientID == "" {
		return nil, errors.New("missing clientID")
	}
	// FIXME: a client that has multiple sessions will only see the last one
	session, found := sm.clientSessions[clientID]
	if !found {
		return nil, errors.New("clientID '" + clientID + "' has no sessions")
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

// SendEvent passs an event to all sessions
// Returns true if at least one session received the event or false if no session are available
func (sm *SessionManager) SendEvent(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	payload, _ := json.Marshal(msg)

	for id, session := range sm.sidSessions {
		_ = id
		_ = session.SendSSE(vocab.MessageTypeEvent, string(payload))
	}
	if len(sm.sidSessions) > 0 {
		stat.Status = api.DeliveryCompleted
	} else {
		stat.Status = api.DeliveryFailed
		stat.Error = "No active sessions"
	}
	return stat
}

// The global session manager instance.
// Init must be called before use.
var sessionmanager = func() *SessionManager {
	sm := &SessionManager{
		sidSessions:    make(map[string]*ClientSession),
		clientSessions: make(map[string]*ClientSession),
	}
	return sm
}()

// GetSessionManager returns the sessionManager singleton
func GetSessionManager() *SessionManager {
	return sessionmanager
}
