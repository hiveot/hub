package mqtthubclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// DefaultTimeoutSec with timeout for connecting and publishing.
const DefaultTimeoutSec = 100 //3 // 100 for testing

// MqttHubSubscription  subscription helper
// This implements ISubscription
type MqttHubSubscription struct {
	topic   string
	handler func(topic string, payload []byte)
	client  pahomqtt.Client
}

func (sub *MqttHubSubscription) Unsubscribe() {
	err := fmt.Errorf("not implemented")
	if err != nil {
		slog.Error("Unsubscribe error", "error", err)
	}
}

// MqttHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the Mqtt messaging system.
type MqttHubClient struct {
	clientID   string
	hostName   string
	port       int
	pahoClient pahomqtt.Client // Paho MQTT Client
	timeout    time.Duration
	nc         *tls.Conn
	isRunning  bool
}

// ClientID the client is authenticated as to the server
func (hc *MqttHubClient) ClientID() string {
	return hc.clientID
}

// Connect to the MQTT broker
func _connect(clientID string, brokerURL string, opts *pahomqtt.ClientOptions) (pahomqtt.Client, error) {

	// set config defaults
	if clientID == "" {
		err := fmt.Errorf("connect - Missing Client ID. Required for MQTT connection")
		return nil, err
	}

	// Support multiple MQTT client IDs
	//brokerURL := fmt.Sprintf("tcp://%s:%d/", hostName, port) // tcp://host:1883 ws://host:1883 tls://host:8883, tcps://awshost:8883/mqtt
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetMaxReconnectInterval(60 * time.Second) // max wait 1 minute for a reconnect
	// Do not use MQTT persistence as not all brokers support it, and it causes problems on the broker if the client ID is
	// randomly generated. CleanSession disables persistence.
	opts.SetCleanSession(true)
	opts.SetKeepAlive(10 * time.Second) // pings to detect a disconnect. Use same as reconnect interval
	//opts.SetKeepAlive(60) // keepalive causes deadlock in v1.1.0. See github issue #126

	opts.SetOnConnectHandler(func(client pahomqtt.Client) {
		slog.Info("mqtt:onConnect: Connected to broker", "url", brokerURL, "clientID", clientID)
		// Subscribe to topics already registered by the app on connect or reconnect
		//mqttClient.resubscribe()
	})
	opts.SetConnectionLostHandler(func(client pahomqtt.Client, err error) {
		slog.Warn("mqtt:onConnectionLost: Disconnected from broker", "url",
			brokerURL, "err", err, "clientID", clientID)
	})
	//if lastWillTopic != "" {
	//	//lastWillTopic := fmt.Sprintf("%s/%s/$state", mqttClient.config.Base, deviceId)
	//	opts.SetWill(lastWillTopic, lastWillValue, 1, false)
	//}
	slog.Info("mqtt:Connect: Connecting to MQTT broker. AutoReconnect and CleanSession are set.",
		"url", brokerURL, "clientID", clientID)

	// FIXME: PahoMqtt disconnects when sending a lot of messages, like on startup of some adapters.
	pahoClient := pahomqtt.NewClient(opts)

	// start listening for messages
	//hc.isRunning = true
	//go mqttClient.messageChanLoop()

	// Auto reconnect doesn't work for initial attempt: https://github.com/eclipse/paho.mqtt.golang/issues/77
	//retryDelaySec := 1
	//for {
	//	token := pahoClient.Connect()
	//	token.Wait()
	//	// Wait to give connection time to settle. Sending a lot of messages causes the connection to fail. Bug?
	//	time.Sleep(1000 * time.Millisecond)
	//	err := token.Error()
	//	if err == nil {
	//		break
	//	}
	//
	//	slog.Error("mqtt:Connect: Connecting to broker failed. Retrying...",
	//		"url", brokerURL, "err", token.Error(), "retryDelaySec", retryDelaySec)
	//	time.Sleep(time.Duration(retryDelaySec) * time.Second)
	//	// slowly increment wait time
	//	if retryDelaySec < 120 {
	//		retryDelaySec++
	//	}
	//}
	return pahoClient, nil
}

