package mqttclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Connecting with UDS for local services. Might not work with autopaho
// FIXME: UDS isn't supported by autopaho
const (
	MqttInMemUDSProd = "@/MqttInMemUDSProd" // production UDS name
	MqttInMemUDSTest = "@/MqttInMemUDSTest" // test server UDS name
)

const (
	ConnStatConnected    = "connected"
	ConnStatConnecting   = "connecting"
	ConnStatUnauthorized = "unauthorized"
	ConnStatDisconnected = "disconnected"
)

// InboxTopicFormat is the INBOX subscription topic used by the client and RPC calls
// _INBOX/{clientID}    (clientID is the unique session clientID, not per-se the loginID)
const INBOX_PREFIX = "INBOX"
const InboxTopicFormat = INBOX_PREFIX + "/%s"
const keepAliveInterval = 30 // seconds
const withQos = 1

// MqttClientConnection provides WoT protocol binding for the MQTT protocol
// This implements the IClientConnection interface.
type MqttClientConnection struct {

	// handler for requests send by clients
	appConnectHandlerPtr atomic.Pointer[messaging.ConnectionHandler]

	// handler for notifications sent by agents
	appNotificationHandler messaging.NotificationHandler
	// handler for requests send by clients
	appRequestHandlerPtr atomic.Pointer[messaging.RequestHandler]
	// handler for responses sent by agents
	appResponseHandlerPtr atomic.Pointer[messaging.ResponseHandler]

	cinfo messaging.ConnectionInfo

	authToken string

	// paho mqtt client
	pahoClient *autopaho.ConnectionManager
	// enable debug logging in the paho client
	logDebug bool
	// handler to obtain a form for the operation
	getForm func(op, thingID, name string) *td.Form

	//
	inboxTopic       string // set on init
	connectionID     string // unique connection ID
	connectionStatus string
	isConnected      atomic.Bool
	lastError        atomic.Pointer[error]
	//
	correlData    map[string]chan *paho.Publish
	subscriptions map[string]bool

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// request timeout
	timeout time.Duration

	// the agent handles requests
	agentRequestHandler func(m *paho.Publish)
}

// ConnectWithToken establishes a connection to the MQTT broker using the paho client
func (cl *MqttClientConnection) ConnectWithToken(token string) (newToken string, err error) {
	// setup TLS
	caCertPool := x509.NewCertPool()
	if cl.cinfo.CaCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", cl.cinfo.ConnectURL))
	} else {
		caCertPool.AddCert(cl.cinfo.CaCert)
	}
	tlsCfg := &tls.Config{
		RootCAs: caCertPool,
		//Certificates:       clientCertList,
		InsecureSkipVerify: cl.cinfo.CaCert == nil,
	}

	//safeConn := packets.NewThreadSafeConn(conn)
	// Setup the Paho client configuration
	hostName, _ := os.Hostname()
	connectID := fmt.Sprintf("%s-%s-%s", cl.cinfo.ClientID, hostName, time.Now().Format("20060102150405.000"))
	logger := log.Default()
	u, err := url.Parse(cl.cinfo.ConnectURL)
	autoCfg := autopaho.ClientConfig{
		BrokerUrls: []*url.URL{u},
		PahoErrors: logger,
		ClientConfig: paho.ClientConfig{
			ClientID: connectID, // instance ID, not the clientID
			//Conn:          safeConn,    // autopaho ignores this :(
			PacketTimeout: cl.timeout,
			Router: paho.NewSingleHandlerRouter(func(m *paho.Publish) {
				cl.handlePahoMessage(m)
			}),
		},
		TlsCfg: tlsCfg,
		// CleanStartOnInitialConnection defaults to false.
		// Setting this to true will clear the session on the first connection.
		//CleanStartOnInitialConnection: true,
		KeepAlive: 20, // Keepalive message should be sent every 20 seconds
	}
	autoCfg.ConnectUsername = cl.cinfo.ClientID
	autoCfg.ConnectPassword = []byte(cl.authToken)
	autoCfg.OnConnectError = cl.onPahoConnectionError
	autoCfg.OnConnectionUp = cl.onPahoConnect
	autoCfg.OnServerDisconnect = func(disconnect *paho.Disconnect) {
		cl.isConnected.Store(false)
		hPtr := cl.appConnectHandlerPtr.Load()
		if hPtr != nil {
			go (*hPtr)(false, nil, cl)
		}
	}

	if cl.logDebug {
		autoCfg.PahoDebug = logger
	}

	// Warning, can't use WithTimeout as it will disconnect the perfectly good
	// connection after the timeout has passed.
	ctx := context.Background()

	pcl, err := autopaho.NewConnection(ctx, autoCfg)

	cl.mux.Lock()
	cl.connectionID = connectID
	cl.pahoClient = pcl
	cl.mux.Unlock()

	// Wait for the connection to come up
	ctx, cancelFn := context.WithTimeout(ctx, time.Second*1)
	err = pcl.AwaitConnection(ctx)
	cancelFn()
	if err != nil {
		// provide a more meaningful error, the actual error is not returned by paho
		cl.mux.RLock()
		errptr := cl.lastError.Load()
		err = *errptr
		cl.mux.RUnlock()
	}
	// the onPahoConnect handler (re)subscribes to the inbox topic
	return token, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *MqttTransportClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeECDSA)
