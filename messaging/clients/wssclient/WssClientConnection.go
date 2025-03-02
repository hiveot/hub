package wssclient

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers/httpserver"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// WssClient manages the connection to a websocket server.
// This implements the IConnection interface.
//
// This supports multiple message formats using a 'messageConverter'. The hiveot
// converts is a straight passthrough of RequestMessage and ResponseMessage, while
// the wotwssConverter maps the messages to the WoT websocket specification.
type WssClient struct {

	// handler for requests send by clients
	appConnectHandlerPtr atomic.Pointer[messaging.ConnectionHandler]

	// handler for requests send by clients
	appRequestHandlerPtr atomic.Pointer[messaging.RequestHandler]
	// handler for responses sent by agents
	appResponseHandlerPtr atomic.Pointer[messaging.ResponseHandler]

	//clientID string
	// Connection information such as clientID, cid, address, protocol etc
	cinfo messaging.ConnectionInfo

	// authentication token
	authToken string
	//caCert    *x509.Certificate

	// clientID is the account ID of the agent or consumer
	//clientID     string
	//connectionID string
	//fullURL      string

	isConnected atomic.Bool
	lastError   atomic.Pointer[error]

	maxReconnectAttempts int // 0 for indefinite

	// convert the request/response to the wss messaging protocol used
	messageConverter messaging.IMessageConverter

	// mutex for controlling writing and closing
	mux sync.RWMutex

	retryOnDisconnect atomic.Bool

	// request timeout
	//timeout time.Duration

	// underlying websocket connection
	wssConn     *websocket.Conn
	wssCancelFn context.CancelFunc
}

// websocket connection status handler
func (cl *WssClient) _onConnectionChanged(connected bool, err error) {

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
func (cl *WssClient) _send(wssMsg any) (err error) {
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

// ConnectWithToken attempts to establish a websocket connection using a valid auth token
func (cl *WssClient) ConnectWithToken(token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.Disconnect()

	cl.authToken = token
	wssCancelFn, wssConn, err := ConnectWSS(cl.cinfo, token,
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
func (cl *WssClient) Disconnect() {
	slog.Debug("Disconnect",
		slog.String("clientID", cl.cinfo.ClientID),
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

// GetConnectionInfo returns the client's connection details
func (c *WssClient) GetConnectionInfo() messaging.ConnectionInfo {
	return c.cinfo
}

// HandleWssMessage processes the websocket message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *WssClient) HandleWssMessage(raw []byte) {

	// both non-agents and agents receive responses
	resp := cl.messageConverter.DecodeResponse(raw)
	if resp != nil {
		hPtr := cl.appResponseHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no response handler set",
				"clientID", cl.cinfo.ClientID,
				"operation", resp.Operation,
			)
			return
		}
		// pass the response to the registered handler
		_ = (*hPtr)(resp)
	} else {
		// only agents receive requests
		req := cl.messageConverter.DecodeRequest(raw)
		if req == nil {
			slog.Warn("HandleWssMessage: Message is not a request or response")
			return
		}
		hPtr := cl.appRequestHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no request handler set",
				"clientID", cl.cinfo.ClientID,
				"operation", req.Operation,
			)
			return
		}
		// return the response to the caller
		resp = (*hPtr)(req, cl)
		_ = cl.SendResponse(resp)
	}
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *WssClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Reconnect attempts to re-establish a dropped connection using the last token
func (cl *WssClient) Reconnect() {
	var err error
	for i := 0; cl.maxReconnectAttempts == 0 || i < cl.maxReconnectAttempts; i++ {
		slog.Warn("Reconnecting attempt",
			slog.String("clientID", cl.cinfo.ClientID),
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
func (cl *WssClient) SendRequest(req *messaging.RequestMessage) error {

	slog.Debug("SendRequest",
		slog.String("clientID", cl.cinfo.ClientID),
		slog.String("correlationID", req.CorrelationID),
		slog.String("operation", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
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
func (cl *WssClient) SendResponse(resp *messaging.ResponseMessage) error {

	slog.Debug("SendResponse",
		slog.String("operation", resp.Operation),
		slog.String("clientID", cl.cinfo.ClientID),
		slog.String("thingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("status", resp.Status),
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
func (cl *WssClient) SetConnectHandler(cb messaging.ConnectionHandler) {
	if cb == nil {
		cl.appConnectHandlerPtr.Store(nil)
	} else {
		cl.appConnectHandlerPtr.Store(&cb)
	}
}

// SetRequestHandler set the application handler for incoming requests
func (cl *WssClient) SetRequestHandler(cb messaging.RequestHandler) {
	if cb == nil {
		cl.appRequestHandlerPtr.Store(nil)
	} else {
		cl.appRequestHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler set the application handler for received responses
func (cl *WssClient) SetResponseHandler(cb messaging.ResponseHandler) {
	if cb == nil {
		cl.appResponseHandlerPtr.Store(nil)
	} else {
		cl.appResponseHandlerPtr.Store(&cb)
	}
}

// NewHiveotWssClient creates a new instance of the websocket client.
//
// messageConverter offers the ability to use any websocket message format that
// can be mapped to a RequestMessage and ResponseMessage. It is used to support
// both hiveot and WoT websocket message formats.
//
//	wssURL is the full websocket connection URL
//	clientID is the authentication ID of the consumer or agent
//	caCert is the server CA for TLS connection validation
//	converter is the message format converter
//	protocol is the protocol ID of the websocket messages
//	timeout is the maximum connection wait time
func NewHiveotWssClient(
	wssURL string, clientID string, caCert *x509.Certificate,
	converter messaging.IMessageConverter,
	timeout time.Duration) *WssClient {

	// ensure the URL has port as 443 is not valid for this
	parts, _ := url.Parse(wssURL)
	if parts.Port() == "" {
		parts.Host = fmt.Sprintf("%s:%d", parts.Hostname(), httpserver.DefaultHttpsPort)
		wssURL = parts.String()
	}

	cinfo := messaging.ConnectionInfo{
		CaCert:       caCert,
		ClientID:     clientID,
		ConnectionID: "wss-" + shortid.MustGenerate(),
		ConnectURL:   wssURL,
		ProtocolType: converter.GetProtocolType(),
		Timeout:      timeout,
	}
	cl := WssClient{
		cinfo:                cinfo,
		maxReconnectAttempts: 0,
		messageConverter:     converter,
	}
	//cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}