// ConnectWithCert to the Hub server
//
//	url of the server.
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
func ConnectWithCert(url string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (hubclient.IHubClient, error) {
	if url == "" {
	}

	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	x509Cert, _ := x509.ParseCertificate(clientCert.Certificate[0])
	_, err := x509Cert.Verify(opts)
	clientCertList := []tls.Certificate{*clientCert}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	_ = tlsConfig
	pahoOpts := pahomqtt.NewClientOptions()
	pahoOpts.TLSConfig = tlsConfig
	pahoClient, err := _connect(clientID, url, pahoOpts)
	hc := NewMqttHubClient(clientID, pahoClient)
	return hc, err
}

// ConnectWithJWT connects to the Hub server using a user JWT credentials secret
// The connection uses the client ID in the JWT token.
//
//	url is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	jwtToken is the token obtained with login or refresh.
//func ConnectWithJWT(url string, myKey *ecdsa.PrivateKey, jwtToken string, caCert *x509.Certificate) (hc *MqttHubClient, err error) {
//	if url == "" {
//	}
//
//	caCertPool := x509.NewCertPool()
//	if caCert != nil {
//		caCertPool.AddCert(caCert)
//	}
//	tlsConfig := &tls.Config{
//		RootCAs:            caCertPool,
//		InsecureSkipVerify: caCert == nil,
//	}
//
//	claims, err := jwt.DecodeUserClaims(jwtToken)
//	if err != nil {
//		err = fmt.Errorf("invalid jwt token: %w", err)
//		return nil, err
//	}
//	clientID := claims.Claims().Name
//	// TODO
//
//	pahoOpts := pahomqtt.NewClientOptions()
//	pahoOpts.SetTLSConfig(tlsConfig)
//	pahoOpts.SetPassword(jwtToken)
//	err = hc.connect(clientID, url, pahoOpts)
//
//	if err == nil {
//		hc, err = NewMqttHubClient(clientID, c)
//	}
//	return hc, err
//}

// ConnectWithNC connects using the given TLS connection
//func ConnectWithNC(nc *tls.Conn) (hc *MqttHubClient, err error) {
//	clientID := "todo" // todo
//	hc, err = NewMqttHubClient(clientID, nc)
//	return hc, err
//}

// ConnectWithKey connects to the Hub server using a key secret with signed nonce
//
// UserID is used for publishing actions
//func ConnectWithKey(url string, clientID string, myKey *ecdsa.PrivateKey, caCert *x509.Certificate) (hc *MqttHubClient, err error) {
//
//	caCertPool := x509.NewCertPool()
//	if caCert != nil {
//		caCertPool.AddCert(caCert)
//	}
//	tlsConfig := &tls.Config{
//		RootCAs:            caCertPool,
//		InsecureSkipVerify: caCert == nil,
//	}
//	// The handler to sign the server issued challenge
//	sigCB := func(nonce []byte) ([]byte, error) {
//		return myKey.Sign(nonce)
//	}
//	nc, err := tls.Dial("tcp", url, tlsConfig)
//	// TODO: implement
//	if err == nil {
//		hc, err = NewHubClient(clientID, nc)
//	}
//	return hc, err
//}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func ConnectWithPassword(
	url string, loginID string, password string, caCert *x509.Certificate) (hc *MqttHubClient, err error) {

	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: caCert == nil,
	}
	// TODO: implement
	pahoOpts := pahomqtt.NewClientOptions()
	pahoOpts.SetTLSConfig(tlsConfig)
	pahoOpts.SetPassword(password)
	pahoClient, err := _connect(url, loginID, pahoOpts)
	if err == nil {
		hc = NewMqttHubClient(loginID, pahoClient)
	}
	return hc, err
}

