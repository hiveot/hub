package mqtthubclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/api/go/hubclient"
	"golang.org/x/exp/slog"
	"log"
	"net/url"
	"os"
	"time"
)

// lower level autopaho functions with connect, pub and sub

const keepAliveInterval = 10 // seconds
const reconnectDelay = 10    // seconds
const withDebug = false

// ConnectWithCert to the Hub server
//
//	url of the server.
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
func ConnectWithCert(
	url string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (
	hubclient.IHubClient, error) {

	if url == "" {
	}

	hc := NewMqttHubClient()
	err := hc.connect(url, clientID, "", "", clientCert, caCert)
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
func ConnectWithNC(nc *tls.Conn) (hc *MqttHubClient, err error) {
	hc = NewMqttHubClient()
	ctx := context.Background()
	clCfg := autopaho.ClientConfig{
		//BrokerUrls:        []*url.URL{broker},
		//TlsCfg:            tlsConfig,
		KeepAlive:         keepAliveInterval,
		ConnectRetryDelay: reconnectDelay,
		ConnectTimeout:    0,
		//WebSocketCfg:      nil,
		OnConnectionUp: nil,
		OnConnectError: nil,
		Debug:          paho.NOOPLogger{},
		//PahoDebug:         nil,
		//PahoErrors:        nil,
		ClientConfig: paho.ClientConfig{
			//ClientID: loginID,
			Router: hc.router,
			Conn:   nc,
		},
	}
	hc.cm, err = autopaho.NewConnection(ctx, clCfg)
	return hc, err
}

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

	hc = NewMqttHubClient()
	err = hc.connect(url, loginID, password, "", nil, caCert)
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

// Connect to the MQTT broker using one of the given authentication methods: password, JWT token or client certificate
// Connect always uses TLS. If no CA cert is provided then any server cert will be accepted (for testing only)
//
//	brokerURL full URL:  tls://host:8883, ssl://host:8883,  wss://host:9001, tcps://awshost:8883/mqtt
//	loginID to identify as using password. Required.
//	password to login with. Empty to not use password auth.
//	jwt token to login with. Empty to not use jwt auth
//	clientCert to login with. Nil to not use client certs
//	caCert of the server to connect to (recommended). Nil to not verify the server connection.
func (hc *MqttHubClient) connect(
	brokerURL string, loginID string,
	password string, jwt string, clientCert *tls.Certificate,
	caCert *x509.Certificate) (err error) {

	// set config defaults
	if loginID == "" {
		err := fmt.Errorf("connect - Missing Login ID")
		return err
	}
	hc.clientID = loginID

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
			return err
		}
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	broker, err := url.Parse(brokerURL)
	if err != nil {
		return err
	}
	hc.router = paho.NewSingleHandlerRouter(func(m *paho.Publish) {
		log.Printf("todo: dispatch this message to subscribers:%v+", m)
	})
	clCfg := autopaho.ClientConfig{
		BrokerUrls:        []*url.URL{broker},
		TlsCfg:            tlsConfig,
		KeepAlive:         keepAliveInterval,
		ConnectRetryDelay: reconnectDelay,
		ConnectTimeout:    0,
		//WebSocketCfg:      nil,
		OnConnectionUp: nil,
		OnConnectError: nil,
		Debug:          paho.NOOPLogger{},
		//PahoDebug:         nil,
		//PahoErrors:        nil,
		ClientConfig: paho.ClientConfig{
			ClientID: loginID,
			Router:   hc.router,
		},
	}
	clCfg.OnConnectionUp = func(cmgr *autopaho.ConnectionManager, cack *paho.Connack) {
		slog.Info("mqtt:onConnect: Connected to broker", "url", brokerURL, "loginID", loginID)
		// does paho re-establishes the session?
		// Subscribe to topics already registered by the app on connect or reconnect
		//mqttClient.resubscribe()
	}
	clCfg.OnConnectError = func(err error) {
		fmt.Printf("error whilst attempting connection: %s\n", err)
	}

	clCfg.OnServerDisconnect = func(d *paho.Disconnect) {
		slog.Warn("mqtt:onConnectionLost: Disconnected from broker", "url", brokerURL,
			"code", d.ReasonCode,
			"loginID", loginID)
	}
	if withDebug {
		clCfg.Debug = log.New(os.Stdout, "autoPaho", log.Ldate|log.Ltime)
		clCfg.PahoDebug = log.New(os.Stdout, "paho", log.Ldate|log.Ltime)
	}
	ctx := context.Background()
	hc.cm, err = autopaho.NewConnection(ctx, clCfg)
	if err != nil {
		return err
	}

	slog.Info("mqtt:Connect: Connecting to MQTT broker. AutoReconnect and CleanSession are set.",
		"url", brokerURL, "clientID", loginID)

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
	return nil
}

// Disconnect from the MQTT broker and unsubscribe from all topics and set
// device state to disconnected
func (mqttClient *MqttHubClient) Disconnect() {
	if mqttClient.cm != nil {
		slog.Warn("Disconnect: Set state to disconnected and close connection")
		//mqttClient.publish("$state", "disconnected")
		time.Sleep(time.Second / 10) // Disconnect doesn't seem to wait for all messages. A small delay ahead helps
		ctx := context.Background()
		_ = mqttClient.cm.Disconnect(ctx)
		mqttClient.cm = nil

		//mqttClient.subscriptions = nil
		//close(mqttClient.messageChannel)     // end the message handler loop
	}
}
