package natshubclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

// PublicUnauthenticatedNKey is the public seed of the unaunthenticated user
const PublicUnauthenticatedNKey = "SUAOXRE662WSIGIMSIFVQNCCIWG673K7GZMB3ZUUIF45BWGMYKECEQQJZE"

// DefaultTimeoutSec with timeout for connecting and publishing.
const DefaultTimeoutSec = 100 //3 // 100 for testing

// Higher level Hub event and action functions

// NatsHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the NATS/Jetstream messaging system.
type NatsHubClient struct {
	clientID string
	myKey    nkeys.KeyPair
	nc       *nats.Conn
	js       nats.JetStreamContext
	timeout  time.Duration
}

// ClientID the client is authenticated as to the server
func (hc *NatsHubClient) ClientID() string {
	return hc.clientID
}

// ConnectWithConn to the hub server using the given nats client connection
func (hc *NatsHubClient) ConnectWithConn(password string, nconn *nats.Conn) (err error) {

	st, _ := nconn.TLSConnectionState()
	_ = st
	slog.Info("ConnectWithConn", "loginID", hc.clientID, "url", st.ServerName)

	// checks
	if hc.clientID == "" {
		err := fmt.Errorf("connect - Missing Login ID")
		return err
	} else if nconn == nil {
		err := fmt.Errorf("connect - missing connection")
		return err
	}
	hc.nc = nconn
	hc.js, err = nconn.JetStream()
	return err
}

// ConnectWithCert to the Hub server
//
//	url of the nats server. "" uses the nats default url
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
func (hc *NatsHubClient) ConnectWithCert(url string, clientCert *tls.Certificate, caCert *x509.Certificate) (err error) {

	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	x509Cert, _ := x509.ParseCertificate(clientCert.Certificate[0])
	_, err = x509Cert.Verify(opts)
	clientCertList := []tls.Certificate{*clientCert}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID),
		nats.Secure(tlsConfig),
		nats.Timeout(hc.timeout))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectWithJWT connects to the Hub server using a NATS user JWT credentials secret
