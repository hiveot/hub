package wssbinding

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// WssBindingClient manages the connection to the hub server using Websockets.
// This implements the IConsumer interface.
type WssBindingClient struct {
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
	messageHandler transports.MessageHandler
	// client side handler that receives agent requests from the hub
	requestHandler transports.RequestHandler
	// map of requestID to delivery status update channel
	correlData map[string]chan *transports.RequestStatus
}

// websocket connection status handler
func (cl *WssBindingClient) _onConnect(connected bool, err error) {

	cl.isConnected.Store(connected)
	if cl.connectHandler != nil {
		cl.connectHandler(connected, err)
	}
	// if retrying is enabled then try on disconnect
	if !connected && cl.retryOnDisconnect.Load() {
		cl.Reconnect()
	}
}

// _rpc publishes and action and waits for a completion or failed progress update.
// This uses a requestID to link actions to progress updates. Only use this for actions
// that support the 'rpc' capabilities (eg, the agent sends the progress update)
func (cl *WssBindingClient) _rpc(correlationID string, request interface{}, output interface{}) (err error) {

	slog.Info("rpc (request)",
		slog.String("clientID", cl.clientID),
		slog.String("correlationID", correlationID),
	)
	rChan := make(chan *transports.RequestStatus)
	cl.mux.Lock()
	cl.correlData[correlationID] = rChan
	cl.mux.Unlock()

	// invoke with query parameters to provide the message ID
	err = cl._send(request)
	if err != nil {
		slog.Warn("rpc: failed sending request",
			"correlationID", correlationID,
			"err", err.Error())
	}
	// wait for reply
	waitCount := 0
	status := vocab.RequestPending // (yeah any rpc)
	var reply any

	// Intermediate status update such as 'applied' are not errors. Wait longer.
	for err == nil {
		// if the hub connection no longer exists then don't wait any longer
		if !cl.IsConnected() {
			err = errors.New("connection lost")
			break
		}

		// wait at most cl.timeout or until delivery completes or fails
		// if the connection breaks while waiting then tlsClient will be nil.
		if time.Duration(waitCount)*time.Second > cl.timeout {
			break
		}
		if status == vocab.RequestCompleted || status == vocab.RequestFailed {
			break
		}
		if waitCount > 0 {
			slog.Info("rpc (wait)",
				slog.Int("count", waitCount),
				slog.String("clientID", cl.clientID),
				slog.String("correlationID", correlationID),
			)
		}
		status, reply, err = cl.WaitForProgressUpdate(rChan, correlationID, time.Second)
		waitCount++
	}

	// ending the wait
	cl.mux.Lock()
	delete(cl.correlData, correlationID)
	cl.mux.Unlock()
	slog.Info("rpc (result)",
		slog.String("clientID", cl.clientID),
		slog.String("correlationID", correlationID),
		slog.String("status", status),
	)

	// check for errors
	if err == nil && status != vocab.RequestCompleted {
		err = errors.New("Delivery not complete. Status: " + status)
	}
	if err != nil {
		slog.Warn("RPC failed", "err", err.Error())
	}
	// only after completion will there be a reply as a result
	if err == nil && output != nil && reply != nil {
		// no choice but to decode
		err = utils.Decode(reply, reply)
	}
	return err
}

// ConnectWithLoginForm invokes login using a form - temporary helper
// intended for testing a connection with a web server
//func (cl *WssBindingClient) ConnectWithLoginForm(password string) error {
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

// Encode and send a message over the websocket
func (cl *WssBindingClient) _send(msg interface{}) error {
	if !cl.IsConnected() {
		// note, it might be trying to reconnect in the background
		err := fmt.Errorf("Not connected to the hub")
		return err
	}
	// websockets do not allow concurrent write
	cl.mux.Lock()
	// TODO: performance of WriteJSON vs jsoniter?
	err := cl.wssConn.WriteJSON(msg)
	cl.mux.Unlock()
	return err
}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *WssBindingClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeEd25519)
//	return k
//}

