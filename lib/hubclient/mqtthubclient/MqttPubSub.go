package mqtthubclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"log/slog"
	"time"
)

// lower level autopaho functions with connect, pub and sub

const withQos = 1

// PahoSubscription implements hubclient.ISubscription interface
type PahoSubscription struct {
	topic string
	pcl   *paho.Client
}

// MqttHubSubscription  subscription helper
// This implements ISubscription
//type MqttHubSubscription struct {
//	topic   string
//	handler func(topic string, payload []byte)
//	pcl     *paho.Client
//}

// Unsubscribe from the subscription
func (sub *PahoSubscription) Unsubscribe() {
	slog.Info("Unsubscribe", "topic", sub.topic)
	u := &paho.Unsubscribe{
		Topics: []string{sub.topic},
		//Properties: nil,
	}
	_, err := sub.pcl.Unsubscribe(context.Background(), u)
	if err != nil {
		slog.Warn("failed unsubscribe", "topic", sub.topic, "err", err.Error())
	}
}

// ParseResponse helper message to parse response and check for errors
func (hc *MqttHubClient) ParseResponse(data []byte, resp interface{}) error {
	var err error
	if data == nil || len(data) == 0 {
		if resp != nil {
			err = errors.New("expected response but none received")
		} else {
			err = nil // all good
		}
	} else {
		if resp == nil {
			err = errors.New("unexpected response")
		} else {
			err = ser.Unmarshal(data, resp)
		}
	}
	return err
}

// Pub publishes a message and returns
func (hc *MqttHubClient) Pub(topic string, payload []byte) (err error) {
	slog.Info("Pub", "topic", topic)
	ctx, cancelFn := context.WithTimeout(context.Background(), hc.timeout)
	defer cancelFn()
	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
	}
	respMsg, err := hc.pcl.Publish(ctx, pubMsg)
	_ = respMsg
	if err != nil {
		return err
	}
	return err
}

// PubRequest publishes a request message and waits for an answer or until timeout
func (hc *MqttHubClient) PubRequest(topic string, payload []byte) (ar hubclient.ActionResponse, err error) {
	slog.Info("PubRequest start:", "topic", topic)

	ctx, cancelFn := context.WithTimeout(context.Background(), hc.timeout)
	defer cancelFn()

	t1 := time.Now()
	ar.Address = topic

	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
		Properties: &paho.PublishProperties{
			ContentType: "json",
			User: paho.UserProperties{{
				Key:   "test",
				Value: "test",
			}},
		},
	}
	// use the inbox as the custom response for this client instance
	// clone of rpc.go to workaround hangup when no response is received #111
	respMsg, err := hc.requestHandler.Request(ctx, pubMsg)
	ar.Duration = time.Now().Sub(t1)
	if err != nil {
		return ar, err
	}
	ar.SentSuccess = true
	ar.Payload = respMsg.Payload
	ar.ReceivedReply = len(respMsg.Payload) > 0

	// test alternative to handling errors since User properties aren't
	// passed through for some reason.
	if respMsg.Properties.ContentType == "error" {
		ar.ErrorReply = errors.New(string(respMsg.Payload))
		err = ar.ErrorReply
	}
	//errResp := respMsg.Properties.User.Get("error")
	//if errResp != "" {
	//	ar.ErrorReply = errors.New(errResp)
	//	err = ar.ErrorReply
	//}
	slog.Info("PubRequest end:",
		slog.String("topic", topic),
		slog.String("ContentType (if any)", respMsg.Properties.ContentType),
		slog.Duration("duration", ar.Duration))
	return ar, err
}

// PubAction sends an action request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *MqttHubClient) PubAction(
	deviceID string, thingID string, actionID string, payload []byte) (
	hubclient.ActionResponse, error) {

	topic := MakeTopic(vocab.MessageTypeAction, deviceID, thingID, actionID, hc.clientID)
	//slog.Info("PubAction", "topic", topic)
	return hc.PubRequest(topic, payload)
}

