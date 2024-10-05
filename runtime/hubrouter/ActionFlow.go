// Package service with digital twin action flow handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	digitwinapi "github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/teris-io/shortid"
	"log/slog"
)

// HandleActionFlow handles the request to invoke an action on a Thing.
// This tracks the progress in the digital twin store and passes the action
// request to the thing agent.
func (svc *HubRouter) HandleActionFlow(
	consumerID string, dThingID string, actionName string, input any, reqID string) (
	status string, output any, messageID string, err error) {
	slog.Info("HandleActionFlow")

	// assign a messageID if none given
	messageID = reqID
	if messageID == "" {
		messageID = "action-" + shortid.MustGenerate()
	}
	err = svc.dtwStore.UpdateActionStart(
		consumerID, dThingID, actionName, input, messageID)
	if err != nil {
		slog.Warn("HandleActionFlow failed.", "err", err.Error())
		return digitwin.StatusFailed, nil, messageID, err
	}

	// Forward the action to the agent
	agentID, thingID := tdd.SplitDigiTwinThingID(dThingID)
	// TODO: Consider injecting the internal services and move these in the transport
	switch agentID {
	case digitwinapi.DirectoryAgentID:
		status, output, err = svc.dirAgent.HandleAction(consumerID, dThingID, actionName, input, messageID)
	case authn.AdminAgentID:
		status, output, err = svc.authnAgent.HandleAction(consumerID, dThingID, actionName, input, messageID)
	case authz.AdminAgentID:
		status, output, err = svc.authzAgent.HandleAction(consumerID, dThingID, actionName, input, messageID)
	default:
		// forward to external services/things
		if svc.tb != nil {
			status, output, err = svc.tb.InvokeAction(
				agentID, thingID, actionName, input, messageID)
		} else {
			err = fmt.Errorf("HandleActionFlow: Agent not reachable. Ignored")
			status = digitwin.StatusFailed
		}
	}
	// update action delivery status
	svc.dtwStore.UpdateActionProgress(agentID, thingID, actionName, status, output)

	return status, output, messageID, err
}
