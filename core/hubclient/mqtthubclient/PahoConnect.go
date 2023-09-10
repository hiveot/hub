package mqtthubclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/api/go/hubclient"
	"golang.org/x/exp/slog"
	"net"
	"net/url"
	"os"
	"time"
)

// lower level autopaho functions with connect, pub and sub

const keepAliveInterval = 10 // seconds
const reconnectDelay = 10 * time.Second
const withDebug = false

// MqttHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the Mqtt messaging system.
type MqttHubClient struct {
	clientID string
	hostName string
	port     int
	pcl      *paho.Client
}

// ConnectWithCert to the Hub server
//
//	brokerURL of the server.
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
func ConnectWithCert(
	brokerURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (
	hc hubclient.IHubClient, err error) {

	conn, err := CreateTLSConnection(brokerURL, clientCert, caCert)
	if err == nil {
		hc, err = ConnectToBroker(clientID, "", conn)
	}
	return hc, err
}

// CreateTLSConnection creates a TLS connection to a MQTT broker, optionally using a client certificate.
//
//	brokerURL full URL:  tls://host:8883, ssl://host:8883,  wss://host:9001, tcps://awshost:8883/mqtt
//	clientCert to login with. Nil to not use client certs
//	caCert of the server to connect to (recommended). Nil to not verify the server connection.
func CreateTLSConnection(
	brokerURL string, clientCert *tls.Certificate, caCert *x509.Certificate) (*tls.Conn, error) {

	// connect always uses TLS
	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	tlsOpts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	// if a client certificate is given then include it
	clientCertList := make([]tls.Certificate, 0)
	if clientCert != nil {
		x509Cert, _ := x509.ParseCertificate(clientCert.Certificate[0])
		clientCertList = append(clientCertList, *clientCert)
		_, err := x509Cert.Verify(tlsOpts)
		if err != nil {
			return nil, err
		}
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	broker, err := url.Parse(brokerURL)
	if err != nil {
		return nil, err
	}
	conn, err := tls.Dial(broker.Scheme, broker.Host, tlsConfig)
	return conn, err
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

// ConnectToBroker connects to a mqtt broker using the pre-established network connection
// This allows creating connections with any available means including tcp/tls/wss/uds/pipes
func ConnectToBroker(loginID string, password string, conn net.Conn) (*MqttHubClient, error) {
	ctx := context.Background()

	// checks
	if loginID == "" {
		err := fmt.Errorf("connect - Missing Login ID")
		return nil, err
	} else if conn == nil {
		err := fmt.Errorf("connect - missing connection")
		return nil, err
	}

	pahoCfg := paho.ClientConfig{
		ClientID: loginID,
		Conn:     conn,
	}
	pahoCfg.OnClientError = func(err error) {
		slog.Warn("ConnectWithNC:OnClientError", "err", err.Error())
	}
	pahoCfg.OnServerDisconnect = func(d *paho.Disconnect) {
		slog.Warn("ConnectWithNC:OnServerDisconnect: Disconnected from broker",
			"code", d.ReasonCode,
			"loginID", loginID)
	}
	pcl := paho.NewClient(pahoCfg)
	hostName, _ := os.Hostname()
	cp := &paho.Connect{
		Password:     []byte(password),
		Username:     loginID,
		ClientID:     fmt.Sprintf("%s-%s-%d", loginID, hostName, time.Now().Unix()/int64(time.Second)),
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
		err = fmt.Errorf("failed to connect. Reason: %d - %s",
			connAck.ReasonCode, connAck.Properties.ReasonString)
		return nil, err
	}
	hc := NewMqttHubClient(pcl)
	return hc, err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func ConnectWithPassword(
	brokerURL string, loginID string, password string, caCert *x509.Certificate) (hc *MqttHubClient, err error) {

	conn, err := CreateTLSConnection(brokerURL, nil, caCert)
	if err == nil {
		hc, err = ConnectToBroker(loginID, password, conn)
	}
	return hc, err
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (mqttClient *MqttHubClient) Disconnect() {
	if mqttClient.pcl != nil {

		slog.Info("Disconnect", "clientID", mqttClient.clientID)
		//mqttClient.publish("$state", "disconnected")
		time.Sleep(time.Second / 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		_ = mqttClient.pcl.Disconnect(&paho.Disconnect{ReasonCode: 0})
		mqttClient.pcl = nil

		//mqttClient.subscriptions = nil
		//close(mqttClient.messageChannel)     // end the message handler loop
	}
}

// NewMqttHubClient creates a new instance of the hub client using the connected paho client
func NewMqttHubClient(pcl *paho.Client) *MqttHubClient {
	hc := &MqttHubClient{
		clientID: pcl.ClientID,
		pcl:      pcl,
	}
	return hc
}
