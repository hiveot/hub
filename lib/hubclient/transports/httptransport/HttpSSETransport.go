package httptransport

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient/transports"
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

// HttpSSETransport manages the hub server connection with hub event and action messaging using autopaho.
// This implements the IHubTransport interface.
type HttpSSETransport struct {
	hostPort string
	ssePath  string
	clientID string
	caCert   *x509.Certificate

	timeout time.Duration // request timeout
	// _ variables are mux protected
	mux             sync.RWMutex
	sseClient       *sse.Client
	tlsClient       *tlsclient.TLSClient
	_authToken      string
	_status         transports.HubTransportStatus
	_sseChan        chan *sse.Event
	_subscriptions  map[string]bool
	_connectHandler func(status transports.HubTransportStatus)
	_messageHandler api.MessageHandler
}

// ConnectSSE establishes a sse session over the Hub HTTPS connection.
// All hub messages are send as type ThingMessage, containing thingID, key, payload and sender
func (tp *HttpSSETransport) ConnectSSE(client *http.Client) error {
	// FIXME. what if serverURL contains a schema?
	sseURL := fmt.Sprintf("https://%s%s", tp.hostPort, tp.ssePath)

	//r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, sseURL, http.NoBody)
	r, _ := tp.tlsClient.NewRequest("GET", sseURL, []byte{})
	//r.Header.Set("Content-Type", "text/event-stream")
	r.Header.Set("Content-Type", "application/json")

	conn := sse.NewConnection(r)

	conn.SubscribeToAll(tp.handleSSEEvent)
	go func() {
		if err := conn.Connect(); err != nil {
			slog.Error("SSE connection failed", "err", err.Error())
		}
	}()
	//c.Connection = client
	//tp.startR3labsSSEListener()
	return nil
}

