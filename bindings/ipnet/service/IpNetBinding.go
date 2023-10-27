package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
)

type IpNetBinding struct {
	// Hub connection
	hc hubclient.IHubClient
	// handler action subscription
	sub transports.ISubscription
}

// ActionHandler handle action requests
func (svc *IpNetBinding) ActionHandler(msg *hubclient.RequestMessage) error {
	return fmt.Errorf("unknown action '%s'", msg.Name)
}

// Start the binding
func (svc *IpNetBinding) Start() (err error) {
	slog.Info("Starting the IpNet binding")

	// register the action handler
	svc.sub, err = svc.hc.SubActions("", svc.ActionHandler)
	return err
}

// Stop the binding
func (svc *IpNetBinding) Stop() {
	slog.Info("Stopping the binding service")
	if svc.sub != nil {
		svc.sub.Unsubscribe()
		svc.sub = nil
	}
}

// NewIpNetBinding creates a new binding instance
func NewIpNetBinding(hc hubclient.IHubClient) *IpNetBinding {

	svc := &IpNetBinding{
		hc: hc,
	}

	return svc
}
