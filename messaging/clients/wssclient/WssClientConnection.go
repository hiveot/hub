package wssclient

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers"
	"github.com/teris-io/shortid"
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

	// handler for notifications sent by agents
	appNotificationHandlerPtr atomic.Pointer[messaging.NotificationHandler]
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
func (cc *WssClient) _onConnectionChanged(connected bool, err error) {

	hPtr := cc.appConnectHandlerPtr.Load()

	cc.isConnected.Store(connected)
	if hPtr != nil {
		(*hPtr)(connected, err, cc)
	}
	// if retrying is enabled then try on disconnect
	if !connected && cc.retryOnDisconnect.Load() {
		cc.Reconnect()
	}
}

// _send publishes a message over websockets
func (cc *WssClient) _send(wssMsg any) (err error) {
	if !cc.isConnected.Load() {
		// note, it might be trying to reconnect in the background
		err := fmt.Errorf("_send: Not connected to the hub")
		return err
	}
	// websockets do not allow concurrent writes
	cc.mux.Lock()
	err = cc.wssConn.WriteJSON(wssMsg)
	cc.mux.Unlock()
	return err
}

// ConnectWithToken attempts to establish a websocket connection using a valid auth token
// If a connection exists it is closed first.
func (cc *WssClient) ConnectWithToken(token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cc.Disconnect()

	cc.authToken = token
	wssCancelFn, wssConn, err := ConnectWSS(cc.cinfo, token,
		cc._onConnectionChanged, cc.HandleWssMessage)

	cc.mux.Lock()
	cc.wssCancelFn = wssCancelFn
	cc.wssConn = wssConn
	cc.mux.Unlock()

	// even if connection failed right now, enable retry
	cc.retryOnDisconnect.Store(true)

	return err
}

// Disconnect from the server
func (cc *WssClient) Disconnect() {
	slog.Debug("Disconnect",
		slog.String("clientID", cc.cinfo.ClientID),
	)
	// dont try to reconnect
	cc.retryOnDisconnect.Store(false)

	cc.mux.Lock()
	defer cc.mux.Unlock()
	if cc.wssCancelFn != nil {
		cc.wssCancelFn()
		cc.wssCancelFn = nil
	}
}

// GetConnectionInfo returns the client's connection details
func (cc *WssClient) GetConnectionInfo() messaging.ConnectionInfo {
	return cc.cinfo
}

// HandleWssMessage processes the websocket message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cc *WssClient) HandleWssMessage(raw []byte) {
	var notif *messaging.NotificationMessage
	var req *messaging.RequestMessage
	var resp *messaging.ResponseMessage

	// both non-agents and agents receive responses
	notif = cc.messageConverter.DecodeNotification(raw)
	if notif == nil {
		resp = cc.messageConverter.DecodeResponse(raw)
		if resp == nil {
			req = cc.messageConverter.DecodeRequest(raw)
		}
	}
	if notif != nil {
		hPtr := cc.appNotificationHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no notification handler set",
				"clientID", cc.cinfo.ClientID,
				"operation", notif.Operation,
			)
			return
		}
		// pass the response to the registered handler
		(*hPtr)(notif)
	} else if resp != nil {
		hPtr := cc.appResponseHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no response handler set",
				"clientID", cc.cinfo.ClientID,
				"operation", resp.Operation,
			)
			return
		}
		// pass the response to the registered handler
		_ = (*hPtr)(resp)
	} else if req != nil {
		hPtr := cc.appRequestHandlerPtr.Load()
		if hPtr == nil {
			slog.Error("HandleWssMessage: no request handler set",
				"clientID", cc.cinfo.ClientID,
				"operation", req.Operation,
			)
			return
		}
		// return the response to the caller
		resp = (*hPtr)(req, cc)
		// responses are optional
		if resp != nil {
			_ = cc.SendResponse(resp)
		}
	} else {
		slog.Warn("HandleWssMessage: Message is not a notification, request or response")
		return
	}
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cc *WssClient) IsConnected() bool {
	return cc.isConnected.Load()
}

