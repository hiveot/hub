package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const refreshPath = vocab.PostRefreshPath

// HttpSSEClient manages the hub server connection with hub event and action messaging using autopaho.
// This implements the IHubClient interface.
type HttpSSEClient struct {
	hostPort string
	ssePath  string
	clientID string
	caCert   *x509.Certificate

	timeout time.Duration // request timeout
	// _ variables are mux protected
	sseCancelFn     context.CancelFunc
	mux             sync.RWMutex
	sseClient       *sse.Client
	tlsClient       *tlsclient.TLSClient
	bearerToken     string
	_status         hubclient.HubTransportStatus
	_sseChan        chan *sse.Event
	_subscriptions  map[string]bool
	_connectHandler func(status hubclient.HubTransportStatus)
	_messageHandler api.MessageHandler
}

// ConnectSSE establishes a sse session over the Hub HTTPS connection.
// All hub messages are send as type ThingMessage, containing thingID, key, payload and sender
func (cl *HttpSSEClient) ConnectSSE(client *http.Client) error {
	// FIXME. what if serverURL contains a schema?
	sseURL := fmt.Sprintf("https://%s%s", cl.hostPort, cl.ssePath)

	//r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, sseURL, http.NoBody)
	req, err := cl.tlsClient.NewRequest("GET", sseURL, []byte{})
	if err != nil {
		return err
	}

	// use context to disconnect the client
	sseCtx, sseCancelFn := context.WithCancel(context.Background())
	cl.sseCancelFn = sseCancelFn
	req = req.WithContext(sseCtx)

	//r.Header.Set("Content-Type", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	// FIXME: how to close this down?
	conn := sse.NewConnection(req)

	conn.SubscribeToAll(cl.handleSSEEvent)
	go func() {
		// connect and report an error if connection ends due to reason other than context cancelled
		if err := conn.Connect(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("SSE connection failed", "err", err.Error())
		}
	}()
	//c.Connection = client
	//cl.startR3labsSSEListener()
	return nil
}

// ConnectWithCert connects to the Hub TLS server using client certifcate authentication
func (cl *HttpSSEClient) ConnectWithCert(kp keys.IHiveKey, cert *tls.Certificate) (token string, err error) {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return "", fmt.Errorf("ConnectWithCert not yet supported by the HTTPS/SSE binding")
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password.
func (cl *HttpSSEClient) ConnectWithPassword(password string) (newToken string, err error) {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	// remove existing connection
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
		cl.tlsClient = nil
	}

	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, cl.caCert, time.Second*120)
	token, err := cl.tlsClient.ConnectWithPassword(cl.clientID, password)
	if err == nil {
		cl.bearerToken = token

		// establish the sse connection if a path is set
		if cl.ssePath != "" {
			err = cl.ConnectSSE(cl.tlsClient.GetHttpClient())
		}
	}
	return token, err
}

// ConnectWithJWT connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
//
//	jwtToken is the token obtained with login or refresh.
func (cl *HttpSSEClient) ConnectWithJWT(jwtToken string) (newToken string, err error) {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
		cl.tlsClient = nil
	}

	cl.bearerToken = jwtToken
	cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, cl.caCert, time.Second*120)
	cl.tlsClient.ConnectWithToken(cl.clientID, jwtToken)
	newToken, err = cl.RefreshToken()

	// establish the sse connection if a path is set
	if err == nil && cl.ssePath != "" {
		err = cl.ConnectSSE(cl.tlsClient.GetHttpClient())
	}

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

// GetClientID returns the client's connection ID
func (cl *HttpSSEClient) GetClientID() string {
	return cl.clientID
}

// GetStatus Return the transport connection info
func (cl *HttpSSEClient) GetStatus() hubclient.HubTransportStatus {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl._status
}

func (cl *HttpSSEClient) GetTlsClient() *tlsclient.TLSClient {
	return cl.tlsClient
}

