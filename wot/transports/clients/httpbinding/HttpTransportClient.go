package httpbinding

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/utils"
	"github.com/hiveot/hub/wot/transports/utils/tlsclient"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// PingMessage can be used by the server to ping the client that the connection is ready
const PingMessage = "ping"

//const HTTPMessageIDHeader = "message-id"

// HTTP Headers
const (
	// StatusHeader for transports that support headers can include a progress status field
	StatusHeader = "status"

	// CorrelationIDHeader for transports that support headers can include a message-ID
	CorrelationIDHeader = "correlation-id"

	// ConnectionIDHeader identifies the client's connection in case of multiple
	// connections in the same session. Used to identify the connection for subscriptions.
	ConnectionIDHeader = "connection-id"

	// DataSchemaHeader for transports that support headers can include a dataschema
	// header to indicate an 'additionalresults' dataschema being returned.
	DataSchemaHeader = "dataschema"
)

// paths not available through forms
const PostAgentPublishProgressPath = "/agent/progress"

// HttpBindingClient is the http/2 client for performing operations on one or more Things.
// This implements the IBindingClient interface.
//
// NOTE: this binding implementation is intended to connect to the hiveOT Hub,
// not for connecting to 3rd party Thing servients. As such, it doesn't use forms
// as the endpoints are well known.
// The use of Forms to perform operation is planned. Thing top level operations
// will be replaced with a single InvokeForm method.
//
// This client has no return channel so it does not support subscribe or observe
// operations. Use the SsescBindingClient or WssBindingClient for this.
type HttpBindingClient struct {
	// http server address and port
	hostPort string
	// the CA certificate to validate the server TLS connection
	caCert *x509.Certificate
	// The client ID of the user of this binding
	clientID string
	// The client connection-id of this instance
	cid string
	// request timeout
	timeout time.Duration

	// callbacks for connection, events and requests
	connectHandler func(connected bool, err error)
	// client side handler that receives messages for consumers
	//messageHandler transports.MessageHandler
	// map of requestID to delivery status update channel
	//requestHandler transports.RequestHandler

	//
	mux sync.RWMutex

	// http2 client for posting messages
	httpClient *http.Client
	// authentication bearer token if authenticated
	bearerToken string
	// custom headers to include in each request
	headers map[string]string

	isConnected atomic.Bool
	lastError   atomic.Pointer[error]
}

// _send a HTTPS method and read response.
//
// If token authentication is enabled then add the bearer token to the header
//
//	method: GET, PUT, POST, ...
//	reqPath: path to invoke
//	contentType of the payload or "" for default (application/json)
//	thingID optional path URI variable
//	name optional path URI variable containing affordance name
//	body contains the serialized request body
//	correlationID: optional correlationID header value
//
// This returns the serialized response data, a response message ID, return status code or an error
func (cl *HttpBindingClient) _send(method string, methodPath string,
	contentType string, thingID string, name string,
	body []byte, correlationID string) (
	resp []byte, headers http.Header, err error) {

	if cl.httpClient == nil {
		err = fmt.Errorf("_send: '%s'. Client is not started", methodPath)
		return nil, nil, err
	}
	// use + as wildcard for thingID to avoid a 404
	// while it not recommended, it is allowed to subscribe/observe all things
	if thingID == "" {
		thingID = "+"
	}
	// use + as wildcard for affordance name to avoid a 404
	// this should not happen very often but it is allowed
	if name == "" {
		name = "+"
	}

	// substitute URI variables in the path
	vars := map[string]string{
		"thingID": thingID,
		"name":    name}
	reqPath := utils.Substitute(methodPath, vars)

	// Caution! a double // in the path causes a 301 and changes post to get
	bodyReader := bytes.NewReader(body)
	fullURL := cl.GetServerURL() + reqPath
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	// set the origin header to the intended destination without the path
	parts, err := url.Parse(fullURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Set("Origin", origin)

	// set the authorization header
	if cl.bearerToken != "" {
		req.Header.Add("Authorization", "bearer "+cl.bearerToken)
	}

	// set other headers
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set(ConnectionIDHeader, cl.cid)
	if correlationID != "" {
		req.Header.Set(CorrelationIDHeader, correlationID)
	}
	for k, v := range cl.headers {
		req.Header.Set(k, v)
	}

	httpResp, err := cl.httpClient.Do(req)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, err
	}
	respBody, err := io.ReadAll(httpResp.Body)
	//respRequestID = httpResp.Header.Get(HTTPMessageIDHeader)
	// response body MUST be closed
	_ = httpResp.Body.Close()
	httpStatus := httpResp.StatusCode

	if httpStatus == 401 {
		err = fmt.Errorf("%s", httpResp.Status)
	} else if httpStatus >= 400 && httpStatus < 500 {
		err = fmt.Errorf("%s: %s", httpResp.Status, respBody)
		if httpResp.Status == "" {
			err = fmt.Errorf("%d (%s): %s", httpResp.StatusCode, httpResp.Status, respBody)
		}
	} else if httpStatus >= 500 {
		err = fmt.Errorf("Error %d (%s): %s", httpStatus, httpResp.Status, respBody)
		slog.Error("_send returned internal server error", "reqPath", reqPath, "err", err.Error())
	} else if err != nil {
		err = fmt.Errorf("_send: Error %s %s: %w", method, reqPath, err)
	}
	return respBody, httpResp.Header, err
}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	kp is the key-pair used to the certificate validation
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
//func (cl *HttpBindingClient) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//	cl.mux.RLock()
//	defer cl.mux.RUnlock()
//	_ = kp
//	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//	return err
//}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *HttpBindingClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeEd25519)
//	return k
//}

