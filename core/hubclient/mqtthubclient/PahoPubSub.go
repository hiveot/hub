package mqtthubclient

import (
	"context"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/autopaho/extensions/rpc"
	"github.com/eclipse/paho.golang/paho"
	"github.com/hiveot/hub/api/go/hubclient"
	"golang.org/x/exp/slog"
	"time"
)

// lower level autopaho functions with connect, pub and sub

const withQos = 1

// PahoSubscription implements hubclient.ISubscription interface
type PahoSubscription struct {
	topic string
	cm    *autopaho.ConnectionManager
}

// Unsubscribe from the subscription
func (sub *PahoSubscription) Unsubscribe() {
	u := &paho.Unsubscribe{
		Topics: []string{sub.topic},
		//Properties: nil,
	}
	_, err := sub.cm.Unsubscribe(context.Background(), u)
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
	respMsg, err := hc.cm.Publish(ctx, pubMsg)
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

	hopts := rpc.HandlerOpts{
		Conn:             hc.cm,
		Router:           hc.router,
		ResponseTopicFmt: "_INBOX.%s", // private inbox for
		ClientID:         hc.clientID,
	}
	handler, err := rpc.NewHandler(ctx, hopts)

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
	suback, err := hc.cm.Subscribe(context.Background(), spacket)
	_ = suback
	hcSub := &PahoSubscription{
		topic: topic,
		cm:    hc.cm,
	}
	return hcSub, err
}

// SubRequest subscribes to action requests and support sending a response
func (hc *MqttHubClient) SubRequest(
	topic string, cb func(req *PahoMsg) error) (hubclient.ISubscription, error) {

	hc.router.RegisterHandler(topic, func(m *paho.Publish) {
		clientID := m.Properties.User.Get("clientID") // experimental
		req := &PahoMsg{
			ClientID:        clientID,
			Topic:           topic,
			Payload:         m.Payload,
			cm:              hc.cm,
			correlationData: m.Properties.CorrelationData,
			responseTopic:   m.Properties.ResponseTopic,
		}
		err := cb(req)
		if err != nil {
			req.SendError(err)
		}
	})

	hcSub := &PahoSubscription{
		topic: topic,
		cm:    hc.cm,
	}

	return hcSub, nil
}
