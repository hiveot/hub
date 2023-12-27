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

// Add a new sessions for the given client after a successful login
// If a session with the given ID already exists, it is returned
// The session contains a hub client for publishing and subscribing messages
func (sm *SessionManager) Add(
	sessionID string, loginID string, expiry time.Time, remoteAddr string) (*ClientSession, error) {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	// TODO: limit max nr of sessions per client/remote addr

	si, exists := sm.sessions[sessionID]
	if !exists {
		si = &ClientSession{
			//SessionID:  sessionID,
			LoginID:    loginID,
			Expiry:     expiry,
			RemoteAddr: remoteAddr,
			hc:         hubclient.NewHubClient(sm.hubURL, loginID, sm.caCert, sm.core),
		}
		sm.sessions[sessionID] = si
		// TODO: save with delay
		err := sm.save()
		return si, err
	}
	return si, nil
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
			"sessionFile", sm.sessionFile, "err", err.Error())
		return err
	} else {
		slog.Info("Restored sessions", "count", len(loadedSessions))
	}
	// restore session only when valid
	now := time.Now()
	for id, si := range loadedSessions {
		if now.Compare(si.Expiry) == -1 {
			// not expired
			si2, _ := sm.Add(id, si.LoginID, si.Expiry, si.RemoteAddr)
			if si.AuthToken != "" {
				// TODO: generate kp for this session?
				si2.hc.ConnectWithToken(nil, si.AuthToken)
				// TODO: renew the token
			}
		}
	}
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
func (sm *SessionManager) save() error {
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