// Disconnect from the server
func (cl *HttpBindingClient) Disconnect() {
	slog.Debug("HttpBindingClient.Disconnect",
		slog.String("clientID", cl.clientID),
	)

	cl.mux.Lock()
	if cl.isConnected.Load() {
		cl.httpClient.CloseIdleConnections()
	}
	cl.mux.Unlock()
}

// GetClientID returns the client's account ID
func (cl *HttpBindingClient) GetClientID() string {
	return cl.clientID
}

// GetCID returns the client's connection ID
func (cl *HttpBindingClient) GetCID() string {
	return cl.cid
}

func (cl *HttpBindingClient) GetConnectionStatus() (bool, string, error) {
	var lastErr error = nil
	// lastError is stored as pointer because atomic.Value cannot switch between error and nil type
	if cl.lastError.Load() != nil {
		lastErrPtr := cl.lastError.Load()
		lastErr = *lastErrPtr
	}
	return cl.isConnected.Load(), cl.cid, lastErr
}

// GetProtocolType returns the type of protocol this client supports
func (cl *HttpBindingClient) GetProtocolType() string {
	return transports.ProtocolTypeHTTP
}

// GetServerURL returns the schema://address:port of the server connection
func (cl *HttpBindingClient) GetServerURL() string {
	hubURL := fmt.Sprintf("https://%s", cl.hostPort)
	return hubURL
}

