package httpsseclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/servers/hiveotsseserver"
	"github.com/hiveot/hub/messaging/servers/httpserver"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/messaging/tputils/tlsclient"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
)

// HiveotSseClient is the http/2 client for connecting a WoT client to a
// WoT server using the HiveOT http and sse protocol.
// This implements the IClientConnection interface.
//
// This can be used by both consumers and agents.
// This is intended to be used together with an SSE return channel.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents. The form contains the
// hiveot RequestMessage and ResponseMessage endpoints. If no form is available
// then use the default hiveot endpoints that are defined with this protocol binding.
type HiveotSseClient struct {

	// handler for requests send by clients
	appConnectHandler messaging.ConnectionHandler

	// handler for notifications sent by agents
	appNotificationHandler messaging.NotificationHandler
	// handler for requests send by clients
	appRequestHandler messaging.RequestHandler
	// handler for responses sent by agents
	appResponseHandler messaging.ResponseHandler

	//clientID string
	// Connection information such as clientID, cid, address, protocol etc
	cinfo messaging.ConnectionInfo

	// CA certificate to verify the server with
	//caCert *x509.Certificate

	// This client's connection ID
	//cid string

	// The full server's base URL sse://host:port/path
	//fullURL string
	// The server host:port
	hostPort string

	// the sse connection path
	ssePath              string
	sseRetryOnDisconnect atomic.Bool
	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	isConnected atomic.Bool

	// RPC timeout
	//timeout time.Duration
	// protected operations
	mux sync.RWMutex
	// http2 client for posting messages
	httpClient *http.Client
	// authentication bearer token if authenticated
	bearerToken string

	// getForm obtains the form for sending a request or notification
	// if nil, then the hiveot protocol envelope and URL are used as fallback
	getForm messaging.GetFormHandler

	// custom headers to include in each request
	headers map[string]string

	lastError atomic.Pointer[error]
}

// Send a HTTPS method and return the http response.
//
// If token authentication is enabled then add the bearer token to the header
//
//	method: GET, PUT, POST, ...
//	reqPath: path to invoke
//	contentType of the payload or "" for default (application/json)
//	thingID optional path URI variable
//	name optional path URI variable containing affordance name
//	body contains the serialized payload
//	correlationID: optional correlationID header value
//
// This returns the raw serialized response data, a response message ID, return status code or an error
func (cc *HiveotSseClient) Send(
	method string, methodPath string, body []byte) (
	resp []byte, headers http.Header, code int, err error) {

	if cc.httpClient == nil {
		err = fmt.Errorf("Send: '%s'. Client is not started", methodPath)
		return nil, nil, 0, err
	}
	// Caution! a double // in the path causes a 301 and changes post to get
	bodyReader := bytes.NewReader(body)
	serverURL := cc.cinfo.ConnectURL
	parts, _ := url.Parse(serverURL)
	parts.Scheme = "https" // the sse path has the sse scheme
	parts.Path = methodPath
	fullURL := parts.String()
	//fullURL := parts.cc.GetServerURL() + reqPath
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		err = fmt.Errorf("Send %s %s failed: %w", method, fullURL, err)
		return nil, nil, 0, err
	}

	// set the origin header to the intended destination without the path
	//parts, err := url.Parse(fullURL)
	origin := fmt.Sprintf("https://%s", parts.Host)
	req.Header.Set("Origin", origin)

	// set the authorization header
	if cc.bearerToken != "" {
		req.Header.Add("Authorization", "bearer "+cc.bearerToken)
	}

	// set other headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(httpserver.ConnectionIDHeader, cc.cinfo.ConnectionID)
	//if correlationID != "" {
	//	req.Header.Set(httpserver.CorrelationIDHeader, correlationID)
	//}
	for k, v := range cc.headers {
		req.Header.Set(k, v)
	}

	httpResp, err := cc.httpClient.Do(req)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, 0, err
	}

	respBody, err := io.ReadAll(httpResp.Body)
	// response body MUST be closed
	_ = httpResp.Body.Close()
	httpStatus := httpResp.StatusCode

	if httpStatus == 401 {
		err = fmt.Errorf("%s", httpResp.Status)
	} else if httpStatus >= 400 && httpStatus < 500 {
		if respBody != nil {
			err = fmt.Errorf("%d (%s): %s", httpResp.StatusCode, httpResp.Status, respBody)
		} else {
			err = fmt.Errorf("%d (%s): Request failed", httpResp.StatusCode, httpResp.Status)
		}
	} else if httpStatus >= 500 {
		err = fmt.Errorf("Error %d (%s): %s", httpStatus, httpResp.Status, respBody)
		slog.Error("Send returned internal server error", "reqPath", methodPath, "err", err.Error())
	} else if err != nil {
		err = fmt.Errorf("Send: Error %s %s: %w", method, methodPath, err)
	}
	return respBody, httpResp.Header, httpStatus, err
}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	kp is the key-pair used to the certificate validation
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
//
//	func (cl *HiveotSseClient) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//		cl.mux.RLock()
//		defer cl.mux.RUnlock()
//		_ = kp
//		cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//		return err
//	}

