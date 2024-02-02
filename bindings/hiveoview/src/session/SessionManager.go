package session

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"github.com/google/uuid"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
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
	// Hub core if known (mqtt or nats)
	core string

	// keys to use for clients that have no public key set
	tokenKP keys.IHiveKey
}

// Close closes the hub connection and event channel, and removes the session
func (sm *SessionManager) Close(sessionID string) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()

	si, found := sm.sessions[sessionID]
	if !found {
		slog.Error("Close. Session not found. This is unexpected.",
			"sessionID", sessionID)
		return errors.New("Session not found")
	}
	si.Close()
	delete(sm.sessions, sessionID)
	return nil
}

// NewHubClient creates a new hub client using the configured URL and core
func (sm *SessionManager) NewHubClient(loginID string) *hubclient.HubClient {
	hc := hubclient.NewHubClient(sm.hubURL, loginID, sm.caCert, sm.core)
	return hc
}

// ConnectWithToken to the Hub using an existing token.
// This returns the session info.
//func ConnectWithToken(clientID string, token string) (*ClientSession, error) {
//}

// GetSession returns the client session if available
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

// GetSessionFromCookie returns the session from the session cookie
// If no active session exists but a valid cookie was found then try to
// activate a session instance.
// If no valid cookie is found then return an error
//
// Passing a response writer is optional. It allows to refresh the auth token in the cookie.
func (sm *SessionManager) GetSessionFromCookie(w http.ResponseWriter, r *http.Request) (*ClientSession, error) {
	var cs *ClientSession
	claims, err := GetSessionCookie(r, &sm.signingKey.PublicKey)
	if err != nil {
		return nil, err
	}
	// return the active session
	cs, err = sm.GetSession(claims.ID)
	if err == nil && cs.IsActive() {
		// success
		return cs, nil
	}
	// try to re-activate the session
	cs = sm.ActivateSession(w, *claims)
	return cs, nil
}

// Init initializes the session manager
//
//		hubURL with address of the hub message bus
//		messaging core to use or "" for auto-detection
//		signingKey for cookies
//		caCert of the messaging server
//	 tokenKP optional keys to use for refreshing tokens of authenticated users
func (sm *SessionManager) Init(hubURL string, core string,
	signingKey *ecdsa.PrivateKey, caCert *x509.Certificate,
	tokenKP keys.IHiveKey) {
	sm.hubURL = hubURL
	sm.caCert = caCert
	sm.core = core
	sm.signingKey = signingKey
	sm.tokenKP = tokenKP
}

// LoginToSession updates or creates a session for the given Hub client.
// This tries to read the current session cookie to determine the session ID
// or creates a session ID if no cookie exists.
//
// This updates the hub client of the existing session if one exists, or
// closes the given client if the existing session is already active in order
// to retain any existing subscriptions for SSE notifications.
// If no session exists then create a new session.
//
// Last, refresh the auth token, generate a JWT token containing claims for
// the session ID, login ID and auth token, and store this token in the
// session cookie. If no public key exists, the key-pair of this service is
// used.
//
// From here on the session is active and can be reactived from the cookie if
// needed, regardless the previous state.
//
//	w is the response write used to store the new cookie
//	r is the request reader used to obtain existing session ID and login ID
//	hc is a connected Hub client for use in the session
//	maxAge is the time in seconds the session is active for. Use 0 to deactivate on browser exit
func (sm *SessionManager) LoginToSession(
	w http.ResponseWriter, r *http.Request, newhc *hubclient.HubClient, maxAge int) {
	var sessionID = ""
	var cs *ClientSession

	// get the existing session jwt, if it exists, to determine the session ID
	claims, _ := GetSessionCookie(r, &sm.signingKey.PublicKey)
	if claims != nil {
		sessionID = claims.ID
		clientID := claims.Subject
		// verify the cookie matches the session
		sm.mux.RLock()
		cs = sm.sessions[sessionID]
		sm.mux.RUnlock()
		// verify if a session exist and has the correct clientID
		if cs != nil {
			if cs.clientID != clientID {
				slog.Error("ERROR, session cookie clientID differs from session clientID. Should not happen.",
					"cookie clientID", clientID,
					"session clientID", cs.clientID,
					"remote addr", r.RemoteAddr)
				return
			}
		}
	} else {
		sessionID = uuid.NewString()
	}
	// if an active client session exists then disconnect the given client
	// and leave the client in place.
	if cs != nil {
		if cs.IsActive() {
			newhc.Disconnect()
			newhc = cs.hc
		} else {
			// replace the existing session HC with the given one.
			cs.hc.Disconnect()
			cs.ReplaceHubClient(newhc) // and subscribe to events
		}
	} else {
		// create a new session
		cs = NewClientSession(sessionID, newhc)
		sm.mux.Lock()
		sm.sessions[sessionID] = cs
		sm.mux.Unlock()
	}

	// last, refresh the auth token and update the cookie
	// if this fails then assign this service's public key, which is good enough
	// for webclient users.
	profileClient := authclient.NewProfileClient(newhc)
	authToken, err := profileClient.RefreshToken()
	if err != nil && sm.tokenKP != nil {
		// try to recover by ensuring a public key exists on the account.
		// this fallback is only useful for users that login using this backend service.
		// This is probable as otherwise a public key would have been set already.
		prof, err := profileClient.GetProfile()
		if err == nil {
			if prof.PubKey == "" {
				pubKey := sm.tokenKP.ExportPublic()
				err = profileClient.UpdatePubKey(pubKey)
			}
			// retry
			authToken, err = profileClient.RefreshToken()
		}
		if err != nil {
			slog.Error("Failed refreshing auth token. Session remains active.",
				"err", err.Error())
		}
		return
	}

	SetSessionCookie(w, sessionID, newhc.ClientID(), authToken, maxAge, sm.signingKey)
}

