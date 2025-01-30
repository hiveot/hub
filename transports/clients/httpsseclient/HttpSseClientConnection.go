package httpsseclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/hiveotsseserver"
	jsoniter "github.com/json-iterator/go"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/url"
	"sync/atomic"
	"time"
)

// WssClientConnection manages the connection to the SSE server using Websockets.
// This implements the IConnection interface.

// SseHttpClientConnection extends the https consumer binding with a return channel
// using the SSE-SC subprotocol for receiving notifications.
//
// The SSE-SC subprotocol uses hiveot message format, which passes ResponseMessage,
// NotificationMessage and RequestMessage (for agents) envelopes over SSE.
// Not further mapping is needed.
//
// The difference between SSE and SSE-SC is that SSE provides subscription/observe in
// the http header during connection, while sse-sc uses a separate REST call to
// subscribe and unsubscribe. In addition, sse-sc uses the SSE eventID to pass
// the operation, thingID and affordance name, supporting multiple devices.
type SseHttpClientConnection struct {
	*HttpClientConnection

	// the sse connection path
	ssePath string

	retryOnDisconnect atomic.Bool

	// handler for closing the sse connection
	sseCancelFn context.CancelFunc
}

// helper to establish the sse connection using the given bearer token
// cl.handleSseEvent will set 'connected' status when the first ping event is
// received from the server. (go-sse doesn't have a connected callback)
func (cl *SseHttpClientConnection) connectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("connectSSE: Missing SSE path")
	}
	// establish the SSE connection for the return channel
	sseURL := fmt.Sprintf("https://%s%s", cl.hostPort, cl.ssePath)
	cl.sseCancelFn, err = ConnectSSE(
		cl.GetClientID(),
		cl.GetConnectionID(),
		sseURL, token, cl.caCert,
		cl.GetTlsClient(),
		cl.handleSSEConnect,
		cl.handleSseEvent,
		cl.timeout)

	return err
}

// ConnectWithPassword connects to the Hub TLS server using the http handler,
// and on success establish an SSE connection using the same TLS client.
//
// This returns an authentication token for use with ConnectWithToken.
func (cl *SseHttpClientConnection) ConnectWithPassword(password string) (newToken string, err error) {
	newToken, err = cl.HttpClientConnection.ConnectWithPassword(password)
	if err == nil {
		err = cl.ConnectWithToken(newToken)
	}
	return newToken, err
}

// ConnectWithLoginForm connects to a HTTP/SSE server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *SseHttpClientConnection) ConnectWithLoginForm(password string) (newToken string, err error) {
	newToken, err = cl.HttpClientConnection.ConnectWithLoginForm(password)
	if err == nil {
		err = cl.ConnectWithToken(newToken)
	}
	return newToken, err
}

// ConnectWithToken sets the bearer token to use with requests and establishes
// an SSE connection.
func (cl *SseHttpClientConnection) ConnectWithToken(token string) error {
	err := cl.HttpClientConnection.ConnectWithToken(token)
	if err != nil {
		return err
	}
	// connectSSE will set 'isConnected' on success
	err = cl.connectSSE(token)
	if err != nil {
		cl.isConnected.Store(false)
		return err
	}
	return err
}

// Disconnect the http and sse connection from the server
func (cl *SseHttpClientConnection) Disconnect() {
	slog.Debug("SseHttpClientConnection.Disconnect",
		slog.String("clientID", cl.GetClientID()),
		slog.String("cid", cl.GetConnectionID()),
	)
	cl.mux.Lock()
	sseCancelFn := cl.sseCancelFn
	cl.sseCancelFn = nil
	cl.mux.Unlock()

	// the connection status will update, if changed, through the sse callback
	if sseCancelFn != nil {
		sseCancelFn()
	}
	cl.HttpClientConnection.Disconnect()
}
func (cl *SseHttpClientConnection) GetConnectURL() string {
	return cl.fullURL
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *SseHttpClientConnection) handleSSEConnect(connected bool, err error) {
	errMsg := ""

	// if the context is cancelled this is not an error
	if err != nil {
		errMsg = err.Error()
	}
	slog.Info("handleSSEConnect",
		slog.String("clientID", cl.GetClientID()),
		slog.String("cid", cl.GetConnectionID()),
		slog.Bool("connected", connected),
		slog.String("err", errMsg))

	var connectionChanged bool = false
	if cl.isConnected.Load() != connected {
		connectionChanged = true
	}
	cl.isConnected.Store(connected)
	if err != nil {
		cl.mux.Lock()
		cl.lastError.Store(&err)
		cl.mux.Unlock()
	}
	cl.mux.RLock()
	handler := cl.appConnectHandler
	cl.mux.RUnlock()

	// Note: this callback can send notifications to the client,
	// so prevent deadlock by running in the background.
	// (caught by readhistory failing for unknown reason)
	if connectionChanged && handler != nil {
		go handler(connected, err, cl)
	}
}