// ConnectWithCert connects to the Hub TLS server using client certifcate authentication
func (tp *HttpSSETransport) ConnectWithCert(kp keys.IHiveKey, cert *tls.Certificate) (token string, err error) {
	tp.mux.RLock()
	defer tp.mux.RUnlock()
	return "", fmt.Errorf("ConnectWithCert not yet supported by the HTTPS/SSE binding")
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password.
func (tp *HttpSSETransport) ConnectWithPassword(password string) error {
	tp.mux.RLock()
	defer tp.mux.RUnlock()
	if tp.tlsClient != nil {
		return fmt.Errorf("already connected")
	}

	cl := tlsclient.NewTLSClient(tp.hostPort, tp.caCert, time.Second*120)
	token, err := cl.ConnectWithPassword(tp.clientID, password)
	if err == nil {
		tp.tlsClient = cl
		tp._authToken = token

		// establish the sse connection if a path is set
		if tp.ssePath != "" {
			err = tp.ConnectSSE(cl.GetHttpClient())
		}
	}
	return err
}

// ConnectWithJWT connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
//
//	jwtToken is the token obtained with login or refresh.
func (tp *HttpSSETransport) ConnectWithJWT(jwtToken string) (err error) {
	tp.mux.RLock()
	defer tp.mux.RUnlock()
	if tp.tlsClient != nil {
		return fmt.Errorf("already connected")
	}

	tp.tlsClient = tlsclient.NewTLSClient(tp.hostPort, tp.caCert, time.Second*120)
	tp.tlsClient.ConnectWithToken(tp.clientID, jwtToken)

	// establish the sse connection if a path is set
	if tp.ssePath != "" {
		err = tp.ConnectSSE(tp.tlsClient.GetHttpClient())
	}

	return err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (tp *HttpSSETransport) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeECDSA)
	return k
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (tp *HttpSSETransport) Disconnect() {
	tp.mux.Lock()
	close(tp._sseChan)
	tp._sseChan = nil
	tp.mux.Unlock()
	tp.tlsClient.Close()
}

// GetStatus Return the transport connection info
func (tp *HttpSSETransport) GetStatus() transports.HubTransportStatus {
	tp.mux.RLock()
	defer tp.mux.RUnlock()
	return tp._status
}

func (tp *HttpSSETransport) GetTlsClient() *tlsclient.TLSClient {
	return tp.tlsClient
}

// handleSSEEvent processes the push-event received from the hub.
// This is passed on to the client, which must provide a delivery applied, completed or error status.
// This sends the delivery  status to the hub using a delivery event.
func (tp *HttpSSETransport) handleSSEEvent(event sse.Event) {
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
	if tp._messageHandler == nil {
		stat.Error = "handleSSEEvent no handler is set, event ignored"
		stat.Status = api.DeliveryFailed
	} else {
		stat = tp._messageHandler(&rxMsg)
	}
	// only actions need a delivery update
	if rxMsg.MessageType == vocab.MessageTypeAction {
		stat.MessageID = rxMsg.MessageID
		tp.SendDeliveryUpdate(rxMsg.ThingID, stat)
	}
}

// SendDeliveryUpdate sends a delivery status update to the hub.
// The hub's digitwin will update the status of the action and notify the original sender.
//
// Intended for agents that have processed an incoming action request and need to send
// a reply and confirm that the action has applied.
func (tp *HttpSSETransport) SendDeliveryUpdate(thingID string, stat api.DeliveryStatus) {
	//thingID := inbox.RawThingID // tbd, is this the inbox service?
	statJSON, _ := json.Marshal(&stat)
	tp.PubEvent(thingID, vocab.EventTypeDeliveryUpdate, statJSON)
}

// PubAction publishes an action message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
func (tp *HttpSSETransport) PubAction(thingID string, key string, payload []byte) (stat api.DeliveryStatus) {
	slog.Debug("PubEvent",
		slog.String("thingID", thingID), slog.String("key", key))
	vars := map[string]string{"thingID": thingID, "key": key}
	eventPath := utils.Substitute(vocab.PostActionPath, vars)
	resp, err := tp.tlsClient.Post(eventPath, payload)
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
func (tp *HttpSSETransport) PubEvent(thingID string, key string, payload []byte) (stat api.DeliveryStatus) {
	slog.Debug("PubEvent",
		slog.String("thingID", thingID), slog.String("key", key))
	vars := map[string]string{"thingID": thingID, "key": key}
	eventPath := utils.Substitute(vocab.PostEventPath, vars)
	resp, err := tp.tlsClient.Post(eventPath, payload)
	if err == nil {
		err = json.Unmarshal(resp, &stat)
	}
	if err != nil {
		stat.Status = api.DeliveryFailed
		stat.Error = err.Error()
	}
	return stat
}

// SetConnectHandler sets the notification handler of connection status changes
func (tp *HttpSSETransport) SetConnectHandler(cb func(status transports.HubTransportStatus)) {
	tp.mux.Lock()
	tp._connectHandler = cb
	tp.mux.Unlock()
}

// SetMessageHandler set the single handler that receives all subscribed events.
// Use 'Subscribe' to set the addresses that this receives events on.
func (tp *HttpSSETransport) SetMessageHandler(cb api.MessageHandler) {
	tp.mux.Lock()
	tp._messageHandler = cb
	tp.mux.Unlock()
}

// startTMaxSSEListener starts the SSE listener using the tmaxmax/go-sse/v2 library
// This will invoke tp._eventHandler if set.
// This ends when _sseChan is closed.
//func (tp *HttpSSETransport) startR3labsSSEListener() {
//
//	tp.sseClient = sse.NewClient(sseURL)
//	tp.sseClient.Connection = client
//	tp.sseClient.Headers["Authorization"] = "bearer " + tp._authToken
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
func (tp *HttpSSETransport) Subscribe(thingID string) error {
	go func() {
		//err := tp.sseClient.SubscribeRaw(func(sseMsg *sse.Event) {
		//	slog.Info("SubscribeRaw received sse event:", "msg", sseMsg)
		//})
		//_ = err
	}()
	// TODO: subscribe to what?
	return nil
}

func (tp *HttpSSETransport) Unsubscribe(thingID string) {

}

// NewHttpSSETransport creates a new instance of the htpp client with a SSE return-channel.
//
// fullURL is the url with schema. If omitted this uses the in-memory UDS address,
// which only works with ConnectWithToken.
//
//	hostPort of broker to connect to, without the scheme
//	ssePath path on the server of the SSE connection handler for setting up a SSE event channel
//	clientID to connect as
//	keyPair with previously saved serialized public/private key pair, or "" to create one
//	caCert of the server to validate the server or nil to not check the server cert
func NewHttpSSETransport(hostPort string, ssePath string, clientID string, caCert *x509.Certificate) *HttpSSETransport {
	tp := HttpSSETransport{
		clientID: clientID,
		caCert:   caCert,
		hostPort: hostPort,
		ssePath:  ssePath,
		_sseChan: make(chan *sse.Event),
	}
	return &tp
}
