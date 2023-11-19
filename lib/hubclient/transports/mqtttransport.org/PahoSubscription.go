package mqtttransport_org

import (
	"context"
	"github.com/eclipse/paho.golang/paho"
	"log/slog"
)

// PahoSubscription implements hubclient.ISubscription interface
type PahoSubscription struct {
	topic    string
	pcl      *paho.Client
	clientID string // the client that is subscribing
}

// Unsubscribe from the subscription
func (sub *PahoSubscription) Unsubscribe() error {
	slog.Info("Unsubscribe",
		slog.String("topic", sub.topic),
		slog.String("clientID", sub.clientID))
	u := &paho.Unsubscribe{
		Topics: []string{sub.topic},
		//Properties: nil,
	}
	_, err := sub.pcl.Unsubscribe(context.Background(), u)
	if err != nil {
		slog.Warn("failed unsubscribe",
			slog.String("topic", sub.topic),
			slog.String("clientID", sub.clientID),
			slog.String("err", err.Error()))
	}
	return err
}