// handleSSEEvent processes the push-event received from the hub.
// This splits the message into notification, response and request
// requests have an operation and correlationID
// responses have no operations and a correlationID
// notifications have an operations and no correlationID
func (cl *SseHttpClientConnection) handleSseEvent(event sse.Event) {

	// no further processing of a ping needed
	if event.Type == hiveotsseserver.SSEPingEvent {
		return
	}

	// Use the hiveot message envelopes for request, response and notification
	if event.Type == transports.MessageTypeRequest {
		req := transports.RequestMessage{}
		_ = jsoniter.UnmarshalFromString(event.Data, &req)
		slog.Info("handle request: ",
			slog.String("thingID", req.ThingID),
			slog.String("name", req.Name),
			slog.String("created", req.Created),
		)
		go func() {
			cl.mux.RLock()
			h := cl.appRequestHandler
			cl.mux.RUnlock()
			h(&req, cl)
		}()
	} else if event.Type == transports.MessageTypeResponse {
		resp := transports.ResponseMessage{}
		_ = jsoniter.UnmarshalFromString(event.Data, &resp)
		// don't block the receiver flow
		slog.Info("handle response: ",
			slog.String("thingID", resp.ThingID),
			slog.String("name", resp.Name),
			slog.String("correlationID", resp.CorrelationID),
			slog.String("created", resp.Updated),
		)
		// don't block the receiver flow
		go func() {
			cl.mux.RLock()
			h := cl.appResponseHandler
			cl.mux.RUnlock()
			_ = h(&resp)
		}()
	} else {
		// everything else is in a different format. Attempt to deliver for
		// compatibility with other protocols (such has hiveoview test client)
		resp := transports.ResponseMessage{}
		resp.Output = event.Data
		resp.Operation = event.Type
		// don't block the receiver flow
		go func() {
			cl.mux.RLock()
			h := cl.appResponseHandler
			cl.mux.RUnlock()
			_ = h(&resp)
		}()
	}
}

// Ping the server and wait for a pong response over the sse return channel
//func (cl *SseHttpClientConnection) Ping() error {
//	req := transports.NewRequestMessage(wot.HTOpPing, "", "", nil, "")
//	resp, err := cl.SendRequest(req, true)
//	if err != nil {
//		return err
//	}
//	if resp.Output == nil {
//		return errors.New("ping returned but no reply received")
//	}
//	return nil
//}

// NewHttpSseClientConnection creates a new instance of the hiveot single-connection
// http/sse subprotocol client. This differs from the WoT sse specification in
// that it supports responses from multiple subscriptions and multiple Things.
//
// This is not interoperable with WoT SSE Thing servers. (it can be but there
// are no such devices)
//
// This extends the HttpClientConnection with a return channel over SSE for receiving
// asynchronous requests or responses.
//
//	fullURL is the full sse connection URL of the server.
//	clientID is the authentication ID of the consumer or agent
//	clientCert is the optional client certificate authentication (todo)
//	caCert is the server CA for TLS connection validation
//	getForm provides the form needed for http request to the server.
//	timeout is the maximum connection wait time
func NewHttpSseClientConnection(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	timeout time.Duration) *SseHttpClientConnection {

	parts, _ := url.Parse(fullURL)
	ssePath := parts.Path
	cl := SseHttpClientConnection{
		HttpClientConnection: NewHttpClientConnection(
			fullURL, clientID, clientCert, caCert, getForm, timeout),
		ssePath:     ssePath,
		sseCancelFn: nil,
	}
	return &cl
}