// ConnectWithToken sets the bearer token to use with requests and establishes
// an SSE connection.
// If a connection exists it is closed first.
func (cc *HiveotSseClient) ConnectWithToken(token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cc.Disconnect()

	err := cc.SetBearerToken(token)
	if err != nil {
		return err
	}
	// connectSSE will set 'isConnected' on success
	err = cc.ConnectSSE(token)
	if err != nil {
		cc.isConnected.Store(false)
		return err
	}
	return err
}

// Disconnect from the server
func (cc *HiveotSseClient) Disconnect() {
	slog.Debug("HiveotSseClient.Disconnect",
		slog.String("clientID", cc.cinfo.ClientID),
	)
	cc.mux.Lock()
	cb := cc.sseCancelFn
	cc.sseCancelFn = nil
	cc.mux.Unlock()

	// the connection status will update, if changed, through the sse callback
	if cb != nil {
		cb()
	}

	cc.mux.Lock()
	defer cc.mux.Unlock()
	if cc.isConnected.Load() {
		cc.httpClient.CloseIdleConnections()
	}
}

// GetConnectionInfo returns the client's connection details
func (cc *HiveotSseClient) GetConnectionInfo() messaging.ConnectionInfo {
	return cc.cinfo
}

// GetDefaultForm return the default http form for the operation
// This simply returns nil for anything else than login, logout, ping or refresh.
func (cc *HiveotSseClient) GetDefaultForm(op, thingID, name string) (f *td.Form) {
	// login has its own URL as it is unauthenticated
	if op == wot.HTOpPing {
		href := httpserver.HttpGetPingPath
		nf := td.NewForm(op, href)
		nf.SetMethodName(http.MethodGet)
		f = &nf
		//} else if op == wot.HTOpLogin {
		//	href := httpserver.HttpPostLoginPath
		//	nf := td.NewForm(op, href)
		//	nf.SetMethodName(http.MethodPost)
		//	f = &nf
		//} else if op == wot.HTOpLogout {
		//	href := httpserver.HttpPostLogoutPath
		//	nf := td.NewForm(op, href)
		//	nf.SetMethodName(http.MethodPost)
		//	f = &nf
		//} else if op == wot.HTOpRefresh {
		//	href := httpserver.HttpPostRefreshPath
		//	nf := td.NewForm(op, href)
		//	nf.SetMethodName(http.MethodPost)
		//	f = &nf
	}
	// everything else has no default form, so falls back to hiveot protocol endpoints
	return f
}

func (cc *HiveotSseClient) GetTlsClient() *http.Client {
	cc.mux.RLock()
	defer cc.mux.RUnlock()
	return cc.httpClient
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cc *HiveotSseClient) IsConnected() bool {
	return cc.isConnected.Load()
}

