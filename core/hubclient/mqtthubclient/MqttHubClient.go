package mqtthubclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/lib/tlsclient"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"time"
)

const keepAliveInterval = 30 // seconds
const reconnectDelay = 10 * time.Second
const withDebug = false

// MqttHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the Mqtt messaging system.
type MqttHubClient struct {
	brokerURL string
	clientID  string
	keys      *ecdsa.PrivateKey
	//hostName string
	//port     int
	conn    net.Conn
	pcl     *paho.Client
	timeout time.Duration // request timeout
}

// ConnectWithConn connects to a mqtt broker using the pre-established network connection
//
// This allows creating connections with any available means including tcp/tls/wss/uds/pipes
//
//	loginID is required and used for authentication
//	password is either the password or a signed JWT token
func (mqttClient *MqttHubClient) ConnectWithConn(
	password string, conn net.Conn) error {
	ctx := context.Background()

	// clients must use a unique connection ID otherwise the previous connection will be dropped
	hostName, _ := os.Hostname()
	connectID := fmt.Sprintf("%s-%s-%s", mqttClient.clientID, hostName, time.Now().Format("20060102150405.000000"))
	slog.Info("ConnectWithConn", "loginID", mqttClient.clientID, "url", conn.RemoteAddr(), "connectID", connectID)

	// checks
	if mqttClient.clientID == "" {
		err := fmt.Errorf("connect - Missing Login ID")
		return err
	} else if conn == nil {
		err := fmt.Errorf("connect - missing connection")
		return err
	}

	pahoCfg := paho.ClientConfig{
		ClientID:      mqttClient.clientID,
		Conn:          conn,
		PacketTimeout: mqttClient.timeout,
	}
	pahoCfg.OnClientError = func(err error) {
		slog.Warn("ConnectWithNC:OnClientError", "err", err.Error())
	}
	pahoCfg.OnServerDisconnect = func(d *paho.Disconnect) {
		slog.Warn("ConnectWithNC:OnServerDisconnect: Disconnected from broker",
			"code", d.ReasonCode,
			"loginID", mqttClient.clientID)
	}
	pcl := paho.NewClient(pahoCfg)
	cp := &paho.Connect{
		Password:     []byte(password),
		Username:     mqttClient.clientID,
		ClientID:     connectID,
		Properties:   nil,
		KeepAlive:    30,
		CleanStart:   true,
		UsernameFlag: true,
		PasswordFlag: password != "",
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*100)
	defer cancelFn()
	connAck, err := pcl.Connect(ctx, cp)
	_ = connAck

	if err != nil {
		err = fmt.Errorf("failed to connect. Reason: %w", err)
		//err = fmt.Errorf("failed to connect. Reason: %d - %s",
		//	connAck.ReasonCode, connAck.Properties.ReasonString)
		return err
	}
	mqttClient.conn = conn
	mqttClient.brokerURL = conn.RemoteAddr().String()
	mqttClient.pcl = pcl
	return err
}

// ConnectWithCert to the Hub server
//
//	brokerURL of the server.
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
func (mqttClient *MqttHubClient) ConnectWithCert(
	brokerURL string, clientCert *tls.Certificate, caCert *x509.Certificate) error {

	conn, err := tlsclient.ConnectTLS(brokerURL, clientCert, caCert)
	if err != nil {
		return err
	}
	err = mqttClient.ConnectWithConn("", conn)
	mqttClient.brokerURL = brokerURL
	return err
}

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
//
//	brokerURL is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	jwtToken is the token obtained with login or refresh.
func (mqttClient *MqttHubClient) ConnectWithToken(brokerURL string, jwtToken string, caCert *x509.Certificate) error {

	conn, err := tlsclient.ConnectTLS(brokerURL, nil, caCert)
	if err != nil {
		return err
	}
	//// no need to verify here, just want to ensure the token is valid and extract the clientID
	//_, err = jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
	//	if mqttClient.keys != nil {
	//		return &mqttClient.keys.PublicKey, nil
	//	}
	//	return nil, nil
	//})
	//if err != nil {
	//	err = fmt.Errorf("invalid jwt token: %w", err)
	//	return err
	//}
	err = mqttClient.ConnectWithConn(jwtToken, conn)
	mqttClient.brokerURL = brokerURL
	return err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (mqttClient *MqttHubClient) ConnectWithPassword(
	brokerURL string, password string, caCert *x509.Certificate) error {

	conn, err := tlsclient.ConnectTLS(brokerURL, nil, caCert)
	if err != nil {
		return err
	}
	err = mqttClient.ConnectWithConn(password, conn)
	mqttClient.brokerURL = brokerURL
	return err
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (mqttClient *MqttHubClient) Disconnect() {
	if mqttClient.pcl != nil {

		slog.Info("Disconnect", "clientID", mqttClient.clientID)
		time.Sleep(time.Second / 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		_ = mqttClient.pcl.Disconnect(&paho.Disconnect{ReasonCode: 0})
		mqttClient.pcl = nil

		//mqttClient.subscriptions = nil
		//close(mqttClient.messageChannel)     // end the message handler loop
	}
}

// NewMqttHubClient creates a new instance of the hub client using the connected paho client
//
//	pcl is option if a paho client already exists. Use on of the client.ConnectXyz method if pcl is nil
func NewMqttHubClient(id string, privKey *ecdsa.PrivateKey) *MqttHubClient {
	hc := &MqttHubClient{
		clientID: id,
		keys:     privKey,
		timeout:  time.Second * 10,
	}
	return hc
}
