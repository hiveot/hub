package session

import (
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"time"
)

// ClientSession of a web client containing a hub connection
type ClientSession struct {
	// The user's session
	//SessionID string
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
}

// ConnectWithPassword reconnects the session to the Hub using the given password
// This obtains a token for restoring the connection at a later time.
func (si *ClientSession) ConnectWithPassword(password string) error {
	err := si.hc.ConnectWithPassword(password)
	if err != nil {
		return err
	}
	authCl := authclient.NewProfileClient(si.hc)
	token, _ := authCl.RefreshToken()
	if err == nil {
		si.AuthToken = token
	}
	return nil
}

// ConnectionStatus returns the status of hub connection
// This returns:
//
//	status transports.ConnectionStatus
//	 * expired when session is expired (and renew failed)
//	 * connected when connected to the hub
//	 * connecting or disconnected when not connected
//	info with a human description
func (si *ClientSession) ConnectionStatus() transports.HubTransportStatus {
	return si.hc.GetStatus()
}

func (si *ClientSession) GetConnection() *hubclient.HubClient {
	return si.hc
}
