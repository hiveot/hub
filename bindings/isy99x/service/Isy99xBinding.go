// Package internal for basic ISY99x Insteon home automation hub access
// This implements common sensors and switches
package service

import (
	"github.com/hiveot/hub/bindings/isy99x/config"
	"github.com/hiveot/hub/bindings/isy99x/service/isyapi"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
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
	IsyAPI *isyapi.IsyAPI // ISY gateway access

	// is gateway currently reachable?
	gwReachable bool

	mu              sync.Mutex
	actionSub       transports.ISubscription
	stopHeartbeatFn func()

	// PrevValues is a map of [thingID/propName] to device values,
	prevValues map[string]string
}

// Start the ISY99x protocol binding
// This publishes a TD for this binding, starts a background polling heartbeat.
func (svc *IsyBinding) Start(hc *hubclient.HubClient) (err error) {
	slog.Warn("Starting Isy99x binding")
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	svc.hc = hc
	// Create the adapter for the ISY99x gateway
	svc.IsyAPI = isyapi.NewIsyAPI(
		svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)

	// subscribe to action requests
	svc.actionSub, err = svc.hc.SubActions("", svc.handleActionRequest)
	if err != nil {
		return err
	}

	// publish this binding's TD document
	td := svc.MakeBindingTD()
	err = svc.hc.PubTD(td)
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	}

	// TODO: subscribe to ISY events instead of polling

	// last, start polling heartbeat
	svc.stopHeartbeatFn = svc.startHeartbeat()
	return err
}

// heartbeat polls the gateway device every X seconds and publishes updates
// This returns a stop function that ends the loop
func (svc *IsyBinding) startHeartbeat() (stopFn func()) {

	var tdCountDown = 0
	var pollCountDown = 0
	var err error

	stopFn = plugin.StartHeartbeat(time.Second, func() {
		onlyChanges := true
		// publish node TDs and values
		tdCountDown--
		if tdCountDown <= 0 {
			err = svc.PollGatewayNodes()
			tdCountDown = svc.config.TDInterval
			// after publishing the TD, republish all values
			onlyChanges = false
		}
		// publish changes to sensor/actuator values
		pollCountDown--
		if pollCountDown <= 0 {
			err = svc.PollValues(onlyChanges)
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
	slog.Warn("Stopping the ISY99x Binding")
	//svc.isRunning.Store(false)
	if svc.actionSub != nil {
		_ = svc.actionSub.Unsubscribe()
		svc.actionSub = nil
	}
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
		prevValues: make(map[string]string),
	}
	return &svc
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
//func Run() error {
//	appConfig := &IsyBindingConfig{ClientID: appID}
//	hc := hubclient.NewHubClient("", caCert, core)
//	err := hc.ConnectWithTokenFile(keysDir)
//	if err == nil {
//		binding := NewIsyBinding(appConfig, hc)
//		err = binding.Start()
//
//		if err == nil {
//			utils.WaitForSignal(context.Background())
//			binding.Stop()
//			return err
//		}
//	}
//	return err
//}
