package wssclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/sse"
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

// WSSClient manages the connection to the hub server using Websockets.
// This implements the IConsumerClient interface.
type WSSClient struct {
	clientID string

	wssURL      string
	wssConn     *websocket.Conn
	wssCancelFn context.CancelFunc
	caCert      *x509.Certificate

	// tlsClient is the TLS client used for sending and receiving messages
	timeout time.Duration // request timeout
	// '_' variables are mux protected
	mux sync.RWMutex
	//tlsClient *tlsclient.TLSClient

	isConnected atomic.Bool

	subscriptions  map[string]bool
	connectHandler func(connected bool, err error)
	// client side handler that receives messages from the server
	messageHandler hubclient.MessageHandler
	// map of requestID to delivery status update channel
	correlData map[string]chan *hubclient.RequestProgress
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

// Send a message over the websocket
func (cl *WSSClient) _send(msg interface{}) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	jsonMsg, _ := jsoniter.Marshal(msg)
	err := cl.wssConn.Write(ctx, websocket.MessageText, jsonMsg)
	cancelFn()
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
	loginURL := fmt.Sprintf("https://%s%s", wssURI.Host, sse.PostLoginPath)
	//cl.mux.Unlock()

	slog.Info("ConnectWithPassword", "clientID", cl.clientID)
	loginMessage := authn.UserLoginArgs{
		ClientID: cl.GetClientID(),
		Password: password,
	}
	// TODO: this is part of the http binding, not the websocket binding
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

	cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
		cl.clientID, cl.wssURL, token, cl.caCert,
		cl.connectHandler, cl.handleWSSMessage)

	return token, err
}

// ConnectWithToken connects to the Hub server using a user bearer token
// and obtain a new token.
//
//	jwtToken is the token previously obtained with login or refresh.
func (cl *WSSClient) ConnectWithToken(token string) (newToken string, err error) {
	//cl.mux.Lock()
	//if cl.tlsClient != nil {
	//	cl.tlsClient.Close()
	//}
	//slog.Info("ConnectWithToken (to hub)", "clientID", cl.clientID, "cid", cl.cid)
	//cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, nil, cl.caCert, cl.timeout, cl.cid)
	////cl._status.HubURL = fmt.Sprintf("https://%s", cl.hostPort)
	//cl.mux.Unlock()
	//cl.tlsClient.SetAuthToken(token)

	// Refresh the auth token and verify the connection works.
	//newToken, err = cl.RefreshToken(token)
	//if err != nil {
	//	return "", err
	//}
	//cl.tlsClient.SetAuthToken(newToken)

	cl.wssCancelFn, cl.wssConn, err = ConnectWSS(
		cl.clientID, cl.wssURL, token, cl.caCert,
		cl.connectHandler, cl.handleWSSMessage)

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

	cl.mux.Lock()
	cl.wssCancelFn()
	cl.mux.Unlock()
}