// handleSSEEvent processes the push-event received from the hub.
// This is passed on to the client, which must provide a delivery applied, completed or error status.
// This sends the delivery  status to the hub using a delivery event.
func (cl *HttpSSEClient) handleSSEEvent(event sse.Event) {
	slog.Info("received SSE msg")
	var stat api.DeliveryStatus
	var rxMsg things.ThingMessage

	slog.Info("startSSEListener. Received message",
		//slog.String("Comment", string(event.Comment)),
		slog.String("LastEventID", string(event.LastEventID)),
		slog.String("event", string(event.Data)),
		slog.String("type", string(event.Type)),
		slog.Int("size", len(event.Data)),
	)
	rxMsg = things.ThingMessage{}
	err := json.Unmarshal([]byte(event.Data), &rxMsg)
	if err != nil {
		slog.Error("ConnectSSE - subscribe; Received non-ThingMessage sse event. Ignored",
			"eventType", event.Type,
			"LastEventID", event.LastEventID)
		return
	}

	// if no message handler is set then this obviously fails
	if cl._messageHandler == nil {
		stat.Error = "handleSSEEvent no handler is set, event ignored"
		stat.Status = api.DeliveryFailed
	} else {
		stat = cl._messageHandler(&rxMsg)
	}
	// only actions need a delivery update
	if rxMsg.MessageType == vocab.MessageTypeAction {
		stat.MessageID = rxMsg.MessageID
		cl.SendDeliveryUpdate(rxMsg.ThingID, stat)
	}
}

// PubAction publishes an action message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
func (cl *HttpSSEClient) PubAction(thingID string, key string, payload []byte) (stat api.DeliveryStatus) {
	slog.Debug("PubEvent",
		slog.String("thingID", thingID), slog.String("key", key))
	vars := map[string]string{"thingID": thingID, "key": key}
	eventPath := utils.Substitute(vocab.PostActionPath, vars)
	resp, err := cl.tlsClient.Post(eventPath, payload)
	if err == nil {
		err = json.Unmarshal(resp, &stat)
	}
	if err != nil {
		stat.Status = api.DeliveryFailed
		stat.Error = err.Error()
	}
	return stat
}

