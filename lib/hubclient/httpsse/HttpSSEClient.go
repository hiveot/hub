package httpsse

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Paths used by this protocol binding - SYNC with HttpSSEClient.ts
//
// These are currently hard coded to allow services and bindings to use the runtime.
//
// The idea is that the TDD defines them but that isn't usable from a service.
// Do the hub runtime services require a TD to be used?
//
// Yes? chicken and egg problem as to get the TD you need to use the directory service
// Is the directory an exception?
// No? hard coding these paths isn't ideal either, even if its in a client library
// Another consideration is that the args and results struct are already hard coded.
// This clearly needs more thoughts. What is needed for machines to use the TD for
// basic functions such as pub/sub actions, events and read the directory?
//
// For now hard-code the runtime paths in this client library. Investigate if they
// can be derived from the TD (that is the whole goal for using WoT TDs after all)
const (
	// Form paths that apply to all TDs at the top level
	// SYNC with HttpSSEClient.ts
	GetReadAllEventsPath         = "/events/{thingID}"
	GetReadAllPropertiesPath     = "/properties/{thingID}"
	PostSubscribeAllEventsPath   = "/subscribe/{thingID}/+"
	PostUnsubscribeAllEventsPath = "/unsubscribe/{thingID}/+"
	PostSubscribeEventPath       = "/subscribe/{thingID}/{key}"
	PostUnsubscribeEventPath     = "/unsubscribe/{thingID}/{key}"
	ConnectSSEPath               = "/sse"

	// Form paths for accessing TD directory
	GetThingPath  = "/tdd/{thingID}"
	GetThingsPath = "/tdd" // query param offset=, limit=

	// Form paths for accessing Actions
	PostInvokeActionPath = "/action/{thingID}/{key}"

	// Form paths for accessing Events
	PostPublishEventPath = "/event/{thingID}/{key}"

	// Form paths for accessing Properties
	PostWritePropertyPath = "/property/{thingID}/{key}"

	// authn service - used in authn TD
	PostLoginPath   = "/login"
	PostLogoutPath  = "/logout"
	PostRefreshPath = "/refresh"
)

// HttpSSEClient manages the connection to the hub server using http/2.
// This implements the IHubClient interface.
// This client creates two http/2 connections, one for posting messages and
// one for a sse connection to establish a return channel.
//
// This clients implements the REST API supported by the digitwin runtime services,
// specifically the directory, inbox, outbox, authn
type HttpSSEClient struct {
	hostPort string
	ssePath  string
	caCert   *x509.Certificate

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

// ClientID returns the client's connection ID
func (hc *HttpSSEClient) ClientID() string {
	return hc._status.ClientID
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
func (cl *HttpSSEClient) ConnectWithPassword(password string) (newToken string, err error) {
	cl.mux.Lock()
	// remove existing connection
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
	}
	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, nil, cl.caCert, cl.timeout)
	loginURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostLoginPath)
	cl.mux.Unlock()

	loginMessage := authn.UserLoginArgs{
		ClientID: cl.ClientID(),
		Password: password,
	}
	argsJSON, _ := json.Marshal(loginMessage)
	resp, statusCode, err2 := cl.tlsClient.Invoke("POST", loginURL, argsJSON, nil)
	if err2 != nil {
		err = fmt.Errorf("%d: Login failed: %s", statusCode, err2)
		return "", err
	}
	reply := authn.UserLoginResp{}
	err = json.Unmarshal(resp, &reply)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: Login to %s has unexpected response message: %s", loginURL, err)
		cl.SetConnectionStatus(hubclient.ConnectFailed, err)
		return "", err
	}
	// store the bearer token further requests
	cl.tlsClient.SetAuthToken(reply.Token)

	// create a second client to establish the sse connection if a path is set
	if cl.ssePath != "" {
		sseURL := fmt.Sprintf("https://%s%s", cl.hostPort, cl.ssePath)
		// use a new http client instance to set an indefinite timeout for the sse connection
		//sseClient := cl.tlsClient.GetHttpClient()
		sseClient := tlsclient.NewHttp2TLSClient(cl.caCert, nil, 0)
		err = cl.ConnectSSE(sseURL, reply.Token, sseClient, cl.handleSSEDisconnect)
	}
	// If the server is reachable. Open the return channel using SSE
	cl.SetConnectionStatus(hubclient.Connected, err)

	return reply.Token, err
}

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// and obtain a new auth token.
//
//	jwtToken is the token previously obtained with login or refresh.
func (cl *HttpSSEClient) ConnectWithToken(token string) (newToken string, err error) {
	cl.mux.Lock()
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
	}
	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, nil, cl._status.CaCert, cl.timeout)
	cl._status.HubURL = fmt.Sprintf("https://%s", cl.hostPort)
	cl.mux.Unlock()
	cl.tlsClient.SetAuthToken(token)

	// Refresh the auth token and verify the connection works.
	newToken, err = cl.RefreshToken(token)
	if err != nil {
		cl.SetConnectionStatus(hubclient.ConnectFailed, err)
		return "", err
	}
	cl.tlsClient.SetAuthToken(newToken)

	// establish the sse connection if a path is set
	if cl.ssePath != "" {
		// if the return channel fails then the client is still considered connected
		sseURL := fmt.Sprintf("https://%s%s", cl.hostPort, cl.ssePath)

		// separate client with a long timeout for sse
		// use a new http client instance to set an indefinite timeout for the sse connection
		//httpClient := cl.tlsClient.GetHttpClient()
		httpClient := tlsclient.NewHttp2TLSClient(cl.caCert, nil, 0)
		err = cl.ConnectSSE(sseURL, newToken, httpClient, cl.handleSSEDisconnect)
	}
	cl.SetConnectionStatus(hubclient.Connected, err)

	return newToken, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (cl *HttpSSEClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeECDSA)
	return k
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (cl *HttpSSEClient) Disconnect() {
	slog.Info("HttpSSEClient.Disconnect")

	cl.SetConnectionStatus(hubclient.Disconnected, nil)

	cl.mux.Lock()
	defer cl.mux.Unlock()

	if cl._sseChan != nil {
		close(cl._sseChan)
		cl._sseChan = nil
	}
	if cl.sseCancelFn != nil {
		cl.sseCancelFn()
		cl.sseCancelFn = nil
	}
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
		cl.tlsClient = nil
	}
}

