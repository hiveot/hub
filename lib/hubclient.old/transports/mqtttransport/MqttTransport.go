package mqtttransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/ser"
	"log"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// InboxTopicFormat is the INBOX subscription topic used by the client and RPC calls
// _INBOX/{clientID}    (clientID is the unique session clientID, not per-se the loginID)
const InboxTopicFormat = transports.MessageTypeINBOX + "/%s"

const keepAliveInterval = 30 // seconds
const reconnectDelay = 10 * time.Second
const withDebug = false
const withQos = 1

// Connecting with UDS for local services. Might not work with autopaho
// FIXME: UDS isn't supported by autopaho
const (
	MqttInMemUDSProd = "@/MqttInMemUDSProd" // production UDS name
	MqttInMemUDSTest = "@/MqttInMemUDSTest" // test server UDS name
)

// MqttHubTransport manages the hub server connection with hub event and action messaging using autopaho.
// This implements the IHubTransport interface.
type MqttHubTransport struct {
	serverURL string
	clientID  string
	caCert    *x509.Certificate
	//conn      net.Conn

	timeout time.Duration // request timeout
	// enable debug logging in the transport
	logDebug bool

	// track the request-response correlations. Access only when setting lock

	// _ variables are mux protected
	mux             sync.RWMutex
	_connectID      string
	_correlData     map[string]chan *paho.Publish
	_status         transports.HubTransportStatus
	_inboxTopic     string // set on first request, cleared on disconnect
	_pahoClient     *autopaho.ConnectionManager
	_subscriptions  map[string]bool
	_connectHandler func(status transports.HubTransportStatus)
	_eventHandler   func(addr string, payload []byte)
	_requestHandler func(addr string, payload []byte) (reply []byte, err error, donotreply bool)
}

// AddressTokens returns the address separator and wildcards
func (tp *MqttHubTransport) AddressTokens() (sep string, wc string, rem string) {
	return "/", "+", "#"
}

// Connect connects to a mqtt broker using the pre-established network connection
// This allows creating connections with any available means including tcp/tls/wss/uds/pipes
//
//	credentials is either the password or a signed JWT token
//	conn network connection to utilize
func (tp *MqttHubTransport) Connect(credentials string) error {
	tp.mux.RLock()
	pcl := tp._pahoClient
	tp.mux.RUnlock()
	if pcl != nil {
		return fmt.Errorf("already connected")
	}

	// clients must use a unique connection ID otherwise the previous connection will be dropped
	hostName, _ := os.Hostname()
	connectID := fmt.Sprintf("%s-%s-%s", tp.clientID, hostName, time.Now().Format("20060102150405.000"))
	slog.Info("Connect",
		"clientID", tp.clientID,
		"URL", tp.serverURL,
		"connectID", connectID)

	// setup TLS
	caCertPool := x509.NewCertPool()
	if tp.caCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", tp.serverURL))
	} else {
		caCertPool.AddCert(tp.caCert)
	}
	tlsCfg := &tls.Config{
		RootCAs: caCertPool,
		//Certificates:       clientCertList,
		InsecureSkipVerify: tp.caCert == nil,
	}

	// Determine URL. TODO: support for UDS
	u, err := url.Parse(tp.serverURL)
	if err != nil {
		return err
	}
	//connectContext, cancelFn := context.WithCancel(context.Background())

	//safeConn := packets.NewThreadSafeConn(conn)
	// Setup the Paho client configuration
	logger := log.Default()
	autoCfg := autopaho.ClientConfig{
		BrokerUrls: []*url.URL{u},
		PahoErrors: logger,
		ClientConfig: paho.ClientConfig{
			ClientID: connectID, // instance ID, not the clientID
			//Conn:          safeConn,    // autopaho ignores this :(
			PacketTimeout: tp.timeout,
			Router: paho.NewSingleHandlerRouter(func(m *paho.Publish) {
				tp.handleMessage(m)
			}),
		},
		TlsCfg: tlsCfg,
		// CleanStartOnInitialConnection defaults to false.
		// Setting this to true will clear the session on the first connection.
		//CleanStartOnInitialConnection: true,
		KeepAlive: 20, // Keepalive message should be sent every 20 seconds
	}
	autoCfg.SetUsernamePassword(tp.clientID, []byte(credentials))
	autoCfg.OnConnectError = tp.onPahoConnectionError
	autoCfg.OnConnectionUp = tp.onPahoConnect
	autoCfg.OnServerDisconnect = tp.onPahoServerDisconnect
	if tp.logDebug {
		autoCfg.PahoDebug = logger
	}

	// Warning, can't use WithTimeout as it will disconnect the perfectly good
	// connection after the timeout has passed.
	ctx := context.Background()
	pcl, err = autopaho.NewConnection(ctx, autoCfg)
	tp.mux.Lock()
	tp._connectID = connectID
	tp._pahoClient = pcl
	tp.mux.Unlock()

	// Wait for the connection to come up
	ctx, cancelFn := context.WithTimeout(ctx, time.Second*1)
	err = pcl.AwaitConnection(ctx)
	cancelFn()
	if err != nil {
		// provide a more meaningful error, the actual error is not returned by paho
		tp.mux.RLock()
		err = tp._status.LastError
		tp.mux.RUnlock()
	}
	return err
}