// Reconnect attempts to re-establish a dropped connection using the last token
// This uses an increasing backoff period up to 15 seconds, starting random between 0-2 seconds
func (cc *WssClient) Reconnect() {
	var err error
	var backoffDuration time.Duration = time.Duration(rand.Uint64N(uint64(time.Second * 2)))

	for i := 0; cc.maxReconnectAttempts == 0 || i < cc.maxReconnectAttempts; i++ {
		slog.Warn("Reconnecting attempt",
			slog.String("clientID", cc.cinfo.ClientID),
			slog.Int("i", i))
		err = cc.ConnectWithToken(cc.authToken)
		if err == nil {
			break
		}
		// retry until max repeat is reached, disconnect is called or authorization failed
		if !cc.retryOnDisconnect.Load() {
			break
		}
		if errors.Is(err, messaging.UnauthorizedError) {
			break
		}
		// the connection timeout doesn't seem to work for some reason
		//
		time.Sleep(backoffDuration)
		// slowly wait longer until 10 sec. FIXME: use random
		if backoffDuration < time.Second*15 {
			backoffDuration += time.Second
		}
	}
	if err != nil {
		slog.Warn("Reconnect failed: ", "err", err.Error())
	}
}

// SendNotification Agent posts a notification over websockets
// This passes the notification as-is as a payload.
//
// This posts the JSON-encoded NotificationMessage on the well-known hiveot notification href.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
func (cc *WssClient) SendNotification(notif *messaging.NotificationMessage) error {

	slog.Debug("SendNotification",
		slog.String("clientID", cc.cinfo.ClientID),
		slog.String("correlationID", notif.CorrelationID),
		slog.String("operation", notif.Operation),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
	)
	// convert the operation into a protocol message
	wssMsg, err := cc.messageConverter.EncodeNotification(notif)
	if err != nil {
		slog.Error("SendNotification: unknown request", "op", notif.Operation)
		return err
	}
	err = cc._send(wssMsg)
	if err != nil {
		slog.Warn("SendNotification failed",
			"clientID", cc.cinfo.ClientID,
			"err", err.Error())
	}
	return err
}

// SendRequest send a request message over websockets
// This transforms the request to the protocol message and sends it to the server.
func (cc *WssClient) SendRequest(req *messaging.RequestMessage) error {

	slog.Debug("SendRequest",
		slog.String("clientID", cc.cinfo.ClientID),
		slog.String("correlationID", req.CorrelationID),
		slog.String("operation", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
	)

	// convert the operation into a protocol message
	wssMsg, err := cc.messageConverter.EncodeRequest(req)
	if err != nil {
		slog.Error("SendRequest: unknown request", "op", req.Operation)
		return err
	}
	err = cc._send(wssMsg)
	return err
}

// SendResponse send a response message over websockets
// This transforms the response to the protocol message and sends it to the server.
// Responses without correlationID are subscription notifications.
func (cc *WssClient) SendResponse(resp *messaging.ResponseMessage) error {

	slog.Debug("SendResponse",
		slog.String("operation", resp.Operation),
		slog.String("clientID", cc.cinfo.ClientID),
		slog.String("thingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("error", resp.Error),
		slog.String("correlationID", resp.CorrelationID),
	)

	// convert the operation into a protocol message
	wssMsg, err := cc.messageConverter.EncodeResponse(resp)
	if err != nil {
		slog.Error("SendResponse: cant convert response",
			"op", resp.Operation,
			"err", err)
		return err
	}
	err = cc._send(wssMsg)
	return err
}

// SetConnectHandler set the application handler for connection status updates
func (cc *WssClient) SetConnectHandler(cb messaging.ConnectionHandler) {
	if cb == nil {
		cc.appConnectHandlerPtr.Store(nil)
	} else {
		cc.appConnectHandlerPtr.Store(&cb)
	}
}

// SetNotificationHandler set the application handler for received notifications
func (cc *WssClient) SetNotificationHandler(cb messaging.NotificationHandler) {
	if cb == nil {
		cc.appNotificationHandlerPtr.Store(nil)
	} else {
		cc.appNotificationHandlerPtr.Store(&cb)
	}
}

// SetRequestHandler set the application handler for incoming requests
func (cc *WssClient) SetRequestHandler(cb messaging.RequestHandler) {
	if cb == nil {
		cc.appRequestHandlerPtr.Store(nil)
	} else {
		cc.appRequestHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler set the application handler for received responses
func (cc *WssClient) SetResponseHandler(cb messaging.ResponseHandler) {
	if cb == nil {
		cc.appResponseHandlerPtr.Store(nil)
	} else {
		cc.appResponseHandlerPtr.Store(&cb)
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
		parts.Host = fmt.Sprintf("%s:%d", parts.Hostname(), servers.DefaultHttpsPort)
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
