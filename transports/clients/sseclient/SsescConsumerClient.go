package sseclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/httpclient"
	"github.com/hiveot/hub/transports/servers/ssescserver"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/url"
	"sync/atomic"
	"time"
)

// SsescConsumerClient extends the https consumer binding with a return channel
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
type SsescConsumerClient struct {
	httpbasicclient.HttpConsumerClient

	// the sse connection path
	ssePath string

	// handler for closing the sse connection
	sseCancelFn context.CancelFunc
	lastError   atomic.Pointer[error]

	// the agent handles requests
	agentRequestHandler func(req transports.RequestMessage)
}

// helper to establish the sse connection using the given bearer token
// cl.handleSseEvent will set 'connected' status when the first ping event is
// received from the server. (go-sse doesn't have a connected callback)
func (cl *SsescConsumerClient) connectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("connectSSE: Missing SSE path")
	}
	// establish the SSE connection for the return channel
	sseURL := fmt.Sprintf("https://%s%s", cl.BaseHostPort, cl.ssePath)
	cl.sseCancelFn, err = ConnectSSE(
		cl.GetClientID(),
		cl.GetConnectionID(),
		sseURL, token, cl.BaseCaCert,
		cl.GetTlsClient(),
		cl.handleSSEConnect,
		cl.handleSseEvent,
		cl.BaseTimeout)

	return err
}

// ConnectWithPassword connects to the Hub TLS server using the http handler,
// and on success establish an SSE connection using the same TLS client.
//
// This returns an authentication token for use with ConnectWithToken.
func (cl *SsescConsumerClient) ConnectWithPassword(password string) (newToken string, err error) {
	newToken, err = cl.HttpConsumerClient.ConnectWithPassword(password)
	if err != nil {
		return "", err
	}
	// connectSSE will set 'isConnected' if successful
	err = cl.connectSSE(newToken)
	if err != nil {
		cl.BaseIsConnected.Store(false)
		return "", err
	}
	return newToken, err
}

// ConnectWithLoginForm connects to a HTTP/SSE server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *SsescConsumerClient) ConnectWithLoginForm(password string) (newToken string, err error) {
	newToken, err = cl.HttpConsumerClient.ConnectWithLoginForm(password)
	if err != nil {
		return "", err
	}
	err = cl.connectSSE(newToken)
	if err != nil {
		cl.BaseIsConnected.Store(false)
		return "", err
	}
	//cl.BaseIsConnected.Store(true)
	return newToken, err
}

// ConnectWithToken sets the bearer token to use with requests and establishes
// an SSE connection.
func (cl *SsescConsumerClient) ConnectWithToken(token string) (newToken string, err error) {
	newToken, err = cl.HttpConsumerClient.ConnectWithToken(token)
	if err != nil {
		return "", err
	}
	err = cl.connectSSE(newToken)
	if err != nil {
		cl.BaseIsConnected.Store(false)
		return "", err
	}
	//cl.BaseIsConnected.Store(true)
	return newToken, err
}

// Disconnect the http and sse connection from the server
func (cl *SsescConsumerClient) Disconnect() {
	slog.Debug("HttpSSEClient.Disconnect",
		slog.String("clientID", cl.GetClientID()),
		slog.String("cid", cl.GetConnectionID()),
	)
	cl.BaseMux.Lock()
	sseCancelFn := cl.sseCancelFn
	cl.sseCancelFn = nil
	cl.BaseMux.Unlock()

	// the connection status will update, if changed, through the sse callback
	if sseCancelFn != nil {
		sseCancelFn()
	}
	cl.HttpConsumerClient.Disconnect()

	if cl.BaseRnrChan.Len() > 0 {
		// maybe an unhandled error in connecting?
		slog.Error("Force closing unhandled RPC calls", "n", cl.BaseRnrChan.Len())
		cl.BaseRnrChan.CloseAll()
	}
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *SsescConsumerClient) handleSSEConnect(connected bool, err error) {
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
	if cl.BaseIsConnected.Load() != connected {
		connectionChanged = true
	}
	cl.BaseIsConnected.Store(connected)
	if err != nil {
		cl.BaseMux.Lock()
		cl.lastError.Store(&err)
		cl.BaseMux.Unlock()
	}
	cl.BaseMux.RLock()
	handler := cl.AppConnectHandler
	cl.BaseMux.RUnlock()

	// Note: this callback can send notifications to the client,
	// so prevent deadlock by running in the background.
	// (caught by readhistory failing for unknown reason)
	if connectionChanged && handler != nil {
		go handler(connected, err)
	}
}

// handleSSEEvent processes the push-event received from the hub.
// This splits the message into notification, response and request
// requests have an operation and correlationID
// responses have no operations and a correlationID
// notifications have an operations and no correlationID
func (cl *SsescConsumerClient) handleSseEvent(event sse.Event) {

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
			cl.agentRequestHandler(req)
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
		go cl.OnResponse(resp)
	} else if event.Type == transports.MessageTypeNotification {
		notif := transports.NotificationMessage{}
		_ = jsoniter.UnmarshalFromString(event.Data, &notif)
		slog.Info("handle notification: ",
			slog.String("thingID", notif.ThingID),
			slog.String("name", notif.Name),
			slog.String("data", notif.ToString(20)),
			slog.String("created", notif.Created),
		)
		// don't block the receiver flow
		go cl.OnNotification(notif)
	} else {
		// everything else is in a different format. Attempt to deliver for
		// compatibility with other protocols (such has hiveoview test client)
		notif := transports.NotificationMessage{}
		notif.Data = event.Data
		notif.Operation = event.Type
		// don't block the receiver flow
		go cl.OnNotification(notif)
	}
}

// Ping the server and wait for a pong response over the sse return channel
func (cl *SsescConsumerClient) Ping() error {
	req := transports.NewRequestMessage(wot.HTOpPing, "", "", nil, "")
	resp, err := cl.SendRequest(req, true)
	if err != nil {
		return err
	}
	if resp.Output == nil {
		return errors.New("ping returned but no reply received")
	}
	return nil
}

// SetSSEPath sets the new sse path to use.
// This allows to change the hub default /ssesc
func (cl *SsescConsumerClient) SetSSEPath(ssePath string) {
	cl.BaseMux.Lock()
	cl.ssePath = ssePath
	cl.BaseMux.Unlock()
}

// Init Initializes the HTTP/SSE-SC consumer client transport.
// Used internally by the constructor.
//
//	fullURL full path of the sse endpoint
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func (cl *SsescConsumerClient) Init(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	timeout time.Duration) {

	cl.HttpConsumerClient.Init(
		fullURL, clientID, clientCert, caCert, getForm, timeout)

	parts, _ := url.Parse(fullURL)
	cl.ssePath = parts.Path
	cl.agentRequestHandler = func(req transports.RequestMessage) {
		slog.Error("Request received but this isn't an agent")
	}
}

// NewSsescConsumerClient creates a new instance of the http consumer with SSE-SC return-channel.
func NewSsescConsumerClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	timeout time.Duration) *SsescConsumerClient {

	cl := SsescConsumerClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}
