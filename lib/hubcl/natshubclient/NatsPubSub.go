package natshubclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
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

// PubTD sends the TD document to the hub
func (hc *NatsHubClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	subject := MakeThingsSubject(hc.clientID, td.ID, vocab.MessageTypeEvent, vocab.EventNameTD)
	slog.Info("PubTD", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// PubEvent sends the event value to the hub
func (hc *NatsHubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	subject := MakeThingsSubject(hc.clientID, thingID, vocab.MessageTypeEvent, eventID)
	slog.Info("PubEvent", "subject", subject)
	err := hc.nc.Publish(subject, payload)
	return err
}

// PubServiceAction sends an action request to a Hub Service on the svc prefix
// Returns the response or an error if the request fails or timed out
func (hc *NatsHubClient) pubAction(subject string, payload []byte) (
	ar hubclient.ActionResponse, err error) {

	t1 := time.Now()
	ar.Address = subject
	resp, err := hc.nc.Request(subject, payload, hc.timeout)
	ar.Duration = time.Now().Sub(t1)
	if err == nil {
		ar.SentSuccess = true
		// todo: an ack is also a reply but has no data
		ar.ReceivedReply = resp.Data != nil
		ar.Payload = resp.Data
		// FIXME: detect an error
		if resp.Header != nil {
			errMsg := resp.Header.Get("error")
			if errMsg != "" {
				ar.ErrorReply = errors.New(errMsg)
			}
		}

	}
	return ar, err
}

// PubServiceAction sends an action request to a Hub Service on the svc prefix
// Returns the response or an error if the request fails or timed out
func (hc *NatsHubClient) PubServiceAction(
	serviceID string, capability string, actionID string, payload []byte) (hubclient.ActionResponse, error) {

	subject := MakeServiceActionSubject(serviceID, capability, actionID, hc.clientID)
	slog.Info("PubServiceAction", "subject", subject)
	return hc.pubAction(subject, payload)
}

// PubThingAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *NatsHubClient) PubThingAction(
	bindingID string, thingID string, actionID string, payload []byte) (hubclient.ActionResponse, error) {

	subject := MakeThingActionSubject(bindingID, thingID, actionID, hc.clientID)
	slog.Info("PubThingAction", "subject", subject)
	return hc.pubAction(subject, payload)
}

// Sub is a low level subscription to a subject
func (hc *NatsHubClient) _NSub(subject string, cb func(msg *nats.Msg)) (sub hubclient.ISubscription, err error) {
	slog.Info("subscribe", "subject", subject, "clientID", hc.clientID)
	nsub, err := hc.nc.Subscribe(subject, cb)
	isValid := nsub.IsValid()
	if err != nil || !isValid {
		err = fmt.Errorf("subscribe to '%s' failed: %w", subject, err)
	}
	sub = &NatsHubSubscription{nsub: nsub}
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
				DeviceID:  pubID,
				ThingID:   thID,
				Timestamp: timeStamp.Unix(),
				Payload:   natsMsg.Data,
			}
			cb(msg)

		}
	}()
	return nil
}

// Sub subscribe to an address.
// Primarily intended for testing
func (hc *NatsHubClient) Sub(subject string, cb func(topic string, data []byte)) (hubclient.ISubscription, error) {

	sub, err := hc._NSub(subject, func(natsMsg *nats.Msg) {
		cb(natsMsg.Subject, natsMsg.Data)
	})
	return sub, err
}

// SubActions subscribes to actions on the given subject
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *NatsHubClient) SubActions(
	subject string, cb func(msg *hubclient.ActionRequest) error) (
	hubclient.ISubscription, error) {

	sub, err := hc._NSub(subject, func(natsMsg *nats.Msg) {
		md, _ := natsMsg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		payload := natsMsg.Data
		deviceID, thID, name, clientID, err := SplitActionSubject(natsMsg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", natsMsg.Subject)
			return
		}
		actionMsg := &hubclient.ActionRequest{
			//SenderID: natsMsg.Header.
			ClientID:  clientID,
			ActionID:  name,
			DeviceID:  deviceID,
			ThingID:   thID,
			Timestamp: timeStamp.Unix(),
			Payload:   payload,
			SendReply: func(payload []byte, err error) error {
				if err != nil {
					errMsg := hubclient.ErrorMessage{Error: err.Error()}
					payload, _ = ser.Marshal(errMsg)
					natsMsg.Header.Set("error", err.Error())
				}
				//return natsMsg.Respond(payload)
				natsMsg.Data = payload
				return natsMsg.RespondMsg(natsMsg)
			},
			SendAck: func() error {
				return natsMsg.Ack()
			},
		}
		natsMsg.Header = nats.Header{}
		natsMsg.Header.Set("received", timeStamp.Format(time.StampMicro))
		err = cb(actionMsg)
		if err != nil {
			errMsg := hubclient.ErrorMessage{Error: err.Error()}
			slog.Error("action error", "subject", subject, "err", err.Error())
			natsMsg.Header.Set("error", err.Error())
			errPayload, _ := ser.Marshal(errMsg)
			_ = natsMsg.Respond(errPayload)
		}
	})
	return sub, err
}

// SubServiceActions subscribes to service RPC requests.
// Intended for use by services to receive requests.
//
//	capability is the name of the capability (thingID) to handle
func (hc *NatsHubClient) SubServiceActions(
	capability string, cb func(msg *hubclient.ActionRequest) error) (hubclient.ISubscription, error) {

	subject := MakeServiceActionSubject(hc.clientID, capability, "", "")
	return hc.SubActions(subject, cb)
}

// SubThingActions subscribes to action requests of a device's Thing.
// Intended for use by device implementors to receive requests for its things.
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *NatsHubClient) SubThingActions(
	thingID string, cb func(msg *hubclient.ActionRequest) error) (hubclient.ISubscription, error) {

	subject := MakeThingActionSubject(hc.clientID, thingID, "", "")
	return hc.SubActions(subject, cb)
}

func (hc *NatsHubClient) SubThingEvents(
	deviceID string, thingID string,
	cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {

	subject := MakeThingsSubject(deviceID, thingID, vocab.MessageTypeEvent, "")
	nsub, err := hc.nc.Subscribe(subject, func(msg *nats.Msg) {

		_, deviceID, thingID, _, name, err := SplitSubject(msg.Subject)
		if err != nil {
			return
		}
		timeStamp := time.Now().Unix()
		md, _ := msg.Metadata()
		if md != nil {
			timeStamp = md.Timestamp.Unix()
		}
		evmsg := &hubclient.EventMessage{
			//SenderID: msg.Header.
			EventID:   name,
			DeviceID:  deviceID,
			ThingID:   thingID,
			Timestamp: timeStamp,
			Payload:   msg.Data,
		}
		cb(evmsg)
	})
	sub := &NatsHubSubscription{nsub: nsub}
	return sub, err
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