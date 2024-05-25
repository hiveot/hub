// Package service for basic ISY99x Insteon home automation hub access
// This implements common sensors and switches
package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
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
	hc     *hubclient.HubClient

	thingID string           // ID of the binding Thing
	isyAPI  *IsyAPI          // methods for communicating met ISY gateway device
	IsyGW   *IsyGatewayThing // ISY gateway access

	// is gateway currently reachable?
	gwReachable bool

	//// product identification map by {cat}.{subcat}
	prodMap map[string]InsteonProduct

	mu              sync.Mutex
	stopHeartbeatFn func()
}

//
//// GetValues returns the property values of the binding Thing
//func (svc *IsyBinding) GetValues(onlyChanges bool) map[string]string {
//	props := make(map[string]string)
//	props[vocab.VocabPollInterval] = fmt.Sprintf("%d", svc.config.PollInterval)
//	props[vocab.VocabGatewayAddress] = svc.config.IsyAddress
//	return props
//}

// HandleActionRequest passes the action request to the associated Thing.
func (svc *IsyBinding) handleActionRequest(tv *things.ThingMessage) (reply []byte, err error) {
	slog.Info("handleActionRequest",
		slog.String("thingID", tv.ThingID),
		slog.String("name", tv.Name),
		slog.String("senderID", tv.SenderID))

	if !svc.isyAPI.IsConnected() {
		return nil, fmt.Errorf("No connection with the gateway")
	}
	isyThing := svc.IsyGW.GetIsyThing(tv.ThingID)
	if isyThing == nil {
		err = fmt.Errorf("handleActionRequest: thing '%s' not found", tv.ThingID)
		slog.Warn(err.Error())
	}
	if isyThing != nil {
		err = isyThing.HandleActionRequest(tv)
	}
	return nil, err
}

// handleConfigRequest for handling binding, gateway and node configuration changes
func (svc *IsyBinding) handleConfigRequest(tv *things.ThingMessage) (err error) {

	slog.Info("handleConfigRequest",
		slog.String("thingID", tv.ThingID),
		slog.String("name", tv.Name),
		slog.String("senderID", tv.SenderID))

	// configuring the binding doesn't require a connection with the gateway
	if tv.ThingID == svc.thingID {
		err = svc.HandleBindingConfig(tv)
		return err
	}

	if !svc.isyAPI.IsConnected() {
		return fmt.Errorf("no connection with the gateway")
	}

	// pass request to the Thing
	isyThing := svc.IsyGW.GetIsyThing(tv.ThingID)
	if isyThing == nil {
		err = fmt.Errorf("handleActionRequest: thing '%s' not found", tv.ThingID)
		slog.Warn(err.Error())
	} else {
		err = isyThing.HandleConfigRequest(tv)
		//
		_ = svc.PublishValues(true)
		// re-submit the TD if the title changes
		if tv.Name == vocab.PropDeviceTitle {
			td := isyThing.GetTD()
			_ = svc.hc.PubTD(td)
		}
	}
	return err
}

// Start the ISY99x protocol binding.
// Connection to the gateway will be made during the heartbeat.
// If no connection can be made the heartbeat will retry periodically until stopped.
//
// This publishes a TD for this binding, starts a background polling heartbeat.
func (svc *IsyBinding) Start(hc *hubclient.HubClient) (err error) {
	slog.Warn("Starting Isy99x binding")
	svc.hc = hc
	svc.thingID = hc.ClientID()
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	svc.prodMap, err = LoadProductMapCSV("")

	//// 'IsyThings' use the 'isy connection' to talk to the gateway
	svc.isyAPI = NewIsyAPI()
	svc.IsyGW = NewIsyGateway(svc.prodMap)
	_ = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
	svc.IsyGW.Init(svc.isyAPI)

	// subscribe to action requests
	svc.hc.SetActionHandler(svc.handleActionRequest)
	svc.hc.SetConfigHandler(svc.handleConfigRequest)

	// last, start polling heartbeat
	svc.stopHeartbeatFn = svc.startHeartbeat()
	return err
}

// Stop the heartbeat and remove subscriptions
// This does not close the given hubclient connection.
func (svc *IsyBinding) Stop() {
	slog.Warn("Stopping the ISY99x Binding")
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
		config: cfg,
	}
	return &svc
}
