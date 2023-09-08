package mqtthubclient

import (
	"context"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"golang.org/x/exp/slog"
)

// lower level autopaho functions with connect, pub and sub

// PahoMsg holds an incoming request message with the ability to send a response or ack
type PahoMsg struct {
	ClientID        string
	Topic           string
	Payload         []byte
	cm              *autopaho.ConnectionManager
	correlationData []byte
	responseTopic   string
}

// SendReply replies to the request with a payload
func (pm *PahoMsg) SendReply(payload []byte) {
	if pm.responseTopic == "" {
		slog.Warn("SendReply without response topic",
			slog.String("topic", pm.Topic),
			slog.String("clientID", pm.ClientID))
	}
	// send response
	props := &paho.PublishProperties{
		//User:            paho.UserProperties{},
		CorrelationData: pm.correlationData,
	}
	resp, err := pm.cm.Publish(context.Background(), &paho.Publish{
		QoS:        withQos,
		Retain:     false,
		Topic:      pm.responseTopic,
		Properties: props,
		Payload:    payload,
	})
	if err != nil {
		slog.Warn("SubRequest. Error publishing response",
			slog.String("err", err.Error()),
			slog.Int("reasonCode", int(resp.ReasonCode)))
	}
}

// SendError replies to the request with an error
func (pm *PahoMsg) SendError(err error) {
	if pm.responseTopic == "" {
		slog.Warn("SendError without response topic",
			slog.String("topic", pm.Topic),
			slog.String("clientID", pm.ClientID))
	}
	// send response
	props := &paho.PublishProperties{
		//User:            paho.UserProperties{},
		CorrelationData: pm.correlationData,
	}
	props.User.Add("error", err.Error())
	_, err = pm.cm.Publish(context.Background(), &paho.Publish{
		QoS:        withQos,
		Retain:     false,
		Topic:      pm.responseTopic,
		Properties: props,
		Payload:    nil,
	})
	if err != nil {
		slog.Warn("unable to send response",
			slog.String("topic", pm.Topic),
			slog.String("err", err.Error()),
			slog.String("responseTopic", pm.responseTopic),
		)
	}
}