// The connection uses the client ID in the JWT token.
//
//	serverURL is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	jwtToken is the token obtained with login or refresh. This is not a decorated token.
func (hc *NatsHubClient) ConnectWithJWT(
	serverURL string, jwtToken string, caCert *x509.Certificate) (err error) {
	if serverURL == "" {
		serverURL = nats.DefaultURL
	}

	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: caCert == nil,
	}
	//
	//claims, err := jwt.Decode(jwtToken)
	//if err != nil {
	//	err = fmt.Errorf("invalid jwt token: %w", err)
	//	return err
	//}
	//clientID := claims.Claims().Name
	jwtSeed, _ := hc.myKey.Seed()
	hc.nc, err = nats.Connect(serverURL,
		nats.Name(hc.clientID), // connection name for logging, debugging
		nats.Secure(tlsConfig),
		nats.CustomInboxPrefix("_INBOX."+hc.clientID),
		nats.UserJWTAndSeed(jwtToken, string(jwtSeed)),
		nats.Token(jwtToken), // JWT token isn't passed through in callout
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectWithToken connects to the Hub server using a NATS user a token obtained at login or refresh
//
//	serverURL is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	token is the token obtained with login or refresh.
func (hc *NatsHubClient) ConnectWithToken(
	serverURL string, token string, caCert *x509.Certificate) (err error) {
	if serverURL == "" {
		serverURL = nats.DefaultURL
	}
	_, err = jwt.Decode(token)
	// if this isn't a valid JWT, try the nkey login and ignore the token
	// TODO: remove this once JWT is properly supported using callouts
	if err != nil {
		err = hc.ConnectWithKey(serverURL, caCert)
		//	err = fmt.Errorf("invalid jwt token: %w", err)
		//	return err
	} else {
		err = hc.ConnectWithJWT(serverURL, token, caCert)
	}
	return err
}

// ConnectWithNC connects using the given nats connection
//func ConnectWithNC(nc *nats.Conn) (hc *NatsHubClient, err error) {
//	clientID := nc.Opts.Name
//	if clientID == "" {
//		return nil, fmt.Errorf("NATS connection has no client ID in opts.Name")
//	}
//	hc, err = NewHubClient(clientID, nc)
//	return hc, err
//}

// ConnectWithKey connects to the Hub server using the client's nkey secret
func (hc *NatsHubClient) ConnectWithKey(url string, caCert *x509.Certificate) error {
	var err error
	if url == "" {
		url = nats.DefaultURL
	}

	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: caCert == nil,
	}
	// The handler to sign the server issued challenge
	sigCB := func(nonce []byte) ([]byte, error) {
		return hc.myKey.Sign(nonce)
	}
	pubKey, _ := hc.myKey.PublicKey()
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID), // connection name for logging
		nats.Secure(tlsConfig),
		nats.Nkey(pubKey, sigCB),
		// client permissions allow this inbox prefix
		nats.CustomInboxPrefix("_INBOX."+hc.clientID),
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))

	if err == nil {
		err = hc.ConnectWithConn("", hc.nc)
	}
	return err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (hc *NatsHubClient) ConnectWithPassword(
	url string, password string, caCert *x509.Certificate) (err error) {

	if url == "" {
		url = nats.DefaultURL
	}
	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: caCert == nil,
	}
	hc.nc, err = nats.Connect(url,
		nats.UserInfo(hc.clientID, password),
		nats.Secure(tlsConfig),
		// client permissions allow this inbox prefix
		nats.Name(hc.clientID),
		nats.CustomInboxPrefix("_INBOX."+hc.clientID),
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectUnauthenticated connects to the Hub server as an unauthenticated user
// Intended for use by IoT devices to perform out-of-band provisioning.
//func ConnectUnauthenticated(url string, caCert *x509.Certificate) (hc *NatsHubClient, err error) {
//	if url == "" {
//		url = nats.DefaultURL
//	}
//	caCertPool := x509.NewCertPool()
//	if caCert != nil {
//		caCertPool.AddCert(caCert)
//	}
//	tlsConfig := &tls.Config{
//		RootCAs:            caCertPool,
//		InsecureSkipVerify: caCert == nil,
//	}
//	nc, err := nats.Connect(url,
//		nats.Secure(tlsConfig),
//		// client permissions allow this inbox prefix
//		nats.CustomInboxPrefix("_INBOX.unauthenticated"),
//	)
//	if err == nil {
//		hc, err = NewHubClient("", nc)
//	}
//	return hc, err
//}

// Disconnect from the Hub server and release all subscriptions
func (hc *NatsHubClient) Disconnect() {
	hc.nc.Close()
}

// Refresh an authentication token.
//
//	clientID is the userID, deviceID or serviceID whose token to refresh.
//	oldToken must be a valid token obtained at login or refresh
//
// This returns a short lived auth token that can be used to authenticate with the hub
// This fails if the token has expired or does not belong to the clientID
//func (hc *NatsHubClient) Refresh(clientID string, oldToken string) (newToken string, err error) {
//	req := &authn.RefreshReq{
//		UserID: clientID,
//		OldToken: oldToken,
//	}
//	msg, _ := ser.Marshal(req)
//	subject := MakeThingsSubject(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
//	slog.Info("PubTD", "subject", subject)
//	err := hc.Publish(subject, payload)
//	resp := &authn.RefreshResp{}
//	err = hubclient.ParseResponse(data, err, resp)
//	if err == nil {
//		authToken = resp.JwtToken
//	}
//	return err
//}

// NewNatsHubClient creates a new instance of the hub client for use
// with the NATS messaging server
func NewNatsHubClient(clientID string, myKey nkeys.KeyPair) *NatsHubClient {

	hc := &NatsHubClient{
		clientID: clientID,
		myKey:    myKey,
		timeout:  time.Duration(DefaultTimeoutSec) * time.Second,
	}
	return hc
}
