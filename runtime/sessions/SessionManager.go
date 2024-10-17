package sessions

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

const DefaultMaxSessionsPerClient = 10

// SessionManager manages authenticated client sessions by their session ID
//
// TODO:
//  1. close session after not being used for X seconds
//  2. Persist sessions between restart (to restore login) - config option?
type SessionManager struct {
	// existing sessions by sessionID
	sidSessions map[string]*ClientSession
	// existing sessions by clientID
	clientSessions map[string][]*ClientSession
	// mutex to access the sessions
	mux sync.RWMutex
	// maximum number of sessions a client is allowed to have. Default above.
	MaxSessionsPerClient int
}

// AddSession creates a new session for the given clientID and remote address.
//
// This returns the new session instance or an error if the client has too many sessions.
func (sm *SessionManager) AddSession(clientID string, remoteAddr string, sessionID string) (*ClientSession, error) {
	var cs *ClientSession

	slog.Debug("AddSession",
		slog.String("clientID", clientID),
		slog.String("remoteAddr", remoteAddr),
		slog.String("sessionID", sessionID))
	if sessionID == "" {
		return nil, fmt.Errorf("AddSession for client '%s' is missing a sessionID", clientID)
	}
	if remoteAddr == "" {
		return nil, fmt.Errorf("AddSession for client '%s' is missing a remoteAddr", clientID)
	}
	if clientID == "" {
		return nil, fmt.Errorf("AddSession for sessionID '%s' is missing a clientID", sessionID)
	}
	cs = NewClientSession(sessionID, clientID, remoteAddr)
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.sidSessions[sessionID] = cs
	existingSessions, found := sm.clientSessions[clientID]
	if !found {
		existingSessions = []*ClientSession{cs}
	} else {
		existingSessions = append(existingSessions, cs)
	}
	if len(existingSessions) > sm.MaxSessionsPerClient {
		err := fmt.Errorf("Client '%s' has too many sessions", clientID)
		slog.Warn("AddSession too many sessions", "err", err.Error, "clientID", clientID)
		return nil, err
	}

	sm.clientSessions[clientID] = existingSessions
	return cs, nil
}

// GetSession returns the client session if available
// An error is returned if the sessionID is not an existing session
func (sm *SessionManager) GetSession(sessionID string) (*ClientSession, error) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

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

	sessions, found := sm.clientSessions[clientID]
	if !found || len(sessions) == 0 {
		return nil, errors.New("clientID '" + clientID + "' has no sessions")
	}
	//sessions[0].UpdateLastActivity()
	return sessions, nil
}

// Remove closes the hub connection and event channel, removes the session
func (sm *SessionManager) Remove(sessionID string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	si, found := sm.sidSessions[sessionID]
	if !found {
		slog.Info("Remove. Session was already closed.", "sessionID", sessionID)
		return errors.New("Session not found")
	}
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
		} else {
			sm.clientSessions[si.clientID] = sessions
		}
	}
	return nil
}

// RemoveAll closes all sessions
func (sm *SessionManager) RemoveAll() {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	slog.Info("RemoveAll. Closing remaining sessions", "count", len(sm.sidSessions))
	sm.sidSessions = make(map[string]*ClientSession)
	sm.clientSessions = make(map[string][]*ClientSession)
}

func NewSessionmanager() *SessionManager {
	sm := &SessionManager{
		sidSessions:          make(map[string]*ClientSession),
		clientSessions:       make(map[string][]*ClientSession),
		MaxSessionsPerClient: DefaultMaxSessionsPerClient,
	}
	return sm
}
