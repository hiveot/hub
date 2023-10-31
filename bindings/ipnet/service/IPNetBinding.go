package service

import (
	"fmt"
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/thing"
	"log/slog"
	"time"
)

type IPNetBinding struct {
	config *config.IPNetConfig
	// Hub connection
	hc *hubclient.HubClient
	// handler action subscription
	sub transports.ISubscription

	// discovered devices
	devicesMap      map[string]*IPDeviceInfo
	stopHeartbeatFn func()
}

// ActionHandler handle action requests
func (svc *IPNetBinding) ActionHandler(msg *thing.ThingValue) ([]byte, error) {
	return nil, fmt.Errorf("unknown action '%s'", msg.Name)
}

// Start the binding
func (svc *IPNetBinding) Start(hc *hubclient.HubClient) (err error) {
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	slog.Warn("Starting the IpNet binding", "logLevel", svc.config.LogLevel)

	svc.hc = hc

	// register the action handler
	svc.sub, err = svc.hc.SubActions("", svc.ActionHandler)

	// publish this binding's TD document
	td := svc.MakeBindingTD()
	err = svc.hc.PubTD(td)
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	}

	// start polling in the background
	svc.stopHeartbeatFn = svc.startHeartbeat()

	slog.Info("ipnet started")

	return err
}

// heartbeat polls the gateway device every X seconds and publishes updates
// This returns a stop function that ends the loop
func (svc *IPNetBinding) startHeartbeat() (stopFn func()) {
	interval := time.Duration(svc.config.PollInterval) * time.Second
	stopFn = plugin.StartHeartbeat(interval, func() {
		svc.Poll()
	})
	return stopFn
}

// Stop the binding
func (svc *IPNetBinding) Stop() {
	slog.Warn("Stopping the IPNet binding")
	if svc.sub != nil {
		_ = svc.sub.Unsubscribe()
		svc.sub = nil
	}
	if svc.stopHeartbeatFn != nil {
		svc.stopHeartbeatFn()
		svc.stopHeartbeatFn = nil
	}
}

// NewIpNetBinding creates a new binding instance
func NewIpNetBinding(cfg *config.IPNetConfig) *IPNetBinding {

	svc := &IPNetBinding{
		config:     cfg,
		devicesMap: make(map[string]*IPDeviceInfo),
	}

	return svc
}
