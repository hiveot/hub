package session

import (
	"crypto/ed25519"
	"crypto/x509"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot"
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
	hc transports.IAgentConnection
	// disable persistence from state service (for testing)
	noState bool

	// timeout for hub connections
	timeout time.Duration
}

// add a new session with the given clientID and send a session count event
// This sets the getForm handler for the client for using this session TD directory
func (sm *WebSessionManager) _addSession(
	r *http.Request, cid string, hc transports.IConsumerConnection) (
	cs *WebClientSession, err error) {

	// if the browser does not provide a CID until after the first connection,
	// then browser refresh (F5) will replace this non-cid connection.
	// (htmx sse-connect doesn't provide a cid in spite of hx-header field)
	//
	// FIXME: the first connection (without cid) doesn't shutdown until it gains
	// a sse connection and loses it again.
	clientID := hc.GetClientID()

	sm.mux.Lock()
	clcid := clientID + "-" + cid
	existingSession := sm.sessions[clcid]
	if existingSession != nil {
		// Ugly hack!
		// this is an attempt to add a new connection using an existing cid.
		// it can happen if a client invokes connect-with-password or token, and
		// provides an existing or empty cid.
		// Instead of throwing out the session, replace the connection.
		cs = existingSession
		exhccid := existingSession.hc.GetConnectionID()
		slog.Warn("session with clcid already exists. replace its connection.",
			slog.String("clientID", clientID),
			slog.String("clcid", cs.clcid),
			slog.String("ex hc.cid", exhccid),
			//slog.String("existing HC CID", existingSession.hc.GetCid()),
		)

		// WARNING: disconnect can take a while to call back into SM to remove the
		// session (plus its waiting for a lock).
		// If the connection has been replaced then it won't match so ignore the
		// callback if the connection differs.
		existingSession.ReplaceConnection(hc)
	} else {
		cs = NewWebClientSession(cid, hc, r.RemoteAddr, sm.noState, sm.onClose)
		hccid := hc.GetConnectionID()
		slog.Info("_addSession",
			slog.String("clientID", clientID),
			slog.String("clcid", cs.clcid),
			slog.String("remoteAdddr", r.RemoteAddr),
			slog.String("hc.cid", hccid),
			slog.Int("nr sessions", len(sm.sessions)),
		)
	}
	sm.sessions[cs.clcid] = cs
	hc.SetGetForm(cs.GetForm)
	nrSessions := len(sm.sessions)
	sm.mux.Unlock()

	//Update the session cookie with the new auth token (default 14 days)
	//maxAge := time.Hour * 24 * 14
	//err = SetSessionCookie(w, clientID, newToken, maxAge, sm.signingKey)

	// publish the new nr of sessions
	notif := transports.NewNotificationResponse(wot.HTOpEvent, src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions)
	_ = sm.hc.SendNotification(notif)
	return cs, err
}

// _removeSession removes the session  and sends a session event.
// Make sure the connection is closed before calling this
func (sm *WebSessionManager) _removeSession(cs *WebClientSession) {
	// do not call back into cs as this callback takes place in a locked section.
	isConnected := cs.hc.IsConnected()
	hccid := cs.hc.GetConnectionID()
	slog.Info("_removeSession",
		slog.String("clientID", cs.GetClientID()),
		slog.String("clcid", cs.GetCLCID()),
		slog.Bool("isConnected", isConnected),
		slog.String("hc.cid", hccid),
	)
	if isConnected {
		slog.Warn("_removeSession. session still has a connection")
	}

	sm.mux.Lock()
	_, hasSession := sm.sessions[cs.clcid]
	if !hasSession {
		slog.Error("_removeSession is missing",
			"clientID", cs.GetClientID(),
			"clcid", cs.clcid)
	}

	delete(sm.sessions, cs.clcid)
	nrSessions := len(sm.sessions)
	sm.mux.Unlock()

	// 5. publish the new nr of sessions
	go func() {
		notif := transports.NewNotificationResponse(wot.HTOpEvent, src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions)
		_ = sm.hc.SendNotification(notif)
	}()
}

