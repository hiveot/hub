package httpclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/base"
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
	"sync/atomic"
	"time"
)

// HttpConsumerClient is the http/2 client for connecting a WoT consumer to a
// WoT server.
// This implements the IAgentTransport interface.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents.
//
// While this client can be used stand-alone, it is intended for use as a base
// for http subprotocols SSE-SC and WSS
// See SsescTransportClient and WSSTransportClient.
type HttpConsumerClient struct {
	base.BaseClient

	// http2 client for posting messages
	httpClient *http.Client
	// authentication bearer token if authenticated
	bearerToken string
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
//	requestID: optional requestID header value
//
// This returns the raw serialized response data, a response message ID, return status code or an error
func (cl *HttpConsumerClient) _send(method string, methodPath string,
	contentType string, thingID string, name string,
	body []byte, requestID string) (
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
	reqPath := tputils.Substitute(methodPath, vars)

	// Caution! a double // in the path causes a 301 and changes post to get
	bodyReader := bytes.NewReader(body)
	serverURL := cl.GetServerURL()
	parts, _ := url.Parse(serverURL)
	parts.Path = reqPath
	fullURL := parts.String()
	//fullURL := parts.cl.GetServerURL() + reqPath
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
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
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set(httpserver.ConnectionIDHeader, cl.GetConnectionID())
	if requestID != "" {
		req.Header.Set(httpserver.RequestIDHeader, requestID)
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
//func (cl *HttpConsumerClient) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//	cl.mux.RLock()
//	defer cl.mux.RUnlock()
//	_ = kp
//	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//	return err
//}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *HttpConsumerClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeEd25519)
//	return k
//}

// GetDefaultForm return the default http form for the operation
// This simply returns nil for anything else than login.
func (cl *HttpConsumerClient) GetDefaultForm(op, thingID, name string) (f td.Form) {
	// login has its own URL as it is unauthenticated
	if op == wot.HTOpLogin {
		href := httpserver.HttpPostLoginPath
		nf := td.NewForm(op, href)
		nf.SetMethodName(http.MethodPost)
		f = nf
	}
	// everything else has no default form, so falls back to hiveot protocol endpoints
	return f
}

// Disconnect from the server
func (cl *HttpConsumerClient) Disconnect() {
	slog.Debug("HttpConsumerClient.Disconnect",
		slog.String("clientID", cl.GetClientID()),
	)

	cl.BaseMux.Lock()
	defer cl.BaseMux.Unlock()
	if cl.BaseIsConnected.Load() {
		cl.httpClient.CloseIdleConnections()
	}
}

func (cl *HttpConsumerClient) GetTlsClient() *http.Client {
	cl.BaseMux.RLock()
	defer cl.BaseMux.RUnlock()
	return cl.httpClient
}

// PubRequest publishes a request message to the server.
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
func (cl *HttpConsumerClient) PubRequest(req transports.RequestMessage) error {

	var dataJSON []byte
	var method string
	var href string
	var output any

	if req.Operation == "" && req.RequestID == "" {
		err := fmt.Errorf("SendMessage: missing both operation and requestID")
		slog.Error(err.Error())
		return err
	}

	// the getForm callback provides the method and URL to invoke for this operation.
	// use the hiveot fallback if not available
	f := cl.BaseGetForm(req.Operation, req.ThingID, req.Name)
	if f != nil {
		method, _ = f.GetMethodName()
		href, _ = f.GetHRef()
	}

	if f == nil {
		// fallback to sending the hiveot request envelope
		// FIXME: this is temporary as agents don't use forms.
		dataJSON = cl.Marshal(req)
		method = http.MethodPost
		href = httpserver.HiveOTPostRequestHRef
	} else if req.Input != nil {
		dataJSON = cl.Marshal(req.Input)
	}
	// use + as wildcard for thingID to avoid a 404
	// while not recommended, it is allowed to subscribe/observe all things
	if req.ThingID == "" {
		req.ThingID = "+"
	}
	// use + as wildcard for affordance name to avoid a 404
	// this should not happen very often but it is allowed
	if req.Name == "" {
		req.Name = "+"
	}

	// substitute URI variables in the path, if any
	vars := map[string]string{
		"thingID":   req.ThingID,
		"name":      req.Name,
		"operation": req.Operation}
	reqPath := tputils.Substitute(href, vars)

	// note, if the request is the hiveot fallback path with RequestMessage, then
	// the response will be the ResponseMessage envelope instead of the raw payload.
	outputRaw, headers, err := cl._send(
		method, reqPath, "", req.ThingID, req.Name, dataJSON, req.RequestID)

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
		// responsemessage envelope already. Pass it to the handler of responses.
		resp := transports.ResponseMessage{}
		err = jsoniter.Unmarshal(outputRaw, &resp)
		resp.RequestID = req.RequestID // just to be sure
		go func() {
			cl.OnResponse(resp)
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
			err = cl.Unmarshal(outputRaw, &output)
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
					req.Operation, req.ThingID, req.Name, output, nil, req.RequestID)
				// pass a response to the sync or asyncn handler of responses
				cl.OnResponse(resp)
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

func (cl *HttpConsumerClient) Init(
	fullURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	timeout time.Duration) {
	baseHostPort := ""
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
	urlParts, err := url.Parse(fullURL)
	if err != nil {
		slog.Error("Invalid URL")
	} else {
		baseHostPort = urlParts.Host
	}
	cl.BaseClient = base.BaseClient{
		BaseCaCert:       caCert,
		BaseClientID:     clientID,
		BaseConnectionID: clientID + "." + shortid.MustGenerate(),
		BaseProtocolType: transports.ProtocolTypeHTTPS,
		BaseFullURL:      fullURL,
		BaseHostPort:     baseHostPort,
		BaseTimeout:      timeout,
		BaseRnrChan:      base.NewRnRChan(),
	}

	if getForm == nil {
		getForm = cl.GetDefaultForm
	}
	//
	cl.BaseGetForm = getForm
	cl.headers = make(map[string]string)
	cl.httpClient = tlsclient.NewHttp2TLSClient(caCert, clientCert, timeout)
	cl.BasePubRequest = cl.PubRequest
}

// NewHttpTransportClient creates a new instance of the http binding client
//
// This uses TD forms to perform an operation.
//
//	fullURL of server to connect to, including the schema
//	clientID to connect as; for logging and ConnectWithPassword. It is ignored if auth token is used.
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	timeout for waiting for response. 0 to use the default.
func NewHttpTransportClient(
	fullURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	timeout time.Duration) *HttpConsumerClient {

	cl := HttpConsumerClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}
