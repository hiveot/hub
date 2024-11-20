package wssclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

const PostLoginPath = "/authn/login"

// WSSClient manages the connection to the hub server using Websockets.
// This implements the IConsumerClient interface.
type WSSClient struct {
	clientID string

	wssURL      string
	wssConn     *websocket.Conn
	wssCancelFn context.CancelFunc
	caCert      *x509.Certificate

	timeout time.Duration // rpc timeout
	mux     sync.RWMutex

	isConnected          atomic.Bool
	retryOnDisconnect    atomic.Bool
	lastError            atomic.Pointer[error]
	maxReconnectAttempts int // 0 for indefinite
	token                string

	subscriptions  map[string]bool
	connectHandler func(connected bool, err error)
	// client side handler that receives consumer facing messages from the hub
	messageHandler hubclient.MessageHandler
	// client side handler that receives agent requests from the hub
	requestHandler hubclient.RequestHandler
	// map of requestID to delivery status update channel
	correlData map[string]chan *hubclient.RequestStatus
}

// websocket connection status handler
func (cl *WSSClient) _onConnect(connected bool, err error) {

	cl.isConnected.Store(connected)
	if cl.connectHandler != nil {
		cl.connectHandler(connected, err)
	}
	// if retrying is enabled then try on disconnect
	if !connected && cl.retryOnDisconnect.Load() {
		cl.Reconnect()
	}
}

// ConnectWithLoginForm invokes login using a form - temporary helper
// intended for testing a connection with a web server
//func (cl *WSSClient) ConnectWithLoginForm(password string) error {
//	formMock := url.Values{}
//	formMock.Add("loginID", cl.clientID)
//	formMock.Add("password", password)
//	fullURL := fmt.Sprintf("https://%s/login", cl.hostPort)
//
//	//PostForm should return a cookie that should be used in the sse connection
//	resp, err := cl.tlsClient.GetHttpClient().PostForm(fullURL, formMock)
//	if err == nil {
//		// get the session token from the cookie
//		cookie := resp.Request.Header.Get("cookie")
//		kvList := strings.Split(cookie, ",")
//		token := ""
//		for _, kv := range kvList {
//			kvParts := strings.SplitN(kv, "=", 2)
//			if kvParts[0] == "session" {
//				token = kvParts[1]
//				break
//			}
//		}
//		cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
//			cl.clientID, cl.wssURL, token, cl.caCert,
//			cl.onConnect, cl.onMessage)
//	}
//	return err
//}

// Send a request and wait for a response.
//
// This creates a requestID to link the request to a response and the client timeout
// settings for maximum wait time.
//
// This returns the response (action) status message as returned by the hub, or an error if sending fails.
func (cl *WSSClient) _request(wssMsg interface{}, correlationID string) (stat hubclient.RequestStatus, err error) {

	if correlationID == "" {
		correlationID = shortid.MustGenerate()
	}
	rChan := make(chan *hubclient.RequestStatus)
	cl.mux.Lock()
	cl.correlData[correlationID] = rChan
	cl.mux.Unlock()

	err = cl._send(wssMsg)
	stat.CorrelationID = correlationID
	if err != nil {
		stat.Status = vocab.RequestFailed
		stat.Error = err.Error()
	} else {
		stat.Status = vocab.RequestPending

		waitCount := 0

		// Intermediate status update such as 'applied' are not errors. Wait longer.
		for {
			// if the hub return channel closed then don't bother waiting for a result
			if !cl.IsConnected() {
				err = fmt.Errorf("lost connection to the Hub")
				break
			}

			// wait at most cl.timeout or until delivery completes or fails
			// if the connection breaks while waiting then tlsClient will be nil.
			if time.Duration(waitCount)*time.Second > cl.timeout {
				break
			}
			if stat.Status == vocab.RequestCompleted || stat.Status == vocab.RequestFailed {
				break
			}
			if waitCount > 0 {
				slog.Info("Rpc (wait)",
					slog.Int("count", waitCount),
					slog.String("clientID", cl.clientID),
					slog.String("correlationID", correlationID),
				)
			}
			stat, err = cl.WaitForProgressUpdate(rChan, correlationID, time.Second)
			waitCount++
		}
	}
	cl.mux.Lock()
	delete(cl.correlData, correlationID)
	cl.mux.Unlock()

	slog.Info("Rpc (result)",
		slog.String("clientID", cl.clientID),
		slog.String("requestID", correlationID),
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
		slog.Error("RPC failed", "err", err.Error())
	}
	return stat, err
}

