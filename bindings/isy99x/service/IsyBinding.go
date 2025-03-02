// Package service for basic ISY99x Insteon home automation hub access
// This implements common sensors and switches
package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/service/isy"
	"github.com/hiveot/hub/lib/exposedthing"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot/td"
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
	ag     *messaging.Agent

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

// GetBindingPropValues returns the property/event values of this binding
func (svc *IsyBinding) GetBindingPropValues(onlyChanges bool) map[string]any {

	// update the values
	pv := svc.propValues
	pv.SetValue(vocab.PropDevicePollinterval, svc.config.PollInterval)
	pv.SetValue(vocab.PropNetAddress, svc.config.IsyAddress)
	pv.SetValue("loginName", svc.config.LoginName)
	pv.SetValue(vocab.PropDeviceMake, "Hive Of Things")
	pv.SetValue(vocab.PropNetConnection, svc.isyAPI.IsConnected())

	values := pv.GetValues(onlyChanges)
	return values
}

// onIsyEvent publishes the event sent by one of the ISY thing.
func (svc *IsyBinding) onIsyEvent(thingID string, evName string, value any) {
	_ = svc.ag.PubEvent(thingID, evName, value)
}

// HandleWriteBindingProperty configures the binding.
func (svc *IsyBinding) HandleWriteBindingProperty(
	req *messaging.RequestMessage) *messaging.ResponseMessage {

	err := fmt.Errorf("unknown configuration request '%s' from '%s'", req.Name, req.SenderID)
	// connection settings to connect to the gateway
	// FIXME: persist this configuration
	switch req.Name {
	case vocab.PropNetAddress:
		svc.config.IsyAddress = req.ToString(0)
		err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
		if err == nil {
			svc.IsyGW.Init(svc.isyAPI)
		}
	case "loginName":
		svc.config.LoginName = req.ToString(0)
		err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
		if err == nil {
			svc.IsyGW.Init(svc.isyAPI)
		}
	case "password":
		svc.config.Password = req.ToString(0)
		err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
		if err == nil {
			svc.IsyGW.Init(svc.isyAPI)
		}
	}
	return req.CreateResponse(nil, err)
}

// MakeBindingTD generates a TD document for this binding containing properties,
// event and action definitions.
func (svc *IsyBinding) MakeBindingTD() *td.TD {
	tdi := td.NewTD(svc.thingID, "ISY99x binding", vocab.ThingService)

	// binding attributes
	prop := tdi.AddProperty(vocab.PropNetConnection, "Connected", "Device is connected", vocab.WoTDataTypeBool).
		SetAtType(vocab.PropNetConnection)
	prop.Description = "Whether the Binding has a connection to an ISY gateway"
	//
	prop = tdi.AddProperty(vocab.PropDeviceMake, "Manufacturer", "Device Manufacturer", vocab.WoTDataTypeString).
		SetAtType(vocab.PropDeviceMake)
	prop.Description = "Developer of the binding"

	// TODO: persist configuration
	//binding config
	prop = tdi.AddProperty(vocab.PropDevicePollinterval, "Poll Interval", "Poll for updates in seconds", vocab.WoTDataTypeInteger).
		SetAtType(vocab.PropDevicePollinterval)
	prop.Description = "Interval the binding polls the gateway for data value updates."
	prop.Unit = vocab.UnitSecond
	prop.ReadOnly = false
	//
	prop = tdi.AddPropertyAsString("loginName", "Login name", "ISY99x gateway login name.")
	prop.ReadOnly = false
	//
	prop = tdi.AddPropertyAsString("password", "Password", "ISY99x gateway password")
	prop.ReadOnly = false
	prop.WriteOnly = true
	//
	prop = tdi.AddProperty(vocab.PropNetAddress, "Network Address", "", vocab.WoTDataTypeString).
		SetAtType(vocab.PropNetAddress)
	prop.Description = "ISY99x gateway IP address; empty to auto discover."
	prop.ReadOnly = false

	// binding events
	tdi.AddEvent(vocab.PropNetConnection,
		"Connection changed", "Connection with ISY gateway has changed",
		&td.DataSchema{Type: vocab.WoTDataTypeBool},
	).SetAtType(vocab.PropNetConnection)

	// no binding actions
	return tdi
}

// Start the ISY99x protocol binding.
// Connection to the gateway will be made during the heartbeat.
// If no connection can be made the heartbeat will retry periodically until stopped.
//
// This publishes a TD for this binding, starts a background polling heartbeat.
func (svc *IsyBinding) Start(ag *messaging.Agent) (err error) {
	slog.Info("Starting Isy99x binding")
	svc.ag = ag
	svc.thingID = ag.GetClientID()
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	svc.prodMap, err = LoadProductMapCSV("")

	//// 'IsyThings' use the 'isy connection' to talk to the gateway
	svc.isyAPI = isy.NewIsyAPI()
	svc.IsyGW = NewIsyGateway(svc.prodMap, svc.onIsyEvent)
	_ = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
	svc.IsyGW.Init(svc.isyAPI)

	// subscribe to action and property write requests
	svc.ag.SetRequestHandler(svc.handleRequest)

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
				_ = svc.ag.PubEvent(svc.thingID, vocab.PropNetConnection, isConnected)
			}
			err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
			if err == nil {
				// re-establish the gateway device connection
				svc.IsyGW.Init(svc.isyAPI)
			}
			isConnected = svc.isyAPI.IsConnected()
			if isConnected {
				_ = svc.ag.PubEvent(svc.thingID, vocab.PropNetConnection, isConnected)
			}
		}
		svc.wasConnected = isConnected

		// publish node TDs and values
		if isConnected && tdCountDown <= 0 {
			err = svc.PublishTDs()
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
			err = svc.PublishAllThingValues(!forceRepublish)
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
