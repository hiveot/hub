package sessions

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/keys"
	"log/slog"
	"sync"
)

// SessionManager tracks client authentication sessions
//
// TODO:
//  1. Persist sessions between restart (to restore login) - config option?
//  3. Move clientSessions with the sub-protocol bindings that manage their own sessions
type SessionManager struct {
	// existing sessions by sessionID (remoteAddr)
	sidSessions map[string]*ClientSession
	// existing sessions by clientID
	clientSessions map[string][]*ClientSession
	// mutex to access the sessions
	mux sync.RWMutex
}

// NewSession creates a new session for the given clientID and remote address.
// The clientID must already have been authorized first.
// sessionID is a required sessionID. This will fail if missing.
//
// This returns the new session instance or an error if the client has too many sessions.
func (sm *SessionManager) NewSession(clientID string, remoteAddr string, sessionID string) (*ClientSession, error) {
	var cs *ClientSession

	slog.Debug("NewSession",
		slog.String("clientID", clientID),
		slog.String("remoteAddr", remoteAddr),
		slog.String("sessionID", sessionID))
	if sessionID == "" {
		return nil, fmt.Errorf("NewSession for client '%s' is missing a sessionID", clientID)
	}
	cs = NewClientSession(sessionID, clientID)
	sm.mux.Lock()
	sm.sidSessions[sessionID] = cs
	existingSessions, found := sm.clientSessions[clientID]
	if !found {
		existingSessions = []*ClientSession{cs}
	} else {
		existingSessions = append(existingSessions, cs)
	}
	sm.clientSessions[clientID] = existingSessions
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
	// remove the session from the clientID to sessions map
	sessions, found := sm.clientSessions[si.clientID]
	if found {
		for i, session := range sessions {
			if session.sessionID == sessionID {
				sessions[i] = sessions[len(sessions)-1]
				sessions = sessions[:len(sessions)-1]
				break
			}
		}
		if len(sessions) == 0 {
			delete(sm.clientSessions, si.clientID)
		}
	}
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
	sm.clientSessions = make(map[string][]*ClientSession)
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
	session.UpdateLastActivity()
	return session, nil
}

// GetSessionsByClientID returns the latest client sessions by client's ID.
// An error is returned if the clientID does not have a session
// Intended for lookup of agents or consumers to send directed messages.
func (sm *SessionManager) GetSessionsByClientID(clientID string) ([]*ClientSession, error) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	if clientID == "" {
		return nil, errors.New("missing clientID")
	}
	sessions, found := sm.clientSessions[clientID]
	if !found || len(sessions) == 0 {
		return nil, errors.New("clientID '" + clientID + "' has no sessions")
	}
	//sessions[0].UpdateLastActivity()
	return sessions, nil
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

// HandleActionFlow passes an action request to the session of the given agent.
// This returns a delivery status or an error if the request cannot be delivered.
//
// messageID is optional and can be useful to link to a previous action.
//func (sm *SessionManager) HandleActionFlow(
//	agentID string, thingID string, name string, data any, messageID string) (
//	status string, output any, err error) {
//
//	sm.mux.RLock()
//	defer sm.mux.RUnlock()
//
//	for id, session := range sm.sidSessions {
//		_ = id
//		if session.clientID == agentID && len(session.endpoints) > 0 {
//			ep := session.endpoints[0]
//			status, output, err = ep.HandleActionFlow(
//				agentID, thingID, name, data, messageID)
//			if err == nil {
//				return status, output, err
//			}
//		}
//	}
//	slog.Warn("HandleActionFlow. Agent not connected",
//		"agentID", agentID, "thingID", thingID, "action", name)
//	return digitwin.StatusFailed, nil, fmt.Errorf("agent not connected")
//}

// PublishEvent sends an event to subscribers.
// This is send-and-forget so it doesn't return anything.
// messageID is optional and can be useful to link to a previous action.
//func (sm *SessionManager) PublishEvent(
//	dThingID string, name string, data any, messageID string) {
//	sm.mux.RLock()
//	defer sm.mux.RUnlock()
//
//	for _, session := range sm.sidSessions {
//		for _, ep := range session.endpoints {
//			ep.PublishEvent(dThingID, name, data, messageID)
//		}
//	}
//}

// PublishProperty sends an property change notification to subscribers.
// This is send-and-forget so it doesn't return anything.
// messageID is optional and can be useful to link to a previous action.
//func (sm *SessionManager) PublishProperty(
//	dThingID string, name string, data any, messageID string) {
//	sm.mux.RLock()
//	defer sm.mux.RUnlock()
//
//	for _, session := range sm.sidSessions {
//		for _, ep := range session.endpoints {
//			ep.PublishProperty(dThingID, name, data, messageID)
//		}
//	}
//}

// The global session manager instance.
// Init must be called before use.
var sessionmanager = func() *SessionManager {
	sm := &SessionManager{
		sidSessions:    make(map[string]*ClientSession),
		clientSessions: make(map[string][]*ClientSession),
	}
	return sm
}()

// GetSessionManager returns the sessionManager singleton
func GetSessionManager() *SessionManager {
	return sessionmanager
}