// GetClientID returns the client's account ID
func (cl *WSSClient) GetClientID() string {
	return cl.clientID
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

// InvokeAction publishes an action message and waits for an answer or until timeout
// An error is returned if delivery failed or succeeded but the action itself failed
func (cl *WSSClient) InvokeAction(
	dThingID string, name string, input interface{}, output interface{}, requestID string) (
	stat hubclient.RequestProgress) {

	slog.Info("InvokeAction",
		slog.String("clientID (me)", cl.clientID),
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
	)
	msg := InvokeActionMessage{
		ThingID:   dThingID,
		Operation: vocab.WotOpInvokeAction,
		Name:      name,
		RequestID: requestID,
		Input:     input,
		Timestamp: time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	stat.Progress = vocab.RequestDelivered
	stat.ThingID = dThingID
	stat.Name = name
	stat.RequestID = requestID
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
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
	msg := ObservePropertyMessage{
		ThingID:   thingID,
		Operation: vocab.WoTOpObserveProperty,
		Name:      name,
	}
	err := cl._send(msg)
	return err
}

// PubProgressUpdate agent publishes a progress update message to the digital twin
// The digital twin will update the request status and notify the sender.
// This returns an error if the connection with the server is broken
func (cl *WSSClient) PubProgressUpdate(stat hubclient.RequestProgress) error {
	slog.Debug("PubProgressUpdate",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", stat.ThingID),
		slog.String("name", stat.Name),
		slog.String("progress", stat.Progress),
		slog.String("requestID", stat.RequestID))

	msg := ActionStatusMessage{
		ThingID:   stat.ThingID,
		Name:      stat.Name,
		RequestID: stat.RequestID,
		Progress:  stat.Progress,
		Error:     stat.Error,
		Output:    stat.Reply,
		Timestamp: time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	return err
}

// PubEvent publishes an event message and return
// This returns an error if the connection with the server is broken
func (cl *WSSClient) PubEvent(
	thingID string, name string, data any, requestID string) error {

	slog.Debug("PubEvent",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("data", data),
		//slog.String("requestID", requestID),
	)
	msg := EventMessage{
		ThingID:   thingID,
		Operation: vocab.WoTOpObserveProperty,
		Name:      name,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().Format(utils.RFC3339Milli),
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

// PubProperty agent publishes a property value update.
// Intended for use by agents to property changes
func (cl *WSSClient) PubProperty(thingID string, name string, value any) error {
	slog.Info("PubProperty",
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("value", value))

	msg := PropertyMessage{
		ThingID:   thingID,
		Operation: vocab.WotOpPublishProperty,
		Name:      name,
		Data:      value,
		Timestamp: time.Now().Format(utils.RFC3339Milli),
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
// The resulting token can be used with 'ConnectWithJWT'
func (cl *WSSClient) RefreshToken(oldToken string) (newToken string, err error) {
	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))
	return "", fmt.Errorf("not implemented")
}

// Rpc publishes and action and waits for a completion or failed progress update.
// This uses a requestID to link actions to progress updates. Only use this for actions
// that support the 'rpc' capabilities (eg, the agent sends the progress update)
func (cl *WSSClient) Rpc(
	thingID string, name string, args interface{}, resp interface{}) (err error) {

	// a requestID is needed before the action is published in order to match it with the reply
	requestID := "rpc-" + shortid.MustGenerate()

	slog.Info("Rpc (request)",
		slog.String("clientID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
	)

	rChan := make(chan *hubclient.RequestProgress)
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
		if time.Duration(waitCount)*time.Second > cl.timeout {
			break
		}
		if stat.Progress == vocab.RequestCompleted || stat.Progress == vocab.RequestFailed {
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
		slog.String("status", stat.Progress),
	)

	// check for errors
	if err == nil {
		if stat.Error != "" {
			err = errors.New(stat.Error)
		} else if stat.Progress != vocab.RequestCompleted {
			err = errors.New("Delivery not complete. Progress: " + stat.Progress)
		}
	}
	if err != nil {
		slog.Error("RPC failed",
			"thingID", thingID, "name", name, "err", err.Error())
	}
	// only once completed will there be a reply as a result
	if err == nil && resp != nil {
		// no choice but to decode
		err = utils.Decode(stat.Reply, resp)
	}
	return err
}

// SendOperation is temporary transition to support using TD forms
func (cl *WSSClient) SendOperation(
	thingID string, name string, op tdd.Form, data any) (stat hubclient.RequestProgress) {

	requestID := ""
	slog.Info("InvokeOperation",
		slog.String("clientID (me)", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
	)
	// TODO: pick the message format based on the operation
	msg := InvokeActionMessage{
		ThingID:   thingID,
		Operation: op.GetOperation(),
		Name:      name,
		RequestID: requestID,
		Input:     data,
		Timestamp: time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	stat.Progress = vocab.RequestDelivered
	stat.ThingID = thingID
	stat.Name = name
	stat.RequestID = requestID
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

// SetMessageHandler set the single handler that receives all messages from the hub.
func (cl *WSSClient) SetMessageHandler(cb hubclient.MessageHandler) {
	cl.mux.Lock()
	cl.messageHandler = cb
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
	msg := SubscribeMessage{
		ThingID:   thingID,
		Operation: vocab.WotOpSubscribeEvent,
		Name:      name,
	}
	err := cl._send(msg)
	return err
}

// Unmarshal decodes the wire format to native data
func (cl *WSSClient) Unmarshal(raw []byte, reply interface{}) error {
	err := jsoniter.Unmarshal(raw, reply)
	return err
}

// WaitForProgressUpdate waits for an async progress update message or until timeout
// This returns the status or an error if the timeout has passed
func (cl *WSSClient) WaitForProgressUpdate(
	statChan chan *hubclient.RequestProgress, requestID string, timeout time.Duration) (
	stat hubclient.RequestProgress, err error) {

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
		timeout: timeout,

		correlData: make(map[string]chan *hubclient.RequestProgress),
		// max message size for bulk reads is 10MB.
	}

	return &cl
}