// LoginWithForm invokes login using a form - temporary helper
// intended for testing a connection to a web server.
//
// This sets the bearer token for further requests. It requires the server
// to set a session cookie in response to the login.
//func (cl *HiveotSseClient) LoginWithForm(
//	password string) (newToken string, err error) {
//
//	// FIXME: does this client need a cookie jar???
//	formMock := url.Values{}
//	formMock.Add("loginID", cl.GetClientID())
//	formMock.Add("password", password)
//
//	var loginHRef string
//	f := cl.getForm(wot.HTOpLoginWithForm, "", "")
//	if f != nil {
//		loginHRef, _ = f.GetHRef()
//	}
//	loginURL, err := url.Parse(loginHRef)
//	if err != nil {
//		return "", err
//	}
//	if loginURL.Host == "" {
//		loginHRef = cl.fullURL + loginHRef
//	}
//
//	//PostForm should return a cookie that should be used in the http connection
//	if loginHRef == "" {
//		return "", errors.New("Login path not found in getForm")
//	}
//	resp, err := cl.httpClient.PostForm(loginHRef, formMock)
//	if err != nil {
//		return "", err
//	}
//
//	// get the session token from the cookie
//	//cookie := resp.Request.Header.Get("cookie")
//	cookie := resp.Header.Get("cookie")
//	kvList := strings.Split(cookie, ",")
//
//	for _, kv := range kvList {
//		kvParts := strings.SplitN(kv, "=", 2)
//		if kvParts[0] == "session" {
//			cl.bearerToken = kvParts[1]
//			break
//		}
//	}
//	if cl.bearerToken == "" {
//		slog.Error("No session cookie was received on login")
//	}
//	return cl.bearerToken, err
//}

// LoginWithPassword posts a login request to the TLS server using a login ID and
// password and obtain an auth token for use with SetBearerToken.
//
// FIXME: use a WoT standardized auth method
//
// If the connection fails then any existing connection is cancelled.
//func (cl *HiveotSseClient) LoginWithPassword(password string) (newToken string, err error) {
//
//	slog.Info("ConnectWithPassword",
//		"clientID", cl.GetClientID(), "connectionID", cl.GetConnectionID())
//
//	// FIXME: figure out how a standard login method is used to obtain an auth token
//	loginMessage := map[string]string{
//		"login":    cl.GetClientID(),
//		"password": password,
//	}
//	f := cl.getForm(wot.HTOpLogin, "", "")
//	if f == nil {
//		err = fmt.Errorf("missing form for login operation")
//		slog.Error(err.Error())
//		return "", err
//	}
//	method, _ := f.GetMethodName()
//	href, _ := f.GetHRef()
//
//	dataJSON, _ := jsoniter.Marshal(loginMessage)
//	outputRaw, _, _, err := cl.Send(method, href, dataJSON)
//
//	if err == nil {
//		err = jsoniter.Unmarshal(outputRaw, &newToken)
//	}
//	// store the bearer token further requests
//	// when login fails this clears the existing token. Someone else
//	// logging in cannot continue on a previously valid token.
//	cl.mux.Lock()
//	cl.bearerToken = newToken
//	cl.mux.Unlock()
//	//cl.BaseIsConnected.Store(true)
//	if err != nil {
//		slog.Warn("connectWithPassword failed: " + err.Error())
//	}
//
//	return newToken, err
//}

