package natshubclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/core/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// PublicUnauthenticatedNKey is the public seed of the unaunthenticated user
const PublicUnauthenticatedNKey = "SUAOXRE662WSIGIMSIFVQNCCIWG673K7GZMB3ZUUIF45BWGMYKECEQQJZE"

// NatsHubSubscription nats subscription helper
// This implements ISubscription
type NatsHubSubscription struct {
	nsub *nats.Subscription
}

func (ns *NatsHubSubscription) Unsubscribe() {
	err := ns.nsub.Unsubscribe()
	if err != nil {
		slog.Error("Unsubscribe error", "error", err)
	}
}

// HubNatsClient manages the hub server connection with nats based pub/sub messaging
// This implements the IHubClient interface.
// This implementation is based on the NATS/Jetstream messaging system.
type HubNatsClient struct {
	//instanceName string
	clientID   string
	nc         *nats.Conn
	js         nats.JetStreamContext
	timeoutSec int
	// myKey is the clients private nkey, needed to connect using JWT tokens.
	myKey nkeys.KeyPair
	// public 'unauthenticated' user nkey for connecting to login with password
	unauthenticatedKey nkeys.KeyPair
}

// ClientID the client is authenticated as to the server
func (hc *HubNatsClient) ClientID() string {
	return hc.clientID
}

// ConnectWithCert to the Hub server
//
//	url of the nats server. "" uses the nats default url
//	clientID to connect as
//	clientCert for certificate based authentication
//	caCert of the server
//func (hc *HubNatsClient) ConnectWithCert(url string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (err error) {
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
//		nats.Name(hc.clientID),
//		nats.Secure(tlsConfig),
//		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
//	if err == nil {
//		hc.js, err = hc.nc.JetStream()
//	}
//	return err
//}

// ConnectWithJWT connects to the Hub server using a NATS user JWT credentials secret
// The connection uses the client ID in the JWT token.
//
//	url is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	jwtToken is the token obtained with login or refresh. This is not a decorated token.
func (hc *HubNatsClient) ConnectWithJWT(url string, jwtToken string, caCert *x509.Certificate) (err error) {
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
		return err
	}
	clientID := claims.Claims().Name
	hc.clientID = clientID

	jwtSeed, err := hc.myKey.Seed()
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID), // connection name for logging, debugging
		nats.Secure(tlsConfig),
		nats.CustomInboxPrefix("_INBOX."+hc.clientID),
		nats.UserJWTAndSeed(jwtToken, string(jwtSeed)), // does this help?
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))

	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectWithNC connects using the given nats connection
func (hc *HubNatsClient) ConnectWithNC(nc *nats.Conn, clientID string) {
	hc.clientID = clientID
	hc.nc = nc
}

// ConnectWithNKey connects to the Hub server using an nkey secret
// ClientID is used for publishing actions
func (hc *HubNatsClient) ConnectWithNKey(url string, clientID string, userKey nkeys.KeyPair, caCert *x509.Certificate) (err error) {
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
		return userKey.Sign(nonce)
	}
	pubKey, _ := userKey.PublicKey()
	hc.clientID = clientID
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID), // connection name for logging
		nats.Secure(tlsConfig),
		nats.Nkey(pubKey, sigCB),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (hc *HubNatsClient) ConnectWithPassword(
	url string, loginID string, password string, caCert *x509.Certificate) (token string, err error) {

	if url == "" {
		url = nats.DefaultURL
	}

	// unauthenticated clients are allowed to invoke NewToken()
	err = hc.ConnectUnauthenticated(url, caCert)
	if err != nil {
		return "", err
	}

	req := authn.NewTokenReq{
		ClientID: loginID,
		Password: password,
	}
	msg, _ := ser.Marshal(req)
	data, err := hc.PubAction(loginID, authn.ClientAuthnCapability, authn.NewTokenAction, msg)

	if err != nil {
		return "", err
	}
	resp := &authn.NewTokenResp{}
	err = hc.ParseResponse(data, err, resp)
	if err != nil {
		return "", err
	}

	// reconnect with the token
	hc.Disconnect()

	//err = hc.ConnectWithJWT(url, userCreds, caCert)
	err = hc.ConnectWithJWT(url, resp.Token, caCert)
	if err != nil {
		slog.Warn("Login attempt failed", "err", err, "loginID", loginID)
		return "", err
	}
	return resp.Token, err
}

