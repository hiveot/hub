package httpsseclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/hiveotsseserver"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
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

// HttpClientConnection is the http/2 client for connecting a WoT client to a
// WoT server. This implements the IClientConnection interface.
//
// This can be used by both consumers and agents.
// This is intended to be used together with an SSE return channel.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents.
type HttpClientConnection struct {

	// handler for requests send by clients
	appConnectHandler transports.ConnectionHandler

	// handler for requests send by clients
	appRequestHandler transports.RequestHandler
	// handler for responses sent by agents
	appResponseHandler transports.ResponseHandler

	clientID string

	// CA certificate to verify the server with
	caCert *x509.Certificate

	// This client's connection ID
	cid string

	// The full server's URL schema://host:port/path
	fullURL string
	// The server host:port
	hostPort string

	isConnected atomic.Bool

	// ProtocolType of subprotocol ProtocolTypeHTTPS/SSESC/WSS
	protocolType string

	// RPC timeout
	timeout time.Duration
	// protected operations
	mux sync.RWMutex
	// http2 client for posting messages
	httpClient *http.Client
	// authentication bearer token if authenticated
	bearerToken string

	// getForm obtains the form for sending a request or notification
	// if nil, then the hiveot protocol envelope and URL are used as fallback
	getForm transports.GetFormHandler

	// custom headers to include in each request

	headers map[string]string

	lastError atomic.Pointer[error]
}