// ConnectUnauthenticated connects to the Hub server as an unauthenticated user
// Intended for use by IoT devices to perform out-of-band provisioning.
//func ConnectUnauthenticated(url string, caCert *x509.Certificate) (hc *MqttHubClient, err error) {
//	caCertPool := x509.NewCertPool()
//	if caCert != nil {
//		caCertPool.AddCert(caCert)
//	}
//	tlsConfig := &tls.Config{
//		RootCAs:            caCertPool,
//		InsecureSkipVerify: caCert == nil,
//	}
//	// TODO
//	if err == nil {
//		hc, err = NewMqttHubClient("", nc)
//	}
//	return hc, err
//}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (mqttClient *MqttHubClient) Disconnect() {
	mqttClient.isRunning = false
	if mqttClient.pahoClient != nil {
		slog.Warn("Disconnect: Set state to disconnected and close connection")
		//mqttClient.publish("$state", "disconnected")
		time.Sleep(time.Second / 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		mqttClient.pahoClient.Disconnect(100)
		mqttClient.pahoClient = nil

		//mqttClient.subscriptions = nil
		//close(mqttClient.messageChannel)     // end the message handler loop
	}
}

// ParseResponse helper message to parse response and detect the error response message
func (hc *MqttHubClient) ParseResponse(data []byte, err error, resp interface{}) error {
	if err != nil {
		return err
	}
	if resp != nil {
		err = ser.Unmarshal(data, resp)
	} else if string(data) == "+ACK" {
		err = nil
	} else if len(data) > 0 {
		err = errors.New("unexpected response")
	}
	// if an error is detect see if it is an error response
	// An error response message has the format: {"error":"message"}
	// TODO: find a more idiomatic way to detect an error
	prefix := "{\"error\":"
	if err != nil || strings.HasPrefix(string(data), prefix) {
		errResp := hubclient.ErrorMessage{}
		err2 := ser.Unmarshal(data, &errResp)
		if err2 == nil && errResp.Error != "" {
			err = errors.New(errResp.Error)
		}
	}
	return err
}

// PubThingAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *MqttHubClient) PubThingAction(bindingID string, thingID string, actionID string, payload []byte) ([]byte, error) {
	topic := MakeThingsTopic(bindingID, thingID, actionID, hc.clientID)
	slog.Info("PubThingAction", "topic", topic)
	resp, err := hc.pahoClient.Request(topic, payload, hc.timeout)
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}

// PubServiceAction sends an action request to a Hub Service on the svc prefix
// Returns the response or an error if the request fails or timed out
func (hc *MqttHubClient) PubServiceAction(serviceID string, capability string, actionID string, payload []byte) ([]byte, error) {
	topic := MakeServiceActionTopic(serviceID, capability, actionID, hc.clientID)
	slog.Info("PubServiceAction", "topic", topic)
	resp, err := hc.pahoClient.Request(topic, payload, hc.timeout)
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}

// PubEvent sends the event value to the hub
func (hc *MqttHubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	topic := MakeThingsTopic(hc.clientID, thingID, vocab.MessageTypeEvent, eventID)
	slog.Info("PubEvent", "topic", topic)
	token := hc.pahoClient.Publish(topic, 1, false, payload)
	token.WaitTimeout(hc.timeout)
	return token.Error()
}

