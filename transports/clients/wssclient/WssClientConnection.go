package wssclient

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// WssClientConnection manages the connection to a websocket server.
// This implements the IConnection interface.
//
// This supports multiple message formats using a 'messageConverter'. The hiveot
// converts is a straight passthrough of RequestMessage and ResponseMessage, while
// the wotwssConverter maps the messages to the WoT websocket specification.
type WssClientConnection struct {

	// handler for requests send by clients
	appConnectHandlerPtr atomic.Pointer[transports.ConnectionHandler]

	// handler for requests send by clients
	appRequestHandlerPtr atomic.Pointer[transports.RequestHandler]
	// handler for responses sent by agents
	appResponseHandlerPtr atomic.Pointer[transports.ResponseHandler]

	// authentication token
	authToken string
	caCert    *x509.Certificate

	// clientID is the account ID of the agent or consumer
	clientID     string
	connectionID string
	fullURL      string

	isConnected  atomic.Bool
	lastError    atomic.Pointer[error]
	protocolType string

	maxReconnectAttempts int // 0 for indefinite

	// convert the request/response to the wss messaging protocol used
	messageConverter transports.IMessageConverter

	// mutex for controlling writing and closing
	mux sync.RWMutex

	retryOnDisconnect atomic.Bool

	// request timeout
	timeout time.Duration

	// underlying websocket connection
	wssConn     *websocket.Conn
	wssCancelFn context.CancelFunc
}

// websocket connection status handler
func (cl *WssClientConnection) _onConnectionChanged(connected bool, err error) {

	hPtr := cl.appConnectHandlerPtr.Load()

	cl.isConnected.Store(connected)
	if hPtr != nil {
		(*hPtr)(connected, err, cl)
	}
	// if retrying is enabled then try on disconnect
	if !connected && cl.retryOnDisconnect.Load() {
		cl.Reconnect()
	}
}

// _send publishes a message over websockets
func (cl *WssClientConnection) _send(wssMsg any) (err error) {
	if !cl.isConnected.Load() {
		// note, it might be trying to reconnect in the background
		err := fmt.Errorf("_send: Not connected to the hub")
		return err
	}
	// websockets do not allow concurrent writes
	cl.mux.Lock()
	err = cl.wssConn.WriteJSON(wssMsg)
	cl.mux.Unlock()
	return err
}

// ConnectWithPassword connects to the TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// FIXME:
// 1. This currently only works on the Hub.
// 2. This has a hard coded paths for auth instead of using the TD
// 3. This should download the TD from the discovered endpoint
func (cl *WssClientConnection) ConnectWithPassword(password string) (newToken string, err error) {
	// Login using the http endpoint

	// TODO: use configurable auth method
	parts, err := url.Parse(cl.fullURL)
	if err != nil {
		return "", err
	}
	loginURL := fmt.Sprintf("https://%s%s", parts.Host, httpserver.HttpPostLoginPath)
	slog.Info("ConnectWithPassword", "clientID", cl.clientID)

	// FIXME: figure out how to discover the login method used to obtain an auth token
	loginMessage := map[string]string{
		"login":    cl.GetClientID(),
		"password": password,
	}
	urlParts, _ := url.Parse(loginURL)
	tlsClient := tlsclient.NewTLSClient(urlParts.Host, nil, cl.caCert, cl.timeout)
	argsJSON, _ := jsoniter.Marshal(loginMessage)
	defer tlsClient.Close()
	resp, statusCode, err2 := tlsClient.Post(httpserver.HttpPostLoginPath, argsJSON)
	if err2 != nil {
		err = fmt.Errorf("%d: Login failed: %s", statusCode, err2)
		return "", err
	}
	token := ""
	err = jsoniter.Unmarshal(resp, &token)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: Login to %s has unexpected response message: %s", loginURL, err)
		return "", err
	}
	err = cl.ConnectWithToken(token)
	return token, err
}

// ConnectWithToken attempts to establish a websocket connection using a valid auth token
func (cl *WssClientConnection) ConnectWithToken(token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.Disconnect()

	cl.authToken = token
	wssCancelFn, wssConn, err := ConnectWSS(
		cl.clientID, cl.connectionID, cl.fullURL, token, cl.caCert,
		cl._onConnectionChanged, cl.HandleWssMessage)

	cl.mux.Lock()
	cl.wssCancelFn = wssCancelFn
	cl.wssConn = wssConn
	cl.mux.Unlock()

	// even if connection failed right now, enable retry
	cl.retryOnDisconnect.Store(true)

	return err
}

// Disconnect from the server
func (cl *WssClientConnection) Disconnect() {
	slog.Debug("Disconnect",
		slog.String("clientID", cl.clientID),
	)
	// dont try to reconnect
	cl.retryOnDisconnect.Store(false)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.wssCancelFn != nil {
		cl.wssCancelFn()
		cl.wssCancelFn = nil
	}
}

// GetClientID returns the client's account ID
func (cl *WssClientConnection) GetClientID() string {
	return cl.clientID
}

