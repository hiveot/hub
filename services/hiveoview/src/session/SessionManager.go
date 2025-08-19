package session

import (
	"crypto/ed25519"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/clients"
	"github.com/hiveot/hub/messaging/clients/authenticator"
	"github.com/hiveot/hub/messaging/servers/httpbasic"
	"github.com/hiveot/hub/services/hiveoview/src"
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
	// this service's agent for publishing events
	ag *messaging.Agent

	// persistence of dashboard configuration
	configStore buckets.IBucketStore

	// timeout for hub connections
	timeout time.Duration
}

// add a new session with the given hub connection and send a session count event
func (sm *WebSessionManager) _addSession(
	r *http.Request, cid string, cc messaging.IClientConnection) (
	cs *WebClientSession, err error) {

	// if the browser does not provide a CID until after the first connection,
	// then browser refresh (F5) will replace this non-cid connection.
	// (htmx sse-connect doesn't provide a cid in spite of hx-header field)
	//
	// FIXME: the first connection (without cid) doesn't shutdown until it gains
	// a sse connection and loses it again.
	cinfo := cc.GetConnectionInfo()

	sm.mux.Lock()
	clcid := cinfo.ClientID + "-" + cid
	existingSession := sm.sessions[clcid]
	if existingSession != nil {
		// Caution: Attempt to add a new connection while one exists with the cid.
		// This it can happen if a client invokes connect-with-password or token, and
		// provides an existing or empty cid.
		// Instead of throwing out the session, replace the consumer connection.
		cs = existingSession
		//exhccid := existingSession.cid //GetConnectionID()
		slog.Warn("session with clcid already exists. replace its connection.",
			slog.String("clientID", cinfo.ClientID),
			slog.String("clcid", cs.clcid),
		)

		// WARNING: disconnect can take a while to call back into SM to remove the
		// session (plus its waiting for a lock).
		// If the connection has been replaced then it won't match so ignore the
		// callback if the connection differs.
		co := messaging.NewConsumer(cc, sm.timeout)
		existingSession.ReplaceConsumer(co)
	} else {
		co := messaging.NewConsumer(cc, sm.timeout)
		clientID := co.GetClientID()
		clientBucket := sm.configStore.GetBucket(clientID)

		cs = NewWebClientSession(cid, co, r.RemoteAddr, clientBucket, sm.onClose)
	}
	sm.sessions[cs.clcid] = cs
	nrSessions := len(sm.sessions)
	sm.mux.Unlock()

	//Update the session cookie with the new auth token (default 14 days)
	//maxAge := time.Hour * 24 * 14
	//err = SetSessionCookie(w, clientID, newToken, maxAge, sm.signingKey)
	slog.Info("_addSession",
		slog.String("clientID", cinfo.ClientID),
		slog.String("clcid", cs.clcid),
		slog.String("remoteAdddr", r.RemoteAddr),
		slog.String("cc.cid", cinfo.ConnectionID),
		slog.Int("nr sessions", len(sm.sessions)),
	)
	_ = sm.ag.PubEvent(src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions)
	return cs, err
}

// _removeSession removes the session  and sends a session event.
// Make sure the connection is closed before calling this
func (sm *WebSessionManager) _removeSession(cs *WebClientSession) {
	// do not call back into cs as this callback takes place in a locked section.
	isConnected := cs.co.IsConnected()
	//hccid := cs.clcid//GetConnectionID()

	if isConnected {
		slog.Warn("_removeSession. session still has a connection")
	}

	sm.mux.Lock()
	sess, hasSession := sm.sessions[cs.clcid]
	if !hasSession {
		slog.Error("_removeSession unknown client connection ID",
			"clientID", cs.GetClientID(),
			"clcid", cs.clcid)
	}
	// close the storage bucket that was created for this session
	err := sess.clientData.dataBucket.Close()
	if err != nil {
		slog.Error("_removeSession: Error closing storage bucket: ", "err", err.Error())
	}

	delete(sm.sessions, cs.clcid)
	nrSessions := len(sm.sessions)
	sm.mux.Unlock()

	// 5. publish the new nr of sessions
	slog.Info("_removeSession",
		slog.String("clientID", cs.GetClientID()),
		slog.String("clcid", cs.GetCLCID()),
		slog.Bool("isConnected", isConnected),
		slog.Int("nrSessions", nrSessions),
		//slog.String("hc.cid", hccid),
	)
	go func() {
		_ = sm.ag.PubEvent(src.HiveoviewServiceID, src.NrActiveSessionsEvent, nrSessions)
	}()
}

