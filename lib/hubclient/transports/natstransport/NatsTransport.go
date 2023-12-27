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
	caCert    *x509.Certificate
	// TLS configuration to use in connecting
	tlsConfig *tls.Config
	timeout   time.Duration
	status    transports.HubTransportStatus

	connectHandler func(status transports.HubTransportStatus)
	eventHandler   func(addr string, payload []byte)
	requestHandler func(addr string, payload []byte) (reply []byte, err error, donotreply bool)
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
	nconn.SetDisconnectErrHandler(nt.onDisconnect)
	nconn.SetReconnectHandler(nt.onConnected)
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
		nats.ConnectHandler(nt.onConnected),
		nats.DisconnectErrHandler(nt.onDisconnect),
		nats.ReconnectHandler(nt.onConnected),
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
		nats.ConnectHandler(nt.onConnected),
		nats.DisconnectErrHandler(nt.onDisconnect),
		nats.ReconnectHandler(nt.onConnected),
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
		nats.ConnectHandler(nt.onConnected),
		nats.DisconnectErrHandler(nt.onDisconnect),
		nats.ReconnectHandler(nt.onConnected),
		nats.UserInfo(nt.clientID, password),
		nats.Secure(nt.tlsConfig),
		// client permissions allow this inbox prefix
		nats.Name(nt.clientID),
		nats.CustomInboxPrefix(vocab.MessageTypeINBOX+"."+nt.clientID),
		nats.Timeout(time.Second*time.Duration(DefaultTimeoutSec)))
	if err == nil {
		nt.js, err = nt.nc.JetStream()
	} else {

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

// GetStatus Return the transport connection info
func (nt *NatsTransport) GetStatus() transports.HubTransportStatus {
	return nt.status
}

// JS Returns the JetStream client (nats specific)
func (nt *NatsTransport) JS() nats.JetStreamContext {
	return nt.js
}

// handle connected to the server
func (nt *NatsTransport) onConnected(c *nats.Conn) {
	nt.status.ConnectionStatus = transports.Connected
	nt.status.LastError = transports.ConnErrNone
	nt.connectHandler(nt.status)
}

// handle disconnect from the server
func (nt *NatsTransport) onDisconnect(c *nats.Conn, err error) {
	// FIXME: how to differentiate between intentional and unintended disconnect?
	// Is it important?
	nt.status.ConnectionStatus = transports.Disconnected
	if err != nil {
		nt.status.LastError = err.Error()
	} else {
		nt.status.LastError = transports.ConnErrNone
	}
	nt.connectHandler(nt.status)
}

// onMessage handles incoming request and event messages
func (nt *NatsTransport) onMessage(msg *nats.Msg) {
	var err error
	if msg.Reply == "" {
		if nt.eventHandler != nil {
			// no reply address, so treat as event
			nt.eventHandler(msg.Subject, msg.Data)
		}
	} else if nt.requestHandler != nil {
		// this is a request-response message
		reply, err, donotreply := nt.requestHandler(msg.Subject, msg.Data)
		msg.Data = reply
		// FIXME, does this work?

		if err != nil {
			if msg.Header == nil {
				msg.Header = nats.Header{}
			}
			msg.Header.Set("error", err.Error())
		}
		if !donotreply {
			err = msg.RespondMsg(msg)
		}
	} else {
		// this client has no handler, ignore the request
		msg.Header.Set("error", "missing handler")
		// this
		//err = msg.Ack()
		err = nil
	}
	if err != nil {
		slog.Error("onMessage: failed sending response",
			"err", err.Error(),
			"subject", msg.Subject)
	}
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

// PubEvent publishes a message and returns
func (nt *NatsTransport) PubEvent(subject string, payload []byte) error {
	slog.Info("PubEvent", "subject", subject)
	err := nt.nc.Publish(subject, payload)
	return err
}

// PubRequest publishes a request message and waits for an answer or until timeout
func (nt *NatsTransport) PubRequest(
	subject string, payload []byte) (data []byte, err error) {

	resp, err := nt.nc.Request(subject, payload, nt.timeout)
	if err != nil {
		return nil, err
	}
	// error responses are stored in the header
	if resp.Header != nil {
		errMsg := resp.Header.Get("error")
		if errMsg != "" {
			err = errors.New(errMsg)
		}
	}
	return resp.Data, err
}

// startEventMessageHandler listens for incoming event messages from a stream
// and invoke a callback handler.
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

// SetConnectHandler sets the notification handler of connection status changes
func (nt *NatsTransport) SetConnectHandler(cb func(status transports.HubTransportStatus)) {
	if cb == nil {
		panic("nil handler not allowed")
	}
	nt.connectHandler = cb
}

// SetEventHandler set the single handler that receives all subscribed events.
// This does not provide routing as in most cases it is unnecessary overhead
// Use 'Subscribe' to set the addresses that this receives events on.
func (nt *NatsTransport) SetEventHandler(cb func(addr string, payload []byte)) {
	nt.eventHandler = cb
}

// SetRequestHandler sets the handler that receives all subscribed requests.
// This does not provide routing as in most cases it is unnecessary overhead
// Use 'Subscribe' to set the addresses that this receives requests on.
func (nt *NatsTransport) SetRequestHandler(
	cb func(addr string, payload []byte) (reply []byte, err error, donotreply bool)) {
	nt.requestHandler = cb
}

// Subscribe to a subject.
// Incoming messages are passed to the event or request handler, depending on whether
// a reply-to address and correlation-ID is set.
func (nt *NatsTransport) Subscribe(subject string) (err error) {
	nsub, err := nt.nc.Subscribe(subject, nt.onMessage)
	isValid := nsub.IsValid()
	if err != nil || !isValid {
		err = fmt.Errorf("subscribe to '%s' failed: %w", subject, err)
	} else {
		// subscribe successful
	}
	return err
}

// SubRequest subscribes to a requests and sends a response
// Intended for actions, config and rpc requests
//func (nt *NatsTransport) SubRequest(
//	subject string, cb func(subject string, payload []byte) (reply []byte, err error)) (
//	transports.ISubscription, error) {
//
//	sub, err := nt._sub(subject, func(natsMsg *nats.Msg) {
//		md, _ := natsMsg.Metadata()
//		timeStamp := time.Now()
//		if md != nil {
//			timeStamp = md.Timestamp
//
//		}
//		natsMsg.Header = nats.Header{}
//		natsMsg.Header.Set("received", timeStamp.Format(time.StampMilli))
//		reply, err := cb(natsMsg.Subject, natsMsg.Data)
//		if err != nil {
//			slog.Error("request error",
//				"subject", natsMsg.Subject, "err", err.Error())
//			// the intent is to pass the error using the header but that doesn't seem to work
//			natsMsg.Header.Set("error", err.Error())
//			// so for now use a json message "{error:text}"
//			errPayload, _ := ser.Marshal(err.Error())
//			natsMsg.Data = []byte("{\"error\":" + string(errPayload) + "}")
//			err = natsMsg.RespondMsg(natsMsg)
//		} else if reply != nil {
//			natsMsg.Data = reply
//			natsMsg.RespondMsg(natsMsg)
//			err = natsMsg.RespondMsg(natsMsg)
//		} else {
//			err = natsMsg.Ack()
//		}
//		if err != nil {
//			slog.Error("SubRequest: failed sending response",
//				"err", err.Error(),
//				"subject", natsMsg.Subject)
//		}
//	})
//	return sub, err
//}

func (nt *NatsTransport) Unsubscribe(subject string) {
	// on-the-fly subscribe-unsubscribe is not the intended use.
	// if there is a good use-case it can be added, but it would mean tracking the subscriptions.
	slog.Warn("unsubscribe is not used", "subject", subject)
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
	tp := &NatsTransport{
		serverURL: url,
		caCert:    caCert,
		clientID:  clientID,
		timeout:   time.Duration(DefaultTimeoutSec) * time.Second,
		tlsConfig: tlsConfig,
		connectHandler: func(status transports.HubTransportStatus) {
			slog.Info("connection status change", "newStatus", status.ConnectionStatus, "last error", status.LastError)
		},
		status: transports.HubTransportStatus{
			CaCert:           caCert,
			HubURL:           url,
			ClientID:         clientID,
			ConnectionStatus: transports.Disconnected,
			LastError:        "",
			Core:             "nats",
		},
	}

	return tp
}
