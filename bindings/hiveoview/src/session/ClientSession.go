package session

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"sync"
	"time"
)

type SSEEvent struct {
	Event   string
	Payload string
}

// DefaultExpiryHours TODO: set default expiry in config
const DefaultExpiryHours = 72

// ClientSession of a web client containing a hub connection
type ClientSession struct {
	// ID of this session
	sessionID string

	// ClientID is the login ID of the user
	clientID string
	// Auth token obtained from the secure cookie and used to re-connect to the hub
	//AuthToken string `json:"auth_token"`
	// Expiry of the session
	//Expiry time.Time `json:"expiry"`
	// remote address
	//RemoteAddr string `json:"remote_addr"`
	lastActivity time.Time

	// The associated hub client for pub/sub
	hc *hubclient.HubClient
	// session mutex for updating sse and activity
	mux sync.RWMutex

	// SSE event channels for this session
	// Each SSE connection is added to this list
	sseClients []chan SSEEvent
}

func (cs *ClientSession) AddSSEClient(c chan SSEEvent) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	cs.sseClients = append(cs.sseClients, c)

	go cs.SendSSE("info", "new client listening")
}

// Close the session
// This closes the hub connection and SSE data channels
func (cs *ClientSession) Close() {
	cs.mux.Lock()
	defer cs.mux.Unlock()
	for _, sseChan := range cs.sseClients {
		close(sseChan)
	}
	cs.hc.Disconnect()
	cs.sseClients = nil
}

// ConnectWithPassword connects the session with the Hub using the given password
//
// This returns a new auth token for restoring the connection at a later time.
//func (cs *ClientSession) ConnectWithPassword(password string) (string, error) {
//
//	err := cs.hc.ConnectWithPassword(password)
//	if err != nil {
//		slog.Debug("ConnectWithPassword failed", "err", err)
//		_ = cs.SendSSE("error", "Connect failed: "+err.Error())
//		return "", err
//	} else {
//		_ = cs.SendSSE("success", "Connected to the Hub")
//	}
//	authCl := authclient.NewProfileClient(cs.hc)
//	authToken, err := authCl.RefreshToken()
//	if err == nil {
//		// for testing
//		go func() {
//			cs.SendSSETestLoop()
//		}()
//	}
//	return authToken, err
//}

// GetStatus returns the status of hub connection
// This returns:
//
//	status transports.ConnectionStatus
//	 * expired when session is expired (and renew failed)
//	 * connected when connected to the hub
//	 * connecting or disconnected when not connected
//	info with a human description
func (cs *ClientSession) GetStatus() transports.HubTransportStatus {
	status := cs.hc.GetStatus()
	return status
}

// GetHubClient returns the hub client connection for use in pub/sub
func (cs *ClientSession) GetHubClient() *hubclient.HubClient {
	return cs.hc
}

// IsActive returns whether the session has a connection to the Hub
func (cs *ClientSession) IsActive() bool {
	status := cs.hc.GetStatus()
	return status.ConnectionStatus == transports.Connected ||
		status.ConnectionStatus == transports.Connecting
}

// onEvent passes incoming events from the Hub to the SSE client(s)
func (cs *ClientSession) onEvent(msg *things.ThingValue) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	// TODO: determine how events are consumed
	// SSE's usually expect an HTML snippet, not json data
	//_ = cs.SendSSE(msg.Name, string(msg.Data))
}

// Reconnect restores the session's connection with the hub
// This returns a refreshed token or an error if connection fails
//func (cs *ClientSession) Reconnect(authToken string) (string, error) {
//	if authToken == "" {
//		return "", errors.New("missing auth token")
//	}
//
//	// TODO: update expiry with a new token
//	// the user's private key is not available. This token auth only works
//	// if it doesn't require one.
//	err := cs.hc.ConnectWithToken(nil, authToken)
//	if err != nil {
//		return "", err
//	}
//	myProfile := authclient.NewProfileClient(cs.hc)
//	newToken, err := myProfile.RefreshToken()
//	if err != nil {
//		slog.Error("Token refresh failed. Continuing as existing token is still valid.",
//			"clientID", cs.hc.ClientID(), "err", err)
//		_ = cs.SendSSE("error", "Authentication refresh failed. ")
//		err = nil
//		return "", nil
//	}
//	cs.mux.Lock()
//	defer cs.mux.Unlock()
//	_ = cs.SendSSE("success", "Authenticated with the Hub")
//
//	return newToken, err
//}

func (cs *ClientSession) RemoveSSEClient(c chan SSEEvent) {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	for i, sseClient := range cs.sseClients {
		if sseClient == c {
			// delete(cs.sseClients,i)
			cs.sseClients = append(cs.sseClients[:i], cs.sseClients[i+1:]...)
			break
		}
	}
}

// ReplaceHubClient replaces this session's hub client
func (cs *ClientSession) ReplaceHubClient(newHC *hubclient.HubClient) {
	// ensure the old client is disconnected
	if cs.hc != nil {
		cs.hc.Disconnect()
		cs.hc.SetEventHandler(nil)
	}
	cs.hc = newHC
	cs.hc.SetEventHandler(cs.onEvent)
}

// SendSSE encodes and sends an SSE event to clients of this session
// Intended to notify the browser of changes.
func (cs *ClientSession) SendSSE(event string, content string) error {
	cs.mux.RLock()
	defer cs.mux.RUnlock()
	slog.Info("sending sse event", "event", event, "nr clients", len(cs.sseClients))
	for _, c := range cs.sseClients {
		c <- SSEEvent{event, content}
	}
	return nil
}

// send sse test counter
func (cs *ClientSession) SendSSETestLoop() {
	// toggle status
	counter := 1
	for {
		time.Sleep(time.Second * 5)
		msg := fmt.Sprintf(
			`<div >Hello world %d</div>`, counter)
		err := cs.SendSSE("counter", msg)
		if err != nil {
			break
		}
		counter++
	}
	slog.Info("counter loop done")
}

// NewClientSession creates a new client session for the given Hub connection
// Intended for use by the session manager.
// note that expiry is a placeholder for now used to refresh auth token.
// it should be obtained from the login authentication/refresh.
func NewClientSession(sessionID string, hc *hubclient.HubClient) *ClientSession {
	cs := ClientSession{
		sessionID: sessionID,
		clientID:  hc.ClientID(),
		hc:        hc,
		// TODO: assess need for buffering
		sseClients:   make([]chan SSEEvent, 0),
		lastActivity: time.Now(),
	}
	hc.SetEventHandler(cs.onEvent)
	return &cs
}
