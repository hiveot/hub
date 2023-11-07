package mqtttransport

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/packets"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/tlsclient"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"
)

const keepAliveInterval = 30 // seconds
const reconnectDelay = 10 * time.Second
const withDebug = false
const withQos = 1
const MqttInMemUDSProd = "@/MqttInMemUDSProd" // production UDS name
const MqttInMemUDSTest = "@/MqttInMemUDSTest" // test server UDS name

// MqttHubTransport manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the Mqtt messaging system.
type MqttHubTransport struct {
	serverURL      string
	clientID       string
	caCert         *x509.Certificate
	conn           net.Conn
	pcl            *paho.Client
	requestHandler *Handler      // PahoRPC request handler
	timeout        time.Duration // request timeout
}

// AddressTokens returns the address separator and wildcards
func (hc *MqttHubTransport) AddressTokens() (sep string, wc string, rem string) {
	return "/", "+", "#"
}

// ConnectWithConn connects to a mqtt broker using the pre-established network connection
// This allows creating connections with any available means including tcp/tls/wss/uds/pipes
//
//	credentials is either the password or a signed JWT token
//	conn network connection to utilize
func (ht *MqttHubTransport) ConnectWithConn(credentials string, conn net.Conn) error {
	ctx := context.Background()

	// clients must use a unique connection ID otherwise the previous connection will be dropped
	hostName, _ := os.Hostname()
	connectID := fmt.Sprintf("%s-%s-%s", ht.clientID, hostName, time.Now().Format("20060102150405.000"))
	slog.Info("ConnectWithConn", "clientID", ht.clientID, "RemoteAddr", conn.RemoteAddr(), "connectID", connectID)

	// checks
	if ht.clientID == "" {
		err := fmt.Errorf("connect - Missing Login ID")
		return err
	} else if conn == nil {
		err := fmt.Errorf("connect - missing connection")
		return err
	}

	safeConn := packets.NewThreadSafeConn(conn)
	pahoCfg := paho.ClientConfig{
		ClientID:      ht.clientID,
		Conn:          safeConn,
		PacketTimeout: ht.timeout,
	}
	pahoCfg.OnClientError = func(err error) {
		// connection closing can cause this error.
		slog.Debug("OnClientError - connection closing",
			slog.String("err", err.Error()),
			slog.String("clientID", pahoCfg.ClientID))
	}
	pahoCfg.OnServerDisconnect = func(d *paho.Disconnect) {
		slog.Warn("ConnectWithNC:OnServerDisconnect: Disconnected from broker",
			"code", d.ReasonCode,
			"loginID", ht.clientID)
	}
	pcl := paho.NewClient(pahoCfg)
	cp := &paho.Connect{
		Password: []byte(credentials),
		Username: ht.clientID,
		ClientID: connectID,
		// TODO: consider including a signed nonce when connecting with key
		Properties:   &paho.ConnectProperties{},
		KeepAlive:    60,
		CleanStart:   true,
		UsernameFlag: true,
		PasswordFlag: credentials != "",
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), ht.timeout)
	defer cancelFn()
	connAck, err := pcl.Connect(ctx, cp)
	_ = connAck

	if err != nil {
		var info string
		if connAck != nil {
			info = fmt.Sprintf("code %d, reason: '%s'", connAck.ReasonCode, connAck.Properties.ReasonString)
		}
		err = fmt.Errorf("%w %s", err, info)
		//err = fmt.Errorf("failed to connect. Reason: %d - %s",
		//	connAck.ReasonCode, connAck.Properties.ReasonString)
		return err
	}
	ht.conn = conn
	ht.pcl = pcl

	// last, create a request handler
	ht.requestHandler, err = NewHandler(ctx, ht.pcl)

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
func (ht *MqttHubTransport) ConnectWithPassword(password string) error {

	conn, err := tlsclient.ConnectTLS(ht.serverURL, nil, ht.caCert)
	if err != nil {
		return err
	}
	err = ht.ConnectWithConn(password, conn)
	return err
}

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
// A private key might be required in future.
// This supports UDS connections with @/path or unix://@/path
//
// TODO: encrypt token with server public key so a MIM won't be able to get the token
//
//	kp is the key-pair of this client
//	jwtToken is the token obtained with login or refresh.
func (ht *MqttHubTransport) ConnectWithToken(kp keys.IHiveKey, jwtToken string) error {
	var conn net.Conn
	var err error
	if strings.HasPrefix(ht.serverURL, "unix://") {
		// mqtt server UDS listener doesn't use TLS
		// url.Parse doesn't recognize the @ in the path
		addr := ht.serverURL[7:]
		conn, err = net.Dial("unix", addr)
	} else if strings.HasPrefix(ht.serverURL, "@") {
		// using a UDS in-memory path without scheme
		conn, err = net.Dial("unix", ht.serverURL)
	} else {
		conn, err = tlsclient.ConnectTLS(ht.serverURL, nil, ht.caCert)
	}
	if err == nil {
		// clientID from the token is used
		err = ht.ConnectWithConn(jwtToken, conn)
	}
	if err != nil {
		err = fmt.Errorf("ConnectWithToken failed: %w", err)
	}
	return err
}

