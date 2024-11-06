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

// PublishAllThingValues reads and publishes the state of the binding, gateway and nodes.
// Set onlyChanges to only publish changed values.
func (svc *IsyBinding) PublishAllThingValues(onlyChanges bool) error {
	slog.Info("PublishThingValues", slog.Bool("onlyChanges", onlyChanges))

	// publish the binding property values
	propMap := svc.GetBindingPropValues(onlyChanges)
	if len(propMap) > 0 {
		// no use continuing if publishing fails
		err := svc.hc.PubMultipleProperties(svc.thingID, propMap)
		if err != nil {
			err = fmt.Errorf("failed publishing ISY binding props: %w", err)
			slog.Error(err.Error())
			return err
		}
	}
	// read the gateway properties
	err := svc.IsyGW.ReadGatewayValues()
	if err != nil {
		err = fmt.Errorf("failed reading ISY: %w", err)
		slog.Error(err.Error())
		return err
	}
	propMap = svc.IsyGW.GetPropValues(onlyChanges)
	if len(propMap) > 0 {
		_ = svc.hc.PubMultipleProperties(svc.IsyGW.GetID(), propMap)
	}

	// read and publish each gateway connected node on the gateway
	_ = svc.IsyGW.ReadIsyNodeValues()
	isyThings := svc.IsyGW.GetIsyThings()
	for _, thing := range isyThings {
		propMap := thing.GetPropValues(onlyChanges)
		if len(propMap) > 0 {
			_ = svc.hc.PubMultipleProperties(thing.GetID(), propMap)
		}
	}
	return nil
}