// Disconnect from the server
func (cl *WssBindingClient) Disconnect() {
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
func (cl *WssBindingClient) GetClientID() string {
	return cl.clientID
}

func (cl *WssBindingClient) GetConnectionStatus() (bool, string, error) {
	var lastErr error = nil
	// lastError is stored as pointer because atomic.Value cannot switch between error and nil type
	if cl.lastError.Load() != nil {
		lastErrPtr := cl.lastError.Load()
		lastErr = *lastErrPtr
	}
	return cl.isConnected.Load(), "", lastErr
}

// GetProtocolType returns the type of protocol this client supports
func (cl *WssBindingClient) GetProtocolType() string {
	return "https"
}

// GetServerURL returns the schema://address:port of the hub connection
func (cl *WssBindingClient) GetServerURL() string {
	hubURL := fmt.Sprintf("https://%s", cl.wssURL)
	return hubURL
}

func (cl *WssBindingClient) opToMessageType(op string) string {
	// yeah not very efficient. todo
	for k, v := range MsgTypeToOp {
		if v == op {
			return k
		}
	}
	return ""
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *WssBindingClient) IsConnected() bool {
	return cl.isConnected.Load()
}

//// Logout from the server and end the session
//func (cl *WssBindingClient) Logout() error {
//	return fmt.Errorf("not implemented")
//}

// Marshal encodes the native data into the wire format
func (cl *WssBindingClient) Marshal(data any) []byte {
	jsonData, _ := jsoniter.Marshal(data)
	return jsonData
}

// RefreshToken refreshes the authentication token
//
// The resulting token can be used with 'ConnectWithToken'
func (cl *WssBindingClient) RefreshToken(oldToken string) (newToken string, err error) {

	slog.Info("RefreshToken", slog.String("clientID", cl.clientID))
	correlationID := shortid.MustGenerate()
	msg := ActionMessage{
		MessageType:   MsgTypeRefresh,
		Data:          nil,
		MessageID:     shortid.MustGenerate(),
		CorrelationID: correlationID,
		SenderID:      cl.clientID,
	}
	err = cl._rpc(correlationID, msg, &newToken)
	return newToken, err
}

// Reconnect attempts to re-establish a dropped connection using the last token
func (cl *WssBindingClient) Reconnect() {
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

// Rpc invokes an action and waits for a completion or failed progress update.
// This uses a correlationID to link actions to progress updates. Only use this for actions
// that support the 'rpc' capabilities (eg, the agent sends the progress update)
func (cl *WssBindingClient) Rpc(form tdd.Form,
	dThingID string, name string, input interface{}, resp interface{}) (err error) {
	correlationID := "rpc-" + shortid.MustGenerate()
	msg := ActionMessage{
		ThingID:       dThingID,
		MessageType:   form.GetOperation(),
		Name:          name,
		CorrelationID: correlationID,
		MessageID:     correlationID,
		Data:          input,
		SenderID:      cl.clientID,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	err = cl._rpc(correlationID, msg, resp)
	if err != nil {
		slog.Error("RPC failed",
			"thingID", dThingID, "name", name, "err", err.Error())
	}
	return err
}

func (cl *WssBindingClient) SendOperation(
	form tdd.Form, dThingID, name string, input interface{},
	output interface{}, correlationID string) (stat transports.RequestStatus) {

	op := form.GetOperation()

	// unpack the operation and split it into separate messages for each operation
	// it would be nice to have a single message envelope instead...
	msg := make(map[string]any)
	msg["thingId"] = dThingID
	msg["name"] = name
	msg["data"] = input
	msg["correlationID"] = correlationID
	msg["messageType"] = cl.opToMessageType(op)
	msg["timestamp"] = time.Now().Format(utils.RFC3339Milli)
	// FIXME: how to add the names for read multiple events/properties/actions?
	switch op {
	case vocab.HTOpReadEvent:
		msg["event"] = name
	case vocab.OpQueryAction, vocab.OpInvokeAction:
		msg["action"] = name
		msg["input"] = input // invoke only
	case vocab.OpObserveProperty, vocab.OpReadProperty:
		msg["property"] = name
	}
	err := cl._send(msg)
	stat.Status = transports.RequestPending
	stat.ThingID = dThingID
	stat.Name = name
	stat.CorrelationID = correlationID
	if err != nil {
		stat.Error = err.Error()
	}
	return stat
}

// SendOperationStatus [agent] sends a operation progress status update to the server.
func (cl *WssBindingClient) SendOperationStatus(stat transports.RequestStatus) {

	slog.Debug("PubActionStatus",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", stat.ThingID),
		slog.String("name", stat.Name),
		slog.String("progress", stat.Status),
		slog.String("requestID", stat.CorrelationID))

	msg := ActionStatusMessage{
		MessageType:   MsgTypeActionStatus,
		ThingID:       stat.ThingID,
		Name:          stat.Name,
		CorrelationID: stat.CorrelationID,
		MessageID:     shortid.MustGenerate(),
		Status:        stat.Status,
		Error:         stat.Error,
		Output:        stat.Output,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	_ = cl._send(msg)

}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *WssBindingClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.mux.Lock()
	cl.connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the handler that receives all consumer facing messages
// from the hub. (events, property updates)
func (cl *WssBindingClient) SetMessageHandler(cb transports.MessageHandler) {
	cl.mux.Lock()
	cl.messageHandler = cb
	cl.mux.Unlock()
}

// SetRequestHandler set the handler that receives all agent facing messages
// from the hub. (write property and invoke action)
func (cl *WssBindingClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.mux.Lock()
	cl.requestHandler = cb
	cl.mux.Unlock()
}

// SetWSSURL updates sets the new websocket URL to use.
func (cl *WssBindingClient) SetWSSURL(wssURL string) {
	cl.mux.Lock()
	cl.wssURL = wssURL
	cl.mux.Unlock()
}

// Unmarshal decodes the wire format to native data
func (cl *WssBindingClient) Unmarshal(raw []byte, reply interface{}) error {
	err := jsoniter.Unmarshal(raw, reply)
	return err
}

// WaitForProgressUpdate waits for an async progress update message or until timeout
// This returns the status or an error if the timeout has passed
func (cl *WssBindingClient) WaitForProgressUpdate(
	statChan chan *transports.RequestStatus, requestID string, timeout time.Duration) (
	status string, data any, err error) {

	var stat transports.RequestStatus
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	select {
	case statC := <-statChan:
		stat = *statC
		break
	case <-ctx.Done():
		err = errors.New("Timeout waiting for status update: requestID=" + requestID)
	}
	return stat.Status, stat.Output, err
}

// NewWssBindingClient creates a new instance of the websocket hub client.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewWssBindingClient(wssURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *WssBindingClient {

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
	cl := WssBindingClient{
		clientID: clientID,
		wssURL:   wssURL,
		caCert:   caCert,

		// max delay 3 seconds before a response is expected
		timeout:              timeout,
		maxReconnectAttempts: 0, // 1 attempt per second

		correlData: make(map[string]chan *transports.RequestStatus),
		// max message size for bulk reads is 10MB.
	}

	return &cl
}
