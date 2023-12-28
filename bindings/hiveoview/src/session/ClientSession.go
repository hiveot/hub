package session

import (
	"errors"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
	"sync"
	"time"
)

// DefaultExpiryHours TODO: set default expiry in config
const DefaultExpiryHours = 72

// ClientSession of a web client containing a hub connection
type ClientSession struct {
	// The user's session
	SessionID string
	// User ID for login
	LoginID string `json:"login_id"`
	// Auth token is obtained from the secure cookie and used to re-connect to the hub
	AuthToken string `json:"auth_token"`
	// Expiry of the session
	Expiry time.Time `json:"expiry"`
	// remote address
	RemoteAddr string `json:"remote_addr"`
	// The associated hub client for pub/sub
	hc *hubclient.HubClient
	// session mutex
	mux sync.RWMutex
}

// ConnectWithPassword connects the session with the Hub using the given password
//
// This obtains a token for restoring the connection at a later time without
// requiring a login. See Reconnect()
func (si *ClientSession) ConnectWithPassword(password string) error {

	err := si.hc.ConnectWithPassword(password)
	if err != nil {
		return err
	}
	authCl := authclient.NewProfileClient(si.hc)
	token, err := authCl.RefreshToken()
	if err == nil {
		si.mux.Lock()
		si.AuthToken = token
		si.mux.Unlock()
	}
	return err
}

// GetStatus returns the status of hub connection
// This returns:
//
//	status transports.ConnectionStatus
//	 * expired when session is expired (and renew failed)
//	 * connected when connected to the hub
//	 * connecting or disconnected when not connected
//	info with a human description
func (si *ClientSession) GetStatus() transports.HubTransportStatus {
	return si.hc.GetStatus()
}

// GetConnection returns the hub client connection for use in pub/sub
func (si *ClientSession) GetConnection() *hubclient.HubClient {
	return si.hc
}

// Reconnect restores the session's connection with the hub
// It returns an error if no valid auth token is available.
// It is recommended to save the session after successful reconnect, to persist
// the auth token.
func (si *ClientSession) Reconnect() error {
	si.mux.Lock()
	defer si.mux.Unlock()

	if si.AuthToken == "" {
		return errors.New("unable to reconnect no auth token")
	}

	// TODO: update expiry with a new token
	// the user's private key is not available. This token auth only works
	// if it doesn't require one.
	err := si.hc.ConnectWithToken(nil, si.AuthToken)
	if err != nil {
		return err
	}
	myProfile := authclient.NewProfileClient(si.hc)
	si.AuthToken, err = myProfile.RefreshToken()
	if err != nil {
		slog.Error("Token refresh failed. Continuing as existing token is still valid.",
			"clientID", si.LoginID, "err", err)
		err = nil
	} else {
		// TODO: use token expiry
		si.Expiry = time.Now().Add(DefaultExpiryHours * time.Hour)
	}
	return err
}

// NewClientSession creates a new client session for the given Hub connection
// Intended for use by the session manager.
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(
	sessionID string, hc *hubclient.HubClient, remoteAddr string, authToken string) *ClientSession {
	cs := ClientSession{
		SessionID:  sessionID,
		LoginID:    hc.ClientID(),
		RemoteAddr: remoteAddr,
		Expiry:     time.Now().Add(DefaultExpiryHours * time.Hour),
		hc:         hc,
		AuthToken:  authToken,
	}
	return &cs
}