// GetConnectionID returns the client's connection ID
func (cl *WssClientConnection) GetConnectionID() string {
	return cl.connectionID
}

// GetProtocolType returns the type of protocol this client supports
func (cl *WssClientConnection) GetProtocolType() string {
	return cl.protocolType
}

// GetConnectURL returns the schema://address:port/path of the server connection
func (cl *WssClientConnection) GetConnectURL() string {
	return cl.fullURL
}

// HandleWssMessage processes the websocket message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *WssClientConnection) HandleWssMessage(raw []byte) {

	req, resp, err := cl.messageConverter.DecodeMessage(raw)
	if err != nil {
		slog.Warn("HandleWssMessage: failed decoding message:", "err", err.Error())
	} else if req != nil {
		hPtr := cl.appRequestHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no request handler set",
				"clientID", cl.clientID,
				"operation", req.Operation,
			)
			return
		}
		// return the response to the caller
		resp = (*hPtr)(req, cl)
		_ = cl.SendResponse(resp)
	} else if resp != nil {
		hPtr := cl.appResponseHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no response handler set",
				"clientID", cl.clientID,
				"operation", resp.Operation,
			)
			return
		}
		_ = (*hPtr)(resp)
	}
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *WssClientConnection) IsConnected() bool {
	return cl.isConnected.Load()
}

// Reconnect attempts to re-establish a dropped connection using the last token
func (cl *WssClientConnection) Reconnect() {
	var err error
	for i := 0; cl.maxReconnectAttempts == 0 || i < cl.maxReconnectAttempts; i++ {
		slog.Warn("Reconnecting attempt",
			slog.String("clientID", cl.clientID),
			slog.Int("i", i))
		err = cl.ConnectWithToken(cl.authToken)
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

// SendRequest send a request message over websockets
// This transforms the request to the protocol message and sends it to the server.
func (cl *WssClientConnection) SendRequest(req *transports.RequestMessage) error {

	slog.Info("SendRequest",
		slog.String("operation", req.Operation),
		slog.String("clientID", cl.clientID),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("correlationID", req.CorrelationID),
	)

	// convert the operation into a protocol message
	wssMsg, err := cl.messageConverter.EncodeRequest(req)
	if err != nil {
		slog.Error("SendRequest: unknown request", "op", req.Operation)
		return err
	}
	err = cl._send(wssMsg)
	return err
}

// SendResponse send a response message over websockets
// This transforms the response to the protocol message and sends it to the server.
// Responses without correlationID are subscription notifications.
func (cl *WssClientConnection) SendResponse(resp *transports.ResponseMessage) error {

	slog.Info("SendResponse",
		slog.String("operation", resp.Operation),
		slog.String("clientID", cl.clientID),
		slog.String("thingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("correlationID", resp.CorrelationID),
	)

	// convert the operation into a protocol message
	wssMsg, err := cl.messageConverter.EncodeResponse(resp)
	if err != nil {
		slog.Error("SendResponse: cant convert response",
			"op", resp.Operation,
			"err", err)
		return err
	}
	err = cl._send(wssMsg)
	return err
}

// SetConnectHandler set the application handler for connection status updates
func (cl *WssClientConnection) SetConnectHandler(cb transports.ConnectionHandler) {
	if cb == nil {
		cl.appConnectHandlerPtr.Store(nil)
	} else {
		cl.appConnectHandlerPtr.Store(&cb)
	}
}

// SetRequestHandler set the application handler for incoming requests
func (cl *WssClientConnection) SetRequestHandler(cb transports.RequestHandler) {
	if cb == nil {
		cl.appRequestHandlerPtr.Store(nil)
	} else {
		cl.appRequestHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler set the application handler for received responses
func (cl *WssClientConnection) SetResponseHandler(cb transports.ResponseHandler) {
	if cb == nil {
		cl.appResponseHandlerPtr.Store(nil)
	} else {
		cl.appResponseHandlerPtr.Store(&cb)
	}
}

// NewHiveotWssClientConnection creates a new instance of the websocket client.
//
// messageConverter offers the ability to use any websocket message format that
// can be mapped to a RequestMessage and ResponseMessage. It is used to support
// both hiveot and WoT websocket message formats.
//
//	fullURL is the full websocket connection URL
//	clientID is the authentication ID of the consumer or agent
//	caCert is the server CA for TLS connection validation
//	converter is the message format converter
//	protocol is the protocol ID of the websocket messages
//	timeout is the maximum connection wait time
func NewHiveotWssClientConnection(
	fullURL string, clientID string, caCert *x509.Certificate,
	converter transports.IMessageConverter, protocol string,
	timeout time.Duration) *WssClientConnection {

	if protocol == "" {
		protocol = transports.ProtocolTypeHiveotWSS
	}

	cl := WssClientConnection{
		caCert:               caCert,
		clientID:             clientID,
		connectionID:         shortid.MustGenerate(),
		fullURL:              fullURL,
		protocolType:         protocol,
		maxReconnectAttempts: 0,
		messageConverter:     converter,
		timeout:              timeout,
	}
	//cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}
