package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/lib/plugin"
	"log/slog"
	"time"
)

// PublishIsyTDs reads and publishes the TD document of the gateway and its nodes.
func (svc *IsyBinding) PublishIsyTDs() (err error) {
	slog.Info("PublishIsyTDs Gateway and Nodes")

	td := svc.GetTD()
	err = svc.hc.PubTD(td)
	if err != nil {
		err = fmt.Errorf("failed publishing thing TD: %w", err)
		slog.Error(err.Error())
		return err
	}
	if !svc.isyAPI.IsConnected() {
		return errors.New("not connected to the gateway")
	}

	td = svc.IsyGW.GetTD()
	if td != nil {
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
		//things := svc.IsyGW.ReadThings()
		for _, thing := range svc.IsyGW.GetIsyThings() {
			td = thing.GetTD()
			err = svc.hc.PubTD(td)
			if err != nil {
				slog.Error("PollIsyTDs", "err", err)
			}
		}
	}
	return err
}

func (svc *IsyBinding) PubEvents(thingID string, evMap map[string]string) {
	for k, v := range evMap {
		vJson, _ := json.Marshal(v)
		svc.hc.PubEvent(thingID, k, vJson)
	}
}

// PublishValues reads and publishes property/event values of the binding, gateway and nodes
// Set onlyChanges to only publish changed values as events
func (svc *IsyBinding) PublishValues(onlyChanges bool) error {
	slog.Info("PublishValues", slog.Bool("onlyChanges", onlyChanges))

	// publish the binding's property and event values
	props, events := svc.GetPropValues(onlyChanges)
	bindingID := svc.hc.ClientID()
	err := svc.hc.PubProps(bindingID, props)
	// no use continuing if publishing fails
	if err != nil {
		err = fmt.Errorf("failed publishing ISY binding props: %w", err)
		slog.Error(err.Error())
		return err
	}
	for k, v := range events {
		payload, _ := json.Marshal(v)
		err = svc.hc.PubEvent(bindingID, k, payload)

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
		_ = svc.hc.PubProps(svc.IsyGW.GetID(), props)
		svc.PubEvents(svc.IsyGW.GetID(), events)
	}

	// read and publish props of each node
	_ = svc.IsyGW.ReadIsyNodeValues()
	isyThings := svc.IsyGW.GetIsyThings()
	for _, thing := range isyThings {
		props = thing.GetPropValues(onlyChanges)
		if len(props) > 0 {
			_ = svc.hc.PubProps(thing.GetID(), props)
			// send an event for each of the changed values
			if onlyChanges {
				svc.PubEvents(thing.GetID(), props)
			}
		}
	}
	return nil
}

// heartbeat polls the gateway device every X seconds and publishes updates
// This returns a stop function that can be used to end the loop
func (svc *IsyBinding) startHeartbeat() (stopFn func()) {

	var tdCountDown = 0
	var pollCountDown = 0
	var isConnected = false
	var onlyChanges = false
	var err error

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		// if no gateway connection exists, try to reestablish a connection to the gateway
		isConnected = svc.isyAPI.IsConnected()
		if !isConnected {
			err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
			if err == nil {
				// re-establish the gateway device connection
				svc.IsyGW.Init(svc.isyAPI)
			}
			isConnected = svc.isyAPI.IsConnected()
		}

		// publish node TDs and values
		tdCountDown--
		if isConnected && tdCountDown <= 0 {
			err = svc.PublishIsyTDs()
			tdCountDown = svc.config.TDInterval
			// after publishing the TD, republish all values
			onlyChanges = false
		}
		// publish changes to sensor/actuator values
		pollCountDown--
		if isConnected && pollCountDown <= 0 {
			err = svc.PublishValues(onlyChanges)
			pollCountDown = svc.config.PollInterval
			// slow down if this fails. Don't flood the logs
			if err != nil {
				pollCountDown = svc.config.PollInterval * 5
			}
			onlyChanges = true
		}
	})
	return stopFn
}
