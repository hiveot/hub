package sseclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Paths used by this protocol binding - SYNC with HttpSSEClient.ts
//
// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
// The hub client will need the TD (ConsumedThing) to determine the paths.
const (
	// deprecated, use forms
	ReadAllEventsPath = "/digitwin/events/{thingID}"

	ConnectSSEPath = "/ssesc"
	// deprecated, use forms
	PostInvokeActionPath = "/digitwin/actions/{thingID}/{name}"
	// deprecated, use forms
	PostObservePropertiesPath = "/ssesc/digitwin/observe/{thingID}/{name}"
	// deprecated, use forms
	PostSubscribeEventPath = "/ssesc/digitwin/subscribe/{thingID}/{name}"
	// deprecated, use forms
	PostUnobservePropertyPath = "/ssesc/digitwin/unobserve/{thingID}/{name}"
	// deprecated, use forms
	PostUnsubscribeEventPath = "/ssesc/digitwin/unsubscribe/{thingID}/{name}"
	// deprecated, use forms
	PostWritePropertyPath = "/digitwin/properties/{thingID}/{name}"

	// Form paths for accessing TD directory
	// deprecated, use forms
	GetThingPath = "/digitwin/directory/{thingID}"
	// deprecated, use forms
	GetAllThingsPath = "/digitwin/directory" // query param offset=, limit=

	// deprecated, use forms
	PostAgentPublishEventPath = "/agent/event/{thingID}/{name}"
	// deprecated, use forms
	PostAgentPublishProgressPath = "/agent/progress"
	// deprecated, use forms
	PostAgentUpdatePropertyPath           = "/agent/property/{thingID}/{name}"
	PostAgentUpdateMultiplePropertiesPath = "/agent/properties/{thingID}"
	// deprecated, use forms
	PostAgentUpdateTDDPath = "/agent/tdd/{thingID}"

	// deprecated authn service - use the generated constants or forms
	PostLoginPath = "/authn/login"
	// deprecated authn service - use the generated constants
	PostLogoutPath = "/authn/logout"
	// deprecated authn service - use the generated constants
	PostRefreshPath = "/authn/refresh"
)

// HttpSSEClient manages the connection to the hub server using http/2.
// This implements the IConsumerClient interface.
// This client creates two http/2 connections, one for posting messages and
// one for a sse connection to establish a return channel.
//
// This clients implements the REST API supported by the digitwin runtime services,
// specifically the directory, inbox, outbox, authn
type HttpSSEClient struct {
	hostPort string
	ssePath  string
	caCert   *x509.Certificate
	clientID string
	// the cid header field to correlate request and return channel connections
	// the server prefixes it with clientID to guarantee uniqueness.
	cid string

	timeout time.Duration // request timeout
	// '_' variables are mux protected
	mux         sync.RWMutex
	sseCancelFn context.CancelFunc

	// sseClient is the TLS client with the SSE connection
	//sseClient *tlsclient.TLSClient
	// tlsClient is the TLS client used for posting events
	tlsClient *tlsclient.TLSClient

	isConnected atomic.Bool
	lastError   atomic.Pointer[error]

	subscriptions  map[string]bool
	connectHandler func(connected bool, err error)
	// client side handler that receives messages for consumers
	messageHandler hubclient.MessageHandler
	// map of requestID to delivery status update channel
	requestHandler hubclient.RequestHandler
	// map of requestID to delivery status update channel
	correlData map[string]chan *hubclient.RequestStatus
}

// helper to establish an sse connection using the given bearer token
func (cl *HttpSSEClient) connectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("Missing SSE path")
	}
	// create a second client to establish the sse connection if a path is set
	sseURL := fmt.Sprintf("https://%s%s", cl.hostPort, cl.ssePath)
	cl.sseCancelFn, err = ConnectSSE(
		cl.clientID, cl.cid,
		sseURL, token, cl.caCert,
		cl.handleSSEConnect, cl.handleSSEEvent)

	return err
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