//	return k
//}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (cl *MqttClientConnection) Disconnect() {
	cl.mux.Lock()
	pcl := cl.pahoClient
	connectID := cl.connectionID

	cl.pahoClient = nil
	cl.isConnected.Store(false)
	err := errors.New("disconnected by user")
	cl.lastError.Store(&err)
	cl.mux.Unlock()

	slog.Info("Disconnecting", "cid", connectID)
	if pcl != nil {
		//time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		err := pcl.Disconnect(context.Background())
		if err != nil {
			slog.Error("disconnect error", "err", err)
		}
	}
}

// GetConnectionInfo returns the connection information
func (cl *MqttClientConnection) GetConnectionInfo() messaging.ConnectionInfo {
	return cl.cinfo
}

//// handle receiving an action status update.
//// This can be a response to a non-rpc request, or an update to a prior RPC
//// request that already received a response.
//// This is passed to the client as a notification.
//func (cl *MqttClientConnection) handleActionStatus(msg *transports.ThingMessage) {
//	// todo
//}

// HandleResponseMessage handles received consumer message
//func (cl *MqttClientConnection) HandleMqttMessage(rxMsg *transports.ThingMessage) {
//
//	switch rxMsg.Operation {
//	case wot.HTOpActionStatus:
//		// the consumer received an async update to the action request
//		// this client is receiving a status update from a previously sent action.
//		cl.handleActionStatus(rxMsg)
//	case ssescserver.SSEOpPong:
//		cl.handlePongMessage(rxMsg)
//	default:
//		// response to a read operation or notification of property or events
//		// no correlationID means this is just a notification without a response expected
//		cl.handleNotificationMessage(rxMsg)
//	}
//}

// Anything that isn't a request or response is passed up as a notification
//func (cl *MqttClientConnection) handleNotificationMessage(msg *transports.ThingMessage) {
//
//	// pass everything else to the message handler. No reply is sent.
//	// Eg" consumer receive event, property and TD updates
//	if cl.BaseNotificationHandler == nil {
//		slog.Warn("handleSseEvent, no message handler registered. Message ignored.",
//			slog.String("operation", msg.Operation),
//			slog.String("thingID", msg.ThingID),
//			slog.String("name", msg.Name),
//			slog.String("clientID", cl.GetClientID()))
//		return
//	}
//	cl.BaseNotificationHandler(msg)
//}

// handlePahoMessage handles incoming mqtt messages.
// This converts it to the standard request,response or notification envelope and
// passes it to the registered handler.
func (cl *MqttClientConnection) handlePahoMessage(m *paho.Publish) {
	slog.Debug("handlePahoMessage", slog.String("topic", m.Topic))

	// run this in the background to allow for reentrancy
	go func() {
		correlationID := ""
		if m.Properties.CorrelationData != nil {
			correlationID = string(m.Properties.CorrelationData)
		}

		// handle the response to a request from this consumer
		// responses have topic starting with the inbox prefix
		if strings.HasPrefix(m.Topic, INBOX_PREFIX) && correlationID != "" {
			resp := messaging.ResponseMessage{}
			resp.MessageType = messaging.MessageTypeResponse
			// mqtt payload are straight hiveot messages (for now)
			err := jsoniter.Unmarshal(m.Payload, &resp)
			if err != nil {
				slog.Warn("handlePahoMessage. Payload unmarshal failed",
					"topic", m.Topic,
					"correlationID", correlationID)
			} else {
				hPtr := cl.appResponseHandlerPtr.Load()
				if hPtr != nil {
					_ = (*hPtr)(&resp)
				}
			}
			return
		}
		// handle request message from consumer (move to agent)
		replyTo := m.Properties.ResponseTopic
		if replyTo != "" && correlationID != "" {
			if cl.agentRequestHandler == nil {
				slog.Error("handlePahoMessage: received request but this is a consumer")
			} else {
				cl.agentRequestHandler(m)
			}
			return
		}

		// this is a notification message with an event, property, TD update
		notif := messaging.ResponseMessage{}
		notif.MessageType = messaging.MessageTypeResponse
		err := jsoniter.Unmarshal(m.Payload, &notif)
		if err != nil {
			slog.Warn("handlePahoMessage. Notification unmarshal failed",
				"topic", m.Topic,
				"correlationID", correlationID)
		} else {
			hPtr := cl.appResponseHandlerPtr.Load()
			if hPtr != nil {
				_ = (*hPtr)(&notif)
			}
		}
	}()
}

