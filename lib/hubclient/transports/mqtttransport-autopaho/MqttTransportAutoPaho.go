package mqtttransportautopaho

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// InboxTopicFormat is the INBOX subscription topic used by the client
// _INBOX/{clientID}
const InboxPrefix = "_INBOX/"
const InboxTopicFormat = "_INBOX/%s"

const keepAliveInterval = 30 // seconds
const reconnectDelay = 10 * time.Second
const withDebug = false
const withQos = 1

// Connecting with UDS for local services. Might not work with autopaho
const MqttInMemUDSProd = "@/MqttInMemUDSProd" // production UDS name
const MqttInMemUDSTest = "@/MqttInMemUDSTest" // test server UDS name

// MqttHubTransport manages the hub server connection with hub event and action
// messaging using autopaho.
// This implements the IHubTransport interface.
type MqttHubTransport struct {
	serverURL string
	clientID  string
	caCert    *x509.Certificate
	//conn      net.Conn
	pcl     *autopaho.ConnectionManager
	timeout time.Duration // request timeout

	// track the request-response correlations
	correlData map[string]chan *paho.Publish
	inboxTopic string // set on first request

	connectHandler func(connected bool, err error)
	eventHandler   func(addr string, payload []byte)
	requestHandler func(addr string, payload []byte) (reply []byte, err error, donotreply bool)

	//
	mux sync.RWMutex
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
	ctx := context.Background()

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
	//safeConn := packets.NewThreadSafeConn(conn)
	// Setup the Paho client configuration
	pahoCfg := autopaho.ClientConfig{
		BrokerUrls: []*url.URL{u},
		ClientConfig: paho.ClientConfig{
			ClientID: tp.clientID,
			//Conn:          safeConn,    // autopaho ignores this :(
			PacketTimeout: tp.timeout,
			Router: paho.NewSingleHandlerRouter(func(m *paho.Publish) {
				tp.handleMessage(m)
			}),
			OnServerDisconnect: func(d *paho.Disconnect) {
				err := errors.New(
					fmt.Sprintf("Server disconnected. reason code %d", d.ReasonCode))
				slog.Warn(err.Error())
				if tp.connectHandler != nil {
					tp.connectHandler(false, err)
				}
			},
		},
		TlsCfg: tlsCfg,
		// CleanStartOnInitialConnection defaults to false.
		// Setting this to true will clear the session on the first connection.
		//CleanStartOnInitialConnection: true,
		KeepAlive: 20, // Keepalive message should be sent every 20 seconds
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			if tp.connectHandler != nil {
				tp.connectHandler(true, nil)
			}
		},
		OnConnectError: func(err error) {
			fmt.Printf("error whilst attempting connection: %s\n", err)
		},
	}
	pahoCfg.SetUsernamePassword(tp.clientID, []byte(credentials))
	pahoCfg.OnClientError = func(err error) {
		// connection closing can cause this error.
		slog.Warn("OnClientError - connection closing",
			slog.String("err", err.Error()),
			slog.String("clientID", pahoCfg.ClientID))
	}
	pahoCfg.OnServerDisconnect = func(d *paho.Disconnect) {
		slog.Warn("OnServerDisconnect: Disconnected from broker",
			"code", d.ReasonCode,
			"loginID", tp.clientID)
	}
	pcl, err := autopaho.NewConnection(ctx, pahoCfg)
	// Wait for the connection to come up
	if err = pcl.AwaitConnection(ctx); err != nil {
		panic(err)
	}

	// subscribe to inbox to receive responses to requests

	//cp := &paho.Connect{
	//	Password: []byte(credentials),
	//	Username: tp.clientID,
	//	ClientID: connectID,
	//	// TODO: consider including a signed nonce when connecting with key
	//	Properties:   &paho.ConnectProperties{},
	//	KeepAlive:    60,
	//	CleanStart:   true,
	//	UsernameFlag: true,
	//	PasswordFlag: credentials != "",
	//}
	//ctx, cancelFn := context.WithTimeout(context.Background(), tp.timeout)
	//defer cancelFn()
	//connAck, err := pcl.Connect(ctx, cp)
	//_ = connAck

	if err != nil {
		var info string
		//if connAck != nil {
		//	info = fmt.Sprintf("code %d, reason: '%s'", connAck.ReasonCode, connAck.Properties.ReasonString)
		//}
		err = fmt.Errorf("%w %s", err, info)
		return err
	}
	//tp.conn = conn
	tp.pcl = pcl

	// create a request handler for request-response messages
	//tp.pahoReqHandler, err = rpc.NewHandler(ctx, tp.pcl)
	// register a single handler for all messages containing type/agent/thing|cap/...
	//tp.pcl.Router.RegisterHandler("+/+/+/#", tp.handleMessage)

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
	if tp.pcl != nil {
		time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		//_ = tp.pcl.Disconnect(&paho.Disconnect{ReasonCode: 0})
		_ = tp.pcl.Disconnect(context.Background())
		//tp.subscriptions = nil
		//close(tp.messageChannel)     // end the message handler loop
	}
}