// GetStatus Return the transport connection info
func (cl *HttpSSEClient) GetStatus() hubclient.TransportStatus {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl._status
}

func (cl *HttpSSEClient) GetTlsClient() *tlsclient.TLSClient {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient
}

// handler when the SSE connection fails.
// The SSE connection determines the connection status.
// If the status is connected then set it to disconnected.
// This invokes the optional callback if provided.
func (cl *HttpSSEClient) handleSSEDisconnect(err error) {
	if err != nil {
		cl.SetConnectionStatus(hubclient.ConnectFailed, err)
	} else {
		cl.SetConnectionStatus(hubclient.Disconnected, nil)
	}
}

// Logout from the server and end the session
func (cl *HttpSSEClient) Logout() error {
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostLogoutPath)
	//_, err := cl.Invoke("POST", serverURL, http.NoBody, nil)
	_, statusCode, err := cl.tlsClient.Post(serverURL, nil)
	_ = statusCode
	return err
}

// Marshal encodes the native data into the wire format
func (cl *HttpSSEClient) Marshal(data any) []byte {
	jsonData, _ := json.Marshal(data)
	return jsonData
}

// Publish an action, event or property message and return the delivery status
//
//	path used to publish PostActionPath/PostEventPath/...
//	thingID to publish as or to: events are published for the thing and actions to publish to the thingID
//	key is the event/action/property key being published or modified
//	data is the native message payload to transfer that will be serialized
//	queryParams optional key-value pairs to pass along as query parameters
func (cl *HttpSSEClient) postMessage(
	postPath string, thingID string, key string, data any, queryParams map[string]string) (
	stat hubclient.DeliveryStatus) {

	vars := map[string]string{
		"thingID": thingID,
		"key":     key}
	messagePath := utils.Substitute(postPath, vars)
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	if cl.tlsClient == nil {
		stat.Progress = hubclient.DeliveryFailed
		stat.Error = "PubAction. Client connection was closed"
		slog.Error(stat.Error)
		return stat
	}
	//resp, err := cl.tlsClient.Post(messagePath, payload)
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, messagePath)
	serData := cl.Marshal(data)
	reply, statusCode, err := cl.tlsClient.Invoke("POST", serverURL, serData, queryParams)

	// TODO: detect difference between not connected and unauthenticated
	// FIXME: should reply payload be a DeliveryStatus for every single message? Is there another way?
	// option 1: delivery updates are always sent async, not as a response. what about messageID?
	// option 2: pass delivery progress and messageID in response headers

	if err == nil && reply != nil && len(reply) > 0 {
		err = cl.Unmarshal(reply, &stat)
	}
	if err != nil {
		stat.Progress = hubclient.DeliveryFailed
		if statusCode == http.StatusUnauthorized {
			stat.Error = "no longer authenticated"
		} else {
			stat.Error = err.Error()
		}
		slog.Error(stat.Error)
	}
	return stat
}