// ConnectWithCert connects to the Hub server using TLS client certificate.
//
//	brokerURL of the server.
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
//func (hc *MqttHubTransport) ConnectWithCert(clientCert tls.Certificate) error {
//
//	conn, err := tlsclient.ConnectTLS(hc.serverURL, &clientCert, hc.caCert)
//	if err != nil {
//		return err
//	}
//	err = hc.ConnectWithConn("", conn)
//	return err
//}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (tp *MqttHubTransport) ConnectWithPassword(password string) error {

	//conn, err := tlsclient.ConnectTLS(tp.serverURL, nil, tp.caCert)
	//if err != nil {
	//	return err
	//}
	//err = tp.ConnectWithConn(password, conn)
	err := tp.Connect(password)
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
func (tp *MqttHubTransport) ConnectWithToken(kp keys.IHiveKey, jwtToken string) error {
	//var conn net.Conn
	//var err error
	//if strings.HasPrefix(tp.serverURL, "unix://") {
	//	// mqtt server UDS listener doesn't use TLS
	//	// url.Parse doesn't recognize the @ in the path
	//	addr := tp.serverURL[7:]
	//	conn, err = net.Dial("unix", addr)
	//} else if strings.HasPrefix(tp.serverURL, "@") {
	//	// using a UDS in-memory path without scheme
	//	conn, err = net.Dial("unix", tp.serverURL)
	//} else {
	//	conn, err = tlsclient.ConnectTLS(tp.serverURL, nil, tp.caCert)
	//}
	//// TODO: pass in a signed auth nonce, if possible
	//if err == nil {
	//	// clientID from the token is used
	//	err = tp.ConnectWithConn(jwtToken, conn)
	//}
	err := tp.Connect(jwtToken)

	if err != nil {
		err = fmt.Errorf("ConnectWithToken failed: %w", err)
	}
	return err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (tp *MqttHubTransport) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeECDSA)
	return k
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (tp *MqttHubTransport) Disconnect() {
	tp.mux.Lock()
	pcl := tp._pahoClient
	connectID := tp._connectID

	tp._pahoClient = nil
	tp._inboxTopic = ""
	tp._connectID = ""
	tp._status.ConnectionStatus = transports.Disconnected
	tp._status.LastError = errors.New("disconnected by user")
	tp.mux.Unlock()

	slog.Info("Disconnecting", "cid", connectID)
	if pcl != nil {
		//time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		err := pcl.Disconnect(context.Background())
		if err != nil {
			slog.Error("disconnect error", "err", err)
		}
	}
}

// GetStatus Return the transport connection info
func (tp *MqttHubTransport) GetStatus() transports.HubTransportStatus {
	tp.mux.RLock()
	defer tp.mux.RUnlock()
	return tp._status
}

// handleMessage handles incoming request, reply and event messages
func (tp *MqttHubTransport) handleMessage(m *paho.Publish) {
	slog.Debug("handleMessage", slog.String("topic", m.Topic))
	// run this in the background to allow for reentrancy
	go func() {
		// handle reply message
		if strings.HasPrefix(m.Topic, transports.MessageTypeINBOX) && m.Properties.CorrelationData != nil {
			// Pass replies to their waiting channel
			cID := string(m.Properties.CorrelationData)
			tp.mux.RLock()
			rChan, _ := tp._correlData[cID]
			tp.mux.RUnlock()
			if rChan == nil {
				slog.Warn("Received reply without matching correlation ID", "corrID", cID)
			} else {
				tp.mux.Lock()
				delete(tp._correlData, cID)
				tp.mux.Unlock()

				rChan <- m
			}
			return
		}

		// handle request message
		replyTo := m.Properties.ResponseTopic
		if replyTo != "" && m.Properties.CorrelationData != nil {
			var reply []byte
			var err error
			var donotreply bool
			// get a reply from the single request handler
			tp.mux.RLock()
			reqHandler := tp._requestHandler
			tp.mux.RUnlock()

			if reqHandler != nil {
				reply, err, donotreply = reqHandler(m.Topic, m.Payload)
				if err != nil {
					slog.Warn("SubRequest: handle request failed.",
						slog.String("err", err.Error()),
						slog.String("topic", m.Topic))
				}
			} else {
				slog.Error("Received request message but no request handler is set.",
					slog.String("clientID", tp.clientID),
					slog.String("topic", m.Topic),
					slog.String("replyTo", replyTo))
				err = errors.New("Cannot handle request. No handler is set")
			}
			if !donotreply {
				err = tp.sendReply(m, reply, err)

				if err != nil {
					slog.Error("SubRequest. Sending reply failed", "err", err)
				}
			}
		} else {
			// this is en event message
			tp.mux.RLock()
			evHandler := tp._eventHandler
			tp.mux.RUnlock()
			if evHandler != nil {
				evHandler(m.Topic, m.Payload)
			}
		}
	}()
}

