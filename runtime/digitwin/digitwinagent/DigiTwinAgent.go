package digitwinagent

import (
	"fmt"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/api/go/inbox"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
)

// DigiTwinAgentID is the connection ID of the digitwin agent used in providing its capabilities
const DigiTwinAgentID = "digitwin"

// DigiTwinAgent agent for access to the API of directory, inbox, or outbox as described
// in the digitwin TDDs.
// This is separate from processing the message ingress flow, which is handled by the service directly.
type DigiTwinAgent struct {
	// connection to the hub used by this agent
	ag hubclient.IHubClient
	// the service itself
	svc *service.DigitwinService
	// handler for directory requests
	directoryHandler api.MessageHandler
	// handler for inbox (action) requests
	inboxHandler api.MessageHandler
	// handler for outbox (events) requests
	outboxHandler api.MessageHandler
}

// HandleMessage dispatches requests to the service capabilities identified by their thingID
func (agent *DigiTwinAgent) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if msg.ThingID == directory.ServiceID {
		return agent.directoryHandler(msg)
	} else if msg.ThingID == inbox.ServiceID {
		return agent.inboxHandler(msg)
	} else if msg.ThingID == outbox.ServiceID {
		return agent.outboxHandler(msg)
	}

	stat.Failed(msg, fmt.Errorf("unknown digitwin service thingID '%s'", msg.ThingID))
	return stat
}

// StartDigiTwinAgent returns a new instance of the agent for the digitwin services.
// This uses the given connected transport for publish events and subscribing to actions
// ag is optional for subscribing to Things. Set to nil use HandleMessage directly.
// the transport must be closed by the called after use.
func StartDigiTwinAgent(svc *service.DigitwinService, cl hubclient.IHubClient) (*DigiTwinAgent, error) {
	var err error
	agent := DigiTwinAgent{ag: cl, svc: svc}
	cl.SetActionHandler(agent.HandleMessage)
	cl.SetEventHandler(agent.HandleMessage)
	// each of the digitwin services implements an API that can be accessed through actions.
	agent.directoryHandler = directory.NewActionHandler(agent.svc.Directory)
	agent.inboxHandler = inbox.NewActionHandler(agent.svc.Inbox)
	agent.outboxHandler = outbox.NewActionHandler(agent.svc.Outbox)
	// agents do not need to subscribe to receive actions directed at them as authenticating is sufficient
	return &agent, err
}