// PubTD sends the TD document to the hub
func (hc *MqttHubClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	topic := MakeThingsTopic(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
	slog.Info("PubTD", "topic", topic)
	token := hc.pahoClient.Publish(topic, 1, false, payload)
	token.WaitTimeout(hc.timeout)
	return token.Error()
}

// subscribe to topics after establishing connection
// The application can already subscribe to topics before the connection is established. If connection is lost then
// this will re-subscribe to those topics as PahoMqtt drops the subscriptions after disconnect.
//func (hc *MqttHubClient) resubscribe() {
//	//
//	slog.Info("mqtt.resubscribe to topics", "n", len(mqttClient.subscriptions))
//	for _, subscription := range mqttClient.subscriptions {
//		// clear existing subscription
//		hc.pahoClient.Unsubscribe(subscription.topic)
//
//		// create a new variable to hold the subscription in the closure
//		newSubscr := subscription
//		token := hc.pahoClient.Subscribe(newSubscr.topic, newSubscr.qos, newSubscr.onMessage)
//		//token := mqttClient.pahoClient.Subscribe(newSubscr.topic, newSubscr.qos, func (c pahomqtt.Client, msg pahomqtt.Message) {
//		//mqttClient.log.Infof("mqtt.resubscribe.onMessage: topic %s, subscription %s", msg.Topic(), newSubscr.topic)
//		//newSubscr.onMessage(c, msg)
//		//})
//		newSubscr.token = token
//	}
//}

// Refresh an authentication token.
//
//	clientID is the userID, deviceID or serviceID whose token to refresh.
//	oldToken must be a valid token obtained at login or refresh
//
// This returns a short lived auth token that can be used to authenticate with the hub
// This fails if the token has expired or does not belong to the clientID
//func (hc *MqttHubClient) Refresh(clientID string, oldToken string) (newToken string, err error) {
//	req := &authn.RefreshReq{
//		UserID: clientID,
//		OldToken: oldToken,
//	}
//	msg, _ := ser.Marshal(req)
//	topic := MakeThingsTopic(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
//	slog.Info("PubTD", "topic", topic)
//	err := hc.Publish(topic, payload)
//	resp := &authn.RefreshResp{}
//	err = hubclient.ParseResponse(data, err, resp)
//	if err == nil {
//		authToken = resp.JwtToken
//	}
//	return err
//}

// SubActions subscribes to actions on the given topic
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *MqttHubClient) SubActions(topic string, cb func(msg *hubclient.ActionMessage) error) (hubclient.ISubscription, error) {

	sub, err := hc.Subscribe(topic, func(msg []byte) {
		timeStamp := time.Now()
		sourceID, thID, name, clientID, err := SplitActionTopic(msg.topic)
		if err != nil {
			slog.Error("unable to handle topic", "err", err, "topic", msg.Topic)
			return
		}
		actionMsg := &hubclient.ActionMessage{
			ClientID:  clientID,
			ActionID:  name,
			BindingID: sourceID,
			ThingID:   thID,
			Timestamp: timeStamp.Unix(),
			Payload:   payload,
			SendReply: func(payload []byte) {
				_ = msg.Respond(payload)
			},
			SendAck: func() {
				_ = msg.Ack()
			},
		}
		err = cb(actionMsg)
		if err != nil {
			errMsg := hubclient.ErrorMessage{Error: err.Error()}
			errPayload, _ := ser.Marshal(errMsg)
			_ = msg.Respond(errPayload)
		}
	})
	return sub, err
}

// SubThingActions subscribes to actions for this device or service on the things prefix
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *MqttHubClient) SubThingActions(thingID string, cb func(msg *hubclient.ActionMessage) error) (hubclient.ISubscription, error) {

	topic := MakeThingActionTopic(hc.clientID, thingID, "", "")
	return hc.SubActions(topic, cb)
}

// SubServiceCapability subscribes to action requests of a service capability
//
//	capability is the name of the capability (thingID) to handle
func (hc *MqttHubClient) SubServiceCapability(capability string, cb func(msg *hubclient.ActionMessage) error) (hubclient.ISubscription, error) {

	topic := MakeServiceTopic(hc.clientID, capability, MessageTypeAction, "")
	return hc.SubActions(topic, cb)
}

// startEventMessageHandler listens for incoming event messages and invoke a callback handler
// this returns when the subscription is no longer valid
func startEventMessageHandler(sub hubclient.ISubscription, cb func(msg *hubclient.EventMessage)) error {

	return nil
}

// SubStream subscribes to events received by the event stream.
//
// This creates an ephemeral pull consumer.
// ReceiveLatest is handy to be up to date on all event instead of quering them separately. Only use this if
// you're going to retrieve them anyways.
//
//	 name of the event stream. "" for default
//		receiveLatest to immediately receive the latest event for each event instance
//func (hc *MqttHubClient) SubStream(name string, receiveLatest bool, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {
//
//}

// NewMqttHubClient instantiates a client for connecting to the Hub using MQTT
func NewMqttHubClient(clientID string, pahoClient pahomqtt.Client) *MqttHubClient {

	hc := &MqttHubClient{
		clientID:   clientID,
		pahoClient: pahoClient,
		timeout:    time.Duration(DefaultTimeoutSec) * time.Second,
	}
	hc.timeout = time.Duration(10) * time.Second // for testing
	return hc
}
