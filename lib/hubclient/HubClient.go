package hubclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

type HubClient struct {
	instanceName string
	clientID     string
	nc           *nats.Conn
	js           nats.JetStreamContext
	timeoutSec   int
}

// publish to NATS
func (hc *HubClient) publish(topic string, payload []byte) error {
	slog.Info("publish", "subject", topic)
	err := hc.nc.Publish(topic, payload)
	return err
}

// subscribe to NATS
func (hc *HubClient) subscribe(topic string, cb func(msg *nats.Msg)) error {
	slog.Info("subscribe", "subject", topic)
	subscription, err := hc.nc.Subscribe(topic, cb)
	_ = subscription
	return err
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
		nats.Name(hc.instanceName),
		nats.Secure(tlsConfig),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	if err == nil {
		// jetstream client
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
		RootCAs: caCertPool,
		//Certificates:       clientCertList,
		InsecureSkipVerify: caCert == nil,
	}
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.instanceName),
		nats.Secure(tlsConfig),
		nats.UserInfo(loginID, password),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	return err
}

// ConnectWithToken connects to the Hub server using a shared secret
func (hc *HubClient) ConnectWithToken(url string, token string) (err error) {
	if url == "" {
		url = nats.DefaultURL
	}
	hc.nc, err = nats.Connect(url,
		nats.Name(hc.instanceName),
		nats.Token(token),
		nats.Timeout(time.Second*time.Duration(hc.timeoutSec)))
	return err
}

// DisConnect from the Hub server
func (hc *HubClient) DisConnect() {
	hc.nc.Close()
}

// Login to the hub with userID and password
func (hc *HubClient) Login(userID, password string) (err error) {
	hc.clientID = userID
	return nil
}

// PubAction sends an action request to the hub
func (hc *HubClient) PubAction(publisherID string, thingID string, actionID string, payload []byte) error {
	subject := MakeSubject(publisherID, thingID, vocab.VocabActionTopic, actionID)
	err := hc.publish(subject, payload)
	return err
}

// PubEvent sends the event value to the hub
func (hc *HubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	subject := MakeSubject(hc.clientID, thingID, vocab.VocabEventTopic, eventID)
	err := hc.publish(subject, payload)
	return err
}

// PubTD sends the TD document to the hub
func (hc *HubClient) PubTD(td *thing.TD) error {
	payload, _ := json.Marshal(td)
	subject := MakeSubject(hc.clientID, td.ID, vocab.VocabEventTopic, vocab.EventNameTD)
	err := hc.publish(subject, payload)
	return err
}

// SubActions subscribes to actions for this device
func (hc *HubClient) SubActions(cb func(tv *thing.ThingValue)) error {

	subject := MakeSubject(hc.clientID, "", vocab.VocabActionTopic, "")

	err := hc.subscribe(subject, func(msg *nats.Msg) {
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
		tv := &thing.ThingValue{
			ID:          name,
			PublisherID: pubID,
			ThingID:     thID,
			Created:     timeStamp.Format(vocab.ISO8601Format),
			Data:        payload,
		}
		cb(tv)
	})
	return err
}

// SubEvent subscribes to an event on subject things.{publisherID}.{thingID}.event.{eventName}, where . is the separator.
// leave publisherID, thingID and/or eventName empty to use wildcards.
//
// This will likely be replaced with subscribing to events in a group instead of a publisher/thing
func (hc *HubClient) SubEvent(
	publisherID, thingID, eventName string, cb func(tv *thing.ThingValue)) error {

	subject := MakeSubject(publisherID, thingID, vocab.VocabEventTopic, eventName)

	err := hc.subscribe(subject, func(msg *nats.Msg) {
		md, _ := msg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		pubID, thID, _, name, err := SplitSubject(msg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", msg.Subject)
			return
		}
		payload := msg.Data
		tv := &thing.ThingValue{
			ID:          name,
			PublisherID: pubID,
			ThingID:     thID,
			Data:        payload,
			Created:     timeStamp.Format(vocab.ISO8601Format),
		}
		cb(tv)
	})
	return err
}

// NewHubClient instantiates a client for connecting to the Hub
func NewHubClient(instanceName string) *HubClient {
	hc := &HubClient{
		instanceName: instanceName,
		timeoutSec:   10,
	}
	return hc
}

// MakeSubject creates an nats subject optionally with nats wildcards
//
//	pubID is the publisher of the subject. Use "" for wildcard
//	thingID is the thing of the subject. Use "" for wildcard
//	stype is the subject type, eg event or action. Use "" for default "event"
func MakeSubject(pubID, thingID, stype, name string) string {
	if pubID == "" {
		pubID = "*"
	}
	if thingID == "" {
		thingID = "*" // nats uses *
	}
	if stype == "" {
		stype = vocab.VocabEventTopic
	}
	if name == "" {
		name = "*" // nats uses *
	}
	subj := fmt.Sprintf("things.%s.%s.%s.%s",
		pubID, thingID, stype, name)
	return subj
}

// SplitSubject separates a subject into its components
//
// subject is a nats subject. eg: things.publisherID.thingID.type.name
//
//		pubID is the publisher of the subject. Use "" for wildcard
//		thingID is the thing of the subject. Use "" for wildcard
//		stype is the subject type, eg event or action. Use "" for default "event"
//	 name is the event or action name
func SplitSubject(subject string) (pubID, thingID, stype, name string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 5 {
		err = errors.New("incomplete subject")
		return
	}
	pubID = parts[1]
	thingID = parts[2]
	stype = parts[3]
	name = parts[4]
	return
}
