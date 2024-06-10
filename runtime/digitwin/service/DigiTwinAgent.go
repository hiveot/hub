package service

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
)

// DigiTwinAgent is the agent for access to the digitwin directory, inbox, and outbox.
// This invokes the generated handlers for each of the services. These handlers are
// generated using the genapi tool. (see cmd/genapi)
type DigiTwinAgent struct {
	// connection to the hub used by this agent
	ag hubclient.IHubClient
	// the service itself
	svc *DigitwinService
	// handler for directory requests
	directoryHandler hubclient.MessageHandler
	// handler for inbox (action) requests
	inboxHandler hubclient.MessageHandler
	// handler for outbox (events) requests
	outboxHandler hubclient.MessageHandler
}

// HandleEvent dispatches events to digitwin sub-services
func (agent *DigiTwinAgent) HandleEvent(msg *things.ThingMessage) (err error) {
	stat := agent.HandleMessage(msg)
	if stat.Error != "" {
		err = errors.New(stat.Error)
	}
	return err
}

// HandleMessage dispatches requests to the service capabilities identified by their thingID
func (agent *DigiTwinAgent) HandleMessage(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	if msg.ThingID == digitwin.DirectoryServiceID {
		return agent.directoryHandler(msg)
	} else if msg.ThingID == digitwin.InboxServiceID {
		return agent.inboxHandler(msg)
	} else if msg.ThingID == digitwin.OutboxServiceID {
		return agent.outboxHandler(msg)
	}

	stat.Failed(msg, fmt.Errorf("unknown digitwin service thingID '%s'", msg.ThingID))
	return stat
}

// StartDigiTwinAgent returns a new instance of the agent for the digitwin services.
// This uses the given connected transport for publish events and subscribing to actions
// ag is optional for subscribing to Things. Set to nil use HandleMessage directly.
// the transport must be closed by the called after use.
func StartDigiTwinAgent(svc *DigitwinService, hc hubclient.IHubClient) (*DigiTwinAgent, error) {
	var err error
	agent := DigiTwinAgent{ag: hc, svc: svc}
	hc.SetActionHandler(agent.HandleMessage)
	hc.SetEventHandler(agent.HandleEvent)
	// each of the digitwin services implements an API that can be accessed through actions.
	agent.directoryHandler = digitwin.NewDirectoryHandler(agent.svc.Directory)
	agent.inboxHandler = digitwin.NewInboxHandler(agent.svc.Inbox)
	agent.outboxHandler = digitwin.NewOutboxHandler(agent.svc.Outbox)
	// agents do not need to subscribe to receive actions directed at them as authenticating is sufficient

	// set permissions for using these services
	err = authz.UserSetPermissions(hc, authz.ThingPermissions{
		AgentID: hc.ClientID(),
		ThingID: digitwin.DirectoryServiceID,
		Deny:    []string{authn.ClientRoleNone},
	})
	err = authz.UserSetPermissions(hc, authz.ThingPermissions{
		AgentID: hc.ClientID(),
		ThingID: digitwin.InboxServiceID,
		Deny:    []string{authn.ClientRoleNone, authn.ClientRoleViewer},
	})
	err = authz.UserSetPermissions(hc, authz.ThingPermissions{
		AgentID: hc.ClientID(),
		ThingID: digitwin.OutboxServiceID,
		Deny:    []string{authn.ClientRoleNone},
	})

	return &agent, err
}
