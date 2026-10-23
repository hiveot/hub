package service

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/bindings/ipnet/config"
	"github.com/hiveot/hub/lib/plugin"
)

type IPNetBinding struct {
	*thing.ExposedThing

	config *config.IPNetConfig

	// discovered devices
	devicesMap      map[string]*IPDeviceInfo
	stopHeartbeatFn func()
}

// ActionHandler handle action requests
// func (svc *IPNetBinding) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

// 	resp = req.CreateResponse(nil, fmt.Errorf("unknown action '%s'", req.Name))
// 	slog.Warn(resp.Error.String())
// 	return resp
// }

// Start the binding
func (svc *IPNetBinding) Start() {
	if svc.config.LogLevel != "" {
		utils.SetLogging(svc.config.LogLevel, "")
	}
	slog.Info("Starting the IpNet binding", "logLevel", svc.config.LogLevel)

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
		ExposedThing: thing.NewExposedThing(thingID, nil),
		config:       cfg,
		devicesMap:   make(map[string]*IPDeviceInfo),
	}
	return svc
}
