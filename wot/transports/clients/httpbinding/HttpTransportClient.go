package httpbinding

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/hiveot/hub/wot/transports"
	"github.com/hiveot/hub/wot/transports/clients/base"
	"github.com/hiveot/hub/wot/transports/utils"
	"github.com/hiveot/hub/wot/transports/utils/tlsclient"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

// HttpTransportClient is the http/2 client for performing operations on one or more Things.
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
type HttpTransportClient struct {
	base.TransportClient

	// getForm obtains the form for sending a request or notification
	getForm func(op string) td.Form

	// http2 client for posting messages
	httpClient *http.Client
	// authentication bearer token if authenticated
	bearerToken string
	// custom headers to include in each request
	headers map[string]string

	lastError atomic.Pointer[error]
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
//	requestID: optional requestID header value
//
// This returns the serialized response data, a response message ID, return status code or an error
func (cl *HttpTransportClient) _send(method string, methodPath string,
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
	req.Header.Set(ConnectionIDHeader, cl.GetConnectionID())
	if requestID != "" {
		req.Header.Set(CorrelationIDHeader, requestID)
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
//func (cl *HttpTransportClient) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//	cl.mux.RLock()
//	defer cl.mux.RUnlock()
//	_ = kp
//	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//	return err
//}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *HttpTransportClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeEd25519)
//	return k
//}

// Disconnect from the server
func (cl *HttpTransportClient) Disconnect() {
	slog.Debug("HttpTransportClient.Disconnect",
		slog.String("clientID", cl.GetClientID()),
	)

	cl.BaseMux.Lock()
	defer cl.BaseMux.Unlock()
	if cl.BaseIsConnected.Load() {
		cl.httpClient.CloseIdleConnections()
	}
}

func (cl *HttpTransportClient) GetTlsClient() *http.Client {
	cl.BaseMux.RLock()
	defer cl.BaseMux.RUnlock()
	return cl.httpClient
}

// SendError sends the notification without a reply.
func (cl *HttpTransportClient) SendError(
	dThingID, name string, err error, requestID string) {

	stat := transports.RequestStatus{
		ThingID:       dThingID,
		Name:          name,
		RequestID:     requestID,
		Status:        transports.StatusCompleted,
		Error:         err.Error(),
		TimeRequested: "",
		TimeEnded:     time.Now().Format(wot.RFC3339Milli),
	}
	_, _, _ = cl.SendOperation(
		wot.HTOpPublishError, dThingID, name, stat, requestID)
}

// SendOperation sends an operation and returns the result
//
// This locates the form for the operation and uses it
// Intended as the base for all sends
func (cl *HttpTransportClient) SendOperation(operation string,
	dThingID, name string, data any, requestID string) ([]byte, http.Header, error) {

	var dataJSON []byte
	var method string
	var href string
	f := cl.getForm(operation)
	if f != nil {
		method, _ = f.GetMethodName()
		href, _ = f.GetHRef()
	}
	if method == "" {
		method = http.MethodGet
	}

	if operation == "" {
		err := fmt.Errorf("SendOperation: missing operation")
		slog.Error(err.Error())
		return nil, nil, err
	} else if href == "" {
		err := fmt.Errorf("SendNotification: Form is missing operation '%s' or href", operation)
		slog.Error(err.Error())
		return nil, nil, err
	}
	if data != nil {
		dataJSON = cl.Marshal(data)
	}
	output, headers, err := cl._send(
		method, href, "", dThingID, name, dataJSON, requestID)
	return output, headers, err
}

// SendNotification sends the notification without a reply.
func (cl *HttpTransportClient) SendNotification(
	operation string, dThingID, name string, data any) error {

	_, _, err := cl.SendOperation(operation, dThingID, name, data, "")
	return err
}

// SendRequest sends an operation and returns the result.
//
// Since the http binding doesn't have a return channel, this only works with
// operations that return their result as http response.
func (cl *HttpTransportClient) SendRequest(operation string,
	dThingID, name string, input interface{}, output interface{}) error {

	// Without a return channel there is no waiting for a result
	raw, _, err := cl.SendOperation(operation, dThingID, name, input, "")

	// in case an immediate result is received then unmarshal it and
	// call it done.
	if raw != nil && output != nil {
		err = cl.Unmarshal(raw, output)
	}
	return err
}

// SendResponse sends the action response message
func (cl *HttpTransportClient) SendResponse(
	dThingID, name string, output any, requestID string) {

	stat := transports.RequestStatus{
		ThingID:       dThingID,
		Name:          name,
		RequestID:     requestID,
		Status:        transports.StatusCompleted,
		Output:        output,
		TimeRequested: "",
		TimeEnded:     time.Now().Format(wot.RFC3339Milli),
	}
	_, _, err := cl.SendOperation(wot.HTOpActionStatus, dThingID, name, stat, requestID)
	_ = err
}

func (cl *HttpTransportClient) Init(
	fullURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm func(op string) td.Form,
	timeout time.Duration) {

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
	cl.TransportClient = base.TransportClient{
		BaseCaCert:       caCert,
		BaseClientID:     clientID,
		BaseConnectionID: clientID + "." + shortid.MustGenerate(),
		BaseProtocolType: transports.ProtocolTypeHTTP,
		BaseFullURL:      fullURL,
		BaseHostPort:     urlParts.Host,
		BaseTimeout:      timeout,
		BaseRnrChan:      utils.NewRnRChan(),
	}
	//
	cl.getForm = getForm
	cl.headers = make(map[string]string)
	cl.httpClient = tlsclient.NewHttp2TLSClient(caCert, clientCert, timeout)
	cl.BaseSendNotification = cl.SendNotification
}

// NewHttpTransportClient creates a new instance of the http binding client
//
// This uses TD forms to perform an operation.
//
//	fullURL of server to connect to, including the schema
//	clientID to connect as; for logging and ConnectWithPassword. It is ignored if auth token is used.
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation
//	timeout for waiting for response. 0 to use the default.
func NewHttpTransportClient(
	fullURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm func(op string) td.Form,
	timeout time.Duration) *HttpTransportClient {

	cl := HttpTransportClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}