// HandlePongMessage handles the response to ping
//func (cl *MqttClientConnection) handlePongMessage(msg *transports.ThingMessage) {
//	cl.BaseRnrChan.HandleResponse(msg)
//}

// InvokeAction invokes an action on a thing and wait for the response
//func (cl *MqttClientConnection) InvokeAction(dThingID, name string, input any, output any) error {
//	return cl.SendRequest(wot.OpInvokeAction, dThingID, name, input, output)
//}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *MqttClientConnection) IsConnected() bool {
	return cl.isConnected.Load()
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *MqttClientConnection) Logout() error {
	err := fmt.Errorf("Not implemented")
	return err
}

// once paho is connected
func (cl *MqttClientConnection) onPahoConnect(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	cl.mux.Lock()
	defer cl.mux.Unlock()

	cl.isConnected.Store(true)
	cl.lastError.Store(nil)
	// resubscribe previous subscriptions
	subList := make([]string, 0, len(cl.subscriptions))
	for topic := range cl.subscriptions {
		subList = append(subList, topic)
	}
	hPtr := cl.appConnectHandlerPtr.Load()

	go func() {
		// (re)subscribe all subscriptions
		for _, topic := range subList {
			err := cl.sub(topic)
			if err != nil {
				slog.Error("onConnect. resubscribe failed", "topic", topic)
			}
		}
		// last but not least, subscribe the inbox to receive responses to requests
		err := cl.SubscribeToTopic(cl.inboxTopic)
		if err != nil {
			slog.Error("Failed inbox subscription. Requests will not receive a response",
				"err", err, "inboxTopic", cl.inboxTopic)
		}

		// now subscriptions have been restored, inform subscriber
		if hPtr != nil {
			(*hPtr)(true, nil, cl)
		}
	}()
}

// paho reports an error but will keep trying until disconnect is called
func (cl *MqttClientConnection) onPahoConnectionError(err error) {
	go func() {
		connStatus := ConnStatConnecting
		connErr := err
		// possible causes:
		// 1. wrong credentials - inform user, dont repeat or do repeat?
		// 2. connection is interrupted - inform user/log, keep repeating
		// 3. server disconnects - inform user/log, keep repeating
		// 4. client disconnects - terminate
		switch et := err.(type) {
		case *autopaho.ConnackError:
			if et.ReasonCode == 134 {
				connStatus = ConnStatUnauthorized
				connErr = fmt.Errorf("Unauthorized: %s", et.Reason)
			} else {
				connStatus = ConnStatConnecting
				connErr = fmt.Errorf("%s: %w", et.Reason, err)
				//connErr = fmt.Errorf("disconnected user '%s': %s", cl.BaseClientID, err.Error())
			}
		default:
			connStatus = ConnStatConnecting
			connErr = fmt.Errorf("disconnected: %w", err)
			slog.Error("connection error", "clientID", cl.cinfo.ClientID, "err", err)
		}
		// notify on change
		cl.mux.RLock()
		oldStatus := cl.connectionStatus
		oldErrPtr := cl.lastError.Load()
		cl.mux.RUnlock()
		if connStatus != oldStatus || connErr != *oldErrPtr {
			cl.mux.Lock()
			//cl.connectionStatus = connStatus
			cl.lastError.Store(&connErr)
			cl.isConnected.Store(false)
			hPtr := cl.appConnectHandlerPtr.Load()
			cl.mux.Unlock()
			if hPtr != nil {
				(*hPtr)(false, connErr, cl)
			}
		}
		slog.Info("onPahoConnectionError", "err", connErr.Error())
		// don't retry on authentication error
		cl.mux.RLock()
		pcl := cl.pahoClient
		cl.mux.RUnlock()
		if connStatus == ConnStatUnauthorized && pcl != nil {
			_ = pcl.Disconnect(context.Background())
		}
	}()
}

