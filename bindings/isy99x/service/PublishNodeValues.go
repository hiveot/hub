package service

import (
	"fmt"
	"log/slog"
)

func (svc *IsyBinding) PubEvents(thingID string, evMap map[string]any) {
	for k, v := range evMap {
		_ = svc.hc.PubEvent(thingID, k, v, "")
	}
}

// PublishNodeValues reads and publishes the state of the binding, gateway and nodes.
// Set onlyChanges to only publish changed values.
func (svc *IsyBinding) PublishNodeValues(onlyChanges bool) error {
	slog.Info("PublishNodeValues", slog.Bool("onlyChanges", onlyChanges))

	// publish the binding's property and event values
	props := svc.GetBindingPropValues(onlyChanges)
	bindingID := svc.hc.GetClientID()
	slog.Info("PublishNodeValues - properties",
		slog.String("thingID", bindingID),
		slog.Int("nrProps", len(props)),
	)
	err := svc.hc.PubProperties(bindingID, props)
	// no use continuing if publishing fails
	if err != nil {
		err = fmt.Errorf("failed publishing ISY binding props: %w", err)
		slog.Error(err.Error())
		return err
	}

	// publish the gateway device properties and events
	err = svc.IsyGW.ReadGatewayValues()
	if err != nil {
		err = fmt.Errorf("failed reading ISY: %w", err)
		slog.Error(err.Error())
		return err
	}
	props = svc.IsyGW.GetPropertyValues(onlyChanges)
	if len(props) > 0 {
		_ = svc.hc.PubProperties(svc.IsyGW.GetID(), props)
	}

	// read and publish props of each node
	_ = svc.IsyGW.ReadIsyNodeValues()
	isyThings := svc.IsyGW.GetIsyThings()
	for _, thing := range isyThings {
		props = thing.GetPropValues(onlyChanges)
		if len(props) > 0 {
			_ = svc.hc.PubProperties(thing.GetID(), props)
		}
	}
	return nil
}
