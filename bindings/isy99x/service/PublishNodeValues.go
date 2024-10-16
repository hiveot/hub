package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

func (svc *IsyBinding) PubEvents(thingID string, evMap map[string]any) {
	for k, v := range evMap {
		_ = svc.hc.PubEvent(thingID, k, v, "")
	}
}

// PublishNodeValues reads and publishes property/event values of the binding, gateway and nodes
// Set onlyChanges to only publish changed values as events
func (svc *IsyBinding) PublishNodeValues(onlyChanges bool) error {
	slog.Info("PublishNodeValues", slog.Bool("onlyChanges", onlyChanges))

	// publish the binding's property and event values
	props, events := svc.GetBindingPropValues(onlyChanges)
	bindingID := svc.hc.GetClientID()
	err := svc.hc.PubProperties(bindingID, props)
	// no use continuing if publishing fails
	if err != nil {
		err = fmt.Errorf("failed publishing ISY binding props: %w", err)
		slog.Error(err.Error())
		return err
	}
	for k, v := range events {
		payload, _ := json.Marshal(v)
		err = svc.hc.PubEvent(bindingID, k, string(payload), "")

	}

	// publish the gateway device values
	err = svc.IsyGW.ReadGatewayValues()
	if err != nil {
		err = fmt.Errorf("failed reading ISY: %w", err)
		slog.Error(err.Error())
		return err
	}
	props, events = svc.IsyGW.GetValues(onlyChanges)
	if len(props) > 0 {
		_ = svc.hc.PubProperties(svc.IsyGW.GetID(), props)
		svc.PubEvents(svc.IsyGW.GetID(), events)
	}

	// read and publish props of each node
	_ = svc.IsyGW.ReadIsyNodeValues()
	isyThings := svc.IsyGW.GetIsyThings()
	for _, thing := range isyThings {
		props = thing.GetPropValues(onlyChanges)
		if len(props) > 0 {
			_ = svc.hc.PubProperties(thing.GetID(), props)
			// send an event for each of the changed values
			if onlyChanges {
				svc.PubEvents(thing.GetID(), props)
			}
		}
	}
	return nil
}