// ConnectUnauthenticated connects to the Hub server as an unauthenticated user
func (hc *HubNatsClient) ConnectUnauthenticated(url string, caCert *x509.Certificate) (err error) {
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
		return hc.unauthenticatedKey.Sign(nonce)
	}
	pubKey, _ := hc.unauthenticatedKey.PublicKey()
	hc.nc, err = nats.Connect(url,
		nats.Name("unauthenticated"),
		nats.Secure(tlsConfig),
		nats.Nkey(pubKey, sigCB),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err

}

// Disconnect from the Hub server and release all subscriptions
func (hc *HubNatsClient) Disconnect() {
	hc.nc.Close()
}

// ParseResponse helper message to parse response and detect the error response message
func (hc *HubNatsClient) ParseResponse(data []byte, err error, resp interface{}) error {
	if err != nil {
		return err
	}
	if resp != nil {
		err = ser.Unmarshal(data, resp)
	} else if string(data) == "+ACK" {
		// nats ack without data
		err = nil
	} else if len(data) > 0 {
		err = errors.New("unexpected response")
	}
	// if an error is detect see if it is an error response
	// An error response message has the format: {"error":"message"}
	// TODO: find a more idiomatic way to detect an error
	prefix := "{\"error\":"
	if err != nil || strings.HasPrefix(string(data), prefix) {
		errResp := hubclient.ErrorMessage{}
		err2 := ser.Unmarshal(data, &errResp)
		if err2 == nil && errResp.Error != "" {
			err = errors.New(errResp.Error)
		}
	}
	return err
}

// Publish to NATS
func (hc *HubNatsClient) Publish(subject string, payload []byte) error {
	slog.Info("publish", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// PubAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *HubNatsClient) PubAction(bindingID string, thingID string, actionID string, payload []byte) ([]byte, error) {
	subject := MakeActionSubject(bindingID, thingID, actionID, hc.clientID)
	slog.Info("PubAction", "subject", subject)
	resp, err := hc.nc.Request(subject, payload, time.Second*time.Duration(hc.timeoutSec))
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}

// PubEvent sends the event value to the hub
func (hc *HubNatsClient) PubEvent(thingID string, eventID string, payload []byte) error {
	subject := MakeSubject(hc.clientID, thingID, vocab.VocabEventTopic, eventID)
	slog.Info("PubEvent", "subject", subject)
	err := hc.Publish(subject, payload)
	return err
}

// PubTD sends the TD document to the hub
func (hc *HubNatsClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	subject := MakeSubject(hc.clientID, td.ID, vocab.VocabEventTopic, vocab.EventNameTD)
	slog.Info("PubTD", "subject", subject)
	err := hc.Publish(subject, payload)
	return err
}

// JS Returns the JetStream client (nats specific)
func (hc *HubNatsClient) JS() nats.JetStreamContext {
	return hc.js
}

// Refresh an authentication token.
//
//	clientID is the userID, deviceID or serviceID whose token to refresh.
//	oldToken must be a valid token obtained at login or refresh
//
// This returns a short lived auth token that can be used to authenticate with the hub
// This fails if the token has expired or does not belong to the clientID
//func (hc *HubNatsClient) Refresh(clientID string, oldToken string) (newToken string, err error) {
//	req := &authn.RefreshReq{
//		ClientID: clientID,
//		OldToken: oldToken,
//	}
//	msg, _ := ser.Marshal(req)
//	subject := MakeSubject(hc.clientID, td.ID, vocab.VocabEventTopic, vocab.EventNameTD)
//	slog.Info("PubTD", "subject", subject)
//	err := hc.Publish(subject, payload)
//	resp := &authn.RefreshResp{}
//	err = hubclient.ParseResponse(data, err, resp)
//	if err == nil {
//		authToken = resp.JwtToken
//	}
//	return err
//}

// SubActions subscribes to actions for this binding
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *HubNatsClient) SubActions(thingID string, cb func(msg *hubclient.ActionMessage) error) (hubclient.ISubscription, error) {

	subject := MakeActionSubject(hc.clientID, thingID, "", "")

	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
		md, _ := natsMsg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		payload := natsMsg.Data
		bindingID, thID, name, clientID, err := SplitActionSubject(natsMsg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", natsMsg.Subject)
			return
		}
		actionMsg := &hubclient.ActionMessage{
			//SenderID: natsMsg.Header.
			ClientID:  clientID,
			ActionID:  name,
			BindingID: bindingID,
			ThingID:   thID,
			Timestamp: timeStamp.Unix(),
			Payload:   payload,
			SendReply: func(payload []byte) {
				_ = natsMsg.Respond(payload)
			},
			SendAck: func() {
				_ = natsMsg.Ack()
			},
		}
		err = cb(actionMsg)
		if err != nil {
			errMsg := hubclient.ErrorMessage{Error: err.Error()}
			errPayload, _ := ser.Marshal(errMsg)
			_ = natsMsg.Respond(errPayload)
		}
	})
	return sub, err
}

