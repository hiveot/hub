package ssescclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/transports/clients/httpclient"
	"github.com/hiveot/hub/transports/utils"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"sync/atomic"
	"time"
)

const SSEPath = "/sse"

// SseTransportClient extends the https binding with the SSE return channel.
//
// This client creates two http/2 connections, one for posting messages and
// one for a sse connection to establish a return channel.
//
// Intended for sse WoT compatibility. Each subscription requires a new instance
// so this is only efficient when few subscriptions are needed.
//
// For a non-official shared connection see the sse-sc transport binding
type SseTransportClient struct {
	httpclient.HttpTransportClient

	// the sse connection path
	ssePath string

	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	lastError atomic.Pointer[error]

	// the subscription for this client
	// sse only has a single on
	subscription string

	// Request and Response channel helper
	rnrChan *tputils.RnRChan
}

// helper to establish an sse connection using the given bearer token
// FIXME: use the http/2 binding connection
func (cl *SseTransportClient) connectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("SseTransportClient: Missing SSE path")
	}
	// create a second client to establish the sse connection if a path is set
	sseURL := fmt.Sprintf("https://%s%s", cl.BaseHostPort, cl.ssePath)
	cl.sseCancelFn, err = ConnectSSE(
		cl.GetClientID(), cl.GetConnectionID(),
		sseURL, token, cl.BaseCaCert,
		cl.handleSSEConnect, cl.handleSseEvent)

	return err
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *SseTransportClient) ConnectWithPassword(password string) (newToken string, err error) {
	newToken, err = cl.HttpTransportClient.ConnectWithPassword(password)
	if err != nil {
		return "", err
	}
	err = cl.connectSSE(newToken)
	if err != nil {
		cl.BaseIsConnected.Store(false)
		return "", err
	}
	cl.BaseIsConnected.Store(true)
	return newToken, err
}

// ConnectWithToken sets the bearer token to use with requests.
func (cl *SseTransportClient) ConnectWithToken(token string) (newToken string, err error) {
	newToken, err = cl.HttpTransportClient.ConnectWithToken(token)
	if err != nil {
		return "", err
	}
	err = cl.connectSSE(newToken)
	if err != nil {
		cl.BaseIsConnected.Store(false)
		return "", err
	}
	cl.BaseIsConnected.Store(true)
	return newToken, err
}

// Disconnect from the server
func (cl *SseTransportClient) Disconnect() {
	slog.Debug("SseTransportClient.Disconnect",
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
	cl.HttpTransportClient.Disconnect()

	if cl.rnrChan.Len() > 0 {
		slog.Error("Force closing unhandled RPC call", "n", cl.rnrChan.Len())
		cl.rnrChan.CloseAll()
	}
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *SseTransportClient) handleSSEConnect(connected bool, err error) {
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
	handler := cl.BaseConnectHandler
	cl.BaseMux.RUnlock()

	if connectionChanged && handler != nil {
		handler(connected, err)
	}
}

// SendRequest sends an operation request and waits for a completion or timeout.
// This uses a correlationID to link actions to progress updates.
func (cl *SseTransportClient) SendRequest(operation string,
	dThingID string, name string, input interface{}, output interface{}) (err error) {

	// a requestID is needed before the action is published in order to match it with the reply
	requestID := "sserpc-" + shortid.MustGenerate()
	rChan := cl.rnrChan.Open(requestID)

	// Without a return channel there is no waiting for a result
	raw, _, err := cl.SendOperation(operation, dThingID, name, input, requestID)

	// If a result is received then leave it there and return the result
	// this is currently not supported
	if raw != nil && output != nil {
		err = jsoniter.Unmarshal(raw, output)
		cl.rnrChan.Close(requestID)
		return err
	}
	err = cl.WaitForResponse(rChan, requestID, output)
	return err
}

// SetSSEPath sets the new sse path to use.
// This allows to change the hub default /ssesc
func (cl *SseTransportClient) SetSSEPath(ssePath string) {
	cl.BaseMux.Lock()
	cl.ssePath = ssePath
	cl.BaseMux.Unlock()
}

// NewSseTransportClient creates a new instance of the http client with SSE return-channel.
//
//	fullURL of server to connect to
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewSseTransportClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm func(op string) td.Form,
	timeout time.Duration) *SseTransportClient {

	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewHttpSSEClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", fullURL))
	} else {
		slog.Debug("NewHttpSSEClient: CA certificate",
			slog.String("destination", fullURL),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}

	cl := SseTransportClient{
		ssePath: SSEPath,
		rnrChan: tputils.NewRnRChan(),
	}
	// initialize the embedded http transport
	cl.HttpTransportClient.Init(
		fullURL, clientID, clientCert, caCert, getForm, timeout)

	return &cl
}