// CreateKeyPair returns a new set of serialized public/private key pair
func (ht *MqttHubTransport) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeECDSA)
	return k
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (ht *MqttHubTransport) Disconnect() {
	if ht.pcl != nil {
		time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		_ = ht.pcl.Disconnect(&paho.Disconnect{ReasonCode: 0})
		//time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		//ht.pcl = nil

		//ht.subscriptions = nil
		//close(ht.messageChannel)     // end the message handler loop
	}
}

// ParseResponse helper message to parse response and check for errors
func (ht *MqttHubTransport) ParseResponse(data []byte, resp interface{}) error {
	var err error
	if data == nil || len(data) == 0 {
		if resp != nil {
			err = fmt.Errorf("ParseResponse: client '%s', expected a response but none received", ht.clientID)
		} else {
			err = nil // all good
		}
	} else {
		if resp == nil {
			err = fmt.Errorf("ParseResponse: client '%s', received response but none was expected. data=%s",
				ht.clientID, data)
		} else {
			err = ser.Unmarshal(data, resp)
		}
	}
	return err
}

// Pub publishes a message and returns
func (ht *MqttHubTransport) Pub(topic string, payload []byte) (err error) {
	slog.Debug("Pub", "topic", topic)
	ctx, cancelFn := context.WithTimeout(context.Background(), ht.timeout)
	defer cancelFn()
	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
	}
	respMsg, err := ht.pcl.Publish(ctx, pubMsg)
	_ = respMsg
	if err != nil {
		return err
	}
	return err
}

// PubRequest publishes a request message and waits for an answer or until timeout
func (ht *MqttHubTransport) PubRequest(topic string, payload []byte) (resp []byte, err error) {
	slog.Info("PubRequest", "topic", topic)

	ctx, cancelFn := context.WithTimeout(context.Background(), ht.timeout)
	defer cancelFn()

	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
		Properties: &paho.PublishProperties{
			ContentType: "json",
			User: paho.UserProperties{{
				Key:   "test",
				Value: "test",
			}},
		},
	}
	// use the inbox as the custom response for this client instance
	// clone of rpc.go to workaround hangup when no response is received #111
	respMsg, err := ht.requestHandler.Request(ctx, pubMsg)
	//ar.Duration = time.Now().Sub(t1)
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
func (ht *MqttHubTransport) sendReply(req *paho.Publish, payload []byte, errResp error) error {

	slog.Debug("sendReply", "topic", req.Topic, "responseTopic", req.Properties.ResponseTopic)

	responseTopic := req.Properties.ResponseTopic
	if responseTopic == "" {
		err2 := fmt.Errorf("sendReply. No response topic. Not sending reply.")
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
	_, err := ht.pcl.Publish(ctx, replyMsg)
	if err != nil {
		slog.Warn("sendReply. Error publishing response",
			slog.String("err", err.Error()))
	}
	return err
}

// Sub subscribes to a topic
func (ht *MqttHubTransport) Sub(topic string, cb func(topic string, msg []byte)) (transports.ISubscription, error) {
	slog.Info("Sub", "topic", topic)
	spacket := &paho.Subscribe{
		Properties: nil,
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {QoS: withQos},
		},
	}
	suback, err := ht.pcl.Subscribe(context.Background(), spacket)
	ht.pcl.Router.RegisterHandler(topic, func(m *paho.Publish) {
		slog.Info("Sub, received Msg:", "topic", m.Topic)
		//clientID := m.Properties.User.Get("clientID") // experimental

		// run this in the background to allow for reentrancy
		go func() {
			cb(m.Topic, m.Payload)
		}()
	})
	_ = suback
	hcSub := &PahoSubscription{
		topic:    topic,
		pcl:      ht.pcl,
		clientID: ht.clientID,
	}
	return hcSub, err
}

// SubRequest subscribes to a requests and sends a response
// Intended for actions, config and rpc requests
func (ht *MqttHubTransport) SubRequest(
	topic string, cb func(topic string, payload []byte) (reply []byte, err error)) (
	transports.ISubscription, error) {

	spacket := &paho.Subscribe{
		Properties: nil,
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {QoS: withQos},
		},
	}
	suback, err := ht.pcl.Subscribe(context.Background(), spacket)
	_ = suback
	ht.pcl.Router.RegisterHandler(topic, func(m *paho.Publish) {

		// run this in the background to allow for reentrancy
		go func() {
			reply, err := cb(m.Topic, m.Payload)

			// TODO: cleanup. Is there something useful in properties.user?
			//clientID:= m.Properties.User.Get("clientID")
			propsJson, _ := json.Marshal(m.Properties.User)
			slog.Debug("SubRequest. Properties.User",
				"props", propsJson)

			if err != nil {
				slog.Warn("SubRequest: handle request failed",
					slog.String("err", err.Error()),
					slog.String("topic", topic))

				err = ht.sendReply(m, nil, err)
			} else {
				err = ht.sendReply(m, reply, err)
			}
			if err != nil {
				slog.Error("SubRequest. Sending reply failed", "err", err)
			}
		}()
	})

	hcSub := &PahoSubscription{
		topic: topic,
		pcl:   ht.pcl,
	}

	return hcSub, err
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
	hc := &MqttHubTransport{
		serverURL: fullURL,
		clientID:  clientID,
		caCert:    caCert,
		timeout:   time.Second * 10, // 10 for testing
	}
	return hc
}
