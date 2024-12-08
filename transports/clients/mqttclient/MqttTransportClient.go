package mqttclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients/base"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/teris-io/shortid"
	"log"
	"log/slog"
	"net/url"
	"os"
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

// MqttTransportClient provides WoT protocol binding for the MQTT protocol
// This implements the IClientConnection interface.
type MqttTransportClient struct {
	base.TransportClient

	// mqtt broker url
	brokerURL string
	// paho mqtt client
	pahoClient *autopaho.ConnectionManager
	// enable debug logging in the paho client
	logDebug bool
	// handler to obtain a form for the operation
	getForm func(op string) td.Form
	// authentication token
	authToken string
	//
	inboxTopic       string // set on first request, cleared on disconnect
	connectID        string // unique connection ID
	connectionStatus string
	lastError        atomic.Pointer[error]
	//
	correlData    map[string]chan *paho.Publish
	subscriptions map[string]bool
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (cl *MqttTransportClient) ConnectWithPassword(password string) (newToken string, err error) {
	// same process using password or token
	newToken, err = cl.ConnectWithToken(password)
	return newToken, err
}

// ConnectWithToken establishes a connection to the MQTT broker using the paho client
func (cl *MqttTransportClient) ConnectWithToken(token string) (newToken string, err error) {
	// setup TLS
	caCertPool := x509.NewCertPool()
	if cl.BaseCaCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", cl.brokerURL))
	} else {
		caCertPool.AddCert(cl.BaseCaCert)
	}
	tlsCfg := &tls.Config{
		RootCAs: caCertPool,
		//Certificates:       clientCertList,
		InsecureSkipVerify: cl.BaseCaCert == nil,
	}

	//safeConn := packets.NewThreadSafeConn(conn)
	// Setup the Paho client configuration
	hostName, _ := os.Hostname()
	connectID := fmt.Sprintf("%s-%s-%s", cl.BaseClientID, hostName, time.Now().Format("20060102150405.000"))
	logger := log.Default()
	u, err := url.Parse(cl.brokerURL)
	autoCfg := autopaho.ClientConfig{
		BrokerUrls: []*url.URL{u},
		PahoErrors: logger,
		ClientConfig: paho.ClientConfig{
			ClientID: connectID, // instance ID, not the clientID
			//Conn:          safeConn,    // autopaho ignores this :(
			PacketTimeout: cl.BaseTimeout,
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
	autoCfg.ConnectUsername = cl.BaseClientID
	autoCfg.ConnectPassword = []byte(cl.authToken)
	autoCfg.OnConnectError = cl.onPahoConnectionError
	autoCfg.OnConnectionUp = cl.onPahoConnect
	autoCfg.OnServerDisconnect = func(disconnect *paho.Disconnect) {
		cl.BaseIsConnected.Store(false)
		if cl.BaseConnectHandler != nil {
			cl.BaseConnectHandler(false, nil)
		}
	}

	if cl.logDebug {
		autoCfg.PahoDebug = logger
	}

	// Warning, can't use WithTimeout as it will disconnect the perfectly good
	// connection after the timeout has passed.
	ctx := context.Background()

	pcl, err := autopaho.NewConnection(ctx, autoCfg)

	cl.BaseMux.Lock()
	cl.connectID = connectID
	cl.pahoClient = pcl
	cl.BaseMux.Unlock()

	// Wait for the connection to come up
	ctx, cancelFn := context.WithTimeout(ctx, time.Second*1)
	err = pcl.AwaitConnection(ctx)
	cancelFn()
	if err != nil {
		// provide a more meaningful error, the actual error is not returned by paho
		cl.BaseMux.RLock()
		errptr := cl.lastError.Load()
		err = *errptr
		cl.BaseMux.RUnlock()
	}
	return token, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *MqttTransportClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeECDSA)
//	return k
//}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (cl *MqttTransportClient) Disconnect() {
	cl.BaseMux.Lock()
	pcl := cl.pahoClient
	connectID := cl.connectID

	cl.pahoClient = nil
	cl.inboxTopic = ""
	cl.connectID = ""
	cl.BaseIsConnected.Store(false)
	err := errors.New("disconnected by user")
	cl.lastError.Store(&err)
	cl.BaseMux.Unlock()

	slog.Info("Disconnecting", "cid", connectID)
	if pcl != nil {
		//time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		err := pcl.Disconnect(context.Background())
		if err != nil {
			slog.Error("disconnect error", "err", err)
		}
	}
}

// GetServerURL returns the schema://address:port of the server connection
func (cl *MqttTransportClient) GetServerURL() string {
	return cl.brokerURL
}

// handlePahoMessage handles incoming mqtt messages
func (cl *MqttTransportClient) handlePahoMessage(m *paho.Publish) {
	slog.Debug("handlePahoMessage", slog.String("topic", m.Topic))
	// run this in the background to allow for reentrancy
	go func() {
		// TODO determine if this is an event or request type message
	}()
}

// InvokeAction invokes an action on a thing and wait for the response
func (cl *MqttTransportClient) InvokeAction(dThingID, name string, input any, output any) error {
	return cl.SendRequest(wot.OpInvokeAction, dThingID, name, input, output)
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *MqttTransportClient) Logout() error {
	err := fmt.Errorf("Not implemented")
	return err
}
func (cl *MqttTransportClient) onPahoConnect(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	cl.BaseMux.Lock()
	defer cl.BaseMux.Unlock()

	cl.BaseIsConnected.Store(true)
	cl.lastError.Store(nil)
	subList := make([]string, 0, len(cl.subscriptions))
	for topic := range cl.subscriptions {
		subList = append(subList, topic)
	}
	connectHandler := cl.BaseConnectHandler

	go func() {
		// (re)subscribe all subscriptions
		for _, topic := range subList {
			err := cl.sub(topic)
			if err != nil {
				slog.Error("onConnect. resubscribe failed", "topic", topic)
			}
		}
		// now subscriptions have been restored, inform subscriber
		if connectHandler != nil {
			connectHandler(true, nil)
		}
	}()
}

// paho reports an error but will keep trying until disconnect is called
func (cl *MqttTransportClient) onPahoConnectionError(err error) {
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
			slog.Error("connection error", "clientID", cl.BaseClientID, "err", err)
		}
		// notify on change
		cl.BaseMux.RLock()
		oldStatus := cl.connectionStatus
		oldErrPtr := cl.lastError.Load()
		cl.BaseMux.RUnlock()
		if connStatus != oldStatus || connErr != *oldErrPtr {
			cl.BaseMux.Lock()
			cl.connectionStatus = connStatus
			cl.lastError.Store(&connErr)
			cl.BaseIsConnected.Store(false)
			connHandler := cl.BaseConnectHandler
			cl.BaseMux.Unlock()
			connHandler(false, connErr)
		}
		slog.Info("onPahoConnectionError", "err", connErr.Error())
		// don't retry on authentication error
		cl.BaseMux.RLock()
		pcl := cl.pahoClient
		cl.BaseMux.RUnlock()
		if connStatus == ConnStatUnauthorized && pcl != nil {
			_ = pcl.Disconnect(context.Background())
		}
	}()
}

