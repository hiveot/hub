package httpclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/base"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

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
	base.BaseTransportClient

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
	reqPath := tputils.Substitute(methodPath, vars)

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
	req.Header.Set(transports.ConnectionIDHeader, cl.GetConnectionID())
	if requestID != "" {
		req.Header.Set(transports.RequestIDHeader, requestID)
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

// GetDefaultForm return the default http form for the operation
// This uses the hiveot hub generic href
func (cl *HttpTransportClient) GetDefaultForm(op string) td.Form {
	f := td.NewForm(op, transports.GenericHttpHRef)
	return f
}

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

// InvokeAction invokes an action on a thing and wait for the response
func (cl *HttpTransportClient) InvokeAction(dThingID, name string, input any, output any) error {
	return cl.SendRequest(wot.OpInvokeAction, dThingID, name, input, output)
}

// SendOperation sends an operation asynchronously.
//
// If a requestID is supplied then it is assumed a response is expected.
// If a result is received then this is passed to the handle BaseRnR channel
// associated with the request.
// If no response channel is opened then the response will be passed to the
// notification handler.
//
// Note that this ignores the http response body.
//
// This locates the form for the operation and uses it
// Intended as the base for all sends
func (cl *HttpTransportClient) SendOperation(
	operation string, dThingID, name string, data any, requestID string) error {

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
		return err
	} else if href == "" {
		err := fmt.Errorf("SendNotification: Form is missing operation '%s' or href", operation)
		slog.Error(err.Error())
		return err
	}
	if data != nil {
		dataJSON = cl.Marshal(data)
	}
	output, headers, err := cl._send(
		method, href, "", dThingID, name, dataJSON, requestID)
	status := headers.Get(transports.StatusHeader)
	if err != nil {
		return err
	}
	if status == "" {
		// this is not a hiveot server so return the output as the result
	} else if status == transports.StatusCompleted || status == "" {
		// request is completed
		if requestID == "" {
			// no response expected. We're done here.
			return nil
		} else {
			// body contains the output
			handled := cl.BaseRnrChan.HandleResponse(requestID, output, true)
			if !handled {
				// no rpc waiting, pass to the notification handler
				msg := transports.NewThingMessage(operation, dThingID, name, data, "")
				msg.RequestID = requestID
				cl.BaseNotificationHandler(msg)
			}
		}
		// not an rpc, handle as notification
	} else if status == transports.StatusFailed {
		// body contains the error details return the error
		errTxt := "request failed"
		if output != nil {
			errTxt = fmt.Sprintf("%v", output)
		}
		return errors.New(errTxt)
	} else {
		// status is pending nothing to do here
	}
	return err
}

// SendResponse sends the action response message
func (cl *HttpTransportClient) SendResponse(
	dThingID, name string, output any, errResp error, requestID string) {

	stat := transports.RequestStatus{
		ThingID:       dThingID,
		Name:          name,
		RequestID:     requestID,
		Status:        transports.StatusCompleted,
		Output:        output,
		TimeRequested: "",
		TimeEnded:     time.Now().Format(wot.RFC3339Milli),
	}
	if errResp != nil {
		stat.Error = errResp.Error()
	}
	err := cl.SendOperation(
		wot.HTOpUpdateActionStatus, dThingID, name, stat, requestID)
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
	cl.BaseTransportClient = base.BaseTransportClient{
		BaseCaCert:       caCert,
		BaseClientID:     clientID,
		BaseConnectionID: clientID + "." + shortid.MustGenerate(),
		BaseProtocolType: transports.ProtocolTypeHTTPS,
		BaseFullURL:      fullURL,
		BaseHostPort:     urlParts.Host,
		BaseTimeout:      timeout,
		BaseRnrChan:      tputils.NewRnRChan(),
	}

	if getForm == nil {
		getForm = cl.GetDefaultForm
	}
	//
	cl.getForm = getForm
	cl.headers = make(map[string]string)
	cl.httpClient = tlsclient.NewHttp2TLSClient(caCert, clientCert, timeout)
	cl.BaseSendOperation = cl.SendOperation
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
	getForm func(op string) td.Form,
	timeout time.Duration) *HttpTransportClient {

	cl := HttpTransportClient{}
	cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}