// ParseResponse helper message to parse response and check for errors
func (cl *MqttClientConnection) _parseResponse(data []byte, resp interface{}) error {
	var err error
	if data == nil || len(data) == 0 {
		if resp != nil {
			err = fmt.Errorf("ParseResponse: client '%s', expected a response but none received",
				cl.cinfo.ClientID)
		} else {
			err = nil // all good
		}
	} else {
		if resp == nil {
			err = fmt.Errorf("ParseResponse: client '%s', received response but none was expected. data=%s",
				cl.cinfo.ClientID, data)
		} else {
			err = jsoniter.Unmarshal(data, resp)
		}
	}
	return err
}

// _pub publishes a message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
//func (cl *MqttClientConnection) _pub(topic string, payload []byte) (resp []byte, err error) {
//	slog.Debug("SendRequest", "topic", topic)
//
//	ctx, cancelFn := context.WithTimeout(context.BackgroundImage(), cl.BaseTimeout)
//	defer cancelFn()
//	//
//	//// FIXME! a deadlock can occur here
//	//cl.mux.RLock()
//	//pcl := cl.pahoClient
//	//cl.mux.RUnlock()
//
//	err = cl._send(topic, payload, cid)
//
//	//pubMsg := &paho.Publish{
//	//	QoS:     withQos,
//	//	Retain:  false,
//	//	Topic:   topic,
//	//	Payload: payload,
//	//	Properties: &paho.PublishProperties{
//	//		CorrelationData: []byte(cid),
//	//		ResponseTopic:   inboxTopic,
//	//		ContentType:     "json",
//	//		User: paho.UserProperties{{
//	//			Key:   "test",
//	//			Value: "test",
//	//		}},
//	//	},
//	//}
//	//_, err = pcl.Publish(ctx, pubMsg)
//	if err != nil {
//		return nil, err
//	}
//
//	// wait for response
//	var respMsg *paho.Publish
//	select {
//	case <-ctx.Done():
//		err = fmt.Errorf("timeout waiting for response")
//		break
//	case respMsg = <-rChan:
//		break
//	}
//	if err != nil {
//		return nil, err
//	}
//
//	// test alternative to handling errors since User properties aren't
//	// passed through for some reason.
//	if respMsg.Properties.ContentType == "error" {
//		err = errors.New(string(respMsg.Payload))
//		return nil, err
//	}
//
//	slog.Debug("SendRequest end:",
//		slog.String("topic", topic),
//		slog.String("ContentType (if any)", respMsg.Properties.ContentType),
//	)
//	return respMsg.Payload, err
//}

//// RefreshToken refreshes the authentication token
//// The resulting token can be used with 'SetBearerToken'
//// This is specific to the Hiveot Hub.
//func (cl *MqttClientConnection) RefreshToken(oldToken string) (newToken string, err error) {
//	return oldToken, fmt.Errorf("not implemented")
//}

// Send a request message to a topic
func (cl *MqttClientConnection) _send(topic string, msg any, correlationID string) error {

	slog.Info("_send", slog.String("topic", topic))

	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	payload, _ := jsoniter.Marshal(msg)
	pahoMsg := paho.Publish{
		QoS:     1,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
		Properties: &paho.PublishProperties{
			CorrelationData: []byte(correlationID),
			ResponseTopic:   cl.inboxTopic,
			ContentType:     "json",
			User: paho.UserProperties{{
				Key:   "test",
				Value: "test",
			}},
		},
	}
	cl.mux.RLock()
	pcl := cl.pahoClient
	cl.mux.RUnlock()
	resp, err := pcl.Publish(ctx, &pahoMsg)
	_ = resp
	cancelFn()
	return err
}

// SendNotification send a notification message
func (cl *MqttClientConnection) SendNotification(resp *messaging.NotificationMessage) error {
	panic("todo: implement")
}

// SendRequest send a request message over websockets
// This transforms the request to the protocol message and sends it to the server.
func (cl *MqttClientConnection) SendRequest(req *messaging.RequestMessage) error {
	panic("todo: implement")
}

// SendResponse send a response message
// This transforms the response to the protocol message and sends it to the server.
// Responses without correlationID are subscription notifications.
func (cl *MqttClientConnection) SendResponse(resp *messaging.ResponseMessage) error {
	panic("todo: implement")
}

// SetConnectHandler set the application handler for connection status updates
func (cl *MqttClientConnection) SetConnectHandler(cb messaging.ConnectionHandler) {
	if cb == nil {
		cl.appConnectHandlerPtr.Store(nil)
	} else {
		cl.appConnectHandlerPtr.Store(&cb)
	}
}

// SetNotificationHandler set the application handler for received notifications
func (cc *MqttClientConnection) SetNotificationHandler(cb messaging.NotificationHandler) {
	cc.mux.Lock()
	cc.appNotificationHandler = cb
	cc.mux.Unlock()
}