// PubAction publishes an action message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
// An error is returned if delivery failed or succeeded but the action itself failed
func (cl *HttpSSEClient) PubAction(thingID string, key string, data any) (stat hubclient.DeliveryStatus) {
	slog.Debug("PubAction",
		slog.String("thingID", thingID),
		slog.String("key", key))
	stat = cl.postMessage(PostInvokeActionPath, thingID, key, data, nil)
	slog.Info("PubAction",
		slog.String("me", cl._status.ClientID),
		slog.String("thingID", thingID),
		slog.String("key", key),
		//slog.String("data", data),
		slog.String("messageID", stat.MessageID),
		slog.String("progress", stat.Progress),
	)
	return stat
}

// PubActionWithQueryParams publishes an action with query parameters
func (cl *HttpSSEClient) PubActionWithQueryParams(
	thingID string, key string, data any, params map[string]string) (stat hubclient.DeliveryStatus) {
	slog.Info("PubActionWithQueryParams",
		slog.String("thingID", thingID),
		slog.String("key", key),
	)
	stat = cl.postMessage(PostInvokeActionPath, thingID, key, data, params)
	return stat
}

// PubEvent publishes an event message and returns
// This returns an error if the connection with the server is broken
func (cl *HttpSSEClient) PubEvent(thingID string, key string, data any) error {
	slog.Info("PubEvent",
		slog.String("me", cl._status.ClientID),
		slog.String("device thingID", thingID),
		slog.String("key", key),
	)
	stat := cl.postMessage(PostPublishEventPath, thingID, key, data, nil)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubProperty publishes a configuration change request
// This is similar to publishing an action but only affects properties.
func (cl *HttpSSEClient) PubProperty(thingID string, key string, data any) (
	stat hubclient.DeliveryStatus) {
	stat = cl.postMessage(PostWritePropertyPath, thingID, key, data, nil)
	slog.Info("PubProperty",
		slog.String("me", cl._status.ClientID),
		slog.String("thingID", thingID),
		slog.String("key", key),
		//slog.String("value", value),
		slog.String("messageID", stat.MessageID),
		slog.String("progress", stat.Progress),
	)
	return stat
}

// PubProps publishes a properties map event
// Intended for use by agents to publish all properties at once
func (cl *HttpSSEClient) PubProps(thingID string, props map[string]any) error {
	return cl.PubEvent(thingID, vocab.EventTypeProperties, props)
}

// PubTD publishes a TD event
func (cl *HttpSSEClient) PubTD(td *things.TD) error {
	// TDs are published in JSON encoding as per spec
	payload, _ := json.Marshal(td)
	return cl.PubEvent(td.ID, vocab.EventTypeTD, string(payload))
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithJWT'
func (cl *HttpSSEClient) RefreshToken(oldToken string) (newToken string, err error) {
	refreshURL := fmt.Sprintf("https://%s%s", cl.hostPort, PostRefreshPath)

	args := authn.UserRefreshTokenArgs{
		ClientID: cl._status.ClientID,
		OldToken: oldToken,
	}
	data, _ := json.Marshal(args)
	// the bearer token holds the old token
	resp, statusCode, err := cl.tlsClient.Invoke("POST", refreshURL, data, nil)

	// set the new token as the bearer token
	// FIXME: differentiate between not connected and unauthenticated
	if err == nil {
		err = json.Unmarshal(resp, &newToken)

		if err == nil {
			// reconnect using the new token
			cl.tlsClient.SetAuthToken(newToken)
		}
	} else if statusCode == http.StatusUnauthorized {
		err = errors.New("Unauthenticated")
	}
	// FIXME: detect difference between connect and unauthenticated
	return newToken, err
}

// Rpc marshals arguments, invokes an action and unmarshal a response.
// Intended to remove some boilerplate
func (cl *HttpSSEClient) Rpc(
	thingID string, key string, args interface{}, resp interface{}) (err error) {

	// a messageID is needed before the action is published in order to match it with the reply
	messageID := "rpc-" + uuid.NewString()
	rChan := make(chan *hubclient.DeliveryStatus)
	cl.mux.Lock()
	cl._correlData[messageID] = rChan
	cl.mux.Unlock()

	// invoke with query parameters to provide the message ID
	qparams := map[string]string{"messageID": messageID}
	stat := cl.PubActionWithQueryParams(thingID, key, args, qparams)

	// Intermediate status update such as 'applied' are not errors. Wait longer.
	for {
		if stat.Progress == hubclient.DeliveryCompleted || stat.Progress == hubclient.DeliveryFailed {
			break
		}
		// after each update wait at most cl.timeout and fail
		stat, err = cl.WaitForStatusUpdate(rChan, messageID, cl.timeout)
	}
	cl.mux.Lock()
	delete(cl._correlData, messageID)
	cl.mux.Unlock()
	if err == nil && stat.Error != "" {
		err = errors.New(stat.Error)
	}
	if err == nil && stat.MessageID != messageID {
		// this only happens if a direct response was received with a different messageID
		// this can happen when the called function does not return a proper delivery status.
		slog.Warn("RPC request messageID does not match response. Bad response from remote service.",
			slog.String("clientID", cl.ClientID()),
			slog.String("service thingID", thingID),
			slog.String("service method", key),
			slog.String("req", messageID),
			slog.String("resp", stat.MessageID))
	}
	// when not completed return an error
	if err == nil && stat.Progress != hubclient.DeliveryCompleted {
		err = errors.New("Delivery not complete. Progress: " + stat.Progress)
	}
	// only once completed will there be a reply as a result
	if err == nil && resp != nil {
		err, _ = stat.Decode(resp)
	}
	return err
}

// SendDeliveryUpdate sends a delivery status update to the hub.
// The hub's inbox will update the status of the action and notify the original sender.
//
// Intended for agents that have processed an incoming action request asynchronously
// and need to send an update on further progress.
func (cl *HttpSSEClient) SendDeliveryUpdate(stat hubclient.DeliveryStatus) {
	slog.Info("SendDeliveryUpdate",
		slog.String("Progress", stat.Progress),
		slog.String("MessageID", stat.MessageID),
	)
	// thing
	_ = cl.PubEvent(digitwin.InboxDThingID, vocab.EventTypeDeliveryUpdate, stat)
}

// SetConnectionStatus updates the current connection status
func (cl *HttpSSEClient) SetConnectionStatus(cstat hubclient.ConnectionStatus, err error) {
	cl.mux.Lock()
	if cl._status.ConnectionStatus == cstat {
		cl.mux.Unlock()
		return
	}

	cl._status.ConnectionStatus = cstat
	if err != nil {
		cl._status.LastError = err
	}
	handler := cl._connectHandler
	newStatus := cl._status
	cl.mux.Unlock()
	if handler != nil {
		handler(newStatus)
	}
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *HttpSSEClient) SetConnectHandler(cb func(status hubclient.TransportStatus)) {
	cl.mux.Lock()
	cl._connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the single handler that receives all messages from the hub.
func (cl *HttpSSEClient) SetMessageHandler(cb hubclient.MessageHandler) {
	cl.mux.Lock()
	cl._messageHandler = cb
	cl.mux.Unlock()
}

// Subscribe subscribes to a single event of one or more thing.
// Use SetEventHandler to receive subscribed events or SetRequestHandler for actions
func (cl *HttpSSEClient) Subscribe(thingID string, key string) error {
	if thingID == "" {
		thingID = "+"
	}
	if key == "" {
		key = "+"
	}
	vars := map[string]string{"thingID": thingID, "key": key}
	subscribePath := utils.Substitute(PostSubscribeEventPath, vars)
	_, _, err := cl.tlsClient.Post(subscribePath, nil)
	return err
}

// Unmarshal decodes the wire format to native data
func (cl *HttpSSEClient) Unmarshal(raw []byte, reply interface{}) error {
	err := json.Unmarshal(raw, reply)
	return err
}

// Unsubscribe from thing event(s)
func (cl *HttpSSEClient) Unsubscribe(thingID string, key string) error {
	if thingID == "" {
		thingID = "+"
	}
	if key == "" {
		key = "+"
	}
	vars := map[string]string{"thingID": thingID, "key": key}
	unsubscribePath := utils.Substitute(PostUnsubscribeEventPath, vars)
	_, _, err := cl.tlsClient.Post(unsubscribePath, nil)
	return err
}

// WaitForStatusUpdate waits for an async status update or until timeout
// This returns the status or an error if the messageID doesn't exist
func (cl *HttpSSEClient) WaitForStatusUpdate(
	statChan chan *hubclient.DeliveryStatus, messageID string, timeout time.Duration) (stat hubclient.DeliveryStatus, err error) {

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
			ClientID:             clientID,
			ConnectionStatus:     hubclient.Disconnected,
			LastError:            nil,
			SupportsCertAuth:     false,
			SupportsPasswordAuth: true,
			SupportsKeysAuth:     false,
			SupportsTokenAuth:    true,
		},
		caCert: caCert,

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