// _send a HTTPS method and return the http response.
//
// If token authentication is enabled then add the bearer token to the header
//
//	method: GET, PUT, POST, ...
//	reqPath: path to invoke
//	contentType of the payload or "" for default (application/json)
//	thingID optional path URI variable
//	name optional path URI variable containing affordance name
//	body contains the raw serialized request body
//	correlationID: optional correlationID header value
//
// This returns the raw serialized response data, a response message ID, return status code or an error
func (cl *HttpClientConnection) _send(method string, methodPath string,
	body []byte, correlationID string) (
	resp []byte, headers http.Header, err error) {

	if cl.httpClient == nil {
		err = fmt.Errorf("_send: '%s'. Client is not started", methodPath)
		return nil, nil, err
	}
	// Caution! a double // in the path causes a 301 and changes post to get
	bodyReader := bytes.NewReader(body)
	serverURL := cl.GetServerURL()
	parts, _ := url.Parse(serverURL)
	parts.Path = methodPath
	fullURL := parts.String()
	//fullURL := parts.cl.GetServerURL() + reqPath
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		err = fmt.Errorf("_send %s %s failed: %w", method, fullURL, err)
		return nil, nil, err
	}

	// set the origin header to the intended destination without the path
	//parts, err := url.Parse(fullURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Set("Origin", origin)

	// set the authorization header
	if cl.bearerToken != "" {
		req.Header.Add("Authorization", "bearer "+cl.bearerToken)
	}

	// set other headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(httpserver.ConnectionIDHeader, cl.GetConnectionID())
	if correlationID != "" {
		req.Header.Set(httpserver.CorrelationIDHeader, correlationID)
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
	//respCorrelationID = httpResp.Header.Get(HTTPMessageIDHeader)
	// response body MUST be closed
	_ = httpResp.Body.Close()
	httpStatus := httpResp.StatusCode

	if httpStatus == 401 {
		err = fmt.Errorf("%s", httpResp.Status)
	} else if httpStatus >= 400 && httpStatus < 500 {
		err = fmt.Errorf("%s: %s", httpResp.Status, fullURL)
		if httpResp.Status == "" {
			err = fmt.Errorf("%d (%s): %s", httpResp.StatusCode, httpResp.Status, respBody)
		}
	} else if httpStatus >= 500 {
		err = fmt.Errorf("Error %d (%s): %s", httpStatus, httpResp.Status, respBody)
		slog.Error("_send returned internal server error", "reqPath", methodPath, "err", err.Error())
	} else if err != nil {
		err = fmt.Errorf("_send: Error %s %s: %w", method, methodPath, err)
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
//
//	func (cl *HttpClientConnection) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//		cl.mux.RLock()
//		defer cl.mux.RUnlock()
//		_ = kp
//		cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//		return err
//	}

// ConnectWithLoginForm invokes login using a form - temporary helper
// intended for testing a connection to a web server.
//
// This sets the bearer token for further requests. It requires the server
// to set a session cookie in response to the login.
func (cl *HttpClientConnection) ConnectWithLoginForm(
	password string) (newToken string, err error) {

	// FIXME: does this client need a cookie jar???
	formMock := url.Values{}
	formMock.Add("loginID", cl.GetClientID())
	formMock.Add("password", password)

	var loginHRef string
	f := cl.getForm(wot.HTOpLoginWithForm, "", "")
	if f != nil {
		loginHRef, _ = f.GetHRef()
	}
	loginURL, err := url.Parse(loginHRef)
	if err != nil {
		return "", err
	}
	if loginURL.Host == "" {
		loginHRef = cl.fullURL + loginHRef
	}

	//PostForm should return a cookie that should be used in the http connection
	if loginHRef == "" {
		return "", errors.New("Login path not found in getForm")
	}
	resp, err := cl.httpClient.PostForm(loginHRef, formMock)
	if err != nil {
		return "", err
	}

	// get the session token from the cookie
	//cookie := resp.Request.Header.Get("cookie")
	cookie := resp.Header.Get("cookie")
	kvList := strings.Split(cookie, ",")

	for _, kv := range kvList {
		kvParts := strings.SplitN(kv, "=", 2)
		if kvParts[0] == "session" {
			cl.bearerToken = kvParts[1]
			break
		}
	}
	if cl.bearerToken == "" {
		slog.Error("No session cookie was received on login")
	}
	return cl.bearerToken, err
}

// ConnectWithPassword connects to the TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *HttpClientConnection) ConnectWithPassword(password string) (newToken string, err error) {

	slog.Info("ConnectWithPassword",
		"clientID", cl.GetClientID(), "connectionID", cl.GetConnectionID())

	// FIXME: figure out how a standard login method is used to obtain an auth token
	loginMessage := map[string]string{
		"login":    cl.GetClientID(),
		"password": password,
	}
	f := cl.getForm(wot.HTOpLogin, "", "")
	if f == nil {
		err = fmt.Errorf("missing form for login operation")
		slog.Error(err.Error())
		return "", err
	}
	method, _ := f.GetMethodName()
	href, _ := f.GetHRef()

	dataJSON, _ := jsoniter.Marshal(loginMessage)
	outputRaw, _, err := cl._send(method, href, dataJSON, "")

	if err != nil {
		slog.Warn("ConnectWithPassword failed", "err", err.Error())
		return "", err
	}
	err = jsoniter.Unmarshal(outputRaw, &newToken)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: unexpected response: %s", err)
		return "", err
	}

	// store the bearer token further requests
	cl.mux.Lock()
	cl.bearerToken = newToken
	cl.mux.Unlock()
	//cl.BaseIsConnected.Store(true)

	return newToken, err
}