func (tp *MqttHubTransport) onPahoConnect(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	tp.mux.Lock()
	tp._status.ConnectionStatus = transports.Connected
	tp._status.LastError = nil
	subList := make([]string, 0, len(tp._subscriptions))
	for topic := range tp._subscriptions {
		subList = append(subList, topic)
	}
	connectStatus := tp._status
	connectHandler := tp._connectHandler
	tp.mux.Unlock()

	go func() {
		// (re)subscribe all subscriptions
		for _, topic := range subList {
			err := tp.sub(topic)
			if err != nil {
				slog.Error("onConnect. resubscribe failed", "topic", topic)
			}
		}
		// now subscriptions have been restored, inform subscriber
		connectHandler(connectStatus)
	}()
}

// paho reports an error but will keep trying until disconnect is called
func (tp *MqttHubTransport) onPahoConnectionError(err error) {
	go func() {
		connStatus := transports.Connecting
		connErr := errors.New(string(transports.Disconnected))
		// possible causes:
		// 1. wrong credentials - inform user, dont repeat or do repeat?
		// 2. connection is interrupted - inform user/log, keep repeating
		// 3. server disconnects - inform user/log, keep repeating
		// 4. client disconnects - terminate
		switch et := err.(type) {
		case *autopaho.ConnackError:
			if et.ReasonCode == 134 {
				connStatus = transports.Unauthorized
				connErr = fmt.Errorf("Unauthorized: %s", et.Reason)
			} else {
				connStatus = transports.Connecting
				connErr = fmt.Errorf("%s: %w", et.Reason, err)
				//connErr = fmt.Errorf("disconnected user '%s': %s", tp.clientID, err.Error())
			}
		default:
			connStatus = transports.Connecting
			connErr = fmt.Errorf("disconnected: %w", err)
			slog.Error("connection error", "clientID", tp.clientID, "err", err)
		}
		// notify on change
		tp.mux.RLock()
		oldStatus := tp._status.ConnectionStatus
		oldErr := tp._status.LastError
		tp.mux.RUnlock()
		if connStatus != oldStatus || connErr != oldErr {
			tp.mux.Lock()
			tp._status.ConnectionStatus = connStatus
			tp._status.LastError = connErr
			connectStatus := tp._status
			connHandler := tp._connectHandler
			tp.mux.Unlock()
			connHandler(connectStatus)
		}
		slog.Info("onPahoConnectionError", "err", connErr.Error())
		// don't retry on authentication error
		tp.mux.RLock()
		pcl := tp._pahoClient
		tp.mux.RUnlock()
		if connStatus == transports.Unauthorized && pcl != nil {
			_ = pcl.Disconnect(context.Background())
		}
	}()
}

// onPahoDisconnect handles a server disconnect
//
//	d is the disconnect packet from the server
func (tp *MqttHubTransport) onPahoServerDisconnect(d *paho.Disconnect) {
	go func() {
		tp.mux.Lock()
		slog.Warn("onPahoServerDisconnect: Disconnected by server. Retrying...",
			"clientID", tp.clientID, "cid", tp._connectID)
		tp._status.ConnectionStatus = transports.Connecting
		tp._status.LastError = errors.New("disconnected by server")
		connStatus := tp._status
		connHandler := tp._connectHandler
		tp.mux.Unlock()
		connHandler(connStatus)
	}()
}

// ParseResponse helper message to parse response and check for errors
func (tp *MqttHubTransport) ParseResponse(data []byte, resp interface{}) error {
	var err error
	if data == nil || len(data) == 0 {
		if resp != nil {
			err = fmt.Errorf("ParseResponse: client '%s', expected a response but none received", tp.clientID)
		} else {
			err = nil // all good
		}
	} else {
		if resp == nil {
			err = fmt.Errorf("ParseResponse: client '%s', received response but none was expected. data=%s",
				tp.clientID, data)
		} else {
			err = ser.Unmarshal(data, resp)
		}
	}
	return err
}