// ConnectWithLoginForm invokes login using a form - temporary helper
// intended for testing a connection with a web server
func (cl *HttpSSEClient) ConnectWithLoginForm(password string) error {
	formMock := url.Values{}
	formMock.Add("loginID", cl.clientID)
	formMock.Add("password", password)
	fullURL := fmt.Sprintf("https://%s/login", cl.hostPort)

	//PostForm should return a cookie that should be used in the sse connection
	resp, err := cl.tlsClient.GetHttpClient().PostForm(fullURL, formMock)
	if err == nil {
		// get the session token from the cookie
		cookie := resp.Request.Header.Get("cookie")
		kvList := strings.Split(cookie, ",")
		token := ""
		for _, kv := range kvList {
			kvParts := strings.SplitN(kv, "=", 2)
			if kvParts[0] == "session" {
				token = kvParts[1]
				break
			}
		}
		//token := resp.Header.Get("token")
		err = cl.connectSSE(token)
	}
	return err
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
func (cl *HttpSSEClient) ConnectWithPassword(password string) (newToken string, err error) {
	//cl.mux.Lock()
	//// remove existing connection
	//if cl.tlsClient != nil {
	//	cl.tlsClient.Close()
	//}
	//cl.tlsClient = tlsclient.NewTLSClient(
	//	cl.hostPort, nil, cl.caCert, cl.timeout, cl.cid)
	loginURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostLoginPath)
	//cl.mux.Unlock()

	slog.Info("ConnectWithPassword", "clientID", cl.clientID, "cid", cl.cid)
	loginMessage := authn.UserLoginArgs{
		ClientID: cl.GetClientID(),
		Password: password,
	}
	argsJSON, _ := json.Marshal(loginMessage)
	resp, _, statusCode, _, err2 := cl.tlsClient.Invoke(
		"POST", loginURL, argsJSON, "", nil)
	if err2 != nil {
		err = fmt.Errorf("%d: Login failed: %s", statusCode, err2)
		return "", err
	}
	token := ""
	err = cl.Unmarshal(resp, &token)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: Login to %s has unexpected response message: %s", loginURL, err)
		return "", err
	}
	// store the bearer token further requests
	cl.tlsClient.SetAuthToken(token)

	// create a second client to establish the sse connection if a path is set
	if cl.ssePath != "" {
		err = cl.connectSSE(token)
	}

	return token, err
}

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// and obtain a new auth token.
//
//	jwtToken is the token previously obtained with login or refresh.
func (cl *HttpSSEClient) ConnectWithToken(token string) (newToken string, err error) {
	//cl.mux.Lock()
	//if cl.tlsClient != nil {
	//	cl.tlsClient.Close()
	//}
	//slog.Info("ConnectWithToken (to hub)", "clientID", cl.clientID, "cid", cl.cid)
	//cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, nil, cl.caCert, cl.timeout, cl.cid)
	////cl._status.HubURL = fmt.Sprintf("https://%s", cl.hostPort)
	//cl.mux.Unlock()
	cl.tlsClient.SetAuthToken(token)

	// Refresh the auth token and verify the connection works.
	newToken, err = cl.RefreshToken(token)
	if err != nil {
		return "", err
	}
	cl.tlsClient.SetAuthToken(newToken)

	// establish the sse connection if a path is set
	if cl.ssePath != "" {
		err = cl.connectSSE(token)
	}

	return newToken, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (cl *HttpSSEClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeEd25519)
	return k
}

// Disconnect from the server
func (cl *HttpSSEClient) Disconnect() {
	slog.Debug("HttpSSEClient.Disconnect",
		slog.String("clientID", cl.clientID),
		slog.String("cid", cl.cid),
	)

	cl.mux.Lock()
	sseCancelFn := cl.sseCancelFn
	cl.sseCancelFn = nil
	tlsClient := cl.tlsClient
	//cl.tlsClient = nil
	cl.mux.Unlock()

	// the connection status will update, if changed, through the sse callback
	if sseCancelFn != nil {
		sseCancelFn()
	}
	if tlsClient != nil {
		tlsClient.Close()
	}
}

// GetClientID returns the client's account ID
func (cl *HttpSSEClient) GetClientID() string {
	return cl.clientID
}