// disconnect all the web client sessions by disconnecting the client side
func (sm *WebSessionManager) CloseAllWebSessions() {
	sm.mux.RLock()
	// shallow copy of the map into an array of remaining sesions
	sessions := make([]*WebClientSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	sm.mux.RUnlock()
	for _, s := range sessions {
		slog.Warn("CloseAllWebSessions", "clientID", s.hc.GetClientID())
		s.hc.Disconnect()
	}
}

// onClose handles closing of the client connection
func (sm *WebSessionManager) onClose(cs *WebClientSession) {
	sm._removeSession(cs)
}

// PostLogin creates a hub connection for the client and adds a new session
//func (sm *WebSessionManager) PostLogin(w http.ResponseWriter, r *http.Request) {
//	// obtain login form fields
//	loginID := r.FormValue("loginID")
//	password := r.FormValue("password")
//	cid := r.Header.Get(hubclient.ConnectionIDHeader)
//
//	if loginID == "" && password == "" {
//		http.Redirect(w, r, src.RenderLoginPath, http.StatusBadRequest)
//		//w.WriteHeader(http.StatusBadRequest)
//		return
//	}
//	_, err := sm.ConnectWithPassword(w, r, loginID, password, cid)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusUnauthorized)
//	}
//}

// ConnectWithPassword logs a consumer in to the hub using the given password.
// If successful this updates the secure cookie with a new auth token and also
// returns this token.
// If a cid is provided in the headers it will create a session using it.
func (sm *WebSessionManager) ConnectWithPassword(
	w http.ResponseWriter, r *http.Request,
	loginID string, password string, cid string) (newToken string, err error) {

	//
	hc, err := clients.NewConsumerClient(sm.hubURL, loginID, sm.caCert, nil, sm.timeout)
	if err == nil {
		newToken, err = hc.ConnectWithPassword(password)
	}
	if err == nil {
		if cid != "" {
			_, err = sm._addSession(r, cid, hc)
		} else {
			hc.Disconnect()
		}
		// Update the session cookie with the new auth token (default 14 days)
		maxAge := time.Hour * 24 * 14
		err = SetSessionCookie(w, loginID, newToken, maxAge, sm.signingKey)

		// this will prevent a redirect from working
		//newTokenJSON, _ := jsoniter.Marshal(newToken)
		//w.Write(newTokenJSON)
	}
	return newToken, err
}

// ConnectWithToken logs-in to the hub using the given valid auth token.
//
// If successful this updates the secure cookie with a new auth token.
//
// !! If a cid is provided it will create a session for it otherwise it must
// be closed by the caller.
//
// The 'cid' field, provided by the client, is used to differentiate between
// multiple client connections. Without it, only a single connection per client is
// accepted which will mess up browser tabs.
//
// Note that at this time there is not yet an SSE connection. Notifications
// for the web browser will be discarded until a SSE connection is established.
func (sm *WebSessionManager) ConnectWithToken(
	w http.ResponseWriter, r *http.Request, loginID string, cid string, authToken string) (
	cs *WebClientSession, err error) {

	slog.Info("ConnectWithToken",
		"clientID", loginID, "cid", cid, "remoteAddr", r.RemoteAddr,
		"nr websessions", len(sm.sessions))
	var newToken string

	hc, err := clients.NewConsumerClient(sm.hubURL, loginID, sm.caCert, nil, sm.timeout)
	if err == nil {
		newToken, err = hc.ConnectWithToken(authToken)
	}
	if err == nil {
		cs, err = sm._addSession(r, cid, hc)
		// Update the session cookie with the new auth token (default 14 days)
		maxAge := time.Hour * 24 * 14
		err = SetSessionCookie(w, loginID, newToken, maxAge, sm.signingKey)
	}

	return cs, err
}
func (sm *WebSessionManager) GetNrSessions() int {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	return len(sm.sessions)
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

// GetSessionFromCookie returns the websession and auth info
// The authentication token comes from the cookie.
//
// This should only be used from the middleware.
//
// If no session exists but a cookie is found then return the cookie claims.
// If no valid cookie is found then return an error
func (sm *WebSessionManager) GetSessionFromCookie(r *http.Request) (
	cs *WebClientSession, clientID string, cid string, authToken string, err error) {

	// The UI is supposed to supply a cid header in sse requests. However,
	// sse-connect ignores the hx-header that contains the cid. As a workaround
	// also check query parameters.
	cid = r.Header.Get(httpserver.ConnectionIDHeader)
	if cid == "" {
		//slog.Error("GetSessionFromCookie: Missing CID")
		cid = r.URL.Query().Get("cid")
	}

	clientID, authToken, err = GetSessionCookie(r, sm.pubKey)
	if err != nil {
		return nil, clientID, cid, authToken, err
	}
	// return the session if active
	cs = sm.GetSession(clientID, cid)
	return cs, clientID, cid, authToken, nil
}

// NewWebSessionManager creates a new instance of the hiveoview service
// session manager.
//
//	signingKey for use with session cookies
//	caCert of the hub
//	hc is the agent service connection for reporting notifications and handling config
//	noState do not try to persist state with the state service (for testing)
//	timeout of hub connections
func NewWebSessionManager(hubURL string,
	signingKey ed25519.PrivateKey, caCert *x509.Certificate,
	hc transports.IAgentConnection, noState bool,
	timeout time.Duration) *WebSessionManager {
	sm := &WebSessionManager{
		sessions:   make(map[string]*WebClientSession),
		mux:        sync.RWMutex{},
		signingKey: signingKey,
		pubKey:     signingKey.Public().(ed25519.PublicKey),
		hubURL:     hubURL,
		caCert:     caCert,
		hc:         hc,
		noState:    noState,
		timeout:    timeout,
	}
	return sm
}
