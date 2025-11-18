package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/agent"
	"github.com/hiveot/hivekit/go/logging"
	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/lib/plugin"
)

type IPNetBinding struct {
	config *config.IPNetConfig
	// Hub connection
	ag *agent.Agent

	// discovered devices
	devicesMap      map[string]*IPDeviceInfo
	stopHeartbeatFn func()
}

// ActionHandler handle action requests
func (svc *IPNetBinding) ActionHandler(req *messaging.RequestMessage,
	_ messaging.IConnection) (resp *messaging.ResponseMessage) {

	resp = req.CreateResponse(nil, fmt.Errorf("unknown action '%s'", req.Name))
	slog.Warn(resp.Error.String())
	return resp
}

// Start the binding
func (svc *IPNetBinding) Start(ag *agent.Agent) (err error) {
	if svc.config.LogLevel != "" {
		logging.SetLogging(svc.config.LogLevel, "")
	}
	slog.Info("Starting the IpNet binding", "logLevel", svc.config.LogLevel)

	svc.ag = ag

	// register the action handler
	svc.ag.SetRequestHandler(svc.ActionHandler)

	// publish this binding's TD document
	err = svc.PubBindingTD()
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	} else {
		svc.PubBindingProps()
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
	slog.Info("Stopping the IPNet binding")
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