func (cl *HttpBindingClient) GetTlsClient() *http.Client {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.httpClient
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *HttpBindingClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Rpc sends an operation and returns the result.
//
// This is the same as SendOperation. If the operation isn't completed it returns
// an error.
//
// Since the http binding doesn't have a return channel, this only works with
// operations that return their result as http response.
func (cl *HttpBindingClient) Rpc(
	form tdd.Form, dThingID, name string, input interface{}, output interface{}) error {

	status, err := cl.SendOperation(form, dThingID, name, input, output, "")
	_ = status
	// there is no return channel receiving result is it.
	return err
}

// SendOperation sends the operation described in the Form.
// The form must describe the HTTP protocol.
func (cl *HttpBindingClient) SendOperation(
	f tdd.Form, dThingID, name string, input interface{}, output interface{},
	correlationID string) (status string, err error) {

	var dataJSON []byte
	operation := f.GetOperation()
	method, _ := f.GetMethodName()
	href, _ := f.GetHRef()

	slog.Info("SendOperation",
		slog.String("op", operation),
		slog.String("method", method),
		slog.String("href", href),
	)
	if method == "" {
		method = http.MethodGet
	}
	if operation == "" || href == "" {
		slog.Error("SendOperation: Form is missing operation or href")
	}
	if input != nil {
		dataJSON, _ = jsoniter.Marshal(input)
	}
	respBody, headers, err := cl._send(
		method, href, "", dThingID, name, dataJSON, correlationID)
	// TODO: the datatype header describes the alternative outputs if provided
	// what to do with this?
	dataSchema := headers.Get(DataSchemaHeader)
	_ = dataSchema

	if err != nil {
		return transports.RequestFailed, err
	}
	// an alternative result is received. For now only RequestStatus is supported.
	if dataSchema == "RequestStatus" {
		stat := transports.RequestStatus{}
		err = jsoniter.Unmarshal(respBody, &stat)
		if stat.Error != "" {
			return transports.RequestFailed, errors.New(stat.Error)
		}
		if stat.Output != nil && output != nil {
			err = jsoniter.Unmarshal(respBody, output)
		}
		return stat.Status, err
	}
	if respBody != nil && output != nil {
		err = jsoniter.Unmarshal(respBody, output)
		return transports.RequestCompleted, err
	}
	return transports.RequestPending, nil
}

// SendOperationStatus [agent] sends a operation progress status update to the server.
//
// NOTE: this message is not defined in the http binding spec for 2 reasons:
// 1. HTTP bindings require the use of a sub-protocol to return data.
// 2. WoT only defines consumer operations and this is an agent operation
func (cl *HttpBindingClient) SendOperationStatus(stat transports.RequestStatus) {
	slog.Debug("SendOperationStatus",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", stat.ThingID),
		slog.String("name", stat.Name),
		slog.String("progress", stat.Status),
		slog.String("correlationID", stat.CorrelationID))

	//stat2 := cl.Pub(http.MethodPost, PostAgentPublishProgressPath,
	//	"", "", stat, stat.CorrelationID)
	//
	dataJSON, _ := jsoniter.Marshal(stat)
	_, _, err := cl._send(
		http.MethodPost, PostAgentPublishProgressPath, "",
		"", "", dataJSON, stat.CorrelationID)

	if err != nil {
		slog.Warn("SendOperationStatus failed", "err", err.Error())
	}
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *HttpBindingClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.mux.Lock()
	cl.connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the handler that receives event type messages send by the server.
// This requires a sub-protocol with a return channel.
func (cl *HttpBindingClient) SetMessageHandler(cb transports.MessageHandler) {
	//cl.mux.Lock()
	//cl.messageHandler = cb
	//cl.mux.Unlock()
}

// SetRequestHandler set the handler that receives requests from the server,
// where a status response is expected.
// This requires a sub-protocol with a return channel.
func (cl *HttpBindingClient) SetRequestHandler(cb transports.RequestHandler) {
	//cl.mux.Lock()
	//cl.requestHandler = cb
	//cl.mux.Unlock()
}

// NewHttpTransportClient creates a new instance of the http binding client
//
//	fullURL of server to connect to, including the schema
//	clientID to connect as; for logging and ConnectWithPassword. It is
//
// ignored if auth token is used.
//
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewHttpTransportClient(
	fullURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *HttpBindingClient {

	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewHttpTransportClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", fullURL))
	} else {
		slog.Debug("NewHttpTransportClient: CA certificate",
			slog.String("destination", fullURL),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}
	urlParts, _ := url.Parse(fullURL)
	cid := shortid.MustGenerate()
	cl := HttpBindingClient{
		//_status: hubclient.TransportStatus{
		//	HubURL:               fmt.Sprintf("https://%s", hostPort),
		caCert:   caCert,
		clientID: clientID,
		cid:      cid,

		// max delay 3 seconds before a response is expected
		timeout:  timeout,
		hostPort: urlParts.Host,
		//
		headers: make(map[string]string),
	}
	cl.httpClient = tlsclient.NewHttp2TLSClient(caCert, clientCert, timeout)

	return &cl
}