// PubEvent publishes a message and returns
func (cl *MqttTransportClient) PubEvent(topic string, payload []byte) (err error) {
	slog.Debug("PubEvent", "topic", topic)
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.BaseTimeout)
	defer cancelFn()
	pubMsg := &paho.Publish{
		QoS:     0, //withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
	}
	cl.BaseMux.RLock()
	pcl := cl.pahoClient
	cl.BaseMux.RUnlock()
	if pcl != nil {
		_, err = pcl.Publish(ctx, pubMsg)
	} else {
		err = errors.New("no connection with the hub")
	}
	return err
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *MqttTransportClient) RefreshToken(oldToken string) (newToken string, err error) {
	return oldToken, fmt.Errorf("not implemented")
}

// SendError sends an error result to a remote consumer.
func (cl *MqttTransportClient) SendError(
	thingID, name string, errResponse string, requestID string) {
	slog.Error("SendError: not implemented")
}

// SendRequest sends a request and waits for a result
// The operation is used to retrieve the form of the Thing whose operation to
// send and determine the endpoint. If no form can be retrieved this falls
// back to the hub known endpoint.
func (cl *MqttTransportClient) SendRequest(
	operation string, thingID, name string, input interface{}, output interface{}) error {
	return fmt.Errorf("not implemented")
}

// SendResponse sends the action response message
func (cl *MqttTransportClient) SendResponse(
	dThingID, name string, output any, requestID string) {
	// FIXME: implement
}

// SendNotification sends the operation as a notification and returns immediately.
func (cl *MqttTransportClient) SendNotification(
	operation string, thingID, name string, input interface{}) error {

	// FIXME: implement message envelope
	err := errors.New("not implemented")
	return err
}

//
//// SendResponse [agent] sends a operation response to the server.
//// (todo)
//func (cl *MqttTransportClient) SendResponse(requestID string, data any) {
//	topic := ""
//	payload, _ := jsoniter.Marshal(data)
//	cl._send(topic, payload)
//}

// sub builds a subscribe packet and submits it
func (cl *MqttTransportClient) sub(topic string) error {
	packet := &paho.Subscribe{
		Properties: nil,
		Subscriptions: []paho.SubscribeOptions{
			{
				Topic: topic,
				QoS:   withQos,
			},
		},
	}
	cl.BaseMux.RLock()
	pcl := cl.pahoClient
	cl.BaseMux.RUnlock()
	suback, err := pcl.Subscribe(context.Background(), packet)
	_ = suback
	return err
}

// NewMqttTransportClient creates a new instance of the mqtt binding client
//
//	fullURL of broker to connect to, including the schema
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler that provides a form for the given operation
//	timeout for waiting for response. 0 to use the default.
func NewMqttTransportClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm func(op string) td.Form,
	timeout time.Duration) *MqttTransportClient {

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

	cl := MqttTransportClient{
		TransportClient: base.TransportClient{
			BaseCaCert:       caCert,
			BaseClientID:     clientID,
			BaseConnectionID: clientID + "." + shortid.MustGenerate(),
			BaseTimeout:      timeout,
			BaseProtocolType: transports.ProtocolTypeMQTTS,
			BaseFullURL:      fullURL,
			BaseRnrChan:      tputils.NewRnRChan(),
		},

		// max delay 3 seconds before a response is expected
		brokerURL:     fullURL,
		getForm:       getForm,
		correlData:    make(map[string]chan *paho.Publish),
		subscriptions: make(map[string]bool),
	}
	cl.BaseConnectHandler = func(connected bool, err error) {
		slog.Info("connection status change",
			"newStatus", connected,
			"lastError", err,
			"clientID", clientID)
	}
	//err = cl.pahoConnect()

	return &cl
}
