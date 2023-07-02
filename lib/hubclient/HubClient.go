package hubclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

type HubClient struct {
	//instanceName string
	clientID   string
	nc         *nats.Conn
	js         nats.JetStreamContext
	timeoutSec int
}

// ConnectWithCert to the Hub server
//
//	url of the nats server. "" uses the nats default url
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
func (hc *HubClient) ConnectWithCert(url string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (err error) {
	if url == "" {
		url = nats.DefaultURL
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
	_, err = x509Cert.Verify(opts)
	clientCertList := []tls.Certificate{*clientCert}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	hc.clientID = clientID
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID),
		nats.Secure(tlsConfig),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
//
// Provide a CA certificate if available. If nil then the connection will still
// use TLS but no server verification will be used (InsecureSkipVerify=true)
func (hc *HubClient) ConnectWithPassword(
	url string, loginID string, password string, caCert *x509.Certificate) (err error) {
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
		nats.Name(hc.clientID),
		nats.Secure(tlsConfig),
		nats.UserInfo(loginID, password),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectWithJWT connects to the Hub server using a user JWT credentials secret
func (hc *HubClient) ConnectWithJWT(url string, token string, caCert *x509.Certificate) (err error) {
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
	// TODO. how is this supposed to work?
	// The user's JWT signed token
	userJWTHandler := func() (string, error) {
		jwtPortion, err := nkeys.ParseDecoratedJWT([]byte(token))
		return jwtPortion, err
	}
	// The handler to sign the server issued challenge
	sigCB := func(nonce []byte) ([]byte, error) {
		kp, err := nkeys.ParseDecoratedNKey([]byte(token))
		sig, _ := kp.Sign(nonce)
		return sig, err
	}
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID),
		nats.Secure(tlsConfig),
		nats.UserJWT(userJWTHandler, sigCB),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectUnauthenticated connects to the Hub server as an unauthenticated user
func (hc *HubClient) ConnectUnauthenticated(url string, caCert *x509.Certificate) (err error) {
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
		//nats.Name(hc.clientID),
		nats.Secure(tlsConfig),
		//nats.UserInfo(loginID, password),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err

}

// ConnectWithNKey connects to the Hub server using an nkey secret
//func (hc *HubClient) ConnectWithNKey(url string, userKey nkeys.KeyPair, caCert *x509.Certificate) (err error) {
//	if url == "" {
//		url = nats.DefaultURL
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
//	// The handler to sign the server issued challenge
//	sigCB := func(nonce []byte) ([]byte, error) {
//		return userKey.Sign(nonce)
//	}
//	pubKey, _ := userKey.PublicKey()
//	hc.nc, err = nats.Connect(url,
//		nats.Name(hc.clientID),
//		nats.Secure(tlsConfig),
//		nats.Nkey(pubKey, sigCB),
//		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
//	if err == nil {
//		hc.js, err = hc.nc.JetStream()
//	}
//	return err
//}

// Disconnect from the Hub server
func (hc *HubClient) Disconnect() {
	hc.nc.Close()
}

// Publish to NATS
func (hc *HubClient) Publish(subject string, payload []byte) error {
	slog.Info("publish", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// PubAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *HubClient) PubAction(publisherID string, thingID string, actionID string, payload []byte) ([]byte, error) {
	subject := MakeSubject(publisherID, thingID, vocab.VocabActionTopic, actionID)
	resp, err := hc.nc.Request(subject, payload, time.Second)
	return resp.Data, err
}

// PubEvent sends the event value to the hub
func (hc *HubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	subject := MakeSubject(hc.clientID, thingID, vocab.VocabEventTopic, eventID)
	err := hc.Publish(subject, payload)
	return err
}

// PubTD sends the TD document to the hub
func (hc *HubClient) PubTD(td *thing.TD) error {
	payload, _ := json.Marshal(td)
	subject := MakeSubject(hc.clientID, td.ID, vocab.VocabEventTopic, vocab.EventNameTD)
	err := hc.Publish(subject, payload)
	return err
}

// SubActions subscribes to actions for this device
func (hc *HubClient) SubActions(cb func(msg *hub.ActionMessage) ([]byte, error)) error {

	subject := MakeSubject(hc.clientID, "", vocab.VocabActionTopic, "")

	err := hc.Subscribe(subject, func(msg *nats.Msg) {
		md, _ := msg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		payload := msg.Data
		pubID, thID, _, name, err := SplitSubject(msg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", msg.Subject)
			return
		}
		action := &hub.ActionMessage{
			//SenderID: msg.Header.
			ActionID:    name,
			PublisherID: pubID,
			ThingID:     thID,
			Timestamp:   timeStamp.Unix(),
			Payload:     payload,
		}
		resp, err := cb(action)
		if err == nil {
			err = msg.Respond(resp)
			if err != nil {
			}
		}
	})
	return err
}

// SubGroup subscribes to an event on subject things.{publisherID}.{thingID}.event.{eventName}, where . is the separator.
// leave publisherID, thingID and/or eventName empty to use wildcards.
//
// This will likely be replaced with subscribing to events in a group instead of a publisher/thing
func (hc *HubClient) SubGroup(
	groupName, thingID, eventID string, cb func(msg *hub.EventMessage)) error {

	//subject := MakeSubject(publisherID, thingID, vocab.VocabEventTopic, eventName)
	subject := groupName

	err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
		md, _ := natsMsg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		pubID, thID, _, name, err := SplitSubject(natsMsg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err,
				"subject", natsMsg.Subject)
			return
		}
		msg := &hub.EventMessage{
			//SenderID: msg.Header.
			EventID:     name,
			PublisherID: pubID,
			ThingID:     thID,
			Timestamp:   timeStamp.Unix(),
			Payload:     natsMsg.Data,
		}
		cb(msg)
	})
	return err
}

// Subscribe to NATS
func (hc *HubClient) Subscribe(subject string, cb func(msg *nats.Msg)) error {
	slog.Info("subscribe", "subject", subject)
	subscription, err := hc.nc.Subscribe(subject, cb)
	_ = subscription
	return err
}

// NewHubClient instantiates a client for connecting to the Hub
func NewHubClient(instanceName string) hub.IHubClient {
	hc := &HubClient{
		clientID:   instanceName,
		timeoutSec: 10,
	}
	return hc
}
