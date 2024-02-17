package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"time"
)

// PublishIsyTDs reads and publishes the TD document of the gateway and its nodes.
func (svc *IsyBinding) PublishIsyTDs() (err error) {
	slog.Info("Polling Gateway and Nodes")

	td := svc.GetTD()
	err = svc.hc.PubTD(td)
	if err != nil {
		err = fmt.Errorf("failed publishing thing TD: %w", err)
		slog.Error(err.Error())
		return err
	}

	td = svc.IsyGW.GetTD()
	err = svc.hc.PubTD(td)
	if err != nil {
		err = fmt.Errorf("failed publishing gateway TD: %w", err)
		slog.Error(err.Error())
		return err
	}

	// read and publish the node TDs
	err = svc.IsyGW.ReadIsyThings()
	if err != nil {
		err = fmt.Errorf("failed reading ISY nodes from gateway: %w", err)
		slog.Error(err.Error())
		return err
	}
	//things := svc.IsyGW.GetThings()
	for _, thing := range svc.IsyGW.GetIsyThings() {
		td = thing.GetTD()
		err = svc.hc.PubTD(td)
		if err != nil {
			slog.Error("PollIsyTDs", "err", err)
		}
	}
	return err
}

// PublishPropValues reads and publishes property values of the binding,gateway and nodes
// Set onlyChanges to only publish changed values.
func (svc *IsyBinding) PublishPropValues(onlyChanges bool) error {
	slog.Info("PollThingPropValues", slog.Bool("onlyChanges", onlyChanges))
	err := svc.IsyGW.ReadGatewayValues()
	if err != nil {
		err = fmt.Errorf("failed reading ISY: %w", err)
		slog.Error(err.Error())
		return err
	}

	// publish binding props
	// TODO: a separate NodeThing for the binding itself?
	bindingID := svc.hc.ClientID()
	//props := svc.GetProps(onlyChanges)
	props := make(map[string]string)
	props[vocab.VocabPollInterval] = fmt.Sprintf("%d", svc.config.PollInterval)
	props[vocab.VocabGatewayAddress] = svc.config.IsyAddress
	err = svc.hc.PubProps(bindingID, props)

	// no use continuing if publishing fails
	if err != nil {
		err = fmt.Errorf("failed publishing ISY binding props: %w", err)
		slog.Error(err.Error())
		return err
	}
	// publish gateway props
	props = svc.IsyGW.GetProps(onlyChanges)
	_ = svc.hc.PubProps(svc.IsyGW.GetID(), props)

	// read and publish props of each node
	_ = svc.IsyGW.ReadIsyNodeValues()
	isyThings := svc.IsyGW.GetIsyThings()
	for _, thing := range isyThings {
		props = thing.GetProps(onlyChanges)
		_ = svc.hc.PubProps(thing.GetID(), props)
	}
	return nil
}

// heartbeat polls the gateway device every X seconds and publishes updates
// This returns a stop function that can be used to end the loop
func (svc *IsyBinding) startHeartbeat() (stopFn func()) {

	var tdCountDown = 0
	var pollCountDown = 0
	var err error

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		onlyChanges := true
		// publish node TDs and values
		tdCountDown--
		if tdCountDown <= 0 {
			err = svc.PublishIsyTDs()
			tdCountDown = svc.config.TDInterval
			// after publishing the TD, republish all values
			onlyChanges = false
		}
		// publish changes to sensor/actuator values
		pollCountDown--
		if pollCountDown <= 0 {
			err = svc.PublishPropValues(onlyChanges)
			pollCountDown = svc.config.PollInterval
			// slow down if this fails. Don't flood the logs
			if err != nil {
				pollCountDown = svc.config.PollInterval * 5
			}
		}
	})
	return stopFn
}