// SendRequest sends a request message and passes the result as a response
// to the registered response handler.
//
// This locates the form for the operation using 'getForm' and uses the result
// to determine the URL to publish the request to and if the hiveot RequestMessage
// envelope is used.
//
// If no form is found then fall back to the hiveot default paths.
// The request input, if any, is json encoded into the body of the request.
// This does not use a RequestMessage envelope to remain http-basic compatible.
//
// The response follows the http-basic specification:
// * code 200: completed; body is output
// * code 201: pending; body is http action status message
// * code 40x: failed ; body is error payload, if present
// * code 50x: failed ; body is error payload, if present
//
// The result is passed to the BaseRnR channel associated with the request just
// like it is done with an async response.
func (cc *HiveotSseClient) SendRequest(req *messaging.RequestMessage) error {

	var inputJSON string
	var method string
	var href string
	var thingID = req.ThingID
	var name = req.Name
	var useRequestEnvelope = true

	if req.Operation == "" && req.CorrelationID == "" {
		err := fmt.Errorf("SendMessage: missing both operation and correlationID")
		slog.Error(err.Error())
		return err
	}

	// the getForm callback provides the method and URL to invoke for this operation.
	// use the hiveot fallback if not available
	// If a form is provided and it doesn't use the hiveot subprotocol then fall
	// back to invoking using http basic using the form href.
	f := cc.getForm(req.Operation, req.ThingID, req.Name)
	if f != nil {
		method, _ = f.GetMethodName()
		href = f.GetHRef()
		subprotocol, _ := f.GetSubprotocol()
		// the SSE-Hiveot subprotocol sends the RequestMessage envelope as payload
		useRequestEnvelope = subprotocol == hiveotsseserver.HiveotSSESchema
	}

	if f == nil {
		// fall back to the 'well known' hiveot request URL
		method = http.MethodPost
		href = hiveotsseserver.DefaultHiveotPostRequestHRef
	}
	// hiveot uses the requestmessage itself as payload on hiveot subprotocol
	if useRequestEnvelope {
		inputJSON, _ = jsoniter.MarshalToString(req)
	} else if req.Input != nil {
		// while http-basic uses the input as payload
		inputJSON, _ = jsoniter.MarshalToString(req.Input)
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
	// intended for use with http-basic forms.
	vars := map[string]string{
		"thingID":   thingID,
		"name":      name,
		"operation": req.Operation}
	reqPath := tputils.Substitute(href, vars)
	outputRaw, headers, code, err := cc.Send(method, reqPath, []byte(inputJSON))
	_ = headers

	// 1. error response
	if err != nil {
		return err
	}
	// follow the HTTP Basic specification
	if code == http.StatusOK {
		resp := req.CreateResponse(nil, nil)
		// unmarshal output. This is either the json encoded output or the ResponseMessage envelope
		if outputRaw == nil || len(outputRaw) == 0 {
			// nothing to unmarshal
		} else if useRequestEnvelope {
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp)
		} else {
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp.Value)
		}
		if err != nil {
			resp.Error = err.Error()
		}

		// pass a direct response to the application handler
		cc.mux.RLock()
		h := cc.appResponseHandler
		cc.mux.RUnlock()
		go func() {
			_ = h(resp)
		}()
	} else if code > 200 && code < 300 {
		// httpbasic servers/things might respond with 201 for pending as per spec
		// this is a notification.
		var notif *messaging.NotificationMessage
		if outputRaw == nil || len(outputRaw) == 0 {
			// no response yet. do not send process a notification
		} else if useRequestEnvelope {
			// hiveot uses NotificationMessage envelopes
			notif = req.CreateNotification()
			err = jsoniter.Unmarshal(outputRaw, &notif)
		} else {
			// output is http basic actionstatus
			tmp := hiveotsseserver.HttpActionStatusMessage{}
			err = jsoniter.Unmarshal(outputRaw, &tmp)
			notif.Data = tmp
		}

		// pass a direct response to the application handler
		if notif != nil {
			cc.mux.RLock()
			h := cc.appNotificationHandler
			cc.mux.RUnlock()
			go func() {
				h(notif)
			}()
		}
	} else {
		// unknown response, create an error response
		resp := req.CreateResponse(nil, nil)
		// unmarshal output. This is either the json encoded output or the ResponseMessage envelope
		if outputRaw == nil {
			// nothing to unmarshal
		} else if useRequestEnvelope {
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp)
		} else {
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp.Value)
		}
		httpProblemDetail := map[string]string{}
		if outputRaw != nil && len(outputRaw) > 0 {
			err = jsoniter.Unmarshal(outputRaw, &httpProblemDetail)
			resp.Error = httpProblemDetail["title"]
			resp.Value = httpProblemDetail["detail"]
		} else {
			resp.Error = "request failed"
		}

		// pass a direct response to the application handler
		cc.mux.RLock()
		h := cc.appResponseHandler
		cc.mux.RUnlock()
		go func() {
			_ = h(resp)
		}()
	}
	return err
}

