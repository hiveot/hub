package session

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"github.com/google/uuid"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/connect"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SessionManager tracks client sessions using session cookies
// TODO:
//  1. close session after not being used for X seconds
//  2. publish a login event on the message bus
type SessionManager struct {
	// existing sessions by sessionID (remoteAddr)
	sessions map[string]*ClientSession
	// mutex to access the sessions
	mux sync.RWMutex
	// signing key for creating and verifying cookies
	signingKey *ecdsa.PrivateKey

	// Hub address
	hubURL string
	// Hub CA certificate
	caCert *x509.Certificate
}

// ActivateNewSession (re)activates a new session for a newly connected hub client.
//
// If a session exists, it will be closed and removed first.
// This requests a session token for storing in the cookie to allow re-opening the session
// after the browser pages is closed or refreshed, without requiring a new password.
// This replaces the session cookie in the browser with a new cookie.
// This subscribes to events of configured agents.
//
// This returns the new session instance or nil with an error if a session could not be created.
func (sm *SessionManager) ActivateNewSession(
	w http.ResponseWriter, r *http.Request, hc hubclient.IHubClient, authToken string) (*ClientSession, error) {
	var cs *ClientSession
	var sessionID string

	slog.Info("ActivateNewSession", slog.String("clientID", hc.ClientID()))
	// 1. close the existing session
	claims, err := GetSessionCookie(r, &sm.signingKey.PublicKey)
	if err == nil && claims.ID != "" {
		sessionID = claims.ID
		cs, err = sm.GetSession(sessionID)
		if cs != nil {
			err = sm.Close(sessionID)
			if err != nil {
				slog.Error("Error closing session. Continuing anyways", "err", err.Error())
			}
		}
	}

	// 2. create a new session using the given connection, if any
	// re-use the session ID if there was one.
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	// create a session for this connection and subscribe to events from configured agents.
	cs = NewClientSession(sessionID, hc, r.RemoteAddr)
	sm.mux.Lock()
	sm.sessions[sessionID] = cs
	sm.mux.Unlock()

	// 3. Get a new auth token from the Hub auth service
	//profileClient := authnclient.NewAuthnUserClient(hc)
	authToken, err = hc.RefreshToken(authToken)

	//if err != nil && sm.tokenKP != nil {
	//	// Oops, refresh failed. This happens if the account has no public key set. (quite common)
	//	// Try to recover by ensuring a public key exists on the account.
	//	// This fallback is only useful in case authenticating takes place through this service,
	//	// as other clients won't have this public key.
	//	prof, err2 := profileClient.GetProfile()
	//	err = err2
	//	if err == nil {
	//		// use this service key-pair
	//		if prof.PubKey == "" {
	//			pubKey := sm.tokenKP.ExportPublic()
	//			err = profileClient.UpdatePubKey(hc.ClientID(), pubKey)
	//		}
	//		// retry getting a token
	//		authToken, err = profileClient.RefreshToken()
	//	}
	//}
	if err != nil {
		slog.Warn("DeliveryFailed refreshing auth token. Session remains active.",
			"err", err.Error())

	}

	// 4. Keep the session for 14 days
	maxAge := 3600 * 24 * 14
	err = SetSessionCookie(w, sessionID, hc.ClientID(), authToken, maxAge, sm.signingKey)
	return cs, err
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

// ConnectWithPassword creates a new hub client and connect it to the hub using password login
// This returns a new token for future logins
func (sm *SessionManager) ConnectWithPassword(loginID string, password string) (
	hc hubclient.IHubClient, newToken string, err error) {

	hc = connect.NewHubClient(sm.hubURL, loginID, sm.caCert)
	newToken, err = hc.ConnectWithPassword(password)
	// subscribe to updates
	return hc, newToken, err
}

// ConnectWithToken creates a new hub client and connect it to the hub using token login
func (sm *SessionManager) ConnectWithToken(loginID string, authToken string) (hubclient.IHubClient, error) {
	hc := connect.NewHubClient(sm.hubURL, loginID, sm.caCert)
	_, err := hc.ConnectWithToken(authToken)
	return hc, err
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

// GetSessionFromCookie returns the session object using the session cookie.
// This should only be used from the middleware, as reconnecting to the hub can change the sessionID.
//
// If no session exists but a cookie is found then return the cookie claims.
// If no valid cookie is found then return an error
func (sm *SessionManager) GetSessionFromCookie(r *http.Request) (*ClientSession, *SessionClaims, error) {
	var cs *ClientSession
	claims, err := GetSessionCookie(r, &sm.signingKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	// return the session if active
	cs, err = sm.GetSession(claims.ID)
	return cs, claims, err
}

// Init initializes the session manager
//
//	hubURL with address of the hub message bus
//	messaging core to use or "" for auto-detection
//	signingKey for cookies
//	caCert of the messaging server
func (sm *SessionManager) Init(
	hubURL string, signingKey *ecdsa.PrivateKey, caCert *x509.Certificate) {
	sm.hubURL = hubURL
	sm.caCert = caCert
	sm.signingKey = signingKey
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