// handleMessage handles incoming request, reply and event messages
func (tp *MqttHubTransport) handleMessage(m *paho.Publish) {

	// run this in the background to allow for reentrancy
	go func() {
		// handle reply message
		if strings.HasPrefix(m.Topic, InboxPrefix) && m.Properties.CorrelationData != nil {
			// Pass replies to their waiting channel
			cID := string(m.Properties.CorrelationData)
			tp.mux.Lock()
			rChan, _ := tp.correlData[cID]
			if rChan == nil {
				slog.Warn("Received reply without matching correlation ID", "corrID", cID)
			} else {
				delete(tp.correlData, cID)
				rChan <- m
			}
			tp.mux.Unlock()
			return
		}

		// handle request message
		replyTo := m.Properties.ResponseTopic
		if replyTo != "" && m.Properties.CorrelationData != nil {
			var reply []byte
			var err error
			var donotreply bool
			// get a reply from the single request handler
			if tp.requestHandler != nil {
				reply, err, donotreply = tp.requestHandler(m.Topic, m.Payload)
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
			if tp.eventHandler != nil {
				tp.eventHandler(m.Topic, m.Payload)
			}
		}
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
	respMsg, err := tp.pcl.Publish(ctx, pubMsg)
	_ = respMsg
	if err != nil {
		return err
	}
	return err
}

// PubRequest publishes a request message and waits for an answer or until timeout
// In order to receive replies, an inbox subscription is added on the first request.
func (tp *MqttHubTransport) PubRequest(topic string, payload []byte) (resp []byte, err error) {
	slog.Info("PubRequest", "topic", topic)

	ctx, cancelFn := context.WithTimeout(context.Background(), tp.timeout)
	defer cancelFn()

	tp.mux.Lock()
	if tp.inboxTopic == "" {
		inboxTopic := fmt.Sprintf(InboxTopicFormat, tp.clientID)
		err = tp.Subscribe(inboxTopic)
		if err != nil {
			slog.Error("Failed inbox subscription",
				"err", err, "inboxTopic", inboxTopic)
			return nil, err
		}
		tp.inboxTopic = inboxTopic
	}
	// from paho rpc.go:
	cid := fmt.Sprintf("%d", time.Now().UnixNano())
	rChan := make(chan *paho.Publish)
	tp.correlData[cid] = rChan
	tp.mux.Unlock()

	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
		Properties: &paho.PublishProperties{
			CorrelationData: []byte(cid),
			ResponseTopic:   tp.inboxTopic,
			ContentType:     "json",
			User: paho.UserProperties{{
				Key:   "test",
				Value: "test",
			}},
		},
	}
	_, err = tp.pcl.Publish(ctx, pubMsg)
	// use the inbox as the custom response for this client instance
	// clone of rpc.go to workaround hangup when no response is received #111
	//respMsg, err := tp.pahoReqHandler.Request(ctx, pubMsg)
	//ar.Duration = time.Now().SubEvent(t1)
	if err != nil {
		return nil, err
	}

	// wait for response
	var respMsg *paho.Publish
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for response")
	case respMsg = <-rChan:
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
func (tp *MqttHubTransport) sendReply(req *paho.Publish, payload []byte, errResp error) error {

	slog.Debug("sendReply", "topic", req.Topic, "responseTopic", req.Properties.ResponseTopic)

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
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn()
	_, err := tp.pcl.Publish(ctx, replyMsg)
	if err != nil {
		slog.Warn("sendReply. Error publishing response",
			slog.String("err", err.Error()))
	}
	return err
}

// SetConnectHandler sets the notification handler of connection status changes
func (tp *MqttHubTransport) SetConnectHandler(cb func(connected bool, err error)) {
	tp.connectHandler = cb
}

// SetEventHandler set the single handler that receives all subscribed events.
// This does not provide routing as in most cases it is unnecessary overhead
// Use 'Subscribe' to set the addresses that this receives events on.
func (tp *MqttHubTransport) SetEventHandler(cb func(addr string, payload []byte)) {
	tp.eventHandler = cb
}

// SetRequestHandler sets the handler that receives all subscribed requests.
// This does not provide routing as in most cases it is unnecessary overhead
// Use 'Subscribe' to set the addresses that this receives requests on.
func (tp *MqttHubTransport) SetRequestHandler(
	cb func(addr string, payload []byte) (reply []byte, err error, donotreply bool)) {

	tp.requestHandler = cb
}

// Subscribe subscribes to a topic.
// Incoming messages are passed to the event or request handler, depending on whether
// a reply-to address and correlation-ID is set.
func (tp *MqttHubTransport) Subscribe(topic string) error {
	slog.Info("Subscribe", "topic", topic)
	packet := &paho.Subscribe{
		Properties: nil,
		Subscriptions: []paho.SubscribeOptions{
			{
				Topic: topic,
				QoS:   withQos,
			},
		},
	}
	suback, err := tp.pcl.Subscribe(context.Background(), packet)
	_ = suback
	return err
}
func (tp *MqttHubTransport) Unsubscribe(topic string) {
	packet := &paho.Unsubscribe{
		Topics: []string{topic},
	}
	ack, err := tp.pcl.Unsubscribe(context.Background(), packet)
	_ = ack
	_ = err
}

// NewMqttTransportAutoPaho creates a new instance of the mqtt client.
//
// fullURL is the url with schema. If omitted this uses the in-memory UDS address,
// which only works with ConnectWithToken.
//
//	fullURL of broker to connect to, starting with "tls://", "wss://", "unix://"
//	clientID to connect as
//	keyPair with previously saved serialized public/private key pair, or "" to create one
//	caCert of the server to validate the server or nil to not check the server cert
func NewMqttTransportAutoPaho(fullURL string, clientID string, caCert *x509.Certificate) *MqttHubTransport {
	if fullURL == "" {
		fullURL = "unix://" + MqttInMemUDSProd
	}
	// checks
	if clientID == "" {
		panic("NewMqttTransport - Missing client ID")
	}
	hc := &MqttHubTransport{
		serverURL:  fullURL,
		clientID:   clientID,
		caCert:     caCert,
		timeout:    time.Second * 10, // 10 for testing
		correlData: make(map[string]chan *paho.Publish),
	}
	return hc
}
