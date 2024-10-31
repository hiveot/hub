// Package service for basic ISY99x Insteon home automation hub access
// This implements common sensors and switches
package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/service/isy"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/wot/exposedthing"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"sync"
	"time"
)

// IsyBinding is the protocol binding for managing the ISY99x Insteon gateway
// TODO: to access multiple gateways, run additional instances,
//
//	or modify this code for multiple isyAPI instances
type IsyBinding struct {

	// Configuration of this protocol binding
	config *config.Isy99xConfig
	hc     hubclient.IHubClient

	thingID      string           // ID of the binding Thing
	isyAPI       *isy.IsyAPI      // methods for communicating met ISY gateway device
	IsyGW        *IsyGatewayThing // device representing the ISY gateway
	wasConnected bool             // prior binding connected status for sending events

	// is gateway currently reachable?
	gwReachable bool

	// product identification map by {cat}.{subcat}
	prodMap map[string]InsteonProduct

	//binding property values
	propValues *exposedthing.ThingValues

	mu              sync.Mutex
	stopHeartbeatFn func()
}

// CreateBindingTD generates a TD document for this binding containing properties,
// event and action definitions.
func (svc *IsyBinding) CreateBindingTD() *tdd.TD {
	td := tdd.NewTD(svc.thingID, "ISY99x binding", vocab.ThingService)

	// binding attributes
	prop := td.AddProperty("", vocab.PropNetConnection,
		"Connected", vocab.WoTDataTypeBool)
	prop.Description = "Whether the Binding has a connection to an ISY gateway"
	//
	prop = td.AddProperty(vocab.PropDeviceMake, vocab.PropDeviceMake,
		"Manufacturer", vocab.WoTDataTypeString)
	prop.Description = "Developer of the binding"

	// TODO: persist configuration
	//binding config
	prop = td.AddProperty(vocab.PropDevicePollinterval, vocab.PropDevicePollinterval,
		"Poll Interval", vocab.WoTDataTypeInteger)
	prop.Description = "Interval the binding polls the gateway for data value updates."
	prop.Unit = vocab.UnitSecond
	prop.ReadOnly = false
	//
	prop = td.AddPropertyAsString("loginName", "", "Login name")
	prop.Description = "ISY99x gateway login name."
	prop.ReadOnly = false
	//
	prop = td.AddPropertyAsString("password", "", "Password")
	prop.Description = "ISY99x gateway password."
	prop.ReadOnly = false
	prop.WriteOnly = true
	//
	prop = td.AddProperty(vocab.PropNetAddress, vocab.PropNetAddress,
		"Network Address", vocab.WoTDataTypeString)
	prop.Description = "ISY99x gateway IP address; empty to auto discover."
	prop.ReadOnly = false

	// binding events
	_ = td.AddEvent("", vocab.PropNetConnection,
		"Connection changed", "Connection with ISY gateway has changed",
		&tdd.DataSchema{Type: vocab.WoTDataTypeBool})

	// no binding actions
	return td
}

// GetBindingPropValues returns the property/event values of this binding
func (svc *IsyBinding) GetBindingPropValues(onlyChanges bool) map[string]any {

	// update the values
	pv := svc.propValues
	pv.SetValue(vocab.PropDevicePollinterval, svc.config.PollInterval)
	pv.SetValue(vocab.PropNetAddress, svc.config.IsyAddress)
	pv.SetValue("loginName", svc.config.LoginName)
	pv.SetValue(vocab.PropDeviceMake, "Hive Of Things")
	pv.SetValue(vocab.PropNetConnection, svc.isyAPI.IsConnected())
	return pv.GetValues(onlyChanges)
}

