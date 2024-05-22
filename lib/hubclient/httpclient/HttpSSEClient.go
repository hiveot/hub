package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
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
	sseCancelFn        context.CancelFunc
	mux                sync.RWMutex
	sseClient          *sse.Client
	tlsClient          *tlsclient.TLSClient
	bearerToken        string
	_maxSSEMessageSize int
	_status            hubclient.HubTransportStatus
	_sseChan           chan *sse.Event
	_subscriptions     map[string]bool
	_connectHandler    func(status hubclient.HubTransportStatus)
	// client side handler that receives actions from the server
	_actionHandler api.MessageHandler
	// client side handler that receives all non-action messages from the server
	_eventHandler api.MessageHandler
	// map of messageID to delivery status update channel
	_correlData map[string]chan *api.DeliveryStatus
}

// ClientID returns the client's connection ID
func (cl *HttpSSEClient) ClientID() string {
	return cl.clientID
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

	conn := sse.NewConnection(req)
	// increase buffer size to 1M
	// TODO: make limit configurable
	//https://github.com/tmaxmax/go-sse/issues/32
	newBuf := make([]byte, 0, 1024*65)
	conn.Buffer(newBuf, cl._maxSSEMessageSize)

	//conn.Parser.Buffer = make([]byte, 1000000) // test 1MB buffer

	remover := conn.SubscribeToAll(cl.handleSSEEvent)
	_ = remover
	go func() {
		// connect and report an error if connection ends due to reason other than context cancelled
		if err := conn.Connect(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("SSE connection failed", "err", err.Error())
		}
		remover()
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

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
//
//	jwtToken is the token obtained with login or refresh.
func (cl *HttpSSEClient) ConnectWithToken(jwtToken string) (newToken string, err error) {
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
	var stat api.DeliveryStatus

	rxMsg := &things.ThingMessage{}
	err := json.Unmarshal([]byte(event.Data), rxMsg)
	if err != nil {
		slog.Error("handleSSEEvent; Received non-ThingMessage sse event. Ignored",
			"eventType", event.Type,
			"LastEventID", event.LastEventID)
		return
	}
	stat.MessageID = rxMsg.MessageID
	slog.Info("handleSSEEvent. Received message",
		//slog.String("Comment", string(event.Comment)),
		slog.String("me", cl.clientID),
		slog.String("messageType", rxMsg.MessageType),
		slog.String("thingID", rxMsg.ThingID),
		slog.String("key", rxMsg.Key),
		slog.String("messageID", rxMsg.MessageID),
		slog.String("senderID", rxMsg.SenderID),
	)

	// always handle rpc response
	if rxMsg.MessageType == vocab.MessageTypeEvent && rxMsg.Key == vocab.EventTypeDeliveryUpdate {
		// this client is receiving a delivery update from a previous action.
		cl.mux.RLock()
		err = json.Unmarshal(rxMsg.Data, &stat)
		rChan, _ := cl._correlData[stat.MessageID]
		cl.mux.RUnlock()
		if rChan != nil {
			rChan <- &stat
			// if status == DeliveryCompleted || status == DeliveryFailed {
			cl.mux.Lock()
			delete(cl._correlData, rxMsg.MessageID)
			cl.mux.Unlock()
			return
		} else if cl._eventHandler != nil {
			// pass event to client as this is an unsolicited event
			// it could be a delayed confirmation of delivery
			stat = cl._eventHandler(rxMsg)
		} else {
			// missing rpc or message handler
			slog.Error("handleSSEEvent, no handler registered for client",
				"clientID", cl.ClientID())
			stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, delivery update ignored"))
		}
	} else if rxMsg.MessageType == vocab.MessageTypeEvent && cl._eventHandler != nil {
		// pass event to handler, if set
		stat = cl._eventHandler(rxMsg)
	} else if rxMsg.MessageType == vocab.MessageTypeAction && cl._actionHandler != nil {
		// pass action to agent for delivery to thing
		stat = cl._actionHandler(rxMsg)
	} else {
		slog.Error("handleSSEEvent, no handler registered for client",
			"clientID", cl.ClientID())
		stat.Failed(rxMsg, fmt.Errorf("handleSSEEvent no handler is set, message ignored"))
	}
	// only actions need a delivery update
	if rxMsg.MessageType == vocab.MessageTypeAction {
		cl.SendDeliveryUpdate(rxMsg.ThingID, stat)
	}
}

// PubAction publishes an action message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
// An error is returned if delivery failed or succeeded but the action itself failed
func (cl *HttpSSEClient) PubAction(thingID string, key string, payload []byte) (stat api.DeliveryStatus, err error) {
	slog.Info("PubAction",
		slog.String("thingID", thingID), slog.String("key", key))
	vars := map[string]string{"thingID": thingID, "key": key}
	eventPath := utils.Substitute(vocab.PostActionPath, vars)
	// the reply is a delivery status
	resp, err := cl.tlsClient.Post(eventPath, payload)
	if err == nil {
		err = json.Unmarshal(resp, &stat)
	}
	if err != nil {
		stat.Status = api.DeliveryFailed
		stat.Error = err.Error()
	} else if stat.Error != "" {
		// if the status contains an error return it instead
		err = errors.New(stat.Error)
	}
	return stat, err
}

// PubActionWithQueryParams publishes an action with query parameters
func (cl *HttpSSEClient) PubActionWithQueryParams(
	thingID string, key string, payload []byte, params map[string]string) (stat api.DeliveryStatus, err error) {
	slog.Info("PubActionWithQueryParams",
		slog.String("thingID", thingID),
		slog.String("key", key),
	)

	vars := map[string]string{"thingID": thingID, "key": key}
	actionPath := utils.Substitute(vocab.PostActionPath, vars)
	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, actionPath)
	// the reply is a delivery status
	reply, err := cl.tlsClient.Invoke("POST", serverURL, payload, params)
	if err == nil {
		err = json.Unmarshal(reply, &stat)
	}
	if err != nil {
		stat.Status = api.DeliveryFailed
		stat.Error = err.Error()
	} else if stat.Error != "" {
		// if the status contains an error return it instead
		err = errors.New(stat.Error)
	}
	return stat, err
}

// PubEvent publishes an event message and returns
func (cl *HttpSSEClient) PubEvent(
	thingID string, key string, payload []byte) (stat api.DeliveryStatus, err error) {

	slog.Info("PubEvent",
		slog.String("me", cl.clientID),
		slog.String("device thingID", thingID),
		slog.String("key", key),
	)
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
	return stat, err
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
	resp, err := cl.tlsClient.Invoke("POST", refreshURL, data, nil)
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
func (cl *HttpSSEClient) Rpc(
	thingID string, key string, args interface{}, resp interface{}) (err error) {

	var data []byte
	if args != nil {
		data, _ = json.Marshal(args)
	}
	// a messageID is needed before the action is published in order to match it with the reply
	messageID := "rpc-" + uuid.NewString()
	rChan := make(chan *api.DeliveryStatus)
	cl.mux.Lock()
	cl._correlData[messageID] = rChan
	cl.mux.Unlock()

	// invoke with query parameters to provide the message ID
	qparams := map[string]string{"messageID": messageID}
	stat, err := cl.PubActionWithQueryParams(thingID, key, data, qparams)

	// wait for a response on the message channel
	if err == nil && !(stat.Status == api.DeliveryCompleted || stat.Status == api.DeliveryFailed) {
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
	if err == nil && stat.Status != api.DeliveryCompleted {
		// delivery not
		err = errors.New("Delivery not complete. Status: " + stat.Status)
	}
	// only once completed will there be a reply as a result
	if err == nil && resp != nil {
		err = json.Unmarshal(stat.Reply, resp)
	}
	return err
}

// SendDeliveryUpdate sends a delivery status update to the hub.
// The hub's inbox will update the status of the action and notify the original sender.
//
// Intended for agents that have processed an incoming action request and need to send
// a reply and confirm that the action has applied.
func (cl *HttpSSEClient) SendDeliveryUpdate(thingID string, stat api.DeliveryStatus) {
	statJSON, _ := json.Marshal(&stat)
	// thing
	cl.PubEvent(thingID, vocab.EventTypeDeliveryUpdate, statJSON)
}

// SetConnectHandler sets the notification handler of connection status changes
func (cl *HttpSSEClient) SetConnectHandler(cb func(status hubclient.HubTransportStatus)) {
	cl.mux.Lock()
	cl._connectHandler = cb
	cl.mux.Unlock()
}

// SetActionHandler set the single handler that receives all actions directed to this client.
// Actions do not need subscription.
func (cl *HttpSSEClient) SetActionHandler(cb api.MessageHandler) {
	cl.mux.Lock()
	cl._actionHandler = cb
	cl.mux.Unlock()
}

// SetEventHandler set the single handler that receives all subscribed events.
func (cl *HttpSSEClient) SetEventHandler(cb api.MessageHandler) {
	cl.mux.Lock()
	cl._eventHandler = cb
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

// WaitForStatusUpdate waits for an async status update or until timeout
// This returns the status or an error if the messageID doesn't exist
func (cl *HttpSSEClient) WaitForStatusUpdate(
	statChan chan *api.DeliveryStatus, messageID string, timeout time.Duration) (stat api.DeliveryStatus, err error) {

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
// which only works with ConnectWithToken.
//
//	hostPort of broker to connect to, without the scheme
//	clientID to connect as
//	keyPair with previously saved serialized public/private key pair, or "" to create one
//	caCert of the server to validate the server or nil to not check the server cert
func NewHttpSSEClient(hostPort string, clientID string, caCert *x509.Certificate) *HttpSSEClient {
	tp := HttpSSEClient{
		timeout:     time.Minute,
		clientID:    clientID,
		caCert:      caCert,
		hostPort:    hostPort,
		ssePath:     vocab.ConnectSSEPath,
		_sseChan:    make(chan *sse.Event),
		_correlData: make(map[string]chan *api.DeliveryStatus),
		// max message size for bulk reads is 10MB.
		_maxSSEMessageSize: 1024 * 1024 * 10,
	}
	return &tp
}
