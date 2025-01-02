package wssclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/httpclient"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"log/slog"
	"sync/atomic"
	"time"
)

// WssConsumerClient manages the connection to the hub server using Websockets.
// This implements the IConsumer interface.
type WssConsumerClient struct {
	httpclient.HttpConsumerClient

	wssConn              *websocket.Conn
	wssCancelFn          context.CancelFunc
	retryOnDisconnect    atomic.Bool
	lastError            atomic.Pointer[error]
	maxReconnectAttempts int // 0 for indefinite
	token                string

	// optionally define a handler to handle agent messages
	agentRequestHandler func(baseMsg wssserver.BaseMessage, raw []byte)
}

// websocket connection status handler
func (cl *WssConsumerClient) _onConnect(connected bool, err error) {
	cl.BaseMux.RLock()
	connectHandler := cl.AppConnectHandler
	cl.BaseMux.RUnlock()
	cl.BaseIsConnected.Store(connected)
	if connectHandler != nil {
		connectHandler(connected, err)
	}
	// if retrying is enabled then try on disconnect
	if !connected && cl.retryOnDisconnect.Load() {
		cl.Reconnect()
	}
}

//// Encode and send a message over the websocket
//// msg is a websocket protocol message
//func (cl *WssConsumerClient) _send(msg any) error {
//	if !cl.IsConnected() {
//		// note, it might be trying to reconnect in the background
//		err := fmt.Errorf("_send: Not connected to the hub")
//		return err
//	}
//	// websockets do not allow concurrent write
//	cl.BaseMux.Lock()
//	err := cl.wssConn.WriteJSON(msg)
//	cl.BaseMux.Unlock()
//	return err
//}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *WssTransportClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeEd25519)
//	return k
//}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
func (cl *WssConsumerClient) ConnectWithPassword(password string) (newToken string, err error) {
	// Login using the http endpoint
	// FIXME: use forms
	loginURL := fmt.Sprintf("https://%s%s",
		cl.BaseHostPort, httpserver.HttpPostLoginPath)

	slog.Info("ConnectWithPassword", "clientID", cl.BaseClientID)

	// FIXME: figure out how a standard login method is used to obtain an auth token
	loginMessage := map[string]string{
		"login":    cl.GetClientID(),
		"password": password,
	}
	// TODO: this is part of the http binding, not the websocket binding
	// a sacrificial client to get a token
	tlsClient := tlsclient.NewTLSClient(loginURL, nil, cl.BaseCaCert, cl.BaseTimeout)
	argsJSON, _ := json.Marshal(loginMessage)
	defer tlsClient.Close()
	resp, statusCode, err2 := tlsClient.Post(loginURL, argsJSON)
	if err2 != nil {
		err = fmt.Errorf("%d: Login failed: %s", statusCode, err2)
		return "", err
	}
	token := ""
	err = cl.Unmarshal(resp, &token)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: Login to %s has unexpected response message: %s", loginURL, err)
		return "", err
	}
	// with an auth token the connection can be established
	cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
		cl.BaseClientID, cl.BaseFullURL, token, cl.BaseCaCert,
		cl._onConnect, cl.HandleWssMessage)

	if err == nil {
		cl.token = token
		cl.retryOnDisconnect.Store(true)
	}

	return token, err
}

// ConnectWithToken connects to the Hub server using a user bearer token
// and obtain a new token.
//
//	token is the token previously obtained with login or refresh.
func (cl *WssConsumerClient) ConnectWithToken(token string) (newToken string, err error) {

	wssCancelFn, wssConn, err := ConnectWSS(
		cl.BaseClientID, cl.BaseFullURL, token, cl.BaseCaCert,
		cl._onConnect, cl.HandleWssMessage)

	cl.BaseMux.Lock()
	cl.wssCancelFn = wssCancelFn
	cl.wssConn = wssConn
	cl.BaseMux.Unlock()

	if err == nil {
		// Refresh the auth token and verify the connection works.
		newToken, err = cl.RefreshToken(token)
	}
	// once the connection is established enable the retry on disconnect
	if err == nil {
		cl.retryOnDisconnect.Store(true)
	}
	return newToken, err
}

// Disconnect from the server
func (cl *WssConsumerClient) Disconnect() {
	slog.Debug("Disconnect",
		slog.String("clientID", cl.BaseClientID),
	)
	// dont try to reconnect
	cl.retryOnDisconnect.Store(false)

	cl.BaseMux.Lock()
	if cl.wssCancelFn != nil {
		cl.wssCancelFn()
		cl.wssCancelFn = nil
	}
	if cl.BaseRnrChan.Len() > 0 {
		slog.Error("Force closing unhandled RPC call", "n", cl.BaseRnrChan.Len())
		cl.BaseRnrChan.CloseAll()
	}
	cl.BaseMux.Unlock()
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *WssConsumerClient) Logout() error {

	// TODO: find a way to derive this from a form
	slog.Info("Logout", slog.String("clientID", cl.BaseClientID))

	// Use a sacrificial http client to logout
	tlsClient := tlsclient.NewTLSClient(cl.BaseHostPort, nil, cl.BaseCaCert, cl.BaseTimeout)
	tlsClient.SetAuthToken(cl.token)
	defer tlsClient.Close()

	logoutURL := fmt.Sprintf("https://%s%s", cl.BaseHostPort, httpserver.HttpPostLogoutPath)
	_, _, err2 := tlsClient.Post(logoutURL, nil)
	if err2 != nil {
		err := fmt.Errorf("logout failed: %s", err2)
		return err
	}
	return nil
}

