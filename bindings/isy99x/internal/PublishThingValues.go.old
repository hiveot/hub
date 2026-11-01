package internal

// func (svc *IsyBinding) PubEvents(thingID string, evMap map[string]any) {
// 	for k, v := range evMap {
// 		svc.PubEvent(thingID, k, v)
// 	}
// }

// PublishAllThingValues reads and publishes the state of the binding, gateway and nodes.
// Set onlyChanges to only publish changed values.
// func (svc *IsyBinding) PublishAllThingValues(onlyChanges bool) error {
// 	slog.Info("PublishThingValues", slog.Bool("onlyChanges", onlyChanges))

// 	// publish the binding property values
// 	propMap := svc.GetBindingPropValues(onlyChanges)
// 	if len(propMap) > 0 {
// 		// no use continuing if publishing fails
// 		svc.PubProperties(svc.thingID, propMap, true)
// 	}
// 	// read and publish the gateway properties
// 	err := svc.IsyGW.RefreshProperties(onlyChanges)
// 	if err != nil {
// 		err = fmt.Errorf("failed reading ISY: %w", err)
// 		slog.Error(err.Error())
// 		return err
// 	}

// 	// read and publish each thing on the gateway
// 	_ = svc.IsyGW.ReadIsyNodeValues()
// 	isyThings := svc.IsyGW.GetIsyThings()
// 	for _, thing := range isyThings {
// 		thing.ReloadPropertyValues(onlyChanges)
// 	}
// 	return nil
// }
