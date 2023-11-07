package natstransport

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"log/slog"
	"strings"
	"time"
)

//// PublicUnauthenticatedNKey is the public seed of the unaunthenticated user
//const PublicUnauthenticatedNKey = "SUAOXRE662WSIGIMSIFVQNCCIWG673K7GZMB3ZUUIF45BWGMYKECEQQJZE"

// DefaultTimeoutSec with timeout for connecting and publishing.
const DefaultTimeoutSec = 100 //3 // 100 for testing

// NatsTransport is a Hub Client transport for the NATS message server.
// This implements the IHubTransport interface.
type NatsTransport struct {
	clientID string
	//keyPair   string
	nc        *nats.Conn
	js        nats.JetStreamContext
	serverURL string
	timeout   time.Duration
	// TLS configuration to use in connecting
	tlsConfig *tls.Config
}

// AddressTokens returns the address separator and wildcards
func (nt *NatsTransport) AddressTokens() (sep string, wc string, rem string) {
	return ".", "*", ">"
}

// ConnectWithConn to the hub server using the given nats client connection
func (nt *NatsTransport) ConnectWithConn(nconn *nats.Conn) (err error) {

	st, _ := nconn.TLSConnectionState()
	_ = st
	//slog.Info("ConnectWithConn", "loginID", nt.clientID, "url", st.ServerName)

	// checks
	//if nt.clientID == "" {
	//	err := fmt.Errorf("connect - Missing Login ID")
	//	return err
	if nconn == nil {
		err := fmt.Errorf("connect - missing connection")
		return err
	}
	nt.nc = nconn
	nt.js, err = nconn.JetStream()
	return err
}

// ConnectWithJWT connects to the Hub server using a NATS user JWT credentials secret
// The connection uses the client ID in the JWT token.
//
//	serverURL is the server URL to connect to. Eg tls://addr:port/ for tcp or wss://addr:port/ for websockets
//	jwtToken is the token obtained with login or refresh. This is not a decorated token.
func (nt *NatsTransport) ConnectWithJWT(myKey nkeys.KeyPair, jwtToken string) (err error) {
	slog.Info("ConnectWithJWT", "loginID", nt.clientID, "url", nt.serverURL)

	//claims, err := jwt.Decode(jwtToken)
	//if err != nil {
	//	err = fmt.Errorf("invalid jwt token: %w", err)
	//	return err
	//}
	//clientID := claims.Claims().Name
	jwtSeed, _ := myKey.Seed()
	nt.nc, err = nats.Connect(nt.serverURL,
		nats.Name(nt.clientID), // connection name for logging, debugging
		nats.Secure(nt.tlsConfig),
		nats.CustomInboxPrefix(vocab.MessageTypeINBOX+"."+nt.clientID),
		nats.UserJWTAndSeed(jwtToken, string(jwtSeed)),
		nats.Token(jwtToken), // JWT token isn't passed through in callout
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))
	if err == nil {
		nt.js, err = nt.nc.JetStream()
	}
	return err
}

// ConnectWithKey connects to the Hub server using the client's nkey secret
func (nt *NatsTransport) ConnectWithKey(myKey nkeys.KeyPair) error {
	var err error

	if myKey == nil {
		return fmt.Errorf(
			"ConnectWithKey: Client '%s' has no auth key", nt.clientID)
	}

	// The handler to sign the server issued challenge
	sigCB := func(nonce []byte) ([]byte, error) {
		return myKey.Sign(nonce)
	}
	pubKey, _ := myKey.PublicKey()
	nt.nc, err = nats.Connect(nt.serverURL,
		nats.Name(nt.clientID), // connection name for logging
		nats.Secure(nt.tlsConfig),
		nats.Nkey(pubKey, sigCB),
		// client permissions allow this inbox prefix
		nats.CustomInboxPrefix(vocab.MessageTypeINBOX+"."+nt.clientID),
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))

	if err == nil {
		nt.js, err = nt.nc.JetStream()
	}
	return err
}

