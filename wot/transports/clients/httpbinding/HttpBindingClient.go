package httpbinding

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PingMessage can be used by the server to ping the client that the connection is ready
const PingMessage = "ping"

const HTTPMessageIDHeader = "message-id"

// HTTP Headers
const (
	// StatusHeader for transports that support headers can include a progress status field
	StatusHeader = "status"

	// RequestIDHeader for transports that support headers can include a message-ID
	RequestIDHeader = tlsclient.HTTPRequestIDHeader

	// ConnectionIDHeader identifies the client's connection in case of multiple
	// connections in the same session. Used to identify the connection for subscriptions.
	ConnectionIDHeader = tlsclient.HTTPConnectionIDHeader

	// DataSchemaHeader for transports that support headers can include a dataschema
	// header to indicate an 'additionalresults' dataschema being returned.
	DataSchemaHeader = "dataschema"
)

// Paths used by this protocol binding - SYNC with HttpBindingClient.ts
//
// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
// The hub client will need the TD (ConsumedThing) to determine the paths.
const (

	// deprecated authn service - use the generated constants or forms
	PostLoginPath = "/authn/login"
	// deprecated authn service - use the generated constants
	PostLogoutPath = "/authn/logout"
	// deprecated authn service - use the generated constants
	PostRefreshPath = "/authn/refresh"
)

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
// operations. Use the SseBinding for this.
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

// ConnectWithLoginForm invokes login using a form - temporary helper
// intended for testing a connection with a web server.
//
// This sets the bearer token for further requests
func (cl *HttpBindingClient) ConnectWithLoginForm(password string) error {
	formMock := url.Values{}
	formMock.Add("loginID", cl.clientID)
	formMock.Add("password", password)
	fullURL := fmt.Sprintf("https://%s/login", cl.hostPort)

	//PostForm should return a cookie that should be used in the sse connection
	resp, err := cl.httpClient.PostForm(fullURL, formMock)
	if err == nil {
		// get the session token from the cookie
		cookie := resp.Request.Header.Get("cookie")
		kvList := strings.Split(cookie, ",")

		for _, kv := range kvList {
			kvParts := strings.SplitN(kv, "=", 2)
			if kvParts[0] == "session" {
				cl.bearerToken = kvParts[1]
				break
			}
		}
	}
	return err
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *HttpBindingClient) ConnectWithPassword(password string) (newToken string, err error) {

	slog.Info("ConnectWithPassword", "clientID", cl.clientID, "cid", cl.cid)

	// TODO: use TD form auth mechanism
	loginMessage := authn.UserLoginArgs{
		ClientID: cl.GetClientID(),
		Password: password,
	}
	argsJSON, _ := json.Marshal(loginMessage)
	resp, _, err := cl.Invoke(
		http.MethodPost, PostLoginPath, "", argsJSON, nil)
	if err != nil {
		slog.Warn("ConnectWithPassword failed", "err", err.Error())
		return "", err
	}
	token := ""
	err = jsoniter.Unmarshal(resp, &token)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: unexpected response: %s", err)
		return "", err
	}
	// store the bearer token further requests
	cl.mux.Lock()
	cl.bearerToken = token
	cl.mux.Unlock()
	cl.isConnected.Store(true)

	return token, err
}

// ConnectWithToken sets the bearer token to use with requests.
func (cl *HttpBindingClient) ConnectWithToken(token string) (newToken string, err error) {
	cl.mux.Lock()
	cl.bearerToken = token
	cl.mux.Unlock()
	cl.isConnected.Store(true)
	return token, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (cl *HttpBindingClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeEd25519)
	return k
}

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
	return transports.ProtocolTypeHTTPS
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

// Invoke a HTTPS method and read response.
//
// If token authentication is enabled then add the bearer token to the header
//
//	method: GET, PUT, POST, ...
//	reqPath: path to invoke
//	contentType of the payload or "" for default (application/json)
//	body contains the serialized request body
//	qParams: optional map with query parameters
//
// This returns the serialized response data, a response message ID, return status code or an error
func (cl *HttpBindingClient) Invoke(method string, reqPath string,
	contentType string, body []byte, qParams map[string]string) (
	resp []byte, headers http.Header, err error) {

	if cl.httpClient == nil {
		err = fmt.Errorf("Invoke: '%s'. Client is not started", reqPath)
		return nil, nil, err
	}

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

	// optional query parameters
	if qParams != nil {
		qValues := req.URL.Query()
		for k, v := range qParams {
			qValues.Add(k, v)
		}
		req.URL.RawQuery = qValues.Encode()
	}

	// set headers
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	requestID := shortid.MustGenerate()
	if requestID != "" {
		req.Header.Set(HTTPMessageIDHeader, requestID)
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
		slog.Error("Invoke returned internal server error", "reqPath", reqPath, "err", err.Error())
	} else if err != nil {
		err = fmt.Errorf("Invoke: Error %s %s: %w", method, reqPath, err)
	}
	return respBody, httpResp.Header, err
}

// InvokeOperation invokes the operation described in the Form
// The form must describe the HTTP protocol.
func (cl *HttpBindingClient) InvokeOperation(
	f tdd.Form, dThingID, name string, input interface{}, output interface{}) error {

	var dataJSON []byte
	operation := f.GetOperation()
	method, _ := f.GetMethodName()
	href, _ := f.GetHRef()
	params := map[string]string{
		"thingID": dThingID,
		"name":    name,
	}
	href2 := utils.Substitute(href, params)

	slog.Info("InvokeOperation",
		slog.String("op", operation),
		slog.String("method", method),
		slog.String("href", href2),
	)
	if method == "" {
		method = "GET"
	}
	if operation == "" || href == "" {
		slog.Error("InvokeOperation: Form is missing operation, method or href")
	}
	if input != nil {
		dataJSON, _ = jsoniter.Marshal(input)
	}
	respBody, _, err := cl.Invoke(
		method, href2, "", dataJSON, nil)

	slog.Warn("")

	if respBody != nil && err == nil {
		err = jsoniter.Unmarshal(respBody, output)
	}
	return err
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *HttpBindingClient) Logout() error {
	// TODO: find a way to derive this from a form
	slog.Info("Logout", slog.String("clientID", cl.clientID))
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostLogoutPath)
	_, _, err := cl.Invoke("POST", serverURL, "", nil, nil)
	return err
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *HttpBindingClient) RefreshToken(oldToken string) (newToken string, err error) {
	// TODO: find a way to derive this from a form

	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))
	refreshURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostRefreshPath)

	args := authn.UserRefreshTokenArgs{
		ClientID: cl.clientID,
		OldToken: oldToken,
	}
	data, _ := jsoniter.Marshal(args)
	// the bearer token holds the old token
	resp, _, err := cl.Invoke(
		"POST", refreshURL, "", data, nil)

	// set the new token as the bearer token
	if err == nil {
		err = jsoniter.Unmarshal(resp, &newToken)

		if err == nil {
			// reconnect using the new token
			cl.mux.Lock()
			cl.bearerToken = newToken
			cl.mux.Unlock()
		}
	}
	return newToken, err
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

// NewHttpBindingClient creates a new instance of the http binding client
//
//	fullURL of server to connect to, including the schema
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewHttpBindingClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) (*HttpBindingClient, error) {

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
	urlParts, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
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

	return &cl, nil
}