// PubConfig sends an configuration update request to the hub and receives a response
// Returns the response or an error if the request fails or timed out
func (hc *MqttHubClient) PubConfig(
	deviceID string, thingID string, propID string, payload []byte) (
	hubclient.ActionResponse, error) {

	topic := MakeTopic(vocab.MessageTypeConfig, deviceID, thingID, propID, hc.clientID)
	//slog.Info("PubConfig", "topic", topic)
	return hc.PubRequest(topic, payload)
}

// PubEvent sends the event value to the hub
func (hc *MqttHubClient) PubEvent(thingID string, eventID string, payload []byte) error {
	topic := MakeTopic(vocab.MessageTypeEvent, hc.clientID, thingID, eventID, hc.clientID)
	//slog.Info("PubEvent", "topic", topic)
	err := hc.Pub(topic, payload)
	return err
}

// PubServiceRPC sends an RPC request to a Hub Service
// Returns the response or an error if the request fails or timed out
func (hc *MqttHubClient) PubServiceRPC(
	serviceID string, capability string, actionID string, payload []byte) (
	hubclient.ActionResponse, error) {

	topic := MakeTopic(vocab.MessageTypeRPC, serviceID, capability, actionID, hc.clientID)
	//slog.Info("PubServiceAction", "topic", topic)
	return hc.PubRequest(topic, payload)
}

// PubTD publishes an event containing a TD document
func (hc *MqttHubClient) PubTD(td *thing.TD) error {
	payload, _ := ser.Marshal(td)
	topic := MakeTopic(vocab.MessageTypeEvent, hc.clientID, td.ID, vocab.EventNameTD, hc.clientID)
	//slog.Info("PubTD", "topic", topic)
	err := hc.Pub(topic, payload)
	return err
}

// sendReply sends a reply on the response topic of the request
// This uses the same QoS as the request, without retain.
//
//	req is the request to reply to
//	optionally include a payload in the reply
//	optionally include an error message in the reply
func (hc *MqttHubClient) sendReply(req *paho.Publish, payload []byte, errResp error) error {

	slog.Info("sendReply", "topic", req.Topic, "responseTopic", req.Properties.ResponseTopic)

	responseTopic := req.Properties.ResponseTopic
	if responseTopic == "" {
		err2 := fmt.Errorf("sendReply. No response topic. Not sending reply.")
		slog.Error(err2.Error())
	}
	replyMsg := &paho.Publish{
		QoS:    req.QoS,
		Retain: false,
		Topic:  responseTopic,
		Properties: &paho.PublishProperties{
			CorrelationData: req.Properties.CorrelationData,
			User:            req.Properties.User,
			PayloadFormat:   req.Properties.PayloadFormat,
			ContentType:     req.Properties.ContentType,
		},
		Payload: payload,
	}
	if errResp != nil {
		replyMsg.Properties.ContentType = "error" // payload is an error message
		replyMsg.Properties.User.Add("error", errResp.Error())
		// for testing, somehow properties.user is not transferred
		replyMsg.Payload = []byte(errResp.Error())
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn()
	_, err := hc.pcl.Publish(ctx, replyMsg)
	if err != nil {
		slog.Warn("SubRequest. Error publishing response",
			slog.String("err", err.Error()))
	}
	return err
}

// Sub subscribes to a topic
func (hc *MqttHubClient) Sub(topic string, cb func(topic string, msg []byte)) (hubclient.ISubscription, error) {
	slog.Info("Sub", "topic", topic)
	spacket := &paho.Subscribe{
		Properties: nil,
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {QoS: withQos},
		},
	}
	suback, err := hc.pcl.Subscribe(context.Background(), spacket)
	hc.pcl.Router.RegisterHandler(topic, func(m *paho.Publish) {
		slog.Info("Sub, received Msg:", "topic", m.Topic)
		//clientID := m.Properties.User.Get("clientID") // experimental

		// run this in the background to allow for reentrancy
		go func() {
			cb(m.Topic, m.Payload)
		}()
	})
	_ = suback
	hcSub := &PahoSubscription{
		topic: topic,
		pcl:   hc.pcl,
	}
	return hcSub, err
}