// ConnectWithPassword connects to the Hub server using a login ID and password.
func (nt *NatsTransport) ConnectWithPassword(password string) (err error) {

	nt.nc, err = nats.Connect(nt.serverURL,
		nats.UserInfo(nt.clientID, password),
		nats.Secure(nt.tlsConfig),
		// client permissions allow this inbox prefix
		nats.Name(nt.clientID),
		nats.CustomInboxPrefix(vocab.MessageTypeINBOX+"."+nt.clientID),
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))
	if err == nil {
		nt.js, err = nt.nc.JetStream()
	}
	return err
}

// ConnectWithToken connects to the Hub server using a NATS user a token obtained at login or refresh
// If a valid nkey is set and token is empty, a connect with nkey will be done.
//
//	keyPair is the serialized pub/private key of the user
//	token is the token obtained with login or refresh.
func (nt *NatsTransport) ConnectWithToken(keyPair keys.IHiveKey, token string) (err error) {
	slog.Info("ConnectWithToken",
		slog.String("loginID", nt.clientID),
		slog.String("url", nt.serverURL))

	myKey := keyPair.PrivateKey().(nkeys.KeyPair)

	_, err = jwt.Decode(token)
	// if this isn't a valid JWT, try the nkey login and ignore the token
	if err != nil {
		err = nt.ConnectWithKey(myKey)
	} else {
		err = nt.ConnectWithJWT(myKey, token)
	}
	return err
}

// CreateKeyPair returns a new set of serialized public/private keys for the client
func (nt *NatsTransport) CreateKeyPair() (kp keys.IHiveKey) {
	kp = keys.NewNkeysKey()
	return kp
}

// Disconnect from the Hub server and release all subscriptions
func (nt *NatsTransport) Disconnect() {
	nt.nc.Close()
}

// JS Returns the JetStream client (nats specific)
func (nt *NatsTransport) JS() nats.JetStreamContext {
	return nt.js
}

// ParseResponse helper message to parse response and detect the error response message
func (nt *NatsTransport) ParseResponse(data []byte, resp interface{}) error {
	var err error
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
		errResp := ""
		err2 := ser.Unmarshal(data, &errResp)
		if err2 == nil && errResp != "" {
			err = errors.New(errResp)
		}
	}
	return err
}

// Pub low level publish to NATS
func (nt *NatsTransport) Pub(subject string, payload []byte) error {
	slog.Info("Pub", "subject", subject)
	err := nt.nc.Publish(subject, payload)
	return err
}

// PubRequest sends an action request to a Hub Service on the svc prefix
// Returns the response or an error if the request fails or timed out
func (nt *NatsTransport) PubRequest(
	subject string, payload []byte) (data []byte, err error) {

	resp, err := nt.nc.Request(subject, payload, nt.timeout)
	if err != nil {
		return nil, err
	}
	// error responses are stored in the header
	// FIXME: does this work?
	if resp.Header != nil {
		errMsg := resp.Header.Get("error")
		if errMsg != "" {
			err = errors.New(errMsg)
		}
	}
	return resp.Data, err
}

// startEventMessageHandler listens for incoming event messages and invoke a callback handler
// this returns when the subscription is no longer valid
func startEventMessageHandler(nsub *nats.Subscription, cb func(msg *things.ThingValue)) error {
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
			msg := &things.ThingValue{
				//SenderID: msg.Header.
				AgentID:     pubID,
				ThingID:     thID,
				Name:        name,
				CreatedMSec: timeStamp.UnixMilli(),
				Data:        natsMsg.Data,
			}
			cb(msg)

		}
	}()
	return nil
}

