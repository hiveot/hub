package wssclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/base"
	"github.com/hiveot/hub/transports/servers/wssserver"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/url"
	"sync/atomic"
	"time"
)

// WssTransportClient manages the connection to the hub server using Websockets.
// This implements the IConsumer interface.
type WssTransportClient struct {
	base.TransportClient

	wssConn              *websocket.Conn
	wssCancelFn          context.CancelFunc
	retryOnDisconnect    atomic.Bool
	lastError            atomic.Pointer[error]
	maxReconnectAttempts int // 0 for indefinite
	token                string

	subscriptions map[string]bool
}

// websocket connection status handler
func (cl *WssTransportClient) _onConnect(connected bool, err error) {

	cl.BaseIsConnected.Store(connected)
	if cl.BaseConnectHandler != nil {
		cl.BaseConnectHandler(connected, err)
	}
	// if retrying is enabled then try on disconnect
	if !connected && cl.retryOnDisconnect.Load() {
		cl.Reconnect()
	}
}

// Encode and send a message over the websocket
// msg is a websocket protocol message
func (cl *WssTransportClient) _send(msg any) error {
	if !cl.IsConnected() {
		// note, it might be trying to reconnect in the background
		err := fmt.Errorf("_send: Not connected to the hub")
		return err
	}
	// websockets do not allow concurrent write
	cl.BaseMux.Lock()
	err := cl.wssConn.WriteJSON(msg)
	cl.BaseMux.Unlock()
	return err
}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *WssTransportClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeEd25519)
//	return k
//}

// Disconnect from the server
func (cl *WssTransportClient) Disconnect() {
	slog.Debug("Disconnect",
		slog.String("clientID", cl.BaseClientID),
	)
	// dont try to reconnect
	cl.retryOnDisconnect.Store(false)

	cl.BaseMux.Lock()
	if cl.wssCancelFn != nil {
		cl.wssCancelFn()
	}
	if cl.BaseRnrChan.Len() > 0 {
		slog.Error("Force closing unhandled RPC call", "n", cl.BaseRnrChan.Len())
		cl.BaseRnrChan.CloseAll()
	}
	cl.BaseMux.Unlock()
}

// InvokeAction invokes an action on a thing and wait for the response
func (cl *WssTransportClient) InvokeAction(dThingID, name string, input any, output any) error {
	return cl.SendRequest(wot.OpInvokeAction, dThingID, name, input, output)
}

//// Logout from the server and end the session
//func (cl *WssTransportClient) Logout() error {
//	return fmt.Errorf("not implemented")
//}

// Reconnect attempts to re-establish a dropped connection using the last token
func (cl *WssTransportClient) Reconnect() {
	var err error
	for i := 0; cl.maxReconnectAttempts == 0 || i < cl.maxReconnectAttempts; i++ {
		slog.Warn("Reconnecting attempt",
			slog.String("clientID", cl.BaseClientID),
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

// SendError [agent] sends an error response.
func (cl *WssTransportClient) SendError(
	thingID string, name string, errResponse string, requestID string) {

	slog.Debug("SendError",
		slog.String("agentID", cl.BaseClientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID))

	msg := wssserver.ErrorMessage{
		MessageType: wssserver.MsgTypeError,
		ThingID:     thingID,
		Name:        name,
		RequestID:   requestID,
		Title:       errResponse,
		Timestamp:   time.Now().Format(wot.RFC3339Milli),
	}
	_ = cl._send(msg)
}

func (cl *WssTransportClient) SendNotification(
	operation string, dThingID, name string, data any) error {
	// convert the operation into a websocket message and send it to the server
	msg, err := wssserver.OpToMessage(operation, dThingID, name, nil, data, "")
	if err != nil {
		slog.Error("SendNotification: unknown operation", "op", operation)
		return err
	}
	err = cl._send(msg)
	return err
}

// SendRequest sends an operation request and waits for a completion or timeout.
// This uses a correlationID to link actions to progress updates.
func (cl *WssTransportClient) SendRequest(operation string,
	dThingID string, name string, input interface{}, output interface{}) (err error) {

	requestID := "wssrpc-" + shortid.MustGenerate()
	clientID := cl.GetClientID()
	names := []string{}
	wssMsg, err := wssserver.OpToMessage(operation, dThingID, name, names, input, requestID)
	if err != nil {
		slog.Error("SendRequest:Unknown operation", "op", operation)
		return err
	}
	slog.Info("SendRequest (request)",
		slog.String("clientID", clientID),
		slog.String("dThingID", dThingID),
		slog.String("name", name),
		slog.String("requestID", requestID),
	)
	rChan := cl.BaseRnrChan.Open(requestID)

	err = cl._send(wssMsg)

	if err != nil {
		slog.Warn("rpc: failed sending request",
			"correlationID", requestID,
			"err", err.Error())
		return err
	}
	err = cl.WaitForResponse(rChan, requestID, output)
	return err
}

// SendResponse [agent] sends an action status update to the server.
func (cl *WssTransportClient) SendResponse(
	thingID string, name string, output any, requestID string) {

	slog.Debug("SendResponse",
		slog.String("agentID", cl.BaseClientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("requestID", requestID))

	msg := wssserver.ActionStatusMessage{
		MessageType: wssserver.MsgTypeActionStatus,
		ThingID:     thingID,
		Name:        name,
		RequestID:   requestID,
		MessageID:   shortid.MustGenerate(),
		Status:      "completed",
		Output:      output,
		Timestamp:   time.Now().Format(wot.RFC3339Milli),
	}
	_ = cl._send(msg)

}

// NewWssTransportClient creates a new instance of the websocket hub client.
//
//	fullURL of server to connect to, starting with wss:// schema
//	clientID to connect as
//	clientCert optional client certificate to connect with (todo)
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewWssTransportClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *WssTransportClient {

	caCertPool := x509.NewCertPool()
	urlParts, _ := url.Parse(fullURL)

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewWssTransportClient: No CA certificate. InsecureSkipVerify used",
			slog.String("hostPort", urlParts.Host))
	} else {
		slog.Debug("NewWssTransportClient: CA certificate",
			slog.String("hostPort", urlParts.Host),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}
	cl := WssTransportClient{
		TransportClient: base.TransportClient{
			BaseCaCert:       caCert,
			BaseClientID:     clientID,
			BaseConnectionID: clientID + "." + shortid.MustGenerate(),
			BaseProtocolType: transports.ProtocolTypeWSS,
			BaseFullURL:      fullURL,
			BaseHostPort:     urlParts.Host,
			BaseTimeout:      timeout,
			BaseRnrChan:      tputils.NewRnRChan(),
		},

		// max delay 3 seconds before a response is expected
		maxReconnectAttempts: 0, // 1 attempt per second

		// max message size for bulk reads is 10MB.
	}
	cl.BaseSendNotification = cl.SendNotification

	return &cl
}