// SubRequest subscribes to a requests and sends a response
// Intended for actions, config and rpc requests
func (hc *MqttHubClient) SubRequest(
	topic string, cb func(actionMsg *hubclient.RequestMessage) error) (
	hubclient.ISubscription, error) {

	slog.Info("SubRequest", "topic", topic)

	spacket := &paho.Subscribe{
		Properties: nil,
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {QoS: withQos},
		},
	}
	suback, err := hc.pcl.Subscribe(context.Background(), spacket)
	_ = suback
	hc.pcl.Router.RegisterHandler(topic, func(m *paho.Publish) {

		timeStamp := time.Now()
		_, deviceID, thingID, name, clientID, err := SplitTopic(m.Topic)
		// requests MUST contain clientID
		if clientID == "" {
			slog.Warn("SubRequest: Ignored request without clientID", "topic", m.Topic)
			return
		}
		requestMsg := &hubclient.RequestMessage{
			ClientID:  clientID,
			DeviceID:  deviceID,
			ThingID:   thingID,
			ActionID:  name,
			Payload:   m.Payload,
			Timestamp: timeStamp.Unix(),
			SendReply: func(payload []byte, err error) error {
				return hc.sendReply(m, payload, err)
			},
			SendAck: func() error {
				return hc.sendReply(m, nil, nil)
			},
		}
		m.Properties.User.Add("received", timeStamp.Format(time.StampMicro))
		// run this in the background to allow for reentrancy
		go func() {
			err = cb(requestMsg)
			if err != nil {
				slog.Error("SubRequest: handle request failed",
					slog.String("err", err.Error()),
					slog.String("topic", topic))

				err = hc.sendReply(m, nil, err)
			}
		}()
	})

	hcSub := &PahoSubscription{
		topic: topic,
		pcl:   hc.pcl,
	}

	return hcSub, err
}

// SubActions subscribes to actions for this device or service
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *MqttHubClient) SubActions(
	thingID string, cb func(msg *hubclient.RequestMessage) error,
) (hubclient.ISubscription, error) {

	topic := MakeTopic(vocab.MessageTypeAction, hc.clientID, thingID, "", "")
	return hc.SubRequest(topic, cb)
}

// SubConfig subscribes to configuration request for this device or service
//
//	thingID is the device thing or service capability to subscribe to, or "" for wildcard
func (hc *MqttHubClient) SubConfig(
	thingID string, cb func(msg *hubclient.RequestMessage) error,
) (hubclient.ISubscription, error) {

	topic := MakeTopic(vocab.MessageTypeConfig, hc.clientID, thingID, "", "")
	return hc.SubRequest(topic, cb)
}

func (hc *MqttHubClient) SubEvents(
	deviceID string, thingID string,
	cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {

	topic := MakeTopic(vocab.MessageTypeEvent, deviceID, thingID, "", "")

	return hc.Sub(topic, func(topic string, payload []byte) {

		_, deviceID, thingID, name, _, err := SplitTopic(topic)
		if err != nil {
			slog.Info("SplitTopic fail", "topic", topic, "err", err)
			return
		}
		eventMsg := &hubclient.EventMessage{
			DeviceID:  deviceID,
			ThingID:   thingID,
			EventID:   name,
			Payload:   payload,
			Timestamp: time.Now().Unix(),
		}
		cb(eventMsg)
	})
}

// SubServiceRPC subscribes to RPC requests for a service capability
//
//	capability is the name of the capability (thingID) to handle
func (hc *MqttHubClient) SubServiceRPC(
	capability string, cb func(msg *hubclient.RequestMessage) error,
) (hubclient.ISubscription, error) {

	topic := MakeTopic(vocab.MessageTypeRPC, hc.clientID, capability, "", "")
	return hc.SubRequest(topic, cb)
}

// SubStream subscribes to events received by the event stream.
//
// This creates an ephemeral pull consumer.
// ReceiveLatest is handy to be up to date on all event instead of quering them separately. Only use this if
// you're going to retrieve them anyways.
//
//	 name of the event stream. "" for default
//		receiveLatest to immediately receive the latest event for each event instance
func (hc *MqttHubClient) SubStream(
	name string, receiveLatest bool, cb func(msg *hubclient.EventMessage)) (hubclient.ISubscription, error) {

	return nil, fmt.Errorf("not implemented")
}