package session

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
	"os"
	"sync"
	"time"
)

// SessionManager tracks logged-in sessions
// TODO: persist between restarts for testing using 'air'
type SessionManager struct {
	// existing sessions by sessionID
	sessions map[string]*ClientSession
	// path to the session store or "" when not to persist sessions
	sessionFile string
	mux         sync.RWMutex
	// Hub address
	hubURL string
	// Hub CA certificate
	caCert *x509.Certificate
	// Hub core if known (mqtt or nats)
	core string
}

// Add a new session for the given client and create a hub client
// instance for connecting to the Hub as the given user.
//
// If a session with the given ID already exists, it is returned instead.
// The session contains a hub client for publishing and subscribing messages
// authToken is optional when restoring a saved session
func (sm *SessionManager) Add(
	sessionID string, loginID string, remoteAddr string, authToken string) *ClientSession {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	// TODO: limit max nr of sessions per client/remote addr

	cs, exists := sm.sessions[sessionID]
	if !exists {
		hc := hubclient.NewHubClient(sm.hubURL, loginID, sm.caCert, sm.core)
		cs = NewClientSession(sessionID, hc, remoteAddr, authToken)
		sm.sessions[sessionID] = cs
	}
	return cs
}

// ConnectWithToken to the Hub using an existing token.
// This returns the session info.
//func ConnectWithToken(clientID string, token string) (*ClientSession, error) {
//}

// GetSession returns the client session info if available
// Returns the session or an error if it session doesn't exist
func (sm *SessionManager) GetSession(sessionID string) (*ClientSession, error) {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	if sessionID == "" {
		return nil, errors.New("missing sessionID")
	}
	session, found := sm.sessions[sessionID]
	if !found {
		return nil, errors.New("sessionID'" + sessionID + "' not found")
	} else if time.Now().Compare(session.Expiry) >= 0 {
		delete(sm.sessions, sessionID)
		return nil, errors.New("sessionID'" + sessionID + "' has expired")
	}
	return session, nil
}

// Init initializes the session manager
//
//	sessionFile with session storage. Consider using tmpfs for performance.
//	hubURL with address of the hub message bus
func (sm *SessionManager) Init(sessionFile string, hubURL string, core string, caCert *x509.Certificate) error {
	sm.sessionFile = sessionFile
	sm.hubURL = hubURL
	sm.caCert = caCert
	sm.core = core
	err := sm.Load()
	return err
}

// Load restores saved sessions that are not expired
// This return an error if the session file exists and is corrupted.
func (sm *SessionManager) Load() error {
	if sm.sessionFile == "" {
		return nil
	}
	var loadedSessions map[string]*ClientSession
	data, err := os.ReadFile(sm.sessionFile)
	if err != nil {
		// no session file, is okay
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	err = json.Unmarshal(data, &loadedSessions)
	if err != nil {
		slog.Error("Restored sessions are invalid ",
			slog.String("sessionFile", sm.sessionFile),
			slog.String("err", err.Error()))
		return err
	} else {
		slog.Info("Reloaded sessions",
			slog.Int("count", len(loadedSessions)))
	}
	// restore session only when valid
	now := time.Now()
	for id, cs := range loadedSessions {
		if now.Compare(cs.Expiry) >= 0 {
			slog.Info("Dropping expired session",
				slog.String("sessionID", id),
				slog.String("loginID", cs.LoginID))
		} else if cs.AuthToken == "" {
			slog.Info("Dropping session without auth token",
				slog.String("sessionID", id),
				slog.String("loginID", cs.LoginID))
		} else {
			// not expired
			cs2 := sm.Add(id, cs.LoginID, cs.RemoteAddr, cs.AuthToken)
			// TODO: generate kp for this session?
			err = cs2.Reconnect()
			if err != nil {
				slog.Warn("Connect failed for restored session",
					"sessionID", id,
					"loginID", cs.LoginID,
					"err", err.Error())
			} else {
				slog.Info("Connect succeeded for restored session",
					"sessionID", id,
					"loginID", cs.LoginID)
			}
			// TODO: renew the token
		}
	}
	// save the valid sessions
	_ = sm.Save()
	return err
}

// Remove closes a session and disconnect if connected
func (sm *SessionManager) Remove(sessionID string) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	si, found := sm.sessions[sessionID]
	if !found {
		return
	}
	if si.hc != nil {
		si.hc.Disconnect()
	}
	delete(sm.sessions, sessionID)
}

// Save current sessions if a sessionFile is set with Init()
func (sm *SessionManager) Save() error {
	if sm.sessionFile == "" {
		// no session persistence
		return nil
	}

	data, err := json.Marshal(sm.sessions)
	if err == nil {
		err = os.WriteFile(sm.sessionFile, data, 0600)
	}
	return err
}

// The global session manager instance
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
