package mqttbinding

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports"
	jsoniter "github.com/json-iterator/go"
	"log"
	"log/slog"
	"net/url"
	"os"
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

// MqttBindingClient provides WoT protocol binding for the MQTT protocol
type MqttBindingClient struct {
	// mqtt broker url
	brokerURL string
	// the CA certificate to validate the server TLS connection
	caCert *x509.Certificate
	// The client ID of the user of this binding
	clientID string
	// paho mqtt client
	pahoClient *autopaho.ConnectionManager
	// enable debug logging in the paho client
	logDebug bool
	// timeout for packet transfer
	timeout time.Duration
	// authentication token
	authToken string
	//
	inboxTopic       string // set on first request, cleared on disconnect
	connectID        string // unique connection ID
	isConnected      atomic.Bool
	connectionStatus string
	lastError        atomic.Pointer[error]
	//
	correlData    map[string]chan *paho.Publish
	subscriptions map[string]bool

	//
	mux sync.RWMutex
	// callbacks for connection, events and requests
	connectHandler func(connected bool, err error)
	// client side handler that receives messages for consumers
	messageHandler transports.MessageHandler
	// map of requestID to delivery status update channel
	requestHandler transports.RequestHandler
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (cl *MqttBindingClient) ConnectWithPassword(password string) (newToken string, err error) {
	// same process using password or token
	newToken, err = cl.ConnectWithToken(password)
	return newToken, err
}

// ConnectWithToken establishes a connection to the MQTT broker using the paho client
func (cl *MqttBindingClient) ConnectWithToken(token string) (newToken string, err error) {
	// setup TLS
	caCertPool := x509.NewCertPool()
	if cl.caCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", cl.brokerURL))
	} else {
		caCertPool.AddCert(cl.caCert)
	}
	tlsCfg := &tls.Config{
		RootCAs: caCertPool,
		//Certificates:       clientCertList,
		InsecureSkipVerify: cl.caCert == nil,
	}

	//safeConn := packets.NewThreadSafeConn(conn)
	// Setup the Paho client configuration
	hostName, _ := os.Hostname()
	connectID := fmt.Sprintf("%s-%s-%s", cl.clientID, hostName, time.Now().Format("20060102150405.000"))
	logger := log.Default()
	u, err := url.Parse(cl.brokerURL)
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
	autoCfg.ConnectUsername = cl.clientID
	autoCfg.ConnectPassword = []byte(cl.authToken)
	autoCfg.OnConnectError = cl.onPahoConnectionError
	autoCfg.OnConnectionUp = cl.onPahoConnect
	autoCfg.OnServerDisconnect = func(disconnect *paho.Disconnect) {
		cl.isConnected.Store(false)
		if cl.connectHandler != nil {
			cl.connectHandler(false, nil)
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
	cl.connectID = connectID
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
	return token, err
}

// CreateKeyPair returns a new set of serialized public/private key pair
//func (cl *MqttBindingClient) CreateKeyPair() (cryptoKeys keys.IHiveKey) {
//	k := keys.NewKey(keys.KeyTypeECDSA)
//	return k
//}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (cl *MqttBindingClient) Disconnect() {
	cl.mux.Lock()
	pcl := cl.pahoClient
	connectID := cl.connectID

	cl.pahoClient = nil
	cl.inboxTopic = ""
	cl.connectID = ""
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

// handlePahoMessage handles incoming mqtt messages
func (cl *MqttBindingClient) handlePahoMessage(m *paho.Publish) {
	slog.Debug("handlePahoMessage", slog.String("topic", m.Topic))
	// run this in the background to allow for reentrancy
	go func() {
		// TODO determine if this is an event or request type message
	}()
}

// GetClientID returns the client's account ID
func (cl *MqttBindingClient) GetClientID() string {
	return cl.clientID
}

func (cl *MqttBindingClient) GetConnectionStatus() (bool, string, error) {
	var lastErr error = nil
	// lastError is stored as pointer because atomic.Value cannot switch between error and nil type
	if cl.lastError.Load() != nil {
		lastErrPtr := cl.lastError.Load()
		lastErr = *lastErrPtr
	}
	return cl.isConnected.Load(), cl.connectID, lastErr
}

// GetProtocolType returns the type of protocol this client supports
func (cl *MqttBindingClient) GetProtocolType() string {
	return transports.ProtocolTypeMQTT
}

// GetServerURL returns the schema://address:port of the server connection
func (cl *MqttBindingClient) GetServerURL() string {
	return cl.brokerURL
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *MqttBindingClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *MqttBindingClient) Logout() error {
	err := fmt.Errorf("Not implemented")
	return err
}
func (cl *MqttBindingClient) onPahoConnect(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	cl.mux.Lock()
	defer cl.mux.Unlock()

	cl.isConnected.Store(true)
	cl.lastError.Store(nil)
	subList := make([]string, 0, len(cl.subscriptions))
	for topic := range cl.subscriptions {
		subList = append(subList, topic)
	}
	connectHandler := cl.connectHandler

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
func (cl *MqttBindingClient) onPahoConnectionError(err error) {
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
				//connErr = fmt.Errorf("disconnected user '%s': %s", cl.clientID, err.Error())
			}
		default:
			connStatus = ConnStatConnecting
			connErr = fmt.Errorf("disconnected: %w", err)
			slog.Error("connection error", "clientID", cl.clientID, "err", err)
		}
		// notify on change
		cl.mux.RLock()
		oldStatus := cl.connectionStatus
		oldErrPtr := cl.lastError.Load()
		cl.mux.RUnlock()
		if connStatus != oldStatus || connErr != *oldErrPtr {
			cl.mux.Lock()
			cl.connectionStatus = connStatus
			cl.lastError.Store(&connErr)
			cl.isConnected.Store(false)
			connHandler := cl.connectHandler
			cl.mux.Unlock()
			connHandler(false, connErr)
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

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *MqttBindingClient) RefreshToken(oldToken string) (newToken string, err error) {
	return oldToken, fmt.Errorf("not implemented")
}

func (cl *MqttBindingClient) Rpc(form tdd.Form, dThingID, name string, input interface{},
	output interface{}) error {
	return fmt.Errorf("not implemented")
}

// SendOperation sends the operation described in the Form. (todo)
// The form must describe the MQTT protocol.
func (cl *MqttBindingClient) SendOperation(
	form tdd.Form, dThingID, name string, input interface{}, output interface{},
	correlationID string) (status string, err error) {

	// FIXME: implement message envelope
	err = errors.New("not implemented")
	status = transports.RequestFailed
	return status, err
}

// SendOperationStatus [agent] sends a operation progress status update to the server.
// (todo)
func (cl *MqttBindingClient) SendOperationStatus(stat transports.RequestStatus) {
	topic := ""
	payload, _ := jsoniter.Marshal(stat)
	cl.PubEvent(topic, payload)
}

// SetConnectHandler sets the notification handler of connection failure
// Intended to notify the client that a reconnect or relogin is needed.
func (cl *MqttBindingClient) SetConnectHandler(cb func(connected bool, err error)) {
	cl.mux.Lock()
	cl.connectHandler = cb
	cl.mux.Unlock()
}

// SetMessageHandler set the handler that receives event type messages send by the server.
// This requires a sub-protocol with a return channel.
func (cl *MqttBindingClient) SetMessageHandler(cb transports.MessageHandler) {
	cl.mux.Lock()
	cl.messageHandler = cb
	cl.mux.Unlock()
}

// SetRequestHandler set the handler that receives requests from the server,
// where a status response is expected.
// This requires a sub-protocol with a return channel.
func (cl *MqttBindingClient) SetRequestHandler(cb transports.RequestHandler) {
	cl.mux.Lock()
	cl.requestHandler = cb
	cl.mux.Unlock()
}

// sub builds a subscribe packet and submits it
func (cl *MqttBindingClient) sub(topic string) error {
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

// NewMqttBindingClient creates a new instance of the mqtt binding client
//
//	fullURL of broker to connect to, including the schema
//	clientID to connect as
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	timeout for waiting for response. 0 to use the default.
func NewMqttBindingClient(fullURL string, clientID string,
	clientCert *tls.Certificate, caCert *x509.Certificate,
	timeout time.Duration) *MqttBindingClient {

	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewMqttBindingClient: No CA certificate. InsecureSkipVerify used",
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

	cl := MqttBindingClient{
		//_status: hubclient.TransportStatus{
		//	HubURL:               fmt.Sprintf("https://%s", hostPort),
		caCert:   caCert,
		clientID: clientID,

		// max delay 3 seconds before a response is expected
		brokerURL:     fullURL,
		timeout:       timeout,
		correlData:    make(map[string]chan *paho.Publish),
		subscriptions: make(map[string]bool),
		connectHandler: func(connected bool, err error) {
			slog.Info("connection status change",
				"newStatus", connected,
				"lastError", err,
				"clientID", clientID)
		},
	}
	//err = cl.pahoConnect()

	return &cl
}
