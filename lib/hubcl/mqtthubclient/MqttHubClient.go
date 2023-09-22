package mqtthubclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/tlsclient"
	"log/slog"
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
	serverURL string
	caCert    *x509.Certificate
	clientID  string
	privKey   *ecdsa.PrivateKey
	//hostName string
	//port     int
	conn    net.Conn
	pcl     *paho.Client
	timeout time.Duration // request timeout
}

// ClientID the client is authenticated as to the server
func (hc *MqttHubClient) ClientID() string {
	return hc.clientID
}

// ConnectWithConn connects to a mqtt broker using the pre-established network connection
// This allows creating connections with any available means including tcp/tls/wss/uds/pipes
//
//	loginID is required and used for authentication
//	password is either the password or a signed JWT token
func (hc *MqttHubClient) ConnectWithConn(
	password string, conn net.Conn) error {
	ctx := context.Background()

	// clients must use a unique connection ID otherwise the previous connection will be dropped
	hostName, _ := os.Hostname()
	connectID := fmt.Sprintf("%s-%s-%s", hc.clientID, hostName, time.Now().Format("20060102150405.000000"))
	slog.Info("ConnectWithConn", "loginID", hc.clientID, "url", conn.RemoteAddr(), "connectID", connectID)

	// checks
	if hc.clientID == "" {
		err := fmt.Errorf("connect - Missing Login ID")
		return err
	} else if conn == nil {
		err := fmt.Errorf("connect - missing connection")
		return err
	}

	pahoCfg := paho.ClientConfig{
		ClientID:      hc.clientID,
		Conn:          conn,
		PacketTimeout: hc.timeout,
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
			"loginID", hc.clientID)
	}
	pcl := paho.NewClient(pahoCfg)
	cp := &paho.Connect{
		Password: []byte(password),
		Username: hc.clientID,
		ClientID: connectID,
		// TODO: consider including a signed nonce when connecting with key
		Properties:   &paho.ConnectProperties{},
		KeepAlive:    60,
		CleanStart:   true,
		UsernameFlag: true,
		PasswordFlag: password != "",
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), hc.timeout)
	defer cancelFn()
	connAck, err := pcl.Connect(ctx, cp)
	_ = connAck

	if err != nil {
		err = fmt.Errorf("failed to connect. Reason: %w", err)
		//err = fmt.Errorf("failed to connect. Reason: %d - %s",
		//	connAck.ReasonCode, connAck.Properties.ReasonString)
		return err
	}
	hc.conn = conn
	hc.pcl = pcl

	return err
}

// ConnectWithCert to the Hub server
//
//	brokerURL of the server.
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
func (hc *MqttHubClient) ConnectWithCert(clientCert tls.Certificate) error {

	conn, err := tlsclient.ConnectTLS(hc.serverURL, &clientCert, hc.caCert)
	if err != nil {
		return err
	}
	err = hc.ConnectWithConn("", conn)
	return err
}

// ConnectWithToken connects to the Hub server using a user JWT credentials secret
// The token clientID must match that of the client
// A private key might be required in future.
//
//	brokerURL is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	jwtToken is the token obtained with login or refresh.
func (hc *MqttHubClient) ConnectWithToken(jwtToken string) error {

	conn, err := tlsclient.ConnectTLS(hc.serverURL, nil, hc.caCert)
	if err != nil {
		return err
	}
	err = hc.ConnectWithConn(jwtToken, conn)
	return err
}

// ConnectWithTokenFile is a convenience function to read token and key from file and connect to the server
func (hc *MqttHubClient) ConnectWithTokenFile(tokenFile string, keyFile string) error {
	token, err := os.ReadFile(tokenFile)
	if err == nil && keyFile != "" {
		hc.privKey, err = certs.LoadKeysFromPEM(keyFile)
	}
	if err != nil {
		return err
	}
	err = hc.ConnectWithToken(string(token))
	return err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (hc *MqttHubClient) ConnectWithPassword(password string) error {

	conn, err := tlsclient.ConnectTLS(
		hc.serverURL,
		nil,
		hc.caCert)
	if err != nil {
		return err
	}
	err = hc.ConnectWithConn(password, conn)
	return err
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (hc *MqttHubClient) Disconnect() {
	if hc.pcl != nil {

		slog.Info("Disconnect", "clientID", hc.clientID)
		//time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		_ = hc.pcl.Disconnect(&paho.Disconnect{ReasonCode: 0})
		//time.Sleep(time.Millisecond * 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		//hc.pcl = nil

		//hc.subscriptions = nil
		//close(hc.messageChannel)     // end the message handler loop
	}
}

// NewMqttHubClient creates a new instance of the hub client using the connected paho client
//
//	url of broker to connect to, starting with "mqtts" or "mqttwss"
//	id is the client's ID to identify as for the session.
//	privKey for connecting with Key or JWT, and possibly encryption (future)
//	caCert of the server to validate the server or nil to not check the server cert
func NewMqttHubClient(url string, id string, privKey *ecdsa.PrivateKey, caCert *x509.Certificate) *MqttHubClient {
	if url == "" {
		url = "mqtts://localhost:"
	}
	hc := &MqttHubClient{
		serverURL: url,
		caCert:    caCert,
		clientID:  id,
		privKey:   privKey,
		timeout:   time.Second * 10,
	}
	return hc
}
