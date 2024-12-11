package sseclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/httpclient"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"sync/atomic"
	"time"
)

// SsescTransportClient extends the https binding with the SSE-SC return channel.
//
// This client creates two http/2 connections, one for posting messages and
// one for a sse connection to establish a return channel.
//
// This client is for using the hiveot SSE-SC protocol extension.
// The hub server supports both SSE and SSE-SC.
//
// The difference between SSE and SSE-SC is that SSE provides subscription/observe in
// the http header during connection, while sse-sc uses a separate REST call to
// subscribe and unsubscribe. In addition, sse-sc uses the SSE eventID to pass
// the operation, thingID and affordance name, supporting multiple devices.
type SsescTransportClient struct {
	httpclient.HttpTransportClient

	// the sse connection path
	ssePath string

	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	lastError atomic.Pointer[error]

	subscriptions map[string]bool

	// Request and Response channel helper
	rnrChan *tputils.RnRChan
}

// helper to establish an sse connection using the given bearer token
// FIXME: use the http/2 binding connection
func (cl *SsescTransportClient) connectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("Missing SSE path")
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
func (cl *SsescTransportClient) ConnectWithPassword(password string) (newToken string, err error) {
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
func (cl *SsescTransportClient) ConnectWithToken(token string) (newToken string, err error) {
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
func (cl *SsescTransportClient) Disconnect() {
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
	cl.HttpTransportClient.Disconnect()

	if cl.rnrChan.Len() > 0 {
		slog.Error("Force closing unhandled RPC call", "n", cl.rnrChan.Len())
		cl.rnrChan.CloseAll()
	}
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *SsescTransportClient) handleSSEConnect(connected bool, err error) {
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

// SetSSEPath sets the new sse path to use.
// This allows to change the hub default /ssesc
func (cl *SsescTransportClient) SetSSEPath(ssePath string) {
	cl.BaseMux.Lock()
	cl.ssePath = ssePath
	cl.BaseMux.Unlock()
}

// NewSsescTransportClient creates a new instance of the http client with SSE-SC return-channel.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewSsescTransportClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm func(op string) td.Form,
	timeout time.Duration) *SsescTransportClient {

	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewSsescTransportClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", fullURL))
	} else {
		slog.Debug("NewSsescTransportClient: CA certificate",
			slog.String("destination", fullURL),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}

	cl := SsescTransportClient{
		ssePath:       transports.DefaultSSESCPath,
		subscriptions: nil,
		rnrChan:       tputils.NewRnRChan(),
	}
	// initialize the embedded http transport
	cl.HttpTransportClient.Init(
		fullURL, clientID, clientCert, caCert, getForm, timeout)
	//cl.tlsClient = tlsclient.NewTLSClient(
	//	hostPort, clientCert, caCert, timeout, cid)

	return &cl
}
