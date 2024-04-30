package httptransport

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/tmaxmax/go-sse"
	//"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// HttpSSETransport manages the hub server connection with hub event and action messaging using autopaho.
// This implements the IHubTransport interface.
type HttpSSETransport struct {
	serverURL string
	ssePath   string
	clientID  string
	caCert    *x509.Certificate

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
	_eventHandler   func(msg *things.ThingMessage)
	_requestHandler func(msg *things.ThingMessage) (reply []byte, err error, donotreply bool)
}

// ConnectSSE establishes a sse session over the Hub HTTPS connection.
// All hub messages are send as type ThingMessage, containing thingID, key, payload and sender
func (tp *HttpSSETransport) ConnectSSE(client *http.Client) error {
	// FIXME. what if serverURL contains a schema?
	sseURL := "https://" + tp.serverURL + tp.ssePath

	//r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, sseURL, http.NoBody)
	r, _ := tp.tlsClient.NewRequest("GET", sseURL, []byte{})
	//r.Header.Set("Content-Type", "text/event-stream")
	r.Header.Set("Content-Type", "application/json")

	conn := sse.NewConnection(r)

	conn.SubscribeToAll(func(event sse.Event) {
		slog.Info("received SSE msg")

		slog.Info("startSSEListener. Received message",
			//slog.String("Comment", string(event.Comment)),
			slog.String("LastEventID", string(event.LastEventID)),
			slog.String("event", string(event.Data)),
			slog.String("type", string(event.Type)),
			slog.Int("size", len(event.Data)),
		)
		if tp._eventHandler != nil {
			// These events are of type ThingMessage
			msg := things.ThingMessage{}
			err := json.Unmarshal([]byte(event.Data), &msg)
			if err != nil {
				slog.Error("ConnectSSE - subscribe; Received non-ThingMessage sse event. Ignored",
					"eventType", event.Type,
					"LastEventID", event.LastEventID)
				return
			}
			tp._eventHandler(&msg)
		}
	})
	go func() {
		if err := conn.Connect(); err != nil {
			slog.Error("SSE connection failed", "err", err.Error())
		}
	}()
	//c.Connection = client
	//tp.startR3labsSSEListener()
	return nil
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password.
func (tp *HttpSSETransport) ConnectWithPassword(password string) error {
	tp.mux.RLock()
	tp.mux.RUnlock()
	if tp.tlsClient != nil {
		return fmt.Errorf("already connected")
	}
	cl := tlsclient.NewTLSClient(tp.serverURL, tp.caCert, time.Second*120)
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

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
// A private key might be required in future.
// This supports UDS connections with @/path or unix://@/path
//
// TODO: encrypt token with server public key so a MIM won't be able to get the token
// TBD: can server send a connection nonce and verify token signature?
//
//	kp is the key-pair of this client
//	jwtToken is the token obtained with login or refresh.
func (tp *HttpSSETransport) ConnectWithToken(kp keys.IHiveKey, jwtToken string) (err error) {
	tp.mux.RLock()
	tp.mux.RUnlock()
	if tp.tlsClient != nil {
		return fmt.Errorf("already connected")
	}

	tp.tlsClient = tlsclient.NewTLSClient(tp.serverURL, tp.caCert, time.Second*120)
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

// PubEvent publishes a message and returns
func (tp *HttpSSETransport) PubEvent(thingID string, key string, payload []byte) (err error) {
	slog.Debug("PubEvent",
		slog.String("thingID", thingID), slog.String("key", key))
	vars := map[string]string{"thingID": thingID, "key": key}
	eventPath := utils.Substitute(vocab.PostEventPath, vars)
	_, err = tp.tlsClient.Post(eventPath, payload)
	return err
}

// PubRequest publishes an action message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
func (tp *HttpSSETransport) PubRequest(thingID string, key string, payload []byte) (resp []byte, err error) {
	slog.Debug("PubEvent",
		slog.String("thingID", thingID), slog.String("key", key))
	vars := map[string]string{"thingID": thingID, "key": key}
	eventPath := utils.Substitute(vocab.PostActionPath, vars)
	resp, err = tp.tlsClient.Post(eventPath, payload)
	return resp, err
}

// SetConnectHandler sets the notification handler of connection status changes
func (tp *HttpSSETransport) SetConnectHandler(cb func(status transports.HubTransportStatus)) {
	tp.mux.Lock()
	tp._connectHandler = cb
	tp.mux.Unlock()
}

// SetEventHandler set the single handler that receives all subscribed events.
// Use 'Subscribe' to set the addresses that this receives events on.
func (tp *HttpSSETransport) SetEventHandler(cb func(msg *things.ThingMessage)) {
	tp.mux.Lock()
	tp._eventHandler = cb
	tp.mux.Unlock()
}

// SetRequestHandler sets the handler that receives all subscribed requests.
// This does not provide routing as in most cases it is unnecessary overhead
// Use 'Subscribe' to set the addresses that this receives requests on.
func (tp *HttpSSETransport) SetRequestHandler(
	cb func(msg *things.ThingMessage) (reply []byte, err error, donotreply bool)) {

	tp.mux.Lock()
	tp._requestHandler = cb
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

// NewHttpTransport creates a new instance of the htpp client.
//
// fullURL is the url with schema. If omitted this uses the in-memory UDS address,
// which only works with ConnectWithToken.
//
//	fullURL of broker to connect to, starting with "tls://", "wss://", "unix://"
//	ssePath path on the server of the SSE connection handler for setting up a SSE event channel
//	clientID to connect as
//	keyPair with previously saved serialized public/private key pair, or "" to create one
//	caCert of the server to validate the server or nil to not check the server cert
func NewHttpTransport(fullURL string, ssePath string, clientID string, caCert *x509.Certificate) *HttpSSETransport {
	tp := HttpSSETransport{
		clientID:  clientID,
		caCert:    caCert,
		serverURL: fullURL,
		ssePath:   ssePath,
		_sseChan:  make(chan *sse.Event),
	}
	return &tp
}
