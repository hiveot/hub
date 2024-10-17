package httpsse

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/teris-io/shortid"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"sync"
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
	PostAgentUpdatePropertyPath = "/agent/property/{thingID}/{name}"
	// deprecated, use forms
	PostAgentUpdateMultiplePropertiesPath = "/agent/properties/{thingID}"
	// deprecated, use forms
	PostAgentUpdateTDDPath = "/agent/tdd/{thingID}"

	// authn service - used in authn TD
	PostLoginPath   = "/authn/login"
	PostLogoutPath  = "/authn/logout"
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
	cid      string

	timeout time.Duration // request timeout
	// '_' variables are mux protected
	mux         sync.RWMutex
	sseCancelFn context.CancelFunc
	// sseClient is the TLS client with the SSE connection
	//sseClient *tlsclient.TLSClient
	// tlsClient is the TLS client used for posting events
	tlsClient          *tlsclient.TLSClient
	_maxSSEMessageSize int
	_status            hubclient.TransportStatus
	_sseChan           chan *sse.Event
	_subscriptions     map[string]bool
	_connectHandler    func(status hubclient.TransportStatus)
	// client side handler that receives messages from the server
	_messageHandler hubclient.MessageHandler
	// map of messageID to delivery status update channel
	_correlData map[string]chan *hubclient.DeliveryStatus
}

