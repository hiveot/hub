package ssescclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/httpbinding"
	"github.com/hiveot/hub/wot/transports/utils"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// PingMessage can be used by the server to ping the client that the connection is ready
const PingMessage = "ping"

const SSEPath = "/ssesc"

// SsescTransportClient extends the https binding with the SSE return channel.
//
// This client creates two http/2 connections, one for posting messages and
// one for a sse connection to establish a return channel.
//
// This clients implements the REST API supported by the digitwin runtime services,
// specifically the directory, inbox, outbox, authn
type SsescTransportClient struct {
	// the http binding this extends
	httpClient *httpbinding.HttpBindingClient

	// the CA certificate to validate the server TLS connection
	caCert *x509.Certificate
	// The client ID of the user of this binding
	clientID string
	// request timeout
	timeout time.Duration

	fullURL string
	// the sse connection path
	ssePath string

	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	// callbacks for connection, events and requests
	connectHandler func(connected bool, err error)
	// client side handler that receives messages for consumers
	messageHandler transports.MessageHandler
	// map of requestID to delivery status update channel
	requestHandler transports.RequestHandler

	mux sync.RWMutex

	lastError   atomic.Pointer[error]
	isConnected atomic.Bool

	subscriptions map[string]bool

	// map of requestID to delivery status update channel
	correlData map[string]chan *transports.RequestStatus
}

// helper to establish an sse connection using the given bearer token
// FIXME: use the http/2 binding connection
func (cl *SsescTransportClient) connectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("Missing SSE path")
	}
	// create a second client to establish the sse connection if a path is set
	parts, err := url.Parse(cl.fullURL)
	if err != nil {
		return err
	}

	sseURL := fmt.Sprintf("https://%s%s", parts.Host, cl.ssePath)
	_, cid, _ := cl.httpClient.GetConnectionStatus()
	cl.sseCancelFn, err = ConnectSSE(
		cl.clientID, cid,
		sseURL, token, cl.caCert,
		cl.handleSSEConnect, cl.handleSseEvent)

	return err
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *SsescTransportClient) ConnectWithPassword(password string) (newToken string, err error) {
	newToken, err = cl.httpClient.ConnectWithPassword(password)
	if err != nil {
		return "", err
	}
	err = cl.connectSSE(newToken)
	if err != nil {
		return "", err
	}
	cl.isConnected.Store(true)
	return newToken, err
}

// ConnectWithToken sets the bearer token to use with requests.
func (cl *SsescTransportClient) ConnectWithToken(token string) (newToken string, err error) {
	newToken, err = cl.httpClient.ConnectWithToken(token)
	if err != nil {
		return "", err
	}
	err = cl.connectSSE(newToken)
	if err != nil {
		return "", err
	}
	cl.isConnected.Store(true)
	return newToken, err
}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	kp is the key-pair used to the certificate validation
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
//func (cl *HttpSSEClient) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//	cl.mux.RLock()
//	defer cl.mux.RUnlock()
//	_ = kp
//	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//	return err
//}

// Disconnect from the server
func (cl *SsescTransportClient) Disconnect() {
	slog.Debug("HttpSSEClient.Disconnect",
		slog.String("clientID", cl.clientID),
		slog.String("cid", cl.httpClient.GetCID()),
	)

	cl.mux.Lock()
	sseCancelFn := cl.sseCancelFn
	cl.sseCancelFn = nil
	//tlsClient := cl.tlsClient
	//cl.tlsClient = nil
	cl.mux.Unlock()

	// the connection status will update, if changed, through the sse callback
	if sseCancelFn != nil {
		sseCancelFn()
	}
	if cl.httpClient != nil {
		cl.httpClient.Disconnect()
	}
}

// GetClientID returns the client's account ID
func (cl *SsescTransportClient) GetClientID() string {
	return cl.clientID
}

// GetCID returns the client's connection ID
func (cl *SsescTransportClient) GetCID() string {
	return cl.httpClient.GetCID()
}

func (cl *SsescTransportClient) GetConnectionStatus() (bool, string, error) {
	var lastErr error = nil
	// lastError is stored as pointer because atomic.Value cannot switch between error and nil type
	if cl.lastError.Load() != nil {
		lastErrPtr := cl.lastError.Load()
		lastErr = *lastErrPtr
	}
	return cl.isConnected.Load(), cl.GetCID(), lastErr
}

// GetProtocolType returns the type of protocol this client supports
func (cl *SsescTransportClient) GetProtocolType() string {
	return transports.ProtocolTypeSSESC
}

// GetServerURL returns the schema://address:port of the server connection
func (cl *SsescTransportClient) GetServerURL() string {
	return cl.fullURL
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
		slog.String("clientID", cl.clientID),
		slog.String("cid", cl.httpClient.GetCID()),
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
	handler := cl.connectHandler
	cl.mux.RUnlock()

	if connectionChanged && handler != nil {
		handler(connected, err)
	}
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *SsescTransportClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Logout from the server and end the session
func (cl *SsescTransportClient) Logout() error {
	err := cl.httpClient.Logout()
	cl.Disconnect()
	return err
}

// Marshal encodes the native data into the wire format
func (cl *SsescTransportClient) Marshal(data any) []byte {
	jsonData, _ := json.Marshal(data)
	return jsonData
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
func (cl *SsescTransportClient) RefreshToken(oldToken string) (newToken string, err error) {
	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))

	newToken, err = cl.httpClient.RefreshToken(oldToken)
	return newToken, err
}