// ConnectWithToken sets the authentication bearer token to authenticate http requests.
func (cl *HttpClientConnection) ConnectWithToken(token string) error {
	cl.mux.Lock()
	cl.bearerToken = token
	cl.mux.Unlock()
	//cl.BaseIsConnected.Store(true)

	// HTTP connection is not yet established
	//newToken, err = cl.RefreshToken(token)
	return nil
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
//func (cl *HttpClientConnection) RefreshToken(oldToken string) (newToken string, err error) {
//
//	newToken, err = cl.BaseClient.RefreshToken(oldToken)
//	if err == nil {
//		cl.BaseMux.Lock()
//		cl.bearerToken = newToken
//		cl.BaseMux.Unlock()
//	}
//	return newToken, err
//}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
//func (cl *HttpClientConnection) Logout() error {
//	// TODO: can this be derived from a form?
//	slog.Info("Logout",
//		slog.String("clientID", cl.GetClientID()))
//	_, _, err := cl._send(http.MethodPost, httpserver.HttpPostLogoutPath, nil, "")
//	return err
//}

// Disconnect from the server
func (cl *HttpClientConnection) Disconnect() {
	slog.Debug("HttpClientConnection.Disconnect",
		slog.String("clientID", cl.clientID),
	)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.isConnected.Load() {
		cl.httpClient.CloseIdleConnections()
	}
}

// GetDefaultForm return the default http form for the operation
// This simply returns nil for anything else than login.
func (cl *HttpClientConnection) GetDefaultForm(op, thingID, name string) (f *td.Form) {
	// login has its own URL as it is unauthenticated
	if op == wot.HTOpLogin {
		href := httpserver.HttpPostLoginPath
		nf := td.NewForm(op, href)
		nf.SetMethodName(http.MethodPost)
		f = &nf
	}
	// everything else has no default form, so falls back to hiveot protocol endpoints
	return f
}

// GetClientID returns the client's account ID
func (cl *HttpClientConnection) GetClientID() string {
	return cl.clientID
}

// GetConnectionID returns the client's connection ID
func (cl *HttpClientConnection) GetConnectionID() string {
	return cl.cid
}

// GetProtocolType returns the type of protocol this client supports
func (cl *HttpClientConnection) GetProtocolType() string {
	return transports.ProtocolTypeWotHTTPBasic
}

// GetServerURL returns the schema://address:port/path of the server connection
func (cl *HttpClientConnection) GetServerURL() string {
	return cl.fullURL
}

func (cl *HttpClientConnection) GetTlsClient() *http.Client {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.httpClient
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *HttpClientConnection) IsConnected() bool {
	return cl.isConnected.Load()
}

// SendRequest sends a request message to the server.
//
//	If no form is found them send it using the hiveot protocol. The request will
//	be carried in the RequestMessage envelope (as-is) and the response will be
//	received in the ResponseMessage envelope.
//
// If a result is included in the http response then this is passed to the BaseRnR
// channel associated with the request just like it is done with an async response.
//
// This locates the form for the operation using 'getForm' and uses the result
// to determine the URL to publish the request to.
func (cl *HttpClientConnection) SendRequest(req *transports.RequestMessage) error {

	var dataJSON []byte
	var method string
	var href string
	var output any
	var thingID = req.ThingID
	var name = req.Name

	if req.Operation == "" && req.CorrelationID == "" {
		err := fmt.Errorf("SendMessage: missing both operation and correlationID")
		slog.Error(err.Error())
		return err
	}

	// the getForm callback provides the method and URL to invoke for this operation.
	// use the hiveot fallback if not available
	f := cl.getForm(req.Operation, req.ThingID, req.Name)
	if f != nil {
		method, _ = f.GetMethodName()
		href, _ = f.GetHRef()
	}

	if f == nil {
		// fallback to using the hiveot request URL
		method = http.MethodPost
		href = hiveotsseserver.HiveOTPostRequestHRef
	}
	if req.Input != nil {
		dataJSON, _ = jsoniter.Marshal(req.Input)
	}
	// use + as wildcard for thingID to avoid a 404
	// while not recommended, it is allowed to subscribe/observe all things
	if thingID == "" {
		thingID = "+"
	}
	// use + as wildcard for affordance name to avoid a 404
	// this should not happen very often but it is allowed
	if name == "" {
		name = "+"
	}

	// substitute URI variables in the path, if any
	vars := map[string]string{
		"thingID":   thingID,
		"name":      name,
		"operation": req.Operation}
	reqPath := tputils.Substitute(href, vars)

	// note, if the request is the hiveot fallback path with RequestMessage, then
	// the response will be the ResponseMessage envelope instead of the raw payload.
	outputRaw, headers, err := cl._send(method, reqPath, dataJSON, req.CorrelationID)

	// Unfortunately the http binding has no deterministic result format
	// types of responses:
	//	1. error - based on error result; return error
	//	2. raw data - based on response body; handle as completed
	//  3. completed - based on StatusHeader header field
	//	4. failed  - based on StatusHeader header field
	//  5. with body - completed based on reply content
	//	6. other - assume not completed
	// notifications do not return any data
	// response message return error status
	// requests return optionally a response payload

	// 1. error response
	if err != nil {
		return err
	}

	if f == nil {
		// if the response comes from the hiveot endpoint then it contains a
		// ResponseMessage envelope already. Pass it to the handler of responses.
		resp := transports.ResponseMessage{}
		err = jsoniter.Unmarshal(outputRaw, &resp)
		resp.CorrelationID = req.CorrelationID // just to be sure
		go func() {
			cl.mux.RLock()
			h := cl.appResponseHandler
			cl.mux.RUnlock()
			_ = h(&resp)
		}()
	} else {
		// follow the HTTP Basic specification
		// status header indicate the result to consumers
		statusHeader := ""
		if headers != nil {
			statusHeader = headers.Get(httpserver.StatusHeader)
		}
		// having raw output data is treated as completed
		if outputRaw != nil && len(outputRaw) > 0 {
			err = jsoniter.Unmarshal(outputRaw, &output)
		}
		// 2 and 3. request completed
		// not all client respond with a statusHeader
		if statusHeader == transports.StatusCompleted || statusHeader == "" {
			// the synchronous result of the request contains the output and is completed.
			go func() {
				// Handle this in the background to avoid it being blocked, because
				// the caller will have to read the response channel. (caller doesn't know if the result is
				// immediately or asynchronously)
				resp := transports.NewResponseMessage(
					req.Operation, req.ThingID, req.Name, output, nil, req.CorrelationID)
				// pass a direct response to the application handler
				cl.mux.RLock()
				h := cl.appResponseHandler
				cl.mux.RUnlock()
				_ = h(resp)

			}()
		} else if statusHeader == transports.StatusFailed {
			// body contains the error details return the error
			errTxt := "request failed"
			if output != nil {
				errTxt = fmt.Sprintf("%v", output)
			}
			return errors.New(errTxt)
		} else {
			// status is pending no reason to treat it as a response
		}
	}
	return err
}

// SendResponse Agent sends a response using the hiveot protocol
func (cl *HttpClientConnection) SendResponse(resp *transports.ResponseMessage) error {
	var payload []byte

	// Responses do not have forms in WoT - use the generic server response path
	href := hiveotsseserver.HiveOTPostResponseHRef
	method := http.MethodPost

	payload, _ = jsoniter.Marshal(resp)
	_, _, err := cl._send(method, href, payload, resp.CorrelationID)
	return err
}

// SetConnectHandler set the application handler for connection status updates
func (cl *HttpClientConnection) SetConnectHandler(cb transports.ConnectionHandler) {
	cl.mux.Lock()
	cl.appConnectHandler = cb
	cl.mux.Unlock()
}

// SetRequestHandler set the application handler for incoming requests
func (cl *HttpClientConnection) SetRequestHandler(cb transports.RequestHandler) {
	cl.mux.Lock()
	cl.appRequestHandler = cb
	cl.mux.Unlock()
}

// SetResponseHandler set the application handler for received responses
func (cl *HttpClientConnection) SetResponseHandler(cb transports.ResponseHandler) {
	cl.mux.Lock()
	cl.appResponseHandler = cb
	cl.mux.Unlock()
}

// NewHttpClientConnection creates a new instance of the http-basic protocol binding client.
//
// This uses TD forms to perform an operation.
//
//	fullURL of server to connect to, including the schema
//	clientID to connect as; for logging and ConnectWithPassword. It is ignored if auth token is used.
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	timeout for waiting for response. 0 to use the default.
func NewHttpClientConnection(
	fullURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler, timeout time.Duration) *HttpClientConnection {

	var hostPort string
	urlParts, err := url.Parse(fullURL)
	if err != nil {
		slog.Error("Invalid URL")
	} else {
		hostPort = urlParts.Host
	}

	cl := HttpClientConnection{
		clientID:     clientID,
		caCert:       caCert,
		cid:          "http-" + shortid.MustGenerate(),
		fullURL:      fullURL,
		hostPort:     hostPort,
		protocolType: transports.ProtocolTypeWotHTTPBasic,
		timeout:      timeout,
		getForm:      getForm,
		headers:      make(map[string]string),
	}
	if cl.getForm == nil {
		cl.getForm = cl.GetDefaultForm
	}
	cl.httpClient = tlsclient.NewHttp2TLSClient(caCert, clientCert, timeout)
	return &cl
}