// SetRequestHandler set the application handler for incoming requests
func (cl *MqttClientConnection) SetRequestHandler(cb messaging.RequestHandler) {
	if cb == nil {
		cl.appRequestHandlerPtr.Store(nil)
	} else {
		cl.appRequestHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler set the application handler for received responses
func (cl *MqttClientConnection) SetResponseHandler(cb messaging.ResponseHandler) {
	if cb == nil {
		cl.appResponseHandlerPtr.Store(nil)
	} else {
		cl.appResponseHandlerPtr.Store(&cb)
	}
}

// SubscribeToTopic subscribes to a topic.
// Incoming messages are passed to the event or request handler, depending on whether
// a reply-to address and correlation-ID is set.
func (cl *MqttClientConnection) SubscribeToTopic(topic string) error {
	slog.Debug("SubscribeToTopic", "topic", topic)
	err := cl.sub(topic)
	if err != nil {
		return err
	}
	cl.mux.Lock()
	cl.subscriptions[topic] = true
	cl.mux.Unlock()
	return err
}
func (cl *MqttClientConnection) UnsubscribeFromTopic(topic string) {
	packet := &paho.Unsubscribe{
		Topics: []string{topic},
	}
	cl.mux.RLock()
	pcl := cl.pahoClient
	cl.mux.RUnlock()

	ack, err := pcl.Unsubscribe(context.Background(), packet)
	_ = ack
	if err != nil {
		slog.Error("Unable to unsubscribe from topic", "topic", topic)
		return
	}
	cl.mux.Lock()
	delete(cl.subscriptions, topic)
	cl.mux.Unlock()
}

//
//// SendRequest sends a request and waits for a result
//// The operation is used to retrieve the form of the Thing whose operation to
//// send and determine the endpoint. If no form can be retrieved this falls
//// back to the hub known endpoint.
//func (cl *MqttClientConnection) SendRequest(
//	operation string, thingID, name string, input interface{}, output interface{}) error {
//	return fmt.Errorf("not implemented")
//}

//
//// SendResponse [agent] sends a operation response to the server.
//// (todo)
//func (cl *MqttTransportClient) SendResponse(correlationID string, data any) {
//	topic := ""
//	payload, _ := jsoniter.Marshal(data)
//	cl._send(topic, payload)
//}

// sub builds a subscribe packet and submits it
func (cl *MqttClientConnection) sub(topic string) error {
	packet := &paho.Subscribe{
		Properties: nil,
		Subscriptions: []paho.SubscribeOptions{
			{
				Topic: topic,
				QoS:   withQos,
			},
		},
	}
	cl.mux.RLock()
	pcl := cl.pahoClient
	cl.mux.RUnlock()
	suback, err := pcl.Subscribe(context.Background(), packet)
	_ = suback
	return err
}

// NewMqttConsumerClient creates a new instance of the mqtt binding client
//
//	fullURL of broker to connect to, including the schema
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler that provides a form for the given operation
//	timeout for waiting for response. 0 to use the default.
func NewMqttConsumerClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm messaging.GetFormHandler,
	timeout time.Duration) *MqttClientConnection {

	cl := MqttClientConnection{}

	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewMqttTransportClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", fullURL))
	} else {
		slog.Debug("NewHttpSSEClient: CA certificate",
			slog.String("destination", fullURL),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
	}
	if timeout == 0 {
		timeout = time.Second * 3
	}

	cl.cinfo.CaCert = caCert
	cl.cinfo.ClientID = clientID
	cl.connectionID = "mqtt-" + shortid.MustGenerate()
	cl.timeout = timeout
	cl.cinfo.ProtocolType = messaging.ProtocolTypeWotMQTTWSS
	//cl.rnrChan = NewRnRChan()

	// max delay 3 seconds before a response is expected
	cl.cinfo.ConnectURL = fullURL

	cl.getForm = getForm
	cl.correlData = make(map[string]chan *paho.Publish)
	cl.subscriptions = make(map[string]bool)

	// setup an inbox to reply to
	hostName, _ := os.Hostname()
	cl.connectionID = fmt.Sprintf("%s-%s-%s",
		cl.cinfo.ClientID, hostName, time.Now().Format("20060102150405.000"))
	cl.inboxTopic = fmt.Sprintf(InboxTopicFormat, cl.connectionID)

	var onConnection messaging.ConnectionHandler = func(connected bool, err error, c messaging.IConnection) {
		slog.Info("connection status change",
			"newStatus", connected,
			"lastError", err,
			"clientID", clientID)
	}
	cl.appConnectHandlerPtr.Store(&onConnection)
	//err = cl.pahoConnect()

	return &cl
}
