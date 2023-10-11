package service

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/pkg/history"
	"github.com/hiveot/hub/pkg/pubsub"
)

type PubSubEventHandler struct {
	addHistory history.IAddHistory
	serviceSub pubsub.IServicePubSub
}

// Start listening for events
func (svc *PubSubEventHandler) Start() error {
	ctx := context.Background()
	err := svc.serviceSub.SubEvents(ctx, "", "", "",
		func(eventValue thing.ThingValue) {
			logrus.Infof("received event '%s'", eventValue.ID)

			_ = svc.addHistory.AddEvent(ctx, eventValue)
		})
	if err != nil {
		logrus.Errorf("Failed subscribing to event: %s", err)
	}
	return err
}

// Stop releases the add history and pubsub clients
func (svc *PubSubEventHandler) Stop() {
	svc.addHistory.Release()
	svc.serviceSub.Release()
}

// NewSubEventHandler events obtained through pubsub
func NewSubEventHandler(sub pubsub.IServicePubSub, add history.IAddHistory) *PubSubEventHandler {
	pubsubHandler := &PubSubEventHandler{
		addHistory: add,
		serviceSub: sub,
	}
	return pubsubHandler
}