// PubEvent publishes a message and returns
func (tp *MqttHubTransport) PubEvent(topic string, payload []byte) (err error) {
	slog.Debug("PubEvent", "topic", topic)
	ctx, cancelFn := context.WithTimeout(context.Background(), tp.timeout)
	defer cancelFn()
	pubMsg := &paho.Publish{
		QoS:     0, //withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
	}
	tp.mux.RLock()
	pcl := tp._pahoClient
	tp.mux.RUnlock()
	if pcl != nil {
		_, err = pcl.Publish(ctx, pubMsg)
	} else {
		err = errors.New("no connection with the hub")
	}
	return err
}

// PubRequest publishes a request message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
func (tp *MqttHubTransport) PubRequest(topic string, payload []byte) (resp []byte, err error) {
	slog.Debug("PubRequest", "topic", topic)

	ctx, cancelFn := context.WithTimeout(context.Background(), tp.timeout)
	defer cancelFn()

	// FIXME! a deadlock can occur here
	tp.mux.RLock()
	pcl := tp._pahoClient
	inboxTopic := tp._inboxTopic
	connectID := tp._connectID
	tp.mux.RUnlock()

	if pcl == nil {
		return nil, fmt.Errorf("connection lost")
	}
	if inboxTopic == "" {
		inboxTopic = fmt.Sprintf(InboxTopicFormat, connectID)
		if connectID == "" {
			err = fmt.Errorf("can't publish request as connectID is not set. This is unexpected.")
			slog.Error(err.Error())
			return nil, err
		}
		tp.mux.Lock()
		tp._inboxTopic = inboxTopic
		tp.mux.Unlock()
		err = tp.Subscribe(inboxTopic)
		if err != nil {
			slog.Error("Failed inbox subscription",
				"err", err, "inboxTopic", inboxTopic)
			return nil, err
		}
	}
	// from paho rpc.go:
	cid := fmt.Sprintf("%d", time.Now().UnixNano())
	rChan := make(chan *paho.Publish)
	tp.mux.Lock()
	tp._correlData[cid] = rChan
	tp.mux.Unlock()

	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
		Properties: &paho.PublishProperties{
			CorrelationData: []byte(cid),
			ResponseTopic:   inboxTopic,
			ContentType:     "json",
			User: paho.UserProperties{{
				Key:   "test",
				Value: "test",
			}},
		},
	}
	_, err = pcl.Publish(ctx, pubMsg)
	if err != nil {
		return nil, err
	}

	// wait for response
	var respMsg *paho.Publish
	select {
	case <-ctx.Done():
		err = fmt.Errorf("timeout waiting for response")
		break
	case respMsg = <-rChan:
		break
	}
	if err != nil {
		return nil, err
	}

	// test alternative to handling errors since User properties aren't
	// passed through for some reason.
	if respMsg.Properties.ContentType == "error" {
		err = errors.New(string(respMsg.Payload))
		return nil, err
	}

	slog.Debug("PubRequest end:",
		slog.String("topic", topic),
		slog.String("ContentType (if any)", respMsg.Properties.ContentType),
	)
	return respMsg.Payload, err
}

// sendReply sends a reply on the response topic of the request
// This uses the same QoS as the request, without retain.
//
//	req is the request to reply to
//	optionally include a payload in the reply
//	optionally include an error message in the reply
func (tp *MqttHubTransport) sendReply(req *paho.Publish, payload []byte, errResp error) (err error) {

	slog.Debug("sendReply",
		slog.String("topic", req.Topic),
		slog.String("responseTopic", req.Properties.ResponseTopic))

	responseTopic := req.Properties.ResponseTopic
	if responseTopic == "" {
		err2 := fmt.Errorf("sendReply. No response topic. Not sending a reply")
		slog.Error(err2.Error())
	}
	replyMsg := &paho.Publish{
		QoS:    req.QoS,
		Retain: false,
		Topic:  responseTopic,
		Properties: &paho.PublishProperties{
			CorrelationData: req.Properties.CorrelationData,
			User:            req.Properties.User,
			PayloadFormat:   req.Properties.PayloadFormat,
			ContentType:     req.Properties.ContentType,
		},
		Payload: payload,
	}
	if errResp != nil {
		replyMsg.Properties.ContentType = "error" // payload is an error message
		replyMsg.Properties.User.Add("error", errResp.Error())
		// for testing, somehow properties.user is not transferred
		replyMsg.Payload = []byte(errResp.Error())
	}
	tp.mux.RLock()
	pcl := tp._pahoClient
	tp.mux.RUnlock()
	if pcl == nil {
		err = errors.New("connection lost")
	} else {
		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
		defer cancelFn()
		_, err = pcl.Publish(ctx, replyMsg)

		if err != nil {
			slog.Warn("sendReply. Error publishing response",
				slog.String("err", err.Error()))
		}
	}
	return err
}

