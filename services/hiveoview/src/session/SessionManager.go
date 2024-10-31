package session

import (
	"crypto/ed25519"
	"crypto/x509"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/connect"
	"github.com/hiveot/hub/services/hiveoview/src"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// WebSessionManager tracks client sessions using session cookies
// TODO:
//  1. close session after not being used for X seconds
//  2. publish a login event on the message bus
type WebSessionManager struct {
	// existing sessions by ClientConnectionID
	sessions map[string]*WebClientSession
	// mutex to access the sessions
	mux sync.RWMutex
	// signing key for creating and verifying cookies
	signingKey ed25519.PrivateKey
	pubKey     ed25519.PublicKey

	// Hub address
	hubURL string
	// Hub CA certificate
	caCert *x509.Certificate
	// hub client for publishing events
	hc hubclient.IHubClient
}

// add a new session with the given clientID
func (sm *WebSessionManager) _addSession(w http.ResponseWriter, r *http.Request,
	cid string, hc hubclient.IConsumerClient, newToken string) (
	cs *WebClientSession, err error) {

	clientID := hc.GetClientID()
	cs = NewWebClientSession(cid, hc, r.RemoteAddr, sm.onClose)
	slog.Info("_addSession",
		slog.String("clientID", clientID),
		slog.String("clcid", cs.clcid),
		slog.String("remoteAdddr", r.RemoteAddr),
	)
	sm.mux.Lock()
	sm.sessions[cs.clcid] = cs
	nrSessions := len(sm.sessions)
	sm.mux.Unlock()

	// Update the session cookie with the new auth token (default 14 days)
	maxAge := time.Hour * 24 * 14
	err = SetSessionCookie(w, clientID, newToken, maxAge, sm.signingKey)

	// publish the new nr of connections
	go sm.hc.PubEvent(src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions, "")
	return cs, err
}

// onClose handles closing of the client connection
func (sm *WebSessionManager) onClose(cs *WebClientSession) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	// do not call back into cs as this callback takes place in a locked section.
	slog.Info("onClose session",
		slog.String("clientID", cs.GetClientID()),
		slog.String("clcid", cs.GetCLCID()),
	)

	delete(sm.sessions, cs.clcid)
	nrSessions := len(sm.sessions)

	// 5. publish the new nr of sessions
	go sm.hc.PubEvent(src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions, "")
}

// PostLogin creates a hub connection for the client and adds a new session
func (sm *WebSessionManager) PostLogin(w http.ResponseWriter, r *http.Request) {
	// obtain login form fields
	loginID := r.FormValue("loginID")
	password := r.FormValue("password")
	cid := r.Header.Get(hubclient.ConnectionIDHeader)

	if loginID == "" && password == "" {
		http.Redirect(w, r, src.RenderLoginPath, http.StatusBadRequest)
		//w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err := sm.ConnectWithPassword(w, r, loginID, password, cid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
}

// ConnectWithPassword creates a new hub client and connect it to the hub using password login
// If successful this adds the session and updates a session cookie.
func (sm *WebSessionManager) ConnectWithPassword(w http.ResponseWriter, r *http.Request,
	loginID string, password string, cid string) (
	cs *WebClientSession, err error) {

	hc := connect.NewHubClient(sm.hubURL, loginID, sm.caCert)
	newToken, err := hc.ConnectWithPassword(password)
	if err == nil {
		cs, err = sm._addSession(w, r, cid, hc, newToken)
	}
	return cs, err
}

// ConnectWithToken creates a new hub client and connect it to the hub using token login.
//
// On success add a client session for this connection and update the cookie with
// the refreshed token.
//
// The 'cid' header is provided by the client and used to differentiate between
// multiple client connections. Without it, only a single connection per client is
// accepted. Used to link requests to the notification return channel.
//
// Note that at this time there is not yet an SSE connection. Any notifications
// for the web browser will be queued and not yet arrive until its SSE connection
// is established.
func (sm *WebSessionManager) ConnectWithToken(
	w http.ResponseWriter, r *http.Request, loginID string, cid string, authToken string) (
	cs *WebClientSession, err error) {

	// TODO: need a consumerclient instance
	hc := connect.NewHubClient(sm.hubURL, loginID, sm.caCert)
	//hc := NewConsumerClient(sm.hubURL, loginID, sm.caCert)
	newToken, err := hc.ConnectWithToken(authToken)
	if err != nil {
		return nil, err
	}
	cs, err = sm._addSession(w, r, cid, hc, newToken)

	return cs, err
}

// GetSession returns the client session if available
// The 'cid' parameter is provided by the client and used to differentiate between
// multiple client connections. Without it, only a single connection per client is
// accepted.
//
//	clientID is the login ID of the client
//	cid is the connectionID provided by the client
//
// This returns nil if there is not an existing client session
func (sm *WebSessionManager) GetSession(clientID, cid string) *WebClientSession {
	sm.mux.RLock()
	defer sm.mux.RUnlock()

	// The following should match ConnectWithToken
	clientConnID := clientID + "-" + cid

	session, found := sm.sessions[clientConnID]
	if !found {
		return nil
	}
	session.lastActivity = time.Now()
	return session
}

// GetSessionFromCookie returns the session object using the session cookie.
// This should only be used from the middleware, as reconnecting to the hub can change the sessionID.
//
// If no session exists but a cookie is found then return the cookie claims.
// If no valid cookie is found then return an error
func (sm *WebSessionManager) GetSessionFromCookie(r *http.Request) (
	cs *WebClientSession, clientID string, cid string, authToken string, err error) {

	cid = r.Header.Get(hubclient.ConnectionIDHeader)
	clientID, authToken, err = GetSessionCookie(r, sm.pubKey)
	if err != nil {
		return nil, clientID, cid, authToken, err
	}
	// return the session if active
	cs = sm.GetSession(clientID, cid)
	return cs, clientID, cid, authToken, nil
}

func NewWebSessionManager(hubURL string,
	signingKey ed25519.PrivateKey, caCert *x509.Certificate, hc hubclient.IHubClient) *WebSessionManager {
	sm := &WebSessionManager{
		sessions:   make(map[string]*WebClientSession),
		mux:        sync.RWMutex{},
		signingKey: signingKey,
		pubKey:     signingKey.Public().(ed25519.PublicKey),
		hubURL:     hubURL,
		caCert:     caCert,
		hc:         hc,
	}
	return sm
}