// Reconnect attempts to re-establish a dropped connection using the last token
func (cl *WssConsumerClient) Reconnect() {
	var err error
	for i := 0; cl.maxReconnectAttempts == 0 || i < cl.maxReconnectAttempts; i++ {
		slog.Warn("Reconnecting attempt",
			slog.String("clientID", cl.BaseClientID),
			slog.Int("i", i))
		_, err = cl.ConnectWithToken(cl.token)
		if err == nil {
			break
		}
		// retry until max repeat is reached or disconnect is called
		if !cl.retryOnDisconnect.Load() {
			break
		}
		// the connection timeout doesn't seem to work for some reason
		time.Sleep(time.Second)
	}
	if err != nil {
		slog.Warn("Reconnect failed: ", "err", err.Error())
	}
}

// RefreshToken refreshes the authentication token
// This uses the http method
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *WssConsumerClient) RefreshToken(oldToken string) (newToken string, err error) {
	// use the http endpoint to refresh the token
	refreshURL := fmt.Sprintf("https://%s%s", cl.BaseHostPort, httpserver.HttpPostRefreshPath)

	// TODO: this is part of the http binding, not the websocket binding
	// a sacrificial client to get a token
	tlsClient := tlsclient.NewTLSClient(
		cl.BaseHostPort, nil, cl.BaseCaCert, cl.BaseTimeout)
	tlsClient.SetAuthToken(oldToken)
	defer tlsClient.Close()

	argsJSON, _ := json.Marshal(oldToken)
	resp, statusCode, err2 := tlsClient.Post(refreshURL, argsJSON)
	if err2 != nil {
		err = fmt.Errorf("%d: Refresh failed: %s", statusCode, err2)
		return "", err
	}
	err = cl.Unmarshal(resp, &newToken)
	if err != nil {
		return "", err
	}
	cl.token = newToken
	return newToken, err
}

// PubRequest publishes a request message over websockets
func (cl *WssConsumerClient) PubRequest(req transports.RequestMessage) error {

	slog.Info("PubRequest",
		slog.String("operation", req.Operation),
		slog.String("clientID", cl.GetClientID()),
		slog.String("dThingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("correlationID", req.CorrelationID),
	)

	// convert the operation into a websocket message
	wssMsg, err := wssserver.OpToMessage(req.Operation, req.ThingID, req.Name, nil, req.Input,
		req.CorrelationID, cl.GetClientID())
	if err != nil {
		slog.Error("PubRequest: unknown operation", "op", req.Operation)
		return err
	}
	err = cl._send(wssMsg)
	return err
}

// _send publishes a message over websockets
func (cl *WssConsumerClient) _send(wssMsg any) (err error) {
	if !cl.IsConnected() {
		// note, it might be trying to reconnect in the background
		err := fmt.Errorf("_send: Not connected to the hub")
		return err
	}
	// websockets do not allow concurrent write
	cl.BaseMux.Lock()
	err = cl.wssConn.WriteJSON(wssMsg)
	cl.BaseMux.Unlock()
	//err = cl._send(msg)
	return err
}

//func (cl *WssConsumerClient) SendOperation(
//	operation string, dThingID, name string, data any, correlationID string) error {
//
//	slog.Info("SendMessage",
//		slog.String("operation", operation),
//		slog.String("clientID", cl.GetClientID()),
//		slog.String("dThingID", dThingID),
//		slog.String("name", name),
//		slog.String("correlationID", correlationID),
//	)
//
//	// convert the operation into a websocket message and send it to the server
//	msg, err := wssserver.OpToMessage(operation, dThingID, name, nil, data,
//		correlationID, cl.GetClientID())
//	if err != nil {
//		slog.Error("SendOperation: unknown operation", "op", operation)
//		return err
//	}
//	if !cl.IsConnected() {
//		// note, it might be trying to reconnect in the background
//		err := fmt.Errorf("_send: Not connected to the hub")
//		return err
//	}
//	// websockets do not allow concurrent write
//	cl.BaseMux.Lock()
//	err = cl.wssConn.WriteJSON(msg)
//	cl.BaseMux.Unlock()
//	//err = cl._send(msg)
//	return err
//}

// Init Initializes the HTTP/websocket consumer client transport
// For internal use during construction.
//
//	fullURL full path of the sse endpoint
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func (cl *WssConsumerClient) Init(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	timeout time.Duration) {

	cl.HttpConsumerClient.Init(
		fullURL, clientID, clientCert, caCert, getForm, timeout)

	// max delay 3 seconds before a response is expected
	cl.maxReconnectAttempts = 0 // 1 attempt per second
	cl.BasePubRequest = cl.PubRequest
}

// NewWssConsumerClient creates a new instance of the websocket hub client.
func NewWssConsumerClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	timeout time.Duration) *WssConsumerClient {

	cl := WssConsumerClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}
