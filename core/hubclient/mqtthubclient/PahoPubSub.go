package mqtthubclient

import (
	"context"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/extensions/rpc"
	"github.com/hiveot/hub/api/go/hubclient"
	"golang.org/x/exp/slog"
	"time"
)

// lower level autopaho functions with connect, pub and sub

const withQos = 1

// PahoSubscription implements hubclient.ISubscription interface
type PahoSubscription struct {
	topic string
	pcl   *paho.Client
}

// Unsubscribe from the subscription
func (sub *PahoSubscription) Unsubscribe() {
	u := &paho.Unsubscribe{
		Topics: []string{sub.topic},
		//Properties: nil,
	}
	_, err := sub.pcl.Unsubscribe(context.Background(), u)
	if err != nil {
		slog.Warn("failed unsubscribe", "topic", sub.topic, "err", err.Error())
	}
}

// Pub publishes a message and returns
func (hc *MqttHubClient) Pub(topic string, payload []byte) (err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
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

// Request publishes a request message and waits for an answer or until timeout
func (hc *MqttHubClient) Request(topic string, payload []byte) (resp []byte, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()

	//hopts := rpc.HandlerOpts{
	//	Conn:             hc.cm,
	//	Router:           hc.router,
	//	ResponseTopicFmt: "_INBOX.%s", // private inbox for
	//	ClientID:         hc.clientID,
	//}
	handler, err := rpc.NewHandler(ctx, hc.pcl)

	pubMsg := &paho.Publish{
		QoS:     withQos,
		Retain:  false,
		Topic:   topic,
		Payload: payload,
	}
	respMsg, err := handler.Request(ctx, pubMsg)
	if err != nil {
		return nil, err
	}
	return respMsg.Payload, err
}

// Sub subscribes to a topic
func (hc *MqttHubClient) Sub(topic string, cb func(topic string, msg []byte)) (hubclient.ISubscription, error) {
	spacket := &paho.Subscribe{
		Properties: nil,
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {QoS: withQos},
		},
	}
	suback, err := hc.pcl.Subscribe(context.Background(), spacket)
	hc.pcl.Router.RegisterHandler(topic, func(m *paho.Publish) {
		//clientID := m.Properties.User.Get("clientID") // experimental
		cb(m.Topic, m.Payload)
	})
	_ = suback
	hcSub := &PahoSubscription{
		topic: topic,
		pcl:   hc.pcl,
	}
	return hcSub, err
}

// sendReply sends a reply on the response topic of the request
// This uses the same QoS as the request, without retain.
//
//	req is the request to reply to
//	optionally include a payload in the reply
//	optionally include an error message in the reply
func (hc *MqttHubClient) sendReply(req *paho.Publish, payload []byte, errResp error) error {
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
		},
		Payload: payload,
	}
	if errResp != nil {
		replyMsg.Properties.User.Add("error", errResp.Error())
	}
	pubResp, err := hc.pcl.Publish(context.Background(), replyMsg)
	if err != nil {
		slog.Warn("SubRequest. Error publishing response",
			slog.String("err", err.Error()),
			slog.Int("reasonCode", int(pubResp.ReasonCode)))
	}
	return err
}

// SubActions subscribes to action requests and support sending a response
func (hc *MqttHubClient) SubActions(
	topic string, cb func(actionMsg *hubclient.ActionRequest) error) (
	hubclient.ISubscription, error) {

	hc.pcl.Router.RegisterHandler(topic, func(m *paho.Publish) {
		clientID := m.Properties.User.Get("clientID") // experimental
		_, deviceID, thingID, _, name, err := SplitTopic(topic)
		actionMsg := &hubclient.ActionRequest{
			ClientID:  clientID,
			DeviceID:  deviceID,
			ThingID:   thingID,
			ActionID:  name,
			Payload:   m.Payload,
			Timestamp: time.Now().Unix(),
			SendReply: func(payload []byte, err error) error {
				return hc.sendReply(m, payload, err)
			},
			SendAck: func() error {
				return hc.sendReply(m, nil, nil)
			},
		}
		err = cb(actionMsg)
		if err != nil {
			slog.Error("handle request failed",
				slog.String("err", err.Error()),
				slog.String("topic", topic))

			err = hc.sendReply(m, nil, err)
		}
	})

	hcSub := &PahoSubscription{
		topic: topic,
		pcl:   hc.pcl,
	}

	return hcSub, nil
}