// CloseAllWebSessions disconnects all the web client sessions by disconnecting the client side
func (sm *WebSessionManager) CloseAllWebSessions() {
	sm.mux.RLock()
	// shallow copy of the map into an array of remaining sesions
	sessions := make([]*WebClientSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	sm.mux.RUnlock()
	for _, sess := range sessions {
		slog.Warn("CloseAllWebSessions", "clientID", sess.co.GetClientID())
		sess.co.Disconnect()
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
//	_, err := sm.HandleConnectWithPassword(w, r, loginID, password, cid)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusUnauthorized)
//	}
//}

// HandleConnectWithPassword handles the request to log a consumer in to the Hub
// (or any Thing server) using the given password.
// A cid (connection-id) is required to differentiate between browser tabs.
//
// This:
//  1. Logs in to the Hub, obtaining a new auth token to reconnect later.
//  2. Creates a hiveoview session using the given connection-id.
//  3. Set a secure session cookie with the browser that contains the clientID and
//     auth token for reconnecting without password.
func (sm *WebSessionManager) HandleConnectWithPassword(
	w http.ResponseWriter, r *http.Request,
	loginID string, password string, cid string) (newToken string, err error) {

	// Authentication uses its own client that knows the auth protocol
	parts, _ := url.Parse(sm.hubURL)
	authCl := authenticator.NewAuthClient(parts.Host, sm.caCert, cid, sm.timeout)

	// attempt to login
	newToken, err = authCl.LoginWithPassword(loginID, password)

	if err != nil {
		return "", err
	}
	// FIXME: use the session's directory cache to get the form
	cc, err := clients.ConnectWithToken(loginID, newToken, sm.caCert, sm.hubURL, sm.timeout)
	if err == nil {
		if cid != "" {
			_, err = sm._addSession(r, cid, cc)
		} else {
			cc.Disconnect()
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

	slog.Info("SetBearerToken",
		"loginID", loginID, "cid", cid, "remoteAddr", r.RemoteAddr,
		"nr websessions", len(sm.sessions))
	//var newToken string
	cc, err := clients.ConnectWithToken(
		loginID, authToken, sm.caCert, sm.hubURL, sm.timeout)
	if err == nil {
		cs, err = sm._addSession(r, cid, cc)
		// Update the session cookie with the new auth token (default 14 days)
		maxAge := time.Hour * 24 * 14
		err = SetSessionCookie(w, loginID, authToken, maxAge, sm.signingKey)
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

	// The following should match SetBearerToken
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
	cid = r.Header.Get(httpbasic.ConnectionIDHeader)
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
//	hubURL to connect the clients
//	signingKey for use with session cookies
//	caCert of the hub
//	hc is the agent service connection for reporting notifications and handling config
//	configStore data store for client configuration
//	timeout of hub connections
func NewWebSessionManager(
	signingKey ed25519.PrivateKey, caCert *x509.Certificate,
	ag *messaging.Agent, configStore buckets.IBucketStore,
	timeout time.Duration) *WebSessionManager {

	cc := ag.GetConnection()
	cinfo := cc.GetConnectionInfo()
	hubURL := cinfo.ConnectURL
	sm := &WebSessionManager{
		sessions:    make(map[string]*WebClientSession),
		mux:         sync.RWMutex{},
		signingKey:  signingKey,
		pubKey:      signingKey.Public().(ed25519.PublicKey),
		hubURL:      hubURL,
		caCert:      caCert,
		ag:          ag,
		configStore: configStore,
		timeout:     timeout,
	}
	return sm
}
