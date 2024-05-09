package digitwinagent

import (
	"fmt"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/api/go/inbox"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/lib/hubclient/transports"
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
	// connection used by this agent
	tp  transports.IHubTransport
	svc *service.DigitwinService

	directoryHandler api.MessageHandler
	inboxHandler     api.MessageHandler
	outboxHandler    api.MessageHandler
}

// HandleMessage dispatches requests to the service capabilities identified by their thingID
func (agent *DigiTwinAgent) HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	if msg.ThingID == directory.ThingID {
		return agent.directoryHandler(msg)
	} else if msg.ThingID == inbox.ThingID {
		return agent.inboxHandler(msg)
	} else if msg.ThingID == outbox.ThingID {
		return agent.outboxHandler(msg)
	}
	stat.Error = fmt.Sprintf("unknown digitwin service capability '%s'", msg.ThingID)
	stat.Status = api.DeliveryFailed
	return stat
}

// StartDigiTwinAgent returns a new instance of the agent for the digitwin services.
// This uses the given connected transport for publish events and subscribing to actions
// tp is optional for subscribing to Things. Set to nil use HandleMessage directly.
// the transport must be closed by the called after use.
func StartDigiTwinAgent(svc *service.DigitwinService, tp transports.IHubTransport) (*DigiTwinAgent, error) {
	var err error
	agent := DigiTwinAgent{tp: tp, svc: svc}
	// each of the digitwin services implements an API that can be accessed through actions.
	agent.directoryHandler = directory.NewActionHandler(agent.svc.Directory)
	agent.inboxHandler = inbox.NewActionHandler(agent.svc.Inbox)
	agent.outboxHandler = outbox.NewActionHandler(agent.svc.Outbox)
	// agents do not need to subscribe to receive actions directed at them
	// authenticating as the agent is sufficient
	//if tp != nil {
	//	err = agent.tp.Subscribe(directory.ThingID)
	//	if err == nil {
	//		err = agent.tp.Subscribe(inbox.ThingID)
	//	}
	//	if err == nil {
	//		err = agent.tp.Subscribe(outbox.ThingID)
	//	}
	//}
	return &agent, err
}