func (hc *HubNatsClient) SubEvents(thingID string, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {

	subject := MakeSubject(hc.clientID, thingID, "event", "")

	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
		md, _ := natsMsg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		payload := natsMsg.Data
		bindingID, thID, _, name, err := SplitSubject(natsMsg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", natsMsg.Subject)
			return
		}
		eventMsg := &hubclient.EventMessage{
			BindingID: bindingID,
			ThingID:   thID,
			EventID:   name,
			Timestamp: timeStamp.Unix(),
			Payload:   payload,
		}
		cb(eventMsg)
	})
	return sub, err
}

// SubGroup subscribes to events received by a group.
// The client must be a member of the group to be able to create the consumer that receives the events.
// This creates an ephemeral pull consumer.
// ReceiveLatest is handy to be up to date on all event instead of quering them separately. Only use this if
// you're going to retrieve them anyways.
//
//	groupName name of the stream to receive events from.
//	receiveLatest to immediately receive the latest event for each event instance
func (hc *HubNatsClient) SubGroup(groupName string, receiveLatest bool, cb func(msg *hubclient.EventMessage)) error {
	deliverPolicy := nats.DeliverNewPolicy
	if receiveLatest {
		deliverPolicy = nats.DeliverLastPerSubjectPolicy
	}

	// Group event subscription does not need acknowledgements. This will speed up processing.
	// When first connecting, the latest event per subject is received,
	consumerConfig := &nats.ConsumerConfig{
		//Durable: "", // an ephemeral consumer has no name
		//FilterSubject: ">",  // get all events
		AckPolicy:     nats.AckNonePolicy,
		DeliverPolicy: deliverPolicy,
		//DeliverSubject: groupName+"."+hc.clientID,  // is this how this is supposed to be used?
		Description: "group consumer for client " + hc.clientID,
		RateLimit:   1000000, // TODO: configure somewhere. Is 1Mbps kbps a good number?
	}
	consumerInfo, err := hc.js.AddConsumer(groupName, consumerConfig)
	if err != nil {
		return fmt.Errorf("error subscribing to group '%s': %w", groupName, err)
	}

	//subject := MakeSubject(publisherID, thingID, vocab.VocabEventTopic, eventName)
	//subject := groupName
	// bind this consumer to all messages in this group stream (see at the end of the Subscribe)
	sub, err := hc.js.Subscribe(">", func(natsMsg *nats.Msg) {
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
		msg := &hubclient.EventMessage{
			//SenderID: msg.Header.
			EventID:   name,
			BindingID: pubID,
			ThingID:   thID,
			Timestamp: timeStamp.Unix(),
			Payload:   natsMsg.Data,
		}
		cb(msg)
	}, nats.OrderedConsumer(), nats.Bind(groupName, consumerInfo.Name))

	// todo unsubscribe
	_ = sub
	return err
}

// Subscribe to NATS
func (hc *HubNatsClient) Subscribe(subject string, cb func(msg *nats.Msg)) (sub hubclient.ISubscription, err error) {
	slog.Info("subscribe", "subject", subject)
	nsub, err := hc.nc.Subscribe(subject, cb)
	isValid := nsub.IsValid()
	if !isValid {
		err = errors.New("subject " + subject + " not valid")
	}
	sub = &NatsHubSubscription{nsub: nsub}
	return sub, err
}

// NewHubClient instantiates a client for connecting to the Hub using NATS/Jetstream
// Providing a nil nkey will use a newly generated nkey.
//
//	myKey is the optional client's nkey used to connect to the server with the jwt token
func NewHubClient(myKey nkeys.KeyPair) *HubNatsClient {
	if myKey == nil {
		myKey, _ = nkeys.CreateUser()
	}
	noAuthKey, _ := nkeys.FromSeed([]byte(PublicUnauthenticatedNKey))

	hc := &HubNatsClient{
		timeoutSec: 30, // 30sec for debugging. TODO: change to use config
		myKey:      myKey,
		// public 'unauthenticated' user nkey
		unauthenticatedKey: noAuthKey,
	}
	return hc
}