// Send a message over the websocket
func (cl *WSSClient) _send(msg interface{}) error {
	if !cl.IsConnected() {
		// note, it might be trying to reconnect in the background
		err := fmt.Errorf("Not connected to the hub")
		return err
	}
	err := cl.wssConn.WriteJSON(msg)
	return err
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
func (cl *WSSClient) ConnectWithPassword(password string) (newToken string, err error) {
	//cl.mux.Lock()
	//// remove existing connection
	//if cl.tlsClient != nil {
	//	cl.tlsClient.Close()
	//}
	//cl.tlsClient = tlsclient.NewTLSClient(
	//	cl.hostPort, nil, cl.caCert, cl.timeout, cl.cid)
	wssURI, err := url.Parse(cl.wssURL)
	loginURL := fmt.Sprintf("https://%s%s",
		wssURI.Host,
		PostLoginPath)
	//cl.mux.Unlock()

	slog.Info("ConnectWithPassword", "clientID", cl.clientID)
	loginMessage := authn.UserLoginArgs{
		ClientID: cl.GetClientID(),
		Password: password,
	}
	// TODO: this is part of the http binding, not the websocket binding
	// a sacrificial client to get a token
	tlsClient := tlsclient.NewTLSClient(wssURI.Host, nil, cl.caCert, cl.timeout, "")
	argsJSON, _ := json.Marshal(loginMessage)
	resp, _, statusCode, _, err2 := tlsClient.Invoke(
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
	// with an auth token the connection can be established

	cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
		cl.clientID, cl.wssURL, token, cl.caCert,
		cl._onConnect, cl.handleWSSMessage)

	if err == nil {
		cl.token = token
		cl.retryOnDisconnect.Store(true)
	}

	return token, err
}

// ConnectWithToken connects to the Hub server using a user bearer token
// and obtain a new token.
//
//	jwtToken is the token previously obtained with login or refresh.
func (cl *WSSClient) ConnectWithToken(token string) (newToken string, err error) {

	cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
		cl.clientID, cl.wssURL, token, cl.caCert,
		cl._onConnect, cl.handleWSSMessage)

	if err == nil {
		// Refresh the auth token and verify the connection works.
		newToken, err = cl.RefreshToken(token)
	}
	// once the connection is established enable the retry on disconnect
	if err == nil {
		cl.retryOnDisconnect.Store(true)
	}
	return newToken, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (cl *WSSClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeEd25519)
	return k
}

// Disconnect from the server
func (cl *WSSClient) Disconnect() {
	slog.Debug("HttpSSEClient.Disconnect",
		slog.String("clientID", cl.clientID),
	)
	// dont try to reconnect
	cl.retryOnDisconnect.Store(false)

	cl.mux.Lock()
	if cl.wssCancelFn != nil {
		cl.wssCancelFn()
	}
	if len(cl.correlData) != 0 {
		slog.Error("Client is closed but there are still an unhandled RPC call")
	}
	cl.mux.Unlock()
}

// GetClientID returns the client's account ID
func (cl *WSSClient) GetClientID() string {
	return cl.clientID
}

func (cl *WSSClient) GetConnectionStatus() (bool, string, error) {
	var lastErr error = nil
	// lastError is stored as pointer because atomic.Value cannot switch between error and nil type
	if cl.lastError.Load() != nil {
		lastErrPtr := cl.lastError.Load()
		lastErr = *lastErrPtr
	}
	return cl.isConnected.Load(), "", lastErr
}

// GetProtocolType returns the type of protocol this client supports
func (cl *WSSClient) GetProtocolType() string {
	return "https"
}

