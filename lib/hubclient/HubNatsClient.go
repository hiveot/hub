package hubclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// ParseResponse helper message to parse response and detect the error response message
func ParseResponse(data []byte, err error, resp interface{}) error {
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
		errResp := hub.ErrorMessage{}
		err2 := ser.Unmarshal(data, &errResp)
		if err2 == nil && errResp.Error != "" {
			err = errors.New(errResp.Error)
		}
	}
	return err
}

// HubSubscription nats subscription helper
type HubSubscription struct {
	nsub *nats.Subscription
}

func (ns *HubSubscription) Unsubscribe() {
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
func (hc *HubNatsClient) ConnectWithCert(url string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (err error) {
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

// ConnectWithJWT connects to the Hub server using a NATS user JWT credentials secret
// This seems (?) to also work when using jwt with static server setup using nkeys
func (hc *HubNatsClient) ConnectWithJWT(url string, jwtCreds []byte, caCert *x509.Certificate) (err error) {
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
	// Get the userID from the token
	jwtToken, err := jwt.ParseDecoratedJWT(jwtCreds)
	if err != nil {
		return fmt.Errorf("can't get jwt from jwtCreds: %w", err)
	}
	userKP, err := jwt.ParseDecoratedUserNKey(jwtCreds)
	if err != nil {
		return fmt.Errorf("can't get keys from jwtCreds: %w", err)
	}
	jwtSeed, err := userKP.Seed()
	claims, err := jwt.Decode(jwtToken)
	if err != nil {
		err = fmt.Errorf("invalid jwt token: %w", err)
		return err
	}
	clientID := claims.Claims().Name
	hc.clientID = clientID

	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID), // connection name for logging, debugging
		nats.Secure(tlsConfig),
		nats.UserJWTAndSeed(jwtToken, string(jwtSeed)), // does this help?
		// since nats doesnt pass a jwt token to the callout, include it as a regular token
		// the downside is that nonce signing isn't used
		nats.Token(jwtToken),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))

	if err == nil {
		hc.js, err = hc.nc.JetStream()
	}
	return err
}

// ConnectWithNC connects using the given nats connection
func (hc *HubNatsClient) ConnectWithNC(nc *nats.Conn, clientID string) (err error) {
	hc.clientID = clientID
	hc.nc = nc
	return nil
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
	hc.clientID = loginID
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.clientID),
		nats.Secure(tlsConfig),
		nats.UserInfo(hc.clientID, password),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err != nil {
		slog.Warn("Login attempt failed", "loginID", loginID)
		return err
	}
	hc.js, err = hc.nc.JetStream()
	if err != nil {
		slog.Warn("ConnectWithPassword. JetStream not available")
	}
	//
	return err
}

// ConnectWithToken connects to the Hub server using a JWT token.
// NOTE: This differs from NATS JWT.
// The token can be obtained by NewToken and Refresh functions.
//
// Token will be signed by the user public key
//func (hc *HubNatsClient) ConnectWithToken(url string, token []byte, caCert *x509.Certificate) (err error) {
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
//	// Get the userID from the token
//	claims, err := jwt.Decode(string(token))
//	if err != nil {
//		err = fmt.Errorf("invalid jwt token: %w", err)
//		return err
//	}
//	clientID := claims.Claims().Name
//	hc.clientID = clientID
//
//	userJWTHandler := func() (string, error) {
//		jwtPortion, err := nkeys.ParseDecoratedJWT(jwtCreds)
//		return jwtPortion, err
//	}
//	// The handler to sign the server issued challenge
//	sigCB := func(nonce []byte) ([]byte, error) {
//		kp, err := nkeys.ParseDecoratedNKey(jwtCreds)
//		sig, _ := kp.Sign(nonce)
//		return sig, err
//	}
//	hc.nc, err = nats.Connect(url,
//		nats.Name(hc.clientID),                 // connection name for logging, debugging
//		nats.UserJWTAndSeed("myjwt", "myseed"), // does this help?
//		nats.Secure(tlsConfig),
//		nats.UserJWT(userJWTHandler, sigCB),
//		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
//
//	if err == nil {
//		hc.js, err = hc.nc.JetStream()
//	}
//	return err
//}

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

// Disconnect from the Hub server and release all subscriptions
func (hc *HubNatsClient) Disconnect() {
	hc.nc.Close()
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
func (hc *HubNatsClient) SubActions(thingID string, cb func(msg *hub.ActionMessage) error) (hub.ISubscription, error) {

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
		actionMsg := &hub.ActionMessage{
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
			errMsg := hub.ErrorMessage{Error: err.Error()}
			errPayload, _ := ser.Marshal(errMsg)
			_ = natsMsg.Respond(errPayload)
		}
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
func (hc *HubNatsClient) SubGroup(groupName string, receiveLatest bool, cb func(msg *hub.EventMessage)) error {
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
		msg := &hub.EventMessage{
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
func (hc *HubNatsClient) Subscribe(subject string, cb func(msg *nats.Msg)) (sub hub.ISubscription, err error) {
	slog.Info("subscribe", "subject", subject)
	nsub, err := hc.nc.Subscribe(subject, cb)
	isValid := nsub.IsValid()
	if !isValid {
		err = errors.New("subject " + subject + " not valid")
	}
	sub = &HubSubscription{nsub: nsub}
	return sub, err
}

// NewHubClient instantiates a client for connecting to the Hub using NATS/Jetstream
func NewHubClient() *HubNatsClient {
	hc := &HubNatsClient{
		timeoutSec: 30, // 30sec for debugging. TODO: change to use config
	}
	return hc
}
