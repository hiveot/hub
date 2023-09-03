package natshubclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/hubclient"
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

// PublicUnauthenticatedNKey is the public seed of the unaunthenticated user
const PublicUnauthenticatedNKey = "SUAOXRE662WSIGIMSIFVQNCCIWG673K7GZMB3ZUUIF45BWGMYKECEQQJZE"

// DefaultTimeoutSec with timeout for connecting and publishing.
const DefaultTimeoutSec = 100 //3 // 100 for testing

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

// NatsHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the NATS/Jetstream messaging system.
type NatsHubClient struct {
	clientID string
	nc       *nats.Conn
	js       nats.JetStreamContext
	timeout  time.Duration
}

// ClientID the client is authenticated as to the server
func (hc *NatsHubClient) ClientID() string {
	return hc.clientID
}

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

// ParseResponse helper message to parse response and detect the error response message
func (hc *NatsHubClient) ParseResponse(data []byte, err error, resp interface{}) error {
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

// Pub low level publish to NATS
func (hc *NatsHubClient) Pub(subject string, payload []byte) error {
	slog.Info("Pub", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// PubThingAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *NatsHubClient) PubThingAction(bindingID string, thingID string, actionID string, payload []byte) ([]byte, error) {
	subject := MakeThingActionSubject(bindingID, thingID, actionID, hc.clientID)
	slog.Info("PubThingAction", "subject", subject)
	resp, err := hc.nc.Request(subject, payload, hc.timeout)
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}

// PubServiceAction sends an action request to a Hub Service on the svc prefix
// Returns the response or an error if the request fails or timed out
func (hc *NatsHubClient) PubServiceAction(serviceID string, capability string, actionID string, payload []byte) ([]byte, error) {
	subject := MakeServiceActionSubject(serviceID, capability, actionID, hc.clientID)
	slog.Info("PubServiceAction", "subject", subject)
	resp, err := hc.nc.Request(subject, payload, hc.timeout)
	if resp == nil {
		return nil, err
	}
	return resp.Data, err
}

// PubEvent sends the event value to the hub
func (hc *NatsHubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	subject := MakeThingsSubject(hc.clientID, thingID, vocab.MessageTypeEvent, eventID)
	slog.Info("PubEvent", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// PubTD sends the TD document to the hub
func (hc *NatsHubClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	subject := MakeThingsSubject(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
	slog.Info("PubTD", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// JS Returns the JetStream client (nats specific)
func (hc *NatsHubClient) JS() nats.JetStreamContext {
	return hc.js
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

// Sub is a low level subscription to a subject
// Primarily intended for testing
func (hc *NatsHubClient) Sub(subject string, cb func(topic string, data []byte)) (hubclient.ISubscription, error) {

	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
		cb(natsMsg.Subject, natsMsg.Data)
	})
	return sub, err
}

// SubActions subscribes to actions on the given subject
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *NatsHubClient) SubActions(subject string, cb func(msg *hubclient.ActionMessage) error) (hubclient.ISubscription, error) {

	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
		md, _ := natsMsg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		payload := natsMsg.Data
		sourceID, thID, name, clientID, err := SplitActionSubject(natsMsg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", natsMsg.Subject)
			return
		}
		actionMsg := &hubclient.ActionMessage{
			//SenderID: natsMsg.Header.
			ClientID:  clientID,
			ActionID:  name,
			BindingID: sourceID,
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

// SubThingActions subscribes to actions for this device or service on the things prefix
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *NatsHubClient) SubThingActions(thingID string, cb func(msg *hubclient.ActionMessage) error) (hubclient.ISubscription, error) {

	subject := MakeThingActionSubject(hc.clientID, thingID, "", "")
	return hc.SubActions(subject, cb)
}

// SubServiceCapability subscribes to action requests of a service capability
//
//	capability is the name of the capability (thingID) to handle
func (hc *NatsHubClient) SubServiceCapability(capability string, cb func(msg *hubclient.ActionMessage) error) (hubclient.ISubscription, error) {

	subject := MakeServiceSubject(hc.clientID, capability, MessageTypeAction, "")
	return hc.SubActions(subject, cb)
}

// SubEvents subscribe to event
//func (hc *NatsHubClient) SubEvents(thingID string, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {
//
//	subject := MakeThingsSubject(hc.clientID, thingID, "event", "")
//
//	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
//		md, _ := natsMsg.Metadata()
//		timeStamp := time.Now()
//		if md != nil {
//			timeStamp = md.Timestamp
//
//		}
//		payload := natsMsg.Data
//		bindingID, thID, _, name, err := SplitSubject(natsMsg.Subject)
//		if err != nil {
//			slog.Error("unable to handle subject", "err", err, "subject", natsMsg.Subject)
//			return
//		}
//		eventMsg := &hubclient.EventMessage{
//			BindingID: bindingID,
//			ThingID:   thID,
//			EventID:   name,
//			Timestamp: timeStamp.Unix(),
//			Payload:   payload,
//		}
//		cb(eventMsg)
//	})
//	return sub, err
//}

// startEventMessageHandler listens for incoming event messages and invoke a callback handler
// this returns when the subscription is no longer valid
func startEventMessageHandler(nsub *nats.Subscription, cb func(msg *hubclient.EventMessage)) error {
	ci, err := nsub.ConsumerInfo()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	go func() {
		for nsub.IsValid() {

			//natsMsg, err := nsub.NextMsg(time.Second)// invalid subscription type???
			natsMsgs, err := nsub.Fetch(1)
			if err != nil {
				// it is only an error if the subscription hasn't closed
				// error is given when remote side closes connection before the client
				if nsub.IsValid() {
					slog.Error("nsub.Fetch failed", "err", err.Error())
				}
				break
			}
			natsMsg := natsMsgs[0]
			slog.Info("received event msg from consumer ",
				slog.String("consumer", ci.Name),
				slog.String("stream", ci.Stream),
				slog.String("subject", natsMsg.Subject),
			)
			md, _ := natsMsg.Metadata()
			timeStamp := time.Now()
			if md != nil {
				timeStamp = md.Timestamp
			}
			_, pubID, thID, _, name, err := SplitSubject(natsMsg.Subject)
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

		}
	}()
	return nil
}

// SubStream subscribes to events received by the event stream.
//
// This creates an ephemeral pull consumer.
// ReceiveLatest is handy to be up to date on all event instead of quering them separately. Only use this if
// you're going to retrieve them anyways.
//
//	 name of the event stream. "" for default
//		receiveLatest to immediately receive the latest event for each event instance
func (hc *NatsHubClient) SubStream(name string, receiveLatest bool, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {
	if name == "" {
		//name = natsnkeyserver.EventsIntakeStreamName
	}
	deliverPolicy := nats.DeliverNewPolicy
	if receiveLatest {
		// FIXME: deliver has error: "optional filter subject is not set"
		// you'd think optional means its ... well, optional
		deliverPolicy = nats.DeliverLastPerSubjectPolicy
	}

	// Group event subscription does not need acknowledgements. This will speed up processing.
	// When first connecting, the latest event per subject is received,
	consumerConfig := &nats.ConsumerConfig{
		Durable: "", // an ephemeral consumer has no name
		//FilterSubject: ">",  // get all events
		AckPolicy:     nats.AckNonePolicy,
		DeliverPolicy: deliverPolicy,
		//DeliverSubject: groupName+"."+hc.clientID,  // is this how this is supposed to be used?
		Description: "group consumer for client " + hc.clientID,
		//RateLimit:   1000000, // consumers in poll mode cannot have rate limit set
	}
	consumerInfo, err := hc.js.AddConsumer(name, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating consumer for stream '%s': %w", name, err)
	}
	// bind this ephemeral consumer to all messages in this group stream
	// (see at the end of the Subscribe)
	nsub, err := hc.js.PullSubscribe("", "",
		nats.Bind(name, consumerInfo.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("error to PullSubscribe to stream %s: %w", name, err)
	}

	err = startEventMessageHandler(nsub, cb)
	sub := &NatsHubSubscription{nsub: nsub}
	return sub, err
}

// Subscribe to NATS
func (hc *NatsHubClient) Subscribe(subject string, cb func(msg *nats.Msg)) (sub hubclient.ISubscription, err error) {
	slog.Info("subscribe", "subject", subject, "clientID", hc.clientID)
	nsub, err := hc.nc.Subscribe(subject, cb)
	isValid := nsub.IsValid()
	if err != nil || !isValid {
		err = fmt.Errorf("subscribe to '%s' failed: %w", subject, err)
	}
	sub = &NatsHubSubscription{nsub: nsub}
	return sub, err
}

// NewHubClient instantiates a client for connecting to the Hub using NATS/Jetstream
func NewHubClient(clientID string, nc *nats.Conn) (hc *NatsHubClient, err error) {

	hc = &NatsHubClient{
		clientID: clientID,
		nc:       nc,
		timeout:  time.Duration(DefaultTimeoutSec) * time.Second,
	}
	hc.js, err = hc.nc.JetStream()
	hc.timeout = time.Duration(10) * time.Second // for testing
	return hc, err
}