// Sub is a low level subscription to a subject
func (nt *NatsTransport) _sub(subject string, cb func(msg *nats.Msg)) (sub transports.ISubscription, err error) {
	nsub, err := nt.nc.Subscribe(subject, cb)
	isValid := nsub.IsValid()
	if err != nil || !isValid {
		err = fmt.Errorf("subscribe to '%s' failed: %w", subject, err)
	}
	//sub = &NatsHubSubscription{nsub: nsub}
	sub = nsub
	return sub, err
}

// Sub subscribe to an address.
// Primarily intended for testing
func (nt *NatsTransport) Sub(subject string, cb func(subject string, data []byte)) (transports.ISubscription, error) {

	sub, err := nt._sub(subject, func(natsMsg *nats.Msg) {
		cb(natsMsg.Subject, natsMsg.Data)
	})
	return sub, err
}

// SubRequest subscribes to a requests and sends a response
// Intended for actions, config and rpc requests
func (nt *NatsTransport) SubRequest(
	subject string, cb func(subject string, payload []byte) (reply []byte, err error)) (
	transports.ISubscription, error) {

	sub, err := nt._sub(subject, func(natsMsg *nats.Msg) {
		md, _ := natsMsg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		natsMsg.Header = nats.Header{}
		natsMsg.Header.Set("received", timeStamp.Format(time.StampMilli))
		reply, err := cb(natsMsg.Subject, natsMsg.Data)
		if err != nil {
			slog.Error("request error",
				"subject", natsMsg.Subject, "err", err.Error())
			// the intent is to pass the error using the header but that doesn't seem to work
			natsMsg.Header.Set("error", err.Error())
			// so for now use a json message "{error:text}"
			errPayload, _ := ser.Marshal(err.Error())
			natsMsg.Data = []byte("{\"error\":" + string(errPayload) + "}")
			err = natsMsg.RespondMsg(natsMsg)
		} else if reply != nil {
			natsMsg.Data = reply
			natsMsg.RespondMsg(natsMsg)
			err = natsMsg.RespondMsg(natsMsg)
		} else {
			err = natsMsg.Ack()
		}
		if err != nil {
			slog.Error("SubRequest: failed sending response",
				"err", err.Error(),
				"subject", natsMsg.Subject)
		}
	})
	return sub, err
}

// SubStream subscribes to events received by the event stream.
//
// TODO: determine if nats stream can be used as a directory and history service
//
// This creates an ephemeral pull consumer.
// ReceiveLatest is handy to be up to date on all event instead of quering them separately. Only use this if
// you're going to retrieve them anyways.
//
//	 name of the event stream. "" for default
//		receiveLatest to immediately receive the latest event for each event instance
func (nt *NatsTransport) SubStream(name string, receiveLatest bool, cb func(msg *things.ThingValue)) (transports.ISubscription, error) {
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
		//DeliverSubject: groupName+"."+nt.clientID,  // is this how this is supposed to be used?
		Description: "group consumer for client " + nt.clientID,
		//RateLimit:   1000000, // consumers in poll mode cannot have rate limit set
	}
	consumerInfo, err := nt.js.AddConsumer(name, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating consumer for stream '%s': %w", name, err)
	}
	// bind this ephemeral consumer to all messages in this group stream
	// (see at the end of the Subscribe)
	nsub, err := nt.js.PullSubscribe("", "",
		nats.Bind(name, consumerInfo.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("error to PullSubscribe to stream %s: %w", name, err)
	}

	err = startEventMessageHandler(nsub, cb)
	return nsub, err
}

// NewNatsTransport creates a new instance of the hub client for use
// with the NATS messaging server
//
//	url starts with "nats://" schema for using tcp.
//	clientID to connect as
//	keyPair is the serialized keypair or use "" to create a new set.
//	caCert of the server to validate the server or nil to not check the server cert
func NewNatsTransport(url string, clientID string, caCert *x509.Certificate) *NatsTransport {

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

	hc := &NatsTransport{
		serverURL: url,
		clientID:  clientID,
		timeout:   time.Duration(DefaultTimeoutSec) * time.Second,
		tlsConfig: tlsConfig,
	}
	return hc
}