// GetHubURL returns the schema://address:port of the hub connection
func (cl *WSSClient) GetHubURL() string {
	hubURL := fmt.Sprintf("https://%s", cl.wssURL)
	return hubURL
}

// InvokeAction publishes an action message.
// To receive a reply use WaitForProgressUpdate
// An error is returned if there is no connection.
func (cl *WSSClient) InvokeAction(
	dThingID string, name string, input interface{}, output interface{}, correlationID string) (
	stat hubclient.RequestStatus) {

	if correlationID == "" {
		correlationID = shortid.MustGenerate()
	}
	slog.Info("InvokeAction",
		slog.String("clientID (me)", cl.clientID),
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("correlationID", correlationID),
	)
	msg := ActionMessage{
		ThingID:       dThingID,
		MessageType:   MsgTypeInvokeAction,
		Name:          name,
		CorrelationID: correlationID,
		MessageID:     correlationID,
		Data:          input,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	stat.Status = vocab.RequestPending
	stat.ThingID = dThingID
	stat.Name = name
	stat.CorrelationID = correlationID
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}
func (cl *WSSClient) InvokeOperation(
	op tdd.Form, dThingID, name string, input interface{}, output interface{}) error {
	return fmt.Errorf("not implemented")
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *WSSClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Logout from the server and end the session
func (cl *WSSClient) Logout() error {
	return fmt.Errorf("not implemented")
}

// Marshal encodes the native data into the wire format
func (cl *WSSClient) Marshal(data any) []byte {
	jsonData, _ := jsoniter.Marshal(data)
	return jsonData
}

// Observe subscribes to property updates
// Use SetEventHandler to receive observed property updates
// If name is empty then this observes all property changes
func (cl *WSSClient) Observe(thingID string, name string) error {
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
	msg := PropertyMessage{
		ThingID:     thingID,
		MessageType: MsgTypeObserveProperty,
		Name:        name,
	}
	err := cl._send(msg)
	return err
}

// PubActionStatus agent publishes an action progress message to the digital twin.
// The digital twin will update the request status and notify the sender.
// This returns an error if the connection with the server is broken
func (cl *WSSClient) PubActionStatus(stat hubclient.RequestStatus) {
	slog.Debug("PubActionStatus",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", stat.ThingID),
		slog.String("name", stat.Name),
		slog.String("progress", stat.Status),
		slog.String("requestID", stat.CorrelationID))

	msg := ActionStatusMessage{
		ThingID:       stat.ThingID,
		Name:          stat.Name,
		CorrelationID: stat.CorrelationID,
		Status:        stat.Status,
		Error:         stat.Error,
		Output:        stat.Output,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	_ = cl._send(msg)
}

// PubEvent agent publishes an event message and returns
// This returns an error if the connection with the server is broken
func (cl *WSSClient) PubEvent(
	thingID string, name string, data any, correlationID string) error {

	slog.Debug("PubEvent",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("data", data),
		//slog.String("requestID", requestID),
	)
	msg := EventMessage{
		ThingID:       thingID,
		MessageType:   MsgTypePublishEvent,
		Name:          name,
		Data:          data,
		CorrelationID: correlationID,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	return err
}

// PubMultipleProperties agent publishes a batch of property values.
// Intended for use by agents
func (cl *WSSClient) PubMultipleProperties(thingID string, propMap map[string]any) error {
	slog.Info("PubMultipleProperties",
		slog.String("thingID", thingID),
		slog.Int("nr props", len(propMap)),
	)
	return nil
}

// PubProperty agent sends an update of a property value.
// Intended for use by agents
func (cl *WSSClient) PubProperty(thingID string, name string, value any) error {
	slog.Info("PubProperty",
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("value", value))

	msg := PropertyMessage{
		ThingID:     thingID,
		MessageType: MsgTypePropertyReading,
		Name:        name,
		Data:        value,
		Timestamp:   time.Now().Format(utils.RFC3339Milli),
		MessageID:   shortid.MustGenerate(),
	}
	err := cl._send(msg)
	return err
}

// PubTD publishes a TD update.
// This is short for a digitwin directory updateTD action
func (cl *WSSClient) PubTD(thingID string, tdJSON string) error {
	slog.Info("PubTD", slog.String("thingID", thingID))
	stat := cl.InvokeAction(thingID, digitwin.DirectoryTD, tdJSON, nil, "")
	// TODO: track progress or ignore?
	_ = stat
	return nil
}

// RefreshToken refreshes the authentication token
//
// The resulting token can be used with 'ConnectWithToken'
func (cl *WSSClient) RefreshToken(oldToken string) (newToken string, err error) {
	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))
	//stat := cl.InvokeAction(authn.UserDThingID, authn.UserRefreshTokenMethod, oldToken, &newToken, "")
	data := authn.UserRefreshTokenArgs{OldToken: oldToken, ClientID: cl.clientID}
	msg := ActionMessage{
		MessageType:   MsgTypeRefresh,
		Data:          data,
		MessageID:     shortid.MustGenerate(),
		CorrelationID: shortid.MustGenerate(),
	}
	stat, err := cl._request(msg, msg.CorrelationID)
	if err == nil {
		newToken = utils.DecodeAsString(stat.Output)
		cl.token = newToken
	}
	return newToken, err
}

// Reconnect attempts to re-establish a dropped connection using the last token
func (cl *WSSClient) Reconnect() {
	var err error
	for i := 0; cl.maxReconnectAttempts == 0 || i < cl.maxReconnectAttempts; i++ {
		slog.Warn("Reconnecting attempt",
			slog.String("clientID", cl.clientID),
			slog.Int("i", i))
		_, err = cl.ConnectWithToken(cl.token)
		if err == nil {
			break
		}
		// retry until max repeat is reached or disconnect is called
		if !cl.retryOnDisconnect.Load() {
			break
		}
		// the connection timeout doesn't seem to work for some reason
		time.Sleep(time.Second)
	}
	if err != nil {
		slog.Warn("Reconnect failed: ", "err", err.Error())
	}
}

// Rpc publishes and action and waits for a completion or failed progress update.
// This uses a requestID to link actions to progress updates. Only use this for actions
// that support the 'rpc' capabilities (eg, the agent sends the progress update)
func (cl *WSSClient) Rpc(
	thingID string, name string, args interface{}, resp interface{}) (err error) {

	// TODO: share this code with the server _request()

	// a requestID is needed before the action is published in order to match it with the reply
	requestID := "rpc-" + shortid.MustGenerate()

	slog.Info("Rpc (request)",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
	)
	//cl._request(msg)
	rChan := make(chan *hubclient.RequestStatus)
	cl.mux.Lock()
	cl.correlData[requestID] = rChan
	cl.mux.Unlock()

	// invoke with query parameters to provide the message ID
	stat := cl.InvokeAction(thingID, name, args, resp, requestID)
	if stat.Error != "" {
		slog.Warn("InvokeAction: failed",
			"thingID", thingID,
			"name", name,
			"err", stat.Error)
	} else {
		waitCount := 0

		// Intermediate status update such as 'applied' are not errors. Wait longer.
		for {
			// if the hub connection no longer exists then don't wait any longer
			if !cl.IsConnected() {
				stat.Error = "Connection lost"
				stat.Status = ActionStatusFailed
				break
			}

			// wait at most cl.timeout or until delivery completes or fails
			// if the connection breaks while waiting then tlsClient will be nil.
			if time.Duration(waitCount)*time.Second > cl.timeout {
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
	}
	cl.mux.Lock()
	delete(cl.correlData, requestID)
	cl.mux.Unlock()
	slog.Info("Rpc (result)",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
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
func (cl *WSSClient) SendOperation(
	thingID string, name string, op tdd.Form, data any) (stat hubclient.RequestStatus) {

	correlationID := ""
	slog.Info("InvokeOperation",
		slog.String("clientID (me)", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("correlationID", correlationID),
	)
	// TODO: pick the message format based on the operation
	msg := ActionMessage{
		ThingID:       thingID,
		MessageType:   op.GetOperation(),
		Name:          name,
		CorrelationID: correlationID,
		Data:          data,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	stat.Status = vocab.RequestPending
	stat.ThingID = thingID
	stat.Name = name
	stat.CorrelationID = correlationID
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *WSSClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.mux.Lock()
	cl.connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the handler that receives all consumer facing messages
// from the hub. (events, property updates)
func (cl *WSSClient) SetMessageHandler(cb hubclient.MessageHandler) {
	cl.mux.Lock()
	cl.messageHandler = cb
	cl.mux.Unlock()
}

// SetRequestHandler set the handler that receives all agent facing messages
// from the hub. (write property and invoke action)
func (cl *WSSClient) SetRequestHandler(cb hubclient.RequestHandler) {
	cl.mux.Lock()
	cl.requestHandler = cb
	cl.mux.Unlock()
}

// SetWSSURL updates sets the new websocket URL to use.
func (cl *WSSClient) SetWSSURL(wssURL string) {
	cl.mux.Lock()
	cl.wssURL = wssURL
	cl.mux.Unlock()
}

// Subscribe sends a subscribe request to the server
// Use SetEventHandler to receive subscribed events or SetRequestHandler for actions
func (cl *WSSClient) Subscribe(thingID string, name string) error {
	slog.Info("Subscribe",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name))

	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	msg := EventMessage{
		ThingID:     thingID,
		MessageType: MsgTypeSubscribeEvent,
		Name:        name,
	}
	err := cl._send(msg)
	return err
}

// Unmarshal decodes the wire format to native data
func (cl *WSSClient) Unmarshal(raw []byte, reply interface{}) error {
	err := jsoniter.Unmarshal(raw, reply)
	return err
}

// Unobserve sends an unobserve request to the server
func (cl *WSSClient) Unobserve(thingID string, name string) error {
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
	msg := PropertyMessage{
		ThingID:     thingID,
		MessageType: MsgTypeUnobserveProperty,
		Name:        name,
	}
	err := cl._send(msg)
	return err
}

// Unsubscribe sends an unsubscribe request to the server
func (cl *WSSClient) Unsubscribe(thingID string, name string) error {
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
	msg := EventMessage{
		ThingID:     thingID,
		MessageType: MsgTypeUnsubscribeEvent,
		Name:        name,
	}
	err := cl._send(msg)
	return err
}

// WaitForProgressUpdate waits for an async progress update message or until timeout
// This returns the status or an error if the timeout has passed
func (cl *WSSClient) WaitForProgressUpdate(
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

// WriteProperty writes a configuration change request
func (cl *WSSClient) WriteProperty(thingID string, name string, data any) (
	stat hubclient.RequestStatus) {

	slog.Info("WriteProperty",
		slog.String("me", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
	)
	correlationID := shortid.MustGenerate()
	msg := PropertyMessage{
		ThingID:       thingID,
		MessageType:   MsgTypeWriteProperty,
		Name:          name,
		Data:          data,
		CorrelationID: correlationID,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	stat.ThingID = thingID
	stat.Name = name
	stat.CorrelationID = correlationID
	stat.Status = vocab.RequestPending
	if err != nil {
		stat.Error = err.Error()
	}

	return stat
}

// NewWSSClient creates a new instance of the websocket hub client.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewWSSClient(wssURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *WSSClient {

	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewHttpSSEClient: No CA certificate. InsecureSkipVerify used",
			slog.String("wssURL", wssURL))
	} else {
		slog.Debug("NewHttpSSEClient: CA certificate",
			slog.String("wssURL", wssURL),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}
	cl := WSSClient{
		clientID: clientID,
		wssURL:   wssURL,
		caCert:   caCert,

		// max delay 3 seconds before a response is expected
		timeout:              timeout,
		maxReconnectAttempts: 0, // 1 attempt per second

		correlData: make(map[string]chan *hubclient.RequestStatus),
		// max message size for bulk reads is 10MB.
	}

	return &cl
}