func (hc *HttpSSEClient) connect() (err error) {
	return nil
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

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
func (hc *HttpSSEClient) ConnectWithPassword(password string) (newToken string, err error) {
	hc.mux.Lock()
	// remove existing connection
	if hc.tlsClient != nil {
		hc.tlsClient.Close()
	}
	hc.tlsClient = tlsclient.NewTLSClient(hc.hostPort, nil, hc.caCert, hc.timeout, hc.cid)
	loginURL := fmt.Sprintf("https://%s%s", hc.hostPort, PostLoginPath)
	hc.mux.Unlock()

	loginMessage := authn.UserLoginArgs{
		ClientID: hc.GetClientID(),
		Password: password,
	}
	argsJSON, _ := json.Marshal(loginMessage)
	resp, _, statusCode, _, err2 := hc.tlsClient.Invoke(
		"POST", loginURL, argsJSON, "", nil)
	if err2 != nil {
		err = fmt.Errorf("%d: Login failed: %s", statusCode, err2)
		return "", err
	}
	reply := authn.UserLoginResp{}
	err = hc.Unmarshal(resp, &reply)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: Login to %s has unexpected response message: %s", loginURL, err)
		hc.SetConnectionStatus(hubclient.ConnectFailed, err)
		return "", err
	}
	// store the bearer token further requests
	hc.tlsClient.SetAuthToken(reply.Token)

	// create a second client to establish the sse connection if a path is set
	if hc.ssePath != "" {
		sseURL := fmt.Sprintf("https://%s%s", hc.hostPort, hc.ssePath)
		// use a new http client instance to set an indefinite timeout for the sse connection
		//sseClient := cl.tlsClient.GetHttpClient()
		sseClient := tlsclient.NewHttp2TLSClient(hc.caCert, nil, 0)
		// If the server is reachable. Open the return channel using SSE
		err = hc.ConnectSSE(sseURL, reply.Token, sseClient, hc.handleSSEDisconnect)
	}
	hc.SetConnectionStatus(hubclient.Connected, err)

	return reply.Token, err
}

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// and obtain a new auth token.
//
//	jwtToken is the token previously obtained with login or refresh.
func (hc *HttpSSEClient) ConnectWithToken(token string) (newToken string, err error) {
	hc.mux.Lock()
	if hc.tlsClient != nil {
		hc.tlsClient.Close()
	}
	hc.tlsClient = tlsclient.NewTLSClient(hc.hostPort, nil, hc._status.CaCert, hc.timeout, hc.cid)
	hc._status.HubURL = fmt.Sprintf("https://%s", hc.hostPort)
	hc.mux.Unlock()
	hc.tlsClient.SetAuthToken(token)

	// Refresh the auth token and verify the connection works.
	newToken, err = hc.RefreshToken(token)
	if err != nil {
		hc.SetConnectionStatus(hubclient.ConnectFailed, err)
		return "", err
	}
	hc.tlsClient.SetAuthToken(newToken)

	// establish the sse connection if a path is set
	if hc.ssePath != "" {
		// if the return channel fails then the client is still considered connected
		sseURL := fmt.Sprintf("https://%s%s", hc.hostPort, hc.ssePath)

		// separate client with a long timeout for sse
		// use a new http client instance to set an indefinite timeout for the sse connection
		//httpClient := cl.tlsClient.GetHttpClient()
		httpClient := tlsclient.NewHttp2TLSClient(hc.caCert, nil, 0)
		err = hc.ConnectSSE(sseURL, newToken, httpClient, hc.handleSSEDisconnect)
	}
	hc.SetConnectionStatus(hubclient.Connected, err)

	return newToken, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (hc *HttpSSEClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeECDSA)
	return k
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (hc *HttpSSEClient) Disconnect() {
	slog.Info("HttpSSEClient.Disconnect")

	hc.SetConnectionStatus(hubclient.Disconnected, nil)

	hc.mux.Lock()
	defer hc.mux.Unlock()

	if hc._sseChan != nil {
		close(hc._sseChan)
		hc._sseChan = nil
	}
	if hc.sseCancelFn != nil {
		hc.sseCancelFn()
		hc.sseCancelFn = nil
	}
	if hc.tlsClient != nil {
		hc.tlsClient.Close()
		hc.tlsClient = nil
	}
}

// GetClientID returns the client's account ID
func (hc *HttpSSEClient) GetClientID() string {
	return hc.clientID
}

// GetCID returns the client's connection ID
func (hc *HttpSSEClient) GetCID() string {
	return hc.cid
}

// GetProtocolType returns the type of protocol this client supports
func (hc *HttpSSEClient) GetProtocolType() string {
	return "https"
}

// GetStatus Return the transport connection info
func (hc *HttpSSEClient) GetStatus() hubclient.TransportStatus {
	hc.mux.RLock()
	defer hc.mux.RUnlock()
	return hc._status
}

func (hc *HttpSSEClient) GetTlsClient() *tlsclient.TLSClient {
	hc.mux.RLock()
	defer hc.mux.RUnlock()
	return hc.tlsClient
}

// handler when the SSE connection fails.
// The SSE connection determines the connection status.
// If the status is connected then set it to disconnected.
// This invokes the optional callback if provided.
func (hc *HttpSSEClient) handleSSEDisconnect(err error) {
	if err != nil {
		hc.SetConnectionStatus(hubclient.ConnectFailed, err)
	} else {
		hc.SetConnectionStatus(hubclient.Disconnected, nil)
	}
}

// Logout from the server and end the session
func (hc *HttpSSEClient) Logout() error {
	serverURL := fmt.Sprintf("https://%s%s", hc.hostPort, PostLogoutPath)
	//_, err := cl.Invoke("POST", serverURL, http.NoBody, nil)
	_, statusCode, _, err := hc.tlsClient.Post(serverURL, nil, "")
	_ = statusCode
	return err
}

// Marshal encodes the native data into the wire format
func (hc *HttpSSEClient) Marshal(data any) []byte {
	jsonData, _ := json.Marshal(data)
	return jsonData
}

// PubMessage an action, event, property, td or delivery message and return the delivery status
//
//	methodName is http.MethodPost for actions, http.MethodPost/MethodGet for properties
//	path used to publish PostActionPath/PostEventPath/... optionally with {thingID} and/or {name}
//	thingID (optional) to publish as or to: events are published for the thing and actions to publish to the thingID
//	name (optional) is the event/action/property name being published or modified
//	data is the native message payload to transfer that will be serialized
//	messageID optional 'message-id' header value
//	queryParams optional name-value pairs to pass along as query parameters
//
// This returns the response body and optional a response message with delivery status and messageID with a delivery status
func (hc *HttpSSEClient) PubMessage(methodName string, methodPath string,
	thingID string, name string, data any, messageID string, queryParams map[string]string) (
	stat hubclient.DeliveryStatus) {

	progress := ""
	vars := map[string]string{
		"thingID": thingID,
		"name":    name}
	messagePath := utils.Substitute(methodPath, vars)
	hc.mux.RLock()
	defer hc.mux.RUnlock()
	if hc.tlsClient == nil {
		stat.Progress = vocab.ProgressStatusFailed
		stat.Error = "HandleActionFlow. Client connection was closed"
		slog.Error(stat.Error)
		return stat
	}
	//resp, err := cl.tlsClient.Post(messagePath, payload)
	serverURL := fmt.Sprintf("https://%s%s", hc.hostPort, messagePath)
	serData := hc.Marshal(data)

	reply, respMsgID, httpStatus, headers, err :=
		hc.tlsClient.Invoke(methodName, serverURL, serData, messageID, queryParams)

	_ = headers
	// TODO: detect difference between not connected and unauthenticated
	dataSchema := ""
	if headers != nil {
		// set if an alternative output dataschema is used, eg DeliveryStatus result
		dataSchema = headers.Get(hubclient.DataSchemaHeader)
		// when progress is returned without a deliverystatus object
		progress = headers.Get(hubclient.StatusHeader)
	}

	stat.MessageID = respMsgID
	if err != nil {
		stat.Error = err.Error()
		stat.Progress = vocab.ProgressStatusFailed
		if httpStatus == http.StatusUnauthorized {
			err = errors.New("no longer authenticated")
		}
	} else if dataSchema == "DeliveryStatus" {
		// return dataschema contains a progress envelope
		err = hc.Unmarshal(reply, &stat)
	} else if reply != nil && len(reply) > 0 {
		err = hc.Unmarshal(reply, &stat.Reply)
		stat.Progress = vocab.ProgressStatusCompleted
	} else if progress != "" {
		// progress status without delivery status output
		stat.Progress = progress
	} else {
		// not an progress result and no data. assume all went well
		stat.Progress = vocab.ProgressStatusCompleted
	}
	if err != nil {
		slog.Error("PubMessage error", "err", err.Error())
		stat.Error = err.Error()
	}
	return stat
}

// Publish a message using a TD operation and return the delivery status
//
//	 op contains the TD operation for the http binding
//		dThingID to publish as or to: events are published for the thing and actions to publish to the thingID
//		name is the event/action/property name being published or modified
//		data is the native message payload to transfer that will be serialized
//		queryParams optional name-value pairs to pass along as query parameters
func (hc *HttpSSEClient) pubFormMessage(op *tdd.Form,
	dtThingID string, name string, data any, messageID string, queryParams map[string]string) (
	stat hubclient.DeliveryStatus) {

	messagePath, _ := (*op).GetHRef()
	stat = hc.PubMessage(http.MethodPost, messagePath, dtThingID, name, data, messageID, queryParams)
	return stat
}

// InvokeAction publishes an action message and waits for an answer or until timeout
// An error is returned if delivery failed or succeeded but the action itself failed
func (hc *HttpSSEClient) InvokeAction(thingID string, name string, data any, messageID string) (
	stat hubclient.DeliveryStatus) {

	slog.Info("InvokeAction",
		slog.String("me", hc.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("messageID", messageID),
	)
	// FIXME: use TD form for this action
	// FIXME: track message-ID's using headers instead of message envelope
	stat = hc.PubMessage(http.MethodPost, PostInvokeActionPath, thingID, name, data, messageID, nil)

	return stat
}

// Observe subscribes to property updates
// Use SetEventHandler to receive observed property updates
// If name is empty then this observes all property changes
func (hc *HttpSSEClient) Observe(thingID string, name string) error {
	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	vars := map[string]string{"thingID": thingID, "name": name}
	subscribePath := utils.Substitute(PostObservePropertiesPath, vars)
	_, _, _, err := hc.tlsClient.Post(subscribePath, nil, "")
	return err
}

// PubActionWithQueryParams publishes an action with query parameters
//func (cl *HttpSSEClient) PubActionWithQueryParams(
//	thingID string, name string, data any, messageID string, params map[string]string) (
//	stat hubclient.DeliveryStatus) {
//
//	slog.Info("PubActionWithQueryParams",
//		slog.String("thingID", thingID),
//		slog.String("name", name),
//	)
//	stat = cl.PubMessage(http.MethodPost, PostInvokeActionPath, thingID, name, data, messageID, params)
//	return stat
//}

// PubEvent publishes an event message and returns
// This returns an error if the connection with the server is broken
func (hc *HttpSSEClient) PubEvent(thingID string, name string, data any, messageID string) error {
	slog.Info("PubEvent",
		slog.String("agentID", hc.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		//slog.String("messageID", messageID),
	)
	stat := hc.PubMessage(http.MethodPost, PostAgentPublishEventPath, thingID, name, data, messageID, nil)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubProgressUpdate agent publishes a request progress update message to the digital twin
// The digital twin will update the request status and notify the sender.
// This returns an error if the connection with the server is broken
func (hc *HttpSSEClient) PubProgressUpdate(stat hubclient.DeliveryStatus) {
	slog.Debug("PubProgressUpdate",
		slog.String("me", hc.clientID),
		slog.String("progress", stat.Progress),
		slog.String("messageID", stat.MessageID))

	stat2 := hc.PubMessage(http.MethodPost, PostAgentPublishProgressPath,
		"", "", stat, stat.MessageID, nil)
	if stat.Error != "" {
		slog.Warn("PubProgressUpdate failed", "err", stat2.Error)
	}
}

// PubProperties agent publishes a properties map event
// Intended for use by agents to publish all properties at once
func (hc *HttpSSEClient) PubProperties(thingID string, props map[string]any) error {
	slog.Info("PubProperties", slog.String("thingID", thingID))

	// FIXME: get path from forms?
	stat := hc.PubMessage("POST", PostAgentUpdateMultiplePropertiesPath,
		thingID, "", props, "", nil)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubTD publishes a TD update
func (hc *HttpSSEClient) PubTD(thingID string, tdJSON string) error {
	slog.Info("PubTD", slog.String("thingID", thingID))

	// TDs are published in JSON encoding as per spec
	stat := hc.PubMessage("POST", PostAgentUpdateTDDPath, thingID, "", tdJSON, "", nil)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithJWT'
func (hc *HttpSSEClient) RefreshToken(oldToken string) (newToken string, err error) {
	slog.Info("RefreshToken", slog.String("clientID", hc.clientID))
	refreshURL := fmt.Sprintf("https://%s%s", hc.hostPort, PostRefreshPath)

	args := authn.UserRefreshTokenArgs{
		ClientID: hc.clientID,
		OldToken: oldToken,
	}
	data, _ := json.Marshal(args)
	// the bearer token holds the old token
	resp, messageID, httpStatus, headers, err := hc.tlsClient.Invoke(
		"POST", refreshURL, data, "", nil)
	_ = messageID
	_ = headers

	// set the new token as the bearer token
	// FIXME: differentiate between not connected and unauthenticated
	if err == nil {
		err = hc.Unmarshal(resp, &newToken)

		if err == nil {
			// reconnect using the new token
			hc.tlsClient.SetAuthToken(newToken)
		}
	} else if httpStatus == http.StatusUnauthorized {
		err = errors.New("Unauthenticated")
	}
	// FIXME: detect difference between connect and unauthenticated
	return newToken, err
}

// Rpc marshals arguments, invokes an action and unmarshal a response.
// Intended to remove some boilerplate
func (hc *HttpSSEClient) Rpc(
	thingID string, name string, args interface{}, resp interface{}) (err error) {

	// a messageID is needed before the action is published in order to match it with the reply
	messageID := "rpc-" + shortid.MustGenerate()

	slog.Info("Rpc (request)", slog.String("clientID", hc.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("messageID", messageID),
	)

	rChan := make(chan *hubclient.DeliveryStatus)
	hc.mux.Lock()
	hc._correlData[messageID] = rChan
	hc.mux.Unlock()

	// invoke with query parameters to provide the message ID
	stat := hc.InvokeAction(thingID, name, args, messageID)

	// Intermediate status update such as 'applied' are not errors. Wait longer.
	for {
		if stat.Progress == vocab.ProgressStatusCompleted || stat.Progress == vocab.ProgressStatusFailed {
			break
		}
		// wait at most cl.timeout or until delivery completes or fails
		slog.Info("Rpc (wait)",
			slog.String("clientID", hc.clientID),
			slog.String("messageID", messageID),
		)
		stat, err = hc.WaitForStatusUpdate(rChan, messageID, hc.timeout)
	}
	hc.mux.Lock()
	delete(hc._correlData, messageID)
	hc.mux.Unlock()
	slog.Info("Rpc (result)",
		slog.String("clientID", hc.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("messageID", messageID),
		slog.String("status", stat.Progress),
	)

	// check for errors
	if err == nil {
		if stat.Error != "" {
			err = errors.New(stat.Error)
		} else if stat.Progress != vocab.ProgressStatusCompleted {
			err = errors.New("Delivery not complete. Progress: " + stat.Progress)
		}
	}
	// only once completed will there be a reply as a result
	if err == nil && resp != nil {
		err = utils.Decode(stat.Reply, resp)
	}
	return err
}

// SendOperation is temporary transition to support using TD forms
func (cl *HttpSSEClient) SendOperation(
	href string, op tdd.Form, data any) (stat hubclient.DeliveryStatus) {

	slog.Info("SendOperation", "href", href, "op", op)
	panic("Just a placeholder. Dont use this yet. Not implemented")
	return stat
}

// SetConnectionStatus updates the current connection status
func (hc *HttpSSEClient) SetConnectionStatus(cstat hubclient.ConnectionStatus, err error) {

	slog.Info("SetConnectionStatus",
		slog.String("clientID", hc.clientID),
		slog.String("cstat", string(cstat)))

	hc.mux.Lock()
	if hc._status.ConnectionStatus == cstat {
		hc.mux.Unlock()
		return
	}

	hc._status.ConnectionStatus = cstat
	if err != nil {
		hc._status.LastError = err
	}
	handler := hc._connectHandler
	newStatus := hc._status
	hc.mux.Unlock()
	if handler != nil {
		handler(newStatus)
	}
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (hc *HttpSSEClient) SetConnectHandler(cb func(status hubclient.TransportStatus)) {
	hc.mux.Lock()
	hc._connectHandler = cb
	hc.mux.Unlock()
}

// SetMessageHandler set the single handler that receives all messages from the hub.
func (hc *HttpSSEClient) SetMessageHandler(cb hubclient.MessageHandler) {
	hc.mux.Lock()
	hc._messageHandler = cb
	hc.mux.Unlock()
}

// Subscribe subscribes to a single event of one or more thing.
// Use SetEventHandler to receive subscribed events or SetRequestHandler for actions
func (hc *HttpSSEClient) Subscribe(thingID string, name string) error {
	slog.Info("Subscribe",
		slog.String("clientID", hc.clientID),
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
	_, _, _, err := hc.tlsClient.Post(subscribePath, nil, "")
	return err
}

// Unmarshal decodes the wire format to native data
func (hc *HttpSSEClient) Unmarshal(raw []byte, reply interface{}) error {
	err := json.Unmarshal(raw, reply)
	return err
}

// Unobserve thing properties
func (hc *HttpSSEClient) Unobserve(thingID string, name string) error {
	slog.Info("Unobserve",
		slog.String("clientID", hc.clientID),
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
	_, _, _, err := hc.tlsClient.Post(unsubscribePath, nil, "")
	return err
}

// Unsubscribe from thing event(s)
func (hc *HttpSSEClient) Unsubscribe(thingID string, name string) error {
	slog.Info("Unsubscribe",
		slog.String("clientID", hc.clientID),
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
	_, _, _, err := hc.tlsClient.Post(unsubscribePath, nil, "")
	return err
}

// WaitForStatusUpdate waits for an async progress update message or until timeout
// This returns the status or an error if the timeout has passed
func (hc *HttpSSEClient) WaitForStatusUpdate(
	statChan chan *hubclient.DeliveryStatus, messageID string, timeout time.Duration) (
	stat hubclient.DeliveryStatus, err error) {

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	select {
	case statC := <-statChan:
		stat = *statC
		break
	case <-ctx.Done():
		err = errors.New("Timeout waiting for status update: messageID=" + messageID)
	}

	return stat, err
}

// WriteProperty posts a configuration change request
func (hc *HttpSSEClient) WriteProperty(thingID string, name string, data any) (
	stat hubclient.DeliveryStatus) {

	slog.Info("WriteProperty",
		slog.String("me", hc.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
	)

	stat = hc.PubMessage(http.MethodPost, PostWritePropertyPath, thingID, name, data, "", nil)
	return stat
}

// NewHttpSSEClient creates a new instance of the htpp client with a SSE return-channel.
//
// fullURL is the url with schema. If omitted this uses the in-memory UDS address,
// which only works with SetAuthToken.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	keyPair with previously saved serialized public/private key pair, or "" to create one
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewHttpSSEClient(hostPort string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate, timeout time.Duration) *HttpSSEClient {

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
	tp := HttpSSEClient{
		_status: hubclient.TransportStatus{
			HubURL:               fmt.Sprintf("https://%s", hostPort),
			CaCert:               caCert,
			ConnectionStatus:     hubclient.Disconnected,
			LastError:            nil,
			SupportsCertAuth:     false,
			SupportsPasswordAuth: true,
			SupportsKeysAuth:     false,
			SupportsTokenAuth:    true,
		},
		caCert:   caCert,
		clientID: clientID,
		cid:      shortid.MustGenerate(),

		// max delay 3 seconds before a response is expected
		timeout:     timeout,
		hostPort:    hostPort,
		ssePath:     ConnectSSEPath,
		_sseChan:    make(chan *sse.Event),
		_correlData: make(map[string]chan *hubclient.DeliveryStatus),
		// max message size for bulk reads is 10MB.
		_maxSSEMessageSize: 1024 * 1024 * 10,
	}
	return &tp
}
