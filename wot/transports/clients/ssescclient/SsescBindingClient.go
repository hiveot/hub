package ssescclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/httpbinding"
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

// SsescBindingClient extends the https binding with the SSE return channel.
//
// This client creates two http/2 connections, one for posting messages and
// one for a sse connection to establish a return channel.
//
// This clients implements the REST API supported by the digitwin runtime services,
// specifically the directory, inbox, outbox, authn
type SsescBindingClient struct {
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
func (cl *SsescBindingClient) connectSSE(token string) (err error) {
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
func (cl *SsescBindingClient) ConnectWithPassword(password string) (newToken string, err error) {
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
func (cl *SsescBindingClient) ConnectWithToken(token string) (newToken string, err error) {
	newToken, err = cl.httpClient.ConnectWithToken(token)
	cl.isConnected.Store(true)
	return token, err
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
func (cl *SsescBindingClient) Disconnect() {
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
func (cl *SsescBindingClient) GetClientID() string {
	return cl.clientID
}

// GetCID returns the client's connection ID
func (cl *SsescBindingClient) GetCID() string {
	return cl.httpClient.GetCID()
}

func (cl *SsescBindingClient) GetConnectionStatus() (bool, string, error) {
	var lastErr error = nil
	// lastError is stored as pointer because atomic.Value cannot switch between error and nil type
	if cl.lastError.Load() != nil {
		lastErrPtr := cl.lastError.Load()
		lastErr = *lastErrPtr
	}
	return cl.isConnected.Load(), cl.GetCID(), lastErr
}

// GetProtocolType returns the type of protocol this client supports
func (cl *SsescBindingClient) GetProtocolType() string {
	return transports.ProtocolTypeSSESC
}

// GetServerURL returns the schema://address:port of the server connection
func (cl *SsescBindingClient) GetServerURL() string {
	return cl.fullURL
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *SsescBindingClient) handleSSEConnect(connected bool, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	slog.Debug("handleSSEConnect",
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
func (cl *SsescBindingClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Logout from the server and end the session
func (cl *SsescBindingClient) Logout() error {
	err := cl.httpClient.Logout()
	cl.Disconnect()
	return err
}

// Marshal encodes the native data into the wire format
func (cl *SsescBindingClient) Marshal(data any) []byte {
	jsonData, _ := json.Marshal(data)
	return jsonData
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
func (cl *SsescBindingClient) RefreshToken(oldToken string) (newToken string, err error) {
	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))

	newToken, err = cl.httpClient.RefreshToken(oldToken)
	return newToken, err
}

// Rpc sends an operation and waits for a completed or failed progress update.
// This uses a correlationID to link actions to progress updates. Only use this
// for operations that reply using the correlation ID.
func (cl *SsescBindingClient) Rpc(
	op tdd.Form, thingID string, name string, args interface{}, resp interface{}) (err error) {

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
	stat := cl.SendOperation(op, thingID, name, args, resp, correlationID)
	waitCount := 0

	// Intermediate status update such as 'applied' are not errors. Wait longer.
	for {
		// if the hub return channel doesnt exists then don't bother waiting for a result
		if !cl.IsConnected() {
			break
		}

		// wait at most cl.timeout or until delivery completes or fails
		// if the connection breaks while waiting then tlsClient will be nil.
		if time.Duration(waitCount)*time.Second > cl.timeout {
			break
		}
		if stat.Status == vocab.RequestCompleted || stat.Status == vocab.RequestFailed {
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
		stat, err = cl.WaitForProgressUpdate(rChan, correlationID, time.Second)
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
		slog.String("cid", cl.GetClientID()),
		slog.String("status", stat.Status),
	)

	// check for errors
	if err == nil {
		if stat.Error != "" {
			err = errors.New(stat.Error)
		} else if stat.Status != vocab.RequestCompleted {
			err = errors.New("Delivery not complete. Status: " + stat.Status)
		}
	}
	if err != nil {
		slog.Error("RPC failed",
			"thingID", thingID, "name", name, "err", err.Error())
	}
	// only once completed will there be a reply as a result
	if err == nil && resp != nil {
		// no choice but to decode
		err = utils.Decode(stat.Output, resp)
	}
	return err
}

// SendOperation sends the operation described in the given Form.
// The form must describe the HTTP/SSE-SC protocol.
func (cl *SsescBindingClient) SendOperation(
	op tdd.Form, dThingID, name string, input interface{}, output interface{},
	correlationID string) (stat transports.RequestStatus) {

	stat = cl.httpClient.SendOperation(op, dThingID, name, input, output, correlationID)
	return stat
}

// PubOperationStatus [agent] sends a operation progress status update to the server.
func (cl *SsescBindingClient) SendOperationStatus(stat transports.RequestStatus) {
	cl.httpClient.SendOperationStatus(stat)
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *SsescBindingClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.mux.Lock()
	cl.connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the handler that receives event type messages send by the server.
// This requires a sub-protocol with a return channel.
func (cl *SsescBindingClient) SetMessageHandler(cb transports.MessageHandler) {
	cl.mux.Lock()
	cl.messageHandler = cb
	cl.mux.Unlock()
}

// SetRequestHandler set the handler that receives requests from the server,
// where a status response is expected.
// This requires a sub-protocol with a return channel.
func (cl *SsescBindingClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.mux.Lock()
	cl.requestHandler = cb
	cl.mux.Unlock()
}

// SetSSEPath sets the new sse path to use.
// This allows to change the hub default /ssesc
func (cl *SsescBindingClient) SetSSEPath(ssePath string) {
	cl.mux.Lock()
	cl.ssePath = ssePath
	cl.mux.Unlock()
}

// WaitForProgressUpdate waits for an async progress update message or until timeout
// This returns the status or an error if the timeout has passed
func (cl *SsescBindingClient) WaitForProgressUpdate(
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

// NewSsescBindingClient creates a new instance of the http client with SSE-SC return-channel.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewSsescBindingClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *SsescBindingClient {

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
	httpBindingClient := httpbinding.NewHttpBindingClient(fullURL, clientID, clientCert, caCert, timeout)
	cl := SsescBindingClient{

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