// PubEvent publishes a message and returns
func (cl *HttpSSEClient) PubEvent(thingID string, key string, payload []byte) (stat api.DeliveryStatus) {
	slog.Debug("PubEvent",
		slog.String("thingID", thingID), slog.String("key", key))
	vars := map[string]string{"thingID": thingID, "key": key}
	eventPath := utils.Substitute(vocab.PostEventPath, vars)
	resp, err := cl.tlsClient.Post(eventPath, payload)
	if err == nil {
		err = json.Unmarshal(resp, &stat)
	}
	if err != nil {
		stat.Status = api.DeliveryFailed
		stat.Error = err.Error()
	}
	return stat
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithJWT'
func (cl *HttpSSEClient) RefreshToken() (newToken string, err error) {
	refreshURL := fmt.Sprintf("https://%s%s", cl.hostPort, refreshPath)

	args := api.RefreshTokenArgs{
		OldToken: cl.bearerToken, ClientID: cl.clientID,
	}
	data, _ := json.Marshal(args)
	// the bearer token holds the old token
	resp, err := cl.tlsClient.Invoke("POST", refreshURL, data)
	// set the new token as the bearer token
	if err == nil {
		reply := api.RefreshTokenResp{}
		err = json.Unmarshal(resp, &reply)

		if err == nil {
			newToken = reply.Token
			cl.bearerToken = newToken
		}
	}
	return newToken, err
}

// Rpc invokes an action and unmarshal a response.
// Intended to remove some boilerplate
// context is used to timeout the request. Use nil for default of 1 sec
// TODO: under construction
func (cl *HttpSSEClient) Rpc(ctx context.Context,
	thingID string, key string, args interface{}, resp interface{}) (stat api.DeliveryStatus, err error) {

	var cancelFunc context.CancelFunc
	if ctx == nil {
		ctx, cancelFunc = context.WithTimeout(context.Background(), time.Second)
		defer cancelFunc()
	}

	data, _ := json.Marshal(args)
	stat = cl.PubAction(thingID, key, data)

	if stat.Error != "" {
		return stat, errors.New(stat.Error)
	}

	// only once completed will there be a result a return
	if stat.Status == api.DeliveryCompleted {
		err = json.Unmarshal(stat.Reply, resp)
		if err != nil {
			stat.Error = err.Error()
		}
	}
	if stat.Status == api.DeliveryDelivered {
		// TODO: wait for async response if status is delivered to the agent.
		// The agent will send a delivery update asap.
	}
	return stat, err
}

// SendDeliveryUpdate sends a delivery status update to the hub.
// The hub's digitwin will update the status of the action and notify the original sender.
//
// Intended for agents that have processed an incoming action request and need to send
// a reply and confirm that the action has applied.
func (cl *HttpSSEClient) SendDeliveryUpdate(thingID string, stat api.DeliveryStatus) {
	//thingID := inbox.RawThingID // tbd, is this the inbox service?
	statJSON, _ := json.Marshal(&stat)
	cl.PubEvent(thingID, vocab.EventTypeDeliveryUpdate, statJSON)
}

// SetConnectHandler sets the notification handler of connection status changes
func (cl *HttpSSEClient) SetConnectHandler(cb func(status hubclient.HubTransportStatus)) {
	cl.mux.Lock()
	cl._connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the single handler that receives all subscribed events.
// Use 'Subscribe' to set the addresses that this receives events on.
func (cl *HttpSSEClient) SetMessageHandler(cb api.MessageHandler) {
	cl.mux.Lock()
	cl._messageHandler = cb
	cl.mux.Unlock()
}

// startTMaxSSEListener starts the SSE listener using the tmaxmax/go-sse/v2 library
// This will invoke tp._eventHandler if set.
// This ends when _sseChan is closed.
//func (tp *HttpSSEClient) startR3labsSSEListener() {
//
//	tp.sseClient = sse.NewClient(sseURL)
//	tp.sseClient.Connection = client
//	tp.sseClient.Headers["Authorization"] = "bearer " + tp.bearerToken
//
//	tp.sseClient.OnDisconnect(func(c *sse.Client) {
//		slog.Warn("ConnectSSE: disconnected")
//	})
//	go func() {
//		// FIXME: stream name, multiple subscriptions?
//		err := tp.sseClient.Subscribe("", func(sseMsg *sse.Event) {
//
//			slog.Info("startSSEListener. Received message",
//				slog.String("Comment", string(sseMsg.Comment)),
//				slog.String("ID", string(sseMsg.ID)),
//				slog.String("event", string(sseMsg.Event)),
//				slog.Int("size", len(sseMsg.Data)),
//			)
//			if tp._eventHandler != nil {
//				addr := string(sseMsg.ID)
//				parts := strings.Split(addr, "/")
//				key := ""
//				thingID := parts[0]
//				if len(parts) > 1 {
//					key = parts[1]
//				}
//				tp._eventHandler(thingID, key, sseMsg.Data)
//			}
//		})
//		slog.Info("SSE listener ended")
//		if err != nil {
//			slog.Warn("StartSSEListener error", "err", err)
//		}
//	}()
//}

// Subscribe subscribes to a topic.
// Use SetEventHandler to receive subscribed events or SetRequestHandler for actions
func (cl *HttpSSEClient) Subscribe(thingID string) error {
	go func() {
		//err := cl.sseClient.SubscribeRaw(func(sseMsg *sse.Event) {
		//	slog.Info("SubscribeRaw received sse event:", "msg", sseMsg)
		//})
		//_ = err
	}()
	// TODO: subscribe to what?
	return nil
}

func (cl *HttpSSEClient) Unsubscribe(thingID string) {

}

// NewHttpSSEClient creates a new instance of the htpp client with a SSE return-channel.
//
// fullURL is the url with schema. If omitted this uses the in-memory UDS address,
// which only works with ConnectWithToken.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	keyPair with previously saved serialized public/private key pair, or "" to create one
//	caCert of the server to validate the server or nil to not check the server cert
func NewHttpSSEClient(hostPort string, clientID string, caCert *x509.Certificate) *HttpSSEClient {
	tp := HttpSSEClient{
		clientID: clientID,
		caCert:   caCert,
		hostPort: hostPort,
		ssePath:  vocab.ConnectSSEPath,
		_sseChan: make(chan *sse.Event),
	}
	return &tp
}