// HandleBindingConfig configures the binding.
func (svc *IsyBinding) HandleBindingConfig(action *hubclient.ThingMessage) error {
	err := fmt.Errorf("unknown configuration request '%s' from '%s'", action.Name, action.SenderID)
	// connection settings to connect to the gateway
	// FIXME: persist this configuration
	switch action.Name {
	case vocab.PropNetAddress:
		svc.config.IsyAddress = action.DataAsText()
		err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
		if err == nil {
			svc.IsyGW.Init(svc.isyAPI)
		}
	case "loginName":
		svc.config.LoginName = action.DataAsText()
		err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
		if err == nil {
			svc.IsyGW.Init(svc.isyAPI)
		}
	case "password":
		svc.config.Password = action.DataAsText()
		err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
		if err == nil {
			svc.IsyGW.Init(svc.isyAPI)
		}
	}
	return err
}

// Start the ISY99x protocol binding.
// Connection to the gateway will be made during the heartbeat.
// If no connection can be made the heartbeat will retry periodically until stopped.
//
// This publishes a TD for this binding, starts a background polling heartbeat.
func (svc *IsyBinding) Start(hc hubclient.IHubClient) (err error) {
	slog.Info("Starting Isy99x binding")
	svc.hc = hc
	svc.thingID = hc.GetClientID()
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	svc.prodMap, err = LoadProductMapCSV("")

	//// 'IsyThings' use the 'isy connection' to talk to the gateway
	svc.isyAPI = isy.NewIsyAPI()
	svc.IsyGW = NewIsyGateway(svc.prodMap)
	_ = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
	svc.IsyGW.Init(svc.isyAPI)

	// subscribe to action requests
	svc.hc.SetMessageHandler(svc.handleActionRequest)

	// last, start polling heartbeat
	svc.stopHeartbeatFn = svc.startHeartbeat()
	return err
}

// heartbeat polls the gateway device every X seconds and publishes updates
// This returns a stop function that can be used to end the loop
func (svc *IsyBinding) startHeartbeat() (stopFn func()) {

	var tdCountDown = 0
	var pollCountDown = 0
	var republishCountDown = 0
	var isConnected = false
	var forceRepublish bool
	var err error

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		tdCountDown--
		pollCountDown--
		republishCountDown--
		forceRepublish = false

		// if no gateway connection exists, try to reestablish a connection to the gateway
		isConnected = svc.isyAPI.IsConnected()
		if !isConnected {
			// if the connection dropped, send an event
			if svc.wasConnected {
				_ = svc.hc.PubEvent(svc.thingID, vocab.PropNetConnection, isConnected, "")
			}
			err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
			if err == nil {
				// re-establish the gateway device connection
				svc.IsyGW.Init(svc.isyAPI)
			}
			isConnected = svc.isyAPI.IsConnected()
			if isConnected {
				_ = svc.hc.PubEvent(svc.thingID, vocab.PropNetConnection, isConnected, "")
			}
		}
		svc.wasConnected = isConnected

		// publish node TDs and values
		if isConnected && tdCountDown <= 0 {
			err = svc.PublishNodeTDs()
			tdCountDown = svc.config.TDInterval
			// after publishing the TD, republish all values
			forceRepublish = true
		}
		// publish changes to sensor/actuator values
		if isConnected && pollCountDown <= 0 {
			// publish changed values or periodically for publishing all values
			if republishCountDown <= 0 {
				republishCountDown = svc.config.RepublishInterval
				forceRepublish = true
			}
			err = svc.PublishNodeValues(!forceRepublish)
			pollCountDown = svc.config.PollInterval
			// slow down if this fails. Don't flood the logs
			if err != nil {
				pollCountDown = svc.config.PollInterval * 5
			}
		}
	})
	return stopFn
}

// Stop the heartbeat and remove subscriptions
// This does not close the given hubclient connection.
func (svc *IsyBinding) Stop() {
	slog.Info("Stopping the ISY99x Binding")
	//svc.isRunning.Store(false)
	if svc.stopHeartbeatFn != nil {
		svc.stopHeartbeatFn()
		svc.stopHeartbeatFn = nil
	}
	// let processes finish
	time.Sleep(time.Millisecond)
}

// NewIsyBinding creates a new instance of the ISY99x protocol binding service
func NewIsyBinding(cfg *config.Isy99xConfig) *IsyBinding {
	svc := IsyBinding{
		config:     cfg,
		propValues: exposedthing.NewThingValues(),
	}
	return &svc
}
