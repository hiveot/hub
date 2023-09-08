package natshubclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"time"
)

// lower level NATS pub/sub functions

// PublicUnauthenticatedNKey is the public seed of the unaunthenticated user
const PublicUnauthenticatedNKey = "SUAOXRE662WSIGIMSIFVQNCCIWG673K7GZMB3ZUUIF45BWGMYKECEQQJZE"

// DefaultTimeoutSec with timeout for connecting and publishing.
const DefaultTimeoutSec = 100 //3 // 100 for testing

// ConnectWithCert to the Hub server
//
//	url of the nats server. "" uses the nats default url
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
//func (hc *NatsHubClient) ConnectWithCert(url string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (err error) {
//	if url == "" {
//		url = nats.DefaultURL
//	}
//
//	caCertPool := x509.NewCertPool()
//	if caCert != nil {
//		caCertPool.AddCert(caCert)
//	}
//	opts := x509.VerifyOptions{
//		Roots:     caCertPool,
//		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
//	}
//	x509Cert, _ := x509.ParseCertificate(clientCert.Certificate[0])
//	_, err = x509Cert.Verify(opts)
//	clientCertList := []tls.Certificate{*clientCert}
//	tlsConfig := &tls.Config{
//		RootCAs:            caCertPool,
//		Certificates:       clientCertList,
//		InsecureSkipVerify: caCert == nil,
//	}
//	hc.clientID = clientID
//	hc.nc, err = nats.Connect(url,
//		nats.ID(hc.clientID),
//		nats.Secure(tlsConfig),
//		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
//	if err == nil {
//		hc.js, err = hc.nc.JetStream()
//	}
//	return err
//}

// Connect connects to a nats server using automatic detection of the given token.
//
// This does not use server tokens.
// * If token is empty or the public key, use NKeys
// * If token is a JWT token, using JWT
// * Otherwise assume it is a password
//
// UserID is used for publishing actions
func Connect(url string, clientID string, myKey nkeys.KeyPair, token string, caCert *x509.Certificate) (hc *NatsHubClient, err error) {
	pubKey, _ := myKey.PublicKey()
	if token == "" || token == pubKey {
		return ConnectWithNKey(url, clientID, myKey, caCert)
	}
	claims, err := jwt.DecodeUserClaims(token)
	if err == nil && claims.Name == clientID {
		return ConnectWithJWT(url, myKey, token, caCert)
	}
	return ConnectWithPassword(url, clientID, token, caCert)
}

// ConnectWithJWT connects to the Hub server using a NATS user JWT credentials secret
// The connection uses the client ID in the JWT token.
//
//	url is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	jwtToken is the token obtained with login or refresh. This is not a decorated token.
func ConnectWithJWT(url string, myKey nkeys.KeyPair, jwtToken string, caCert *x509.Certificate) (hc *NatsHubClient, err error) {
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

	claims, err := jwt.Decode(jwtToken)
	if err != nil {
		err = fmt.Errorf("invalid jwt token: %w", err)
		return nil, err
	}
	clientID := claims.Claims().Name
	jwtSeed, _ := myKey.Seed()
	nc, err := nats.Connect(url,
		nats.Name(clientID), // connection name for logging, debugging
		nats.Secure(tlsConfig),
		nats.CustomInboxPrefix("_INBOX."+clientID),
		nats.UserJWTAndSeed(jwtToken, string(jwtSeed)),
		nats.Token(jwtToken), // JWT token isn't passed through
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))

	if err == nil {
		hc, err = NewHubClient(clientID, nc)
	}
	return hc, err
}

// ConnectWithNC connects using the given nats connection
func ConnectWithNC(nc *nats.Conn) (hc *NatsHubClient, err error) {
	clientID := nc.Opts.Name
	if clientID == "" {
		return nil, fmt.Errorf("NATS connection has no client ID in opts.Name")
	}
	hc, err = NewHubClient(clientID, nc)
	return hc, err
}

// ConnectWithNKey connects to the Hub server using an nkey secret
//
// UserID is used for publishing actions
func ConnectWithNKey(url string, clientID string, myKey nkeys.KeyPair, caCert *x509.Certificate) (hc *NatsHubClient, err error) {
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
		return myKey.Sign(nonce)
	}
	pubKey, _ := myKey.PublicKey()
	nc, err := nats.Connect(url,
		nats.Name(clientID), // connection name for logging
		nats.Secure(tlsConfig),
		nats.Nkey(pubKey, sigCB),
		// client permissions allow this inbox prefix
		nats.CustomInboxPrefix("_INBOX."+clientID),
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))
	if err == nil {
		hc, err = NewHubClient(clientID, nc)
	}
	return hc, err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func ConnectWithPassword(
	url string, loginID string, password string, caCert *x509.Certificate) (hc *NatsHubClient, err error) {

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
	nc, err := nats.Connect(url,
		nats.UserInfo(loginID, password),
		nats.Secure(tlsConfig),
		// client permissions allow this inbox prefix
		nats.Name(loginID),
		nats.CustomInboxPrefix("_INBOX."+loginID),
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))
	if err == nil {
		hc, err = NewHubClient(loginID, nc)
	}
	return hc, err
}

// ConnectUnauthenticated connects to the Hub server as an unauthenticated user
// Intended for use by IoT devices to perform out-of-band provisioning.
func ConnectUnauthenticated(url string, caCert *x509.Certificate) (hc *NatsHubClient, err error) {
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
	nc, err := nats.Connect(url,
		nats.Secure(tlsConfig),
		// client permissions allow this inbox prefix
		nats.CustomInboxPrefix("_INBOX.unauthenticated"),
	)
	if err == nil {
		hc, err = NewHubClient("", nc)
	}
	return hc, err
}

// Disconnect from the Hub server and release all subscriptions
func (hc *NatsHubClient) Disconnect() {
	hc.nc.Close()
}