// SetConnectHandler sets the notification handler of connection status changes
func (tp *MqttHubTransport) SetConnectHandler(cb func(status transports.HubTransportStatus)) {
	if cb == nil {
		panic("nil handler not allowed")
	}
	tp.mux.Lock()
	tp._connectHandler = cb
	tp.mux.Unlock()
}

// SetEventHandler set the single handler that receives all subscribed events.
// This does not provide routing as in most cases it is unnecessary overhead
// Use 'Subscribe' to set the addresses that this receives events on.
func (tp *MqttHubTransport) SetEventHandler(cb func(addr string, payload []byte)) {
	tp.mux.Lock()
	tp._eventHandler = cb
	tp.mux.Unlock()
}

// SetRequestHandler sets the handler that receives all subscribed requests.
// This does not provide routing as in most cases it is unnecessary overhead
// Use 'Subscribe' to set the addresses that this receives requests on.
func (tp *MqttHubTransport) SetRequestHandler(
	cb func(addr string, payload []byte) (reply []byte, err error, donotreply bool)) {

	tp.mux.Lock()
	tp._requestHandler = cb
	tp.mux.Unlock()
}

// sub builds a subscribe packet and submits it
func (tp *MqttHubTransport) sub(topic string) error {
	packet := &paho.Subscribe{
		Properties: nil,
		Subscriptions: []paho.SubscribeOptions{
			{
				Topic: topic,
				QoS:   withQos,
			},
		},
	}
	tp.mux.RLock()
	pcl := tp._pahoClient
	tp.mux.RUnlock()
	suback, err := pcl.Subscribe(context.Background(), packet)
	_ = suback
	return err
}

// Subscribe subscribes to a topic.
// Incoming messages are passed to the event or request handler, depending on whether
// a reply-to address and correlation-ID is set.
func (tp *MqttHubTransport) Subscribe(topic string) error {
	slog.Debug("Subscribe", "topic", topic)
	err := tp.sub(topic)
	if err != nil {
		return err
	}
	tp.mux.Lock()
	tp._subscriptions[topic] = true
	tp.mux.Unlock()
	return err
}
func (tp *MqttHubTransport) Unsubscribe(topic string) {
	packet := &paho.Unsubscribe{
		Topics: []string{topic},
	}
	tp.mux.RLock()
	pcl := tp._pahoClient
	tp.mux.RUnlock()

	ack, err := pcl.Unsubscribe(context.Background(), packet)
	_ = ack
	if err != nil {
		slog.Error("Unable to unsubscribe from topic", "topic", topic)
		return
	}
	tp.mux.Lock()
	delete(tp._subscriptions, topic)
	tp.mux.Unlock()
}

// NewMqttTransport creates a new instance of the mqtt client.
//
// fullURL is the url with schema. If omitted this uses the in-memory UDS address,
// which only works with ConnectWithToken.
//
//	fullURL of broker to connect to, starting with "tls://", "wss://", "unix://"
//	clientID to connect as
//	keyPair with previously saved serialized public/private key pair, or "" to create one
//	caCert of the server to validate the server or nil to not check the server cert
func NewMqttTransport(fullURL string, clientID string, caCert *x509.Certificate) *MqttHubTransport {
	if fullURL == "" {
		fullURL = "unix://" + MqttInMemUDSProd
	}
	// checks
	if clientID == "" {
		panic("NewMqttTransport - Missing client ID")
	}
	hc := &MqttHubTransport{
		serverURL:      fullURL,
		clientID:       clientID,
		caCert:         caCert,
		timeout:        time.Second * 10, // 10 for testing
		_correlData:    make(map[string]chan *paho.Publish),
		_subscriptions: make(map[string]bool),
		// default, log status
		_connectHandler: func(status transports.HubTransportStatus) {
			slog.Info("connection status change",
				"newStatus", status.ConnectionStatus,
				"lastError", status.LastError,
				"clientID", clientID)
		},
		_status: transports.HubTransportStatus{
			CaCert:           caCert,
			HubURL:           fullURL,
			ClientID:         clientID,
			ConnectionStatus: transports.Disconnected,
			LastError:        nil,
			Core:             "mqtt",
		},
	}
	return hc
}
