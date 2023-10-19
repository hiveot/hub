package service

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
	"log/slog"
)

type PubSubEventHandler struct {
	addHistory *AddHistory
	eventSub   hubclient.ISubscription
	hc         hubclient.IHubClient
}

// Start listening for events
func (svc *PubSubEventHandler) Start() error {
	var err error
	svc.eventSub, err = svc.hc.SubEvents("", "", "",
		func(msg *thing.ThingValue) {
			slog.Info("received event", slog.String("name", msg.Name))
			_ = svc.addHistory.AddEvent(msg)
		})
	if err != nil {
		slog.Error("Failed subscribing to event", "err", err)
	}
	return err
}

// Stop releases the add history and pubsub clients
func (svc *PubSubEventHandler) Stop() {
	svc.eventSub.Unsubscribe()
}

// NewSubEventHandler events obtained through pubsub
func NewSubEventHandler(hc hubclient.IHubClient, add *AddHistory) *PubSubEventHandler {
	pubsubHandler := &PubSubEventHandler{
		addHistory: add,
		hc:         hc,
	}
	return pubsubHandler
}