// Rpc sends an operation and waits for a completed or failed progress update.
// This uses a correlationID to link actions to progress updates. Only use this
// for operations that reply using the correlation ID.
func (cl *SsescTransportClient) Rpc(
	op tdd.Form, thingID string, name string, args interface{}, output interface{}) (err error) {

	// a correlationID is needed before the action is published in order to match it with the reply
	correlationID := "rpc-" + shortid.MustGenerate()

	slog.Info("Rpc (request)",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("correlationID", correlationID),
		slog.String("cid", cl.GetClientID()),
	)

	rChan := make(chan *transports.RequestStatus)
	cl.mux.Lock()
	cl.correlData[correlationID] = rChan
	cl.mux.Unlock()

	// invoke with query parameters to provide the message ID
	status, err := cl.SendOperation(op, thingID, name, args, output, correlationID)
	if err != nil {
		return err
	}
	waitCount := 0

	// Intermediate status update such as 'applied' are not errors. Wait longer.
	for {
		// if the hub return channel doesn't exists then don't bother waiting for a result
		if !cl.IsConnected() {
			break
		}

		// wait at most cl.timeout or until delivery completes or fails
		// if the connection breaks while waiting then tlsClient will be nil.
		if time.Duration(waitCount)*time.Second > cl.timeout {
			break
		}
		if status == transports.RequestCompleted || status == transports.RequestFailed {
			break
		}
		if waitCount > 0 {
			slog.Info("Rpc (wait)",
				slog.Int("count", waitCount),
				slog.String("clientID", cl.clientID),
				slog.String("name", name),
				slog.String("correlationID", correlationID),
			)
		}
		var stat transports.RequestStatus
		stat, err = cl.WaitForProgressUpdate(rChan, correlationID, time.Second)
		status = stat.Status
		if stat.Error != "" {
			err = errors.New(stat.Error)
		} else if status == transports.RequestCompleted {
			if stat.Output != nil {
				err = utils.Decode(stat.Output, output)
				break
			}
		}
		waitCount++
	}
	cl.mux.Lock()
	delete(cl.correlData, correlationID)
	cl.mux.Unlock()
	slog.Info("Rpc (result)",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("correlationID", correlationID),
		slog.String("cid", cl.GetCID()),
		slog.String("status", status),
	)

	// check for errors
	if err == nil && status != transports.RequestCompleted {
		err = errors.New("Delivery not complete. Status: " + status)
	}
	if err != nil {
		slog.Error("RPC failed",
			"thingID", thingID, "name", name, "err", err.Error())
	}
	return err
}

// SendOperation sends the operation described in the given Form.
// The form must describe the HTTP/SSE-SC protocol.
// The returning status object describes the result with status and optionally
// output or an error.
func (cl *SsescTransportClient) SendOperation(
	form tdd.Form, dThingID, name string, input interface{}, output interface{},
	correlationID string) (string, error) {
	// simply pass it to the http binding
	status, err := cl.httpClient.SendOperation(form, dThingID, name, input, output, correlationID)
	return status, err
}

// PubOperationStatus [agent] sends a operation progress status update to the server.
func (cl *SsescTransportClient) SendOperationStatus(stat transports.RequestStatus) {
	cl.httpClient.SendOperationStatus(stat)
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *SsescTransportClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.mux.Lock()
	cl.connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the handler that receives event type messages send by the server.
// This requires a sub-protocol with a return channel.
func (cl *SsescTransportClient) SetMessageHandler(cb transports.MessageHandler) {
	cl.mux.Lock()
	cl.messageHandler = cb
	cl.mux.Unlock()
}

// SetRequestHandler set the handler that receives requests from the server,
// where a status response is expected.
// This requires a sub-protocol with a return channel.
func (cl *SsescTransportClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.mux.Lock()
	cl.requestHandler = cb
	cl.mux.Unlock()
}

// SetSSEPath sets the new sse path to use.
// This allows to change the hub default /ssesc
func (cl *SsescTransportClient) SetSSEPath(ssePath string) {
	cl.mux.Lock()
	cl.ssePath = ssePath
	cl.mux.Unlock()
}

// WaitForProgressUpdate waits for an async progress update message or until timeout
// This returns the status or an error if the timeout has passed
func (cl *SsescTransportClient) WaitForProgressUpdate(
	statChan chan *transports.RequestStatus, requestID string, timeout time.Duration) (
	stat transports.RequestStatus, err error) {

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	select {
	case statC := <-statChan:
		stat = *statC
		break
	case <-ctx.Done():
		err = errors.New("Timeout waiting for status update: requestID=" + requestID)
	}

	return stat, err
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
	timeout time.Duration) *SsescTransportClient {

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

	// establish the http client instance that handles http commands
	httpBindingClient := httpbinding.NewHttpTransportClient(fullURL, clientID, clientCert, caCert, timeout)
	cl := SsescTransportClient{

		httpClient: httpBindingClient,

		//	HubURL:               fmt.Sprintf("https://%s", hostPort),
		caCert:   caCert,
		clientID: clientID,

		// max delay 3 seconds before a response is expected
		timeout: timeout,
		fullURL: fullURL,
		ssePath: SSEPath,

		correlData: make(map[string]chan *transports.RequestStatus),
		// max message size for bulk reads is 10MB.
	}

	//cl.tlsClient = tlsclient.NewTLSClient(
	//	hostPort, clientCert, caCert, timeout, cid)

	return &cl
}