// SendResponse Agent posts a response using the hiveot protocol.
// This passes the response as-is as a payload.
//
// This posts the JSON-encoded ResponseMessage on the well-known hiveot response href.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
func (cc *HiveotSseClient) SendResponse(resp *messaging.ResponseMessage) error {
	outputJSON, _ := jsoniter.MarshalToString(resp)
	_, _, _, err := cc.Send(http.MethodPost,
		hiveotsseserver.DefaultHiveotPostResponseHRef, []byte(outputJSON))
	return err
}

// SendNotification Agent posts a notification using the hiveot protocol.
// This passes the notification as-is as a payload.
//
// This posts the JSON-encoded NotificationMessage on the well-known hiveot notification href.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
//
// This returns an error if the notification could not be delivered to the server
func (cc *HiveotSseClient) SendNotification(msg *messaging.NotificationMessage) error {
	outputJSON, _ := jsoniter.MarshalToString(msg)
	_, _, _, err := cc.Send(http.MethodPost,
		hiveotsseserver.DefaultHiveotPostNotificationHRef, []byte(outputJSON))
	if err != nil {
		slog.Warn("SendNotification failed",
			"clientID", cc.cinfo.ClientID,
			"err", err.Error())
	}
	return err
}

// SetBearerToken sets the authentication bearer token to authenticate http requests.
func (cc *HiveotSseClient) SetBearerToken(token string) error {
	cc.mux.Lock()
	cc.bearerToken = token
	cc.mux.Unlock()
	return nil
}

// SetConnectHandler set the application handler for connection status updates
func (cc *HiveotSseClient) SetConnectHandler(cb messaging.ConnectionHandler) {
	cc.mux.Lock()
	cc.appConnectHandler = cb
	cc.mux.Unlock()
}

// SetNotificationHandler set the application handler for received notifications
func (cc *HiveotSseClient) SetNotificationHandler(cb messaging.NotificationHandler) {
	cc.mux.Lock()
	cc.appNotificationHandler = cb
	cc.mux.Unlock()
}

// SetRequestHandler set the application handler for incoming requests
func (cc *HiveotSseClient) SetRequestHandler(cb messaging.RequestHandler) {
	cc.mux.Lock()
	cc.appRequestHandler = cb
	cc.mux.Unlock()
}

// SetResponseHandler set the application handler for received responses
func (cc *HiveotSseClient) SetResponseHandler(cb messaging.ResponseHandler) {
	cc.mux.Lock()
	cc.appResponseHandler = cb
	cc.mux.Unlock()
}

// NewHiveotSseClient creates a new instance of the http-basic protocol binding client.
//
// This uses TD forms to perform an operation.
//
//	sseURL of the http and sse server to connect to, including the schema
//	clientID to identify as. Must match the auth token
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	timeout for waiting for response. 0 to use the default.
func NewHiveotSseClient(
	sseURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm messaging.GetFormHandler, timeout time.Duration) *HiveotSseClient {

	urlParts, err := url.Parse(sseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host
	ssePath := urlParts.Path

	cinfo := messaging.ConnectionInfo{
		CaCert:       caCert,
		ClientID:     clientID,
		ConnectionID: "http-" + shortid.MustGenerate(),
		ConnectURL:   sseURL,
		ProtocolType: messaging.ProtocolTypeHiveotSSE,
		Timeout:      timeout,
	}
	cl := HiveotSseClient{
		cinfo: cinfo,
		//clientID: clientID,
		//caCert:   caCert,
		//cid:      "http-" + shortid.MustGenerate(),
		//fullURL:  sseURL,
		ssePath:  ssePath,
		hostPort: hostPort,
		//timeout:  timeout,
		getForm: getForm,
		headers: make(map[string]string),
	}
	if cl.getForm == nil {
		cl.getForm = cl.GetDefaultForm
	}
	cl.httpClient = tlsclient.NewHttp2TLSClient(caCert, clientCert, timeout)
	return &cl
}