// GetCID returns the client's connection ID
func (cl *HttpSSEClient) GetCID() string {
	return cl.cid
}

func (cl *HttpSSEClient) GetConnectionStatus() (bool, string, error) {
	var lastErr error = nil
	// lastError is stored as pointer because atomic.Value cannot switch between error and nil type
	if cl.lastError.Load() != nil {
		lastErrPtr := cl.lastError.Load()
		lastErr = *lastErrPtr
	}
	return cl.isConnected.Load(), cl.cid, lastErr
}

// GetProtocolType returns the type of protocol this client supports
func (cl *HttpSSEClient) GetProtocolType() string {
	return "https"
}

// GetHubURL returns the schema://address:port of the hub connection
func (cl *HttpSSEClient) GetHubURL() string {
	hubURL := fmt.Sprintf("https://%s", cl.hostPort)
	return hubURL
}

func (cl *HttpSSEClient) GetTlsClient() *tlsclient.TLSClient {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *HttpSSEClient) handleSSEConnect(connected bool, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	slog.Debug("handleSSEConnect",
		slog.String("clientID", cl.clientID),
		slog.String("cid", cl.cid),
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

// InvokeAction publishes an action message and waits for an answer or until timeout
// An error is returned if delivery failed or succeeded but the action itself failed
func (cl *HttpSSEClient) InvokeAction(
	dThingID string, name string, input interface{}, output interface{}, requestID string) (
	stat hubclient.RequestStatus) {

	slog.Info("InvokeAction",
		slog.String("clientID (me)", cl.clientID),
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
		slog.String("cid", cl.cid),
	)
	// FIXME: use TD form for this action
	// FIXME: track message-ID's using headers instead of message envelope
	stat = cl.PubMessage(http.MethodPost, PostInvokeActionPath, dThingID, name, input, output, requestID)

	return stat
}

// InvokeOperation gets the path from the form and makes the call
func (cl *HttpSSEClient) InvokeOperation(
	op tdd.Form, dThingID, name string, input interface{}, output interface{}) error {

	urlPath, _ := op.GetHRef()
	if urlPath == "" {
		return fmt.Errorf("InvokeOperation. Missing href in form. dThingID='%s', name='%s'", dThingID, name)
	}
	u, err := url.Parse(urlPath)
	if err != nil {
		return err
	}
	// TODO: how to get result of actions here?
	methodName, _ := op.GetMethodName()
	stat := cl.PubMessage(methodName, u.Path, dThingID, name, input, output, "")
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return err
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *HttpSSEClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Logout from the server and end the session
func (cl *HttpSSEClient) Logout() error {
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostLogoutPath)
	//_, err := cl.Invoke("POST", serverURL, http.NoBody, nil)
	_, statusCode, _, err := cl.tlsClient.Post(serverURL, nil, "")
	_ = statusCode
	return err
}

// Marshal encodes the native data into the wire format
func (cl *HttpSSEClient) Marshal(data any) []byte {
	jsonData, _ := json.Marshal(data)
	return jsonData
}

// Observe subscribes to property updates
// Use SetEventHandler to receive observed property updates
// If name is empty then this observes all property changes
func (cl *HttpSSEClient) Observe(thingID string, name string) error {
	slog.Info("Observe",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name))

	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	vars := map[string]string{"thingID": thingID, "name": name}
	subscribePath := utils.Substitute(PostObservePropertiesPath, vars)
	_, _, _, err := cl.tlsClient.Post(subscribePath, nil, "")
	return err
}

// PubActionWithQueryParams publishes an action with query parameters
//func (cl *HttpSSEClient) PubActionWithQueryParams(
//	thingID string, name string, data any, requestID string, params map[string]string) (
//	stat hubclient.RequestStatus) {
//
//	slog.Info("PubActionWithQueryParams",
//		slog.String("thingID", thingID),
//		slog.String("name", name),
//	)
//	stat = cl.PubMessage(http.MethodPost, PostInvokeActionPath, thingID, name, data, requestID, params)
//	return stat
//}

// PubEvent publishes an event message and returns
// This returns an error if the connection with the server is broken
func (cl *HttpSSEClient) PubEvent(thingID string, name string, data any, requestID string) error {
	slog.Debug("PubEvent",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("data", data),
		//slog.String("requestID", requestID),
	)
	stat := cl.PubMessage(http.MethodPost, PostAgentPublishEventPath, thingID, name, data, nil, requestID)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubMessage an action, event, property, td or progress message and return the delivery status
//
//	methodName is http.MethodPost for actions, http.MethodPost/MethodGet for properties
//	path used to publish PostActionPath/PostEventPath/... optionally with {thingID} and/or {name}
//	thingID (optional) to publish as or to: events are published for the thing and actions to publish to the thingID
//	name (optional) is the event/action/property name being published or modified
//	input is the native message payload to transfer that will be serialized
//	output is optional destination for unmarshalling the payload
//	requestID optional 'message-id' header value
//
// This returns the response body and optional a response message with delivery status and requestID with a delivery status
func (cl *HttpSSEClient) PubMessage(methodName string, methodPath string,
	thingID string, name string, input interface{}, output interface{}, requestID string) (
	stat hubclient.RequestStatus) {

	progress := ""
	vars := map[string]string{
		"thingID": thingID,
		"name":    name}
	messagePath := utils.Substitute(methodPath, vars)
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	if cl.tlsClient == nil {
		stat.Status = vocab.RequestFailed
		stat.Error = "PubMessage. Client connection was closed"
		slog.Warn(stat.Error, "clientID", cl.GetClientID(), "thingID", thingID, "name", name, "cid", cl.cid)
		return stat
	}
	//resp, err := cl.tlsClient.Post(messagePath, payload)
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, messagePath)
	serData := cl.Marshal(input)

	reply, respMsgID, httpStatus, headers, err :=
		cl.tlsClient.Invoke(methodName, serverURL, serData, requestID, nil)

	_ = headers
	// TODO: detect difference between not connected and unauthenticated
	dataSchema := ""
	if headers != nil {
		// set if an alternative output dataschema is used, eg RequestStatus result
		dataSchema = headers.Get(hubclient.DataSchemaHeader)
		// when progress is returned without a deliverystatus object
		progress = headers.Get(hubclient.StatusHeader)
	}

	stat.CorrelationID = respMsgID
	if err != nil {
		stat.Error = err.Error()
		stat.Status = vocab.RequestFailed
		if httpStatus == http.StatusUnauthorized {
			err = errors.New("no longer authenticated")
		}
		// FIXME: use actual type
	} else if dataSchema == "RequestStatus" {
		// return dataschema contains a progress envelope
		err = cl.Unmarshal(reply, &stat)
	} else if reply != nil && len(reply) > 0 {
		// TODO: unmarshalling the reply here is useless as there is needs conversion to the correct type
		err = cl.Unmarshal(reply, &stat.Output)
		stat.Status = vocab.RequestCompleted
	} else if progress != "" {
		// progress status without delivery status output
		stat.Status = progress
	} else {
		// not an progress result and no data. assume all went well
		stat.Status = vocab.RequestCompleted
	}
	if err != nil {
		slog.Error("PubMessage error",
			"path", messagePath, "err", err.Error())
		stat.Error = err.Error()
	}
	return stat
}

// PubMultipleProperties agent publishes a batch of property values.
// Intended for use by agents
func (cl *HttpSSEClient) PubMultipleProperties(thingID string, propMap map[string]any) error {
	slog.Info("PubMultipleProperties",
		slog.String("thingID", thingID),
		slog.Int("nr props", len(propMap)),
	)
	// FIXME: get path from forms
	stat := cl.PubMessage("POST", PostAgentUpdateMultiplePropertiesPath,
		thingID, "", propMap, nil, "")
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubRequestStatus agent publishes a request progress update message to the digital twin
// The digital twin will update the request status and notify the sender.
// This returns an error if the connection with the server is broken
func (cl *HttpSSEClient) PubActionStatus(stat hubclient.RequestStatus) {
	slog.Debug("PubActionStatus",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", stat.ThingID),
		slog.String("name", stat.Name),
		slog.String("progress", stat.Status),
		slog.String("requestID", stat.CorrelationID))

	stat2 := cl.PubMessage(http.MethodPost, PostAgentPublishProgressPath,
		"", "", stat, nil, stat.CorrelationID)
	if stat.Error != "" {
		slog.Warn("PubActionStatus failed", "err", stat2.Error)
	}
}

// PubProperty agent publishes a property value update.
// Intended for use by agents to property changes
func (cl *HttpSSEClient) PubProperty(thingID string, name string, value any) error {
	slog.Info("PubProperty",
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("value", value))

	// FIXME: get path from forms
	stat := cl.PubMessage("POST", PostAgentUpdatePropertyPath,
		thingID, name, value, nil, "")
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubTD publishes a TD update.
// This is short for a digitwin directory updateTD action
func (cl *HttpSSEClient) PubTD(thingID string, tdJSON string) error {
	slog.Info("PubTD", slog.String("thingID", thingID))

	err := digitwin.DirectoryUpdateTD(cl, tdJSON)
	return err
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithJWT'
func (cl *HttpSSEClient) RefreshToken(oldToken string) (newToken string, err error) {
	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))
	refreshURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostRefreshPath)

	args := authn.UserRefreshTokenArgs{
		ClientID: cl.clientID,
		OldToken: oldToken,
	}
	data, _ := json.Marshal(args)
	// the bearer token holds the old token
	resp, requestID, httpStatus, headers, err := cl.tlsClient.Invoke(
		"POST", refreshURL, data, "", nil)
	_ = requestID
	_ = headers

	// set the new token as the bearer token
	if err == nil {
		err = cl.Unmarshal(resp, &newToken)

		if err == nil {
			// reconnect using the new token
			cl.tlsClient.SetAuthToken(newToken)
		}
	} else if httpStatus == http.StatusUnauthorized {
		err = errors.New("Unauthenticated")
	}
	return newToken, err
}

// Rpc publishes and action and waits for a completion or failed progress update.
// This uses a requestID to link actions to progress updates. Only use this for actions
// that support the 'rpc' capabilities (eg, the agent sends the progress update)
func (cl *HttpSSEClient) Rpc(
	thingID string, name string, args interface{}, resp interface{}) (err error) {

	// a requestID is needed before the action is published in order to match it with the reply
	requestID := "rpc-" + shortid.MustGenerate()

	slog.Info("Rpc (request)",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
		slog.String("cid", cl.cid),
	)

	rChan := make(chan *hubclient.RequestStatus)
	cl.mux.Lock()
	cl.correlData[requestID] = rChan
	cl.mux.Unlock()

	// invoke with query parameters to provide the message ID
	stat := cl.InvokeAction(thingID, name, args, resp, requestID)
	waitCount := 0

	// Intermediate status update such as 'applied' are not errors. Wait longer.
	for {
		// if the hub return channel doesnt exists then don't bother waiting for a result
		if !cl.IsConnected() {
			break
		}

		// wait at most cl.timeout or until delivery completes or fails
		// if the connection breaks while waiting then tlsClient will be nil.
		if time.Duration(waitCount)*time.Second > cl.timeout || cl.tlsClient == nil {
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
				slog.String("requestID", requestID),
			)
		}
		stat, err = cl.WaitForProgressUpdate(rChan, requestID, time.Second)
		waitCount++
	}
	cl.mux.Lock()
	delete(cl.correlData, requestID)
	cl.mux.Unlock()
	slog.Info("Rpc (result)",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
		slog.String("cid", cl.cid),
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

// SendOperation is temporary transition to support using TD forms
func (cl *HttpSSEClient) SendOperation(
	href string, op tdd.Form, data any) (stat hubclient.RequestStatus) {

	slog.Info("SendOperation", "href", href, "op", op)
	panic("Just a placeholder. Dont use this yet. Not implemented")
	return stat
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *HttpSSEClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.mux.Lock()
	cl.connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the handler that receives all consumer facing messages
// from the hub. (events, property updates)
func (cl *HttpSSEClient) SetMessageHandler(cb hubclient.MessageHandler) {
	cl.mux.Lock()
	cl.messageHandler = cb
	cl.mux.Unlock()
}

// SetRequestHandler set the handler that receives all agent facing messages
// from the hub. (write property and invoke action)
func (cl *HttpSSEClient) SetRequestHandler(cb hubclient.RequestHandler) {
	cl.mux.Lock()
	cl.requestHandler = cb
	cl.mux.Unlock()
}

// SetSSEPath sets the new sse path to use.
// This allows to change the hub default /ssesc
func (cl *HttpSSEClient) SetSSEPath(ssePath string) {
	cl.mux.Lock()
	cl.ssePath = ssePath
	cl.mux.Unlock()
}

// Subscribe subscribes to a single event of one or more thing.
// Use SetEventHandler to receive subscribed events or SetRequestHandler for actions
func (cl *HttpSSEClient) Subscribe(thingID string, name string) error {
	slog.Info("Subscribe",
		slog.String("clientID", cl.clientID),
		slog.String("cid", cl.cid),
		slog.String("thingID", thingID),
		slog.String("name", name))

	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	vars := map[string]string{"thingID": thingID, "name": name}
	subscribePath := utils.Substitute(PostSubscribeEventPath, vars)
	_, _, _, err := cl.tlsClient.Post(subscribePath, nil, "")
	return err
}

// Unmarshal decodes the wire format to native data
func (cl *HttpSSEClient) Unmarshal(raw []byte, reply interface{}) error {
	err := json.Unmarshal(raw, reply)
	return err
}

// Unobserve thing properties
func (cl *HttpSSEClient) Unobserve(thingID string, name string) error {
	slog.Info("Unobserve",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name))

	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	vars := map[string]string{"thingID": thingID, "name": name}
	unsubscribePath := utils.Substitute(PostUnobservePropertyPath, vars)
	_, _, _, err := cl.tlsClient.Post(unsubscribePath, nil, "")
	return err
}

// Unsubscribe from thing event(s)
func (cl *HttpSSEClient) Unsubscribe(thingID string, name string) error {
	slog.Info("Unsubscribe",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name))

	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	vars := map[string]string{"thingID": thingID, "name": name}
	unsubscribePath := utils.Substitute(PostUnsubscribeEventPath, vars)
	_, _, _, err := cl.tlsClient.Post(unsubscribePath, nil, "")
	return err
}

// WaitForProgressUpdate waits for an async progress update message or until timeout
// This returns the status or an error if the timeout has passed
func (cl *HttpSSEClient) WaitForProgressUpdate(
	statChan chan *hubclient.RequestStatus, requestID string, timeout time.Duration) (
	stat hubclient.RequestStatus, err error) {

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

// WriteProperty posts a configuration change request
func (cl *HttpSSEClient) WriteProperty(thingID string, name string, data any) (
	stat hubclient.RequestStatus) {

	slog.Info("WriteProperty",
		slog.String("me", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
	)

	stat = cl.PubMessage(http.MethodPost, PostWritePropertyPath, thingID, name, data, nil, "")
	return stat
}

// NewHttpSSEClient creates a new instance of the http client with a SSE return-channel.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewHttpSSEClient(hostPort string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *HttpSSEClient {

	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewHttpSSEClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", hostPort))
	} else {
		slog.Debug("NewHttpSSEClient: CA certificate",
			slog.String("destination", hostPort),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}
	cid := shortid.MustGenerate()
	cl := HttpSSEClient{
		//_status: hubclient.TransportStatus{
		//	HubURL:               fmt.Sprintf("https://%s", hostPort),
		caCert:   caCert,
		clientID: clientID,
		cid:      cid,

		// max delay 3 seconds before a response is expected
		timeout:  timeout,
		hostPort: hostPort,
		ssePath:  ConnectSSEPath,

		correlData: make(map[string]chan *hubclient.RequestStatus),
		// max message size for bulk reads is 10MB.
	}
	cl.tlsClient = tlsclient.NewTLSClient(
		hostPort, clientCert, caCert, timeout, cid)

	return &cl
}