// ActivateSession creates a new session instance.
//
// If no session cookie exists then this returns nil.
// If a session cookie exists with a session ID then create a new session
// instance if one doesn't already exist.
//
// If the session is not connected then attempt to reconnect.
//
// If a connection exists or is successful then refresh the auth token
// and update the cookie.
//
// Passing a ResponseWriter is optional and allows for refreshing the auth token.
//
// This always returns a session or nil if a session could not be created.
func (sm *SessionManager) ActivateSession(
	w http.ResponseWriter, claims SessionClaims) *ClientSession {

	sm.mux.Lock()
	defer sm.mux.Unlock()
	doRefreshToken := false

	sessionID := claims.ID
	clientID, _ := claims.GetSubject()
	authToken := claims.AuthToken

	cs, exists := sm.sessions[sessionID]
	if !exists {
		// no session exists, so create a new session with hub client
		hc := hubclient.NewHubClient(sm.hubURL, clientID, sm.caCert, sm.core)
		if authToken != "" {
			// FIXME, use a key-pair, needed by NATS
			err := hc.ConnectWithToken(nil, authToken)
			if err != nil {
				slog.Error("ActivateSession. Error connecting to the hub",
					slog.String("sessionID", sessionID),
					slog.String("loginID", clientID),
					slog.String("err", err.Error()))
				// TODO: if authentication failed, the token expired.
				// Remove it the auth token in the cookie
				// TODO: if the server is unreachable, then notify the caller
			} else {
				doRefreshToken = true
				slog.Info("ActivateSession. Connected to the hub",
					slog.String("sessionID", sessionID),
					slog.String("loginID", clientID))
			}
		}
		cs = NewClientSession(sessionID, hc)
		sm.sessions[sessionID] = cs
	} else if cs.IsActive() {
		// an active session exists, no need to connect but do refresh the token
		doRefreshToken = true
	} else {
		// a session exists but is not active. Try to reconnect
		// FIXME: a client key is required in NATS
		// how does the web client get a key ?
		err := cs.hc.ConnectWithToken(nil, authToken)
		doRefreshToken = err == nil
	}
	// refresh the auth token if needed and a response writer is provided
	if doRefreshToken && w != nil {
		// use the auth service 'profile client' capability
		profileClient := authclient.NewProfileClient(cs.hc)
		newToken, err := profileClient.RefreshToken()
		if err == nil {
			SetSessionCookie(w, sessionID, cs.hc.ClientID(), newToken, claims.MaxAge, sm.signingKey)
		} else {
			slog.Error("ActivateSession. Auth token refresh failed", "err", err.Error())
		}
	}
	return cs
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

// GetSession returns the session for the given request
// This tries to reactivate the session if a session cookie with credentials
// is available
//
// Passing a ResponseWriter is optional and allows for updating a cookie.
// This should not be used in an SSE session.
func GetSession(w http.ResponseWriter, r *http.Request) (*ClientSession, error) {
	return sessionmanager.GetSessionFromCookie(w, r)
}
