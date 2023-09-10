package natshubclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// lower level NATS pub/sub functions

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

// JS Returns the JetStream client (nats specific)
func (hc *NatsHubClient) JS() nats.JetStreamContext {
	return hc.js
}

// ParseResponse helper message to parse response and detect the error response message
func (hc *NatsHubClient) ParseResponse(data []byte, resp interface{}) error {
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

// Sub is a low level subscription to a subject
// Primarily intended for testing
func (hc *NatsHubClient) Sub(subject string, cb func(topic string, data []byte)) (hubclient.ISubscription, error) {

	sub, err := hc.Subscribe(subject, func(natsMsg *nats.Msg) {
		cb(natsMsg.Subject, natsMsg.Data)
	})
	return sub, err
}

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
