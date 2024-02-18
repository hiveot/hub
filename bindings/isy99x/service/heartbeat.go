package service

import (
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
	if !svc.ic.IsConnected() {
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
		//things := svc.IsyGW.GetThings()
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

// PublishPropValues reads and publishes property values of the binding,gateway and nodes
// Set onlyChanges to only publish changed values.
func (svc *IsyBinding) PublishPropValues(onlyChanges bool) error {
	slog.Info("PublishPropValues", slog.Bool("onlyChanges", onlyChanges))

	// publish the binding's values
	props := svc.GetPropValues(onlyChanges)

	// the binding ID is that of the publisher
	bindingID := svc.hc.ClientID()
	err := svc.hc.PubProps(bindingID, props)

	// no use continuing if publishing fails
	if err != nil {
		err = fmt.Errorf("failed publishing ISY binding props: %w", err)
		slog.Error(err.Error())
		return err
	}

	// publish the gateway device values
	err = svc.IsyGW.ReadGatewayValues()
	if err != nil {
		err = fmt.Errorf("failed reading ISY: %w", err)
		slog.Error(err.Error())
		return err
	}
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
	var isConnected = false
	var onlyChanges = false
	var err error

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		// if no gateway connection exists, try to reestablish a connection to the gateway
		isConnected = svc.ic.IsConnected()
		if !isConnected {
			err = svc.ic.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
			if err == nil {
				// re-establish the gateway device connection
				svc.IsyGW.Init(svc.ic)
			}
			isConnected = svc.ic.IsConnected()
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
			err = svc.PublishPropValues(onlyChanges)
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
