// Package service with digital twin action flow handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	digitwinapi "github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// ActionFlowRecord holds in progress actions and property writes
type ActionFlowRecord struct {
	MessageType string    // MessageTypeAction or MessageTypeProperty
	MessageID   string    // MessageID of the ongoing action
	AgentID     string    // Agent that is handling the action request
	ThingID     string    // thingID as provided by the agent the action is for
	Name        string    // name of the action as described in the TD
	SenderID    string    // action sender that will receive progress messages
	Progress    string    // current progress of the action
	Updated     time.Time // timestamp to handle expiry
}

// HandleActionFlow handles the request to invoke an action on a Thing.
// This authorizes the request, tracks the progress in the digital twin store
// and passes the action request to the thing agent.
//
// The digitwin acts as a proxy for the action and forwards the request to the
// agent. This can lead to one of these flows:
// 1: The agent is offline => return an error;
// 2: The agent is online and rejects the request => return an error
// 3: The agent is online and accepts the request but has no result yet
// => return a delivery status of 'applied' and no output
// when result is available:
// => update the corresponding property; or send an event with the output
// 4: The agent is online and accepts the request and has a result
// => return a delivery status of 'completed' with output
//
// Pre-requisite: The transport protocol must wait for a response from the agent,
// even thought it uses a uni-directional channel for sending the request.
// SSE, WS, MQTT bindings must use a correlation-id to match request-response messages.
// this is not well-defined in the WoT specs and up to the protocol binding implementation.
func (svc *HubRouter) HandleActionFlow(
	senderID string, dThingID string, actionName string, input any, reqID string) (
	status string, output any, messageID string, err error) {
	slog.Info("HandleActionFlow",
		slog.String("senderID", senderID),
		slog.String("dThingID", dThingID),
		slog.String("actionName", actionName),
		slog.String("messageID", reqID),
	)

	// check if consumer or agent has the right permissions
	hasPerm := svc.authzAgent.HasPermission(senderID, vocab.MessageTypeAction, dThingID, true)
	if !hasPerm {
		err = fmt.Errorf("Client '%s' does not have permission to invoke action '%s' on Thing '%s'",
			senderID, actionName, dThingID)
		slog.Warn("HandleActionFlow: " + err.Error())
		return vocab.ProgressStatusFailed, nil, messageID, err
	}

	// assign a messageID if none given
	messageID = reqID
	if messageID == "" {
		messageID = "action-" + shortid.MustGenerate()
	}

	// Forward the action to the agent
	agentID, thingID := tdd.SplitDigiTwinThingID(dThingID)
	// TODO: Consider injecting the internal services instead of having direct dependencies
	switch agentID {
	case digitwinapi.DirectoryAgentID:
		status, output, err = svc.dtwAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	case authn.AdminAgentID:
		status, output, err = svc.authnAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	case authz.AdminAgentID:
		status, output, err = svc.authzAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	case api.DigitwinServiceID:
		status, output, err = svc.dtwAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	default:
		// FIXME: RPC stateless services don't need a digital twin to be callable
		// Action progress is stored in the digital twin. This is intended for
		// stateful actions. RPC actions are not stored unless a digitwin exists.
		// Intended for storing the last-known action progress.
		// TODO: is storing actions in the store at all useful?
		//  not if a property exists that represents its state.
		//
		// Is there is a use-case for reading the last known action input?
		// maybe for presenting action-in-progress? or is it better to use a property
		// for this as per Ben Francis recommendation? yeah, looks like it.
		//
		err = svc.dtwStore.UpdateActionStart(
			dThingID, actionName, input, messageID, senderID)
		// if no digital twin exists for the thing or service, its progress
		// will not be stored but still be tracked.
		err = nil

		// forward to external services/things and return its response
		if svc.tb != nil {
			status, output, err = svc.tb.InvokeAction(
				agentID, thingID, actionName, input, messageID, senderID)
		} else {
			err = fmt.Errorf("HandleActionFlow: Agent not reachable. Ignored")
			status = vocab.ProgressStatusFailed
		}
		// update action delivery status
		_, _ = svc.dtwStore.UpdateActionProgress(
			agentID, thingID, actionName, status, output)

		// possible response:
		// * on failure: DeliveryStatus error response message; no output
		// * on success: output as per TD
		// * not completed: DeliveryStatus progress response; no output
		//
		//
		if status != vocab.ProgressStatusCompleted && status != vocab.ProgressStatusFailed {
			// store incomplete actions by message ID to support sending updates to the sender
			svc.activeCache[messageID] = &ActionFlowRecord{
				MessageType: vocab.MessageTypeAction,
				AgentID:     agentID,
				ThingID:     thingID,
				MessageID:   messageID,
				Name:        actionName,
				SenderID:    senderID,
				Progress:    status,
				Updated:     time.Now(),
			}
		}
	}

	return status, output, messageID, err
}

// HandleProgressUpdate agent sends a request progress update.
// The message payload contains a DeliveryStatus object
//
// This:
// 1. Validates the request is still ongoing
// 2: Updates the status of the current digital twin action record.
// 3: Forwards the update to the sender, if this is an active request
// 4: Updates the status of the active request cache
//
// If the message is no longer in the active cache then it is ignored.
func (svc *HubRouter) HandleProgressUpdate(agentID string, stat hubclient.DeliveryStatus) error {

	// 1: Validate this is an active action
	svc.mux.Lock()
	actionRecord, found := svc.activeCache[stat.MessageID]
	svc.mux.Unlock()
	if !found {
		err := fmt.Errorf(
			"HandleProgressUpdate: Message '%s' from agent '%s' not in action cache. It is ignored",
			stat.MessageID, agentID)
		slog.Warn(err.Error())
		return err
	}
	thingID := actionRecord.ThingID
	actionName := actionRecord.Name
	// the sender (agents) must be the thing agent
	if agentID != actionRecord.AgentID {
		err := fmt.Errorf(
			"HandleProgressUpdate: status update '%s' of thing '%s' does not come from agent '%s' but from '%s'. Update ignored.",
			stat.MessageID, thingID, actionRecord.AgentID, agentID)
		slog.Warn(err.Error(), "agentID", agentID)
		return err
	}

	slog.Info("HandleProgressUpdate ",
		slog.String("AgentID", agentID),
		slog.String("ThingID", thingID),
		slog.String("Name", actionName),
		slog.String("Progress", stat.Progress),
		slog.String("MessageID", stat.MessageID),
	)

	// 2: Update the action status in the digital twin action record.
	//   (for use with query actions)
	_, _ = svc.dtwStore.UpdateActionProgress(agentID, thingID, actionName, stat.Progress, stat.Reply)

	// 3: Forward the progress update to the original sender
	found, err := svc.tb.PublishProgressUpdate(actionRecord.SenderID, stat, agentID)

	if !found {
		slog.Warn("HandleProgressUpdate. Forwarding to sender failed",
			slog.String("senderID", actionRecord.SenderID),
			slog.String("thingID", thingID),
			slog.String("err", err.Error()),
			slog.String("MessageID", stat.MessageID),
		)
		err = nil
	}

	// 4: Update the active action cache and remove the action when completed or failed
	actionRecord.Progress = stat.Progress
	if stat.Progress == vocab.ProgressStatusCompleted || stat.Progress == vocab.ProgressStatusFailed {
		svc.mux.Lock()
		delete(svc.activeCache, stat.MessageID)
		svc.mux.Unlock()
	}
	return nil
}
