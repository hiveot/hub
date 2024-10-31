// Package service with digital twin action flow handling functions
package hubrouter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
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
	Progress    string    // current progress of the action:
	Updated     time.Time // timestamp to handle expiry
	CLCID       string    // Client connection-id (senderID-cid) from sender to publish progress update
}

// HandleInvokeAction handles the request to invoke an action on a Thing.
// This authorizes the request, tracks the progress in the digital twin store
// and passes the action request to the thing agent.
//
// The digitwin acts as a proxy for the action and forwards the request to the agent
// identified in the dThingID. This can lead to one of these flows:
// 1: The agent is offline => return an error and status Failed
// 2: The agent is online and rejects the request => return an error and status Failed
// 3: The agent is online and accepts the request but has no result yet
// => return a delivery status of 'pending' and no output
// when result is available:
// => update the corresponding property; or send an event with the output
// 4: The agent is online and accepts the request and has a result
// => return a delivery status of 'completed' with output
//
// Pre-requisite: The transport protocol must wait for a response from the agent,
// even thought it uses a uni-directional channel for sending the request.
// SSE, WS, MQTT bindings must use a correlation-id to match request-response messages.
// this is not well-defined in the WoT specs and up to the protocol binding implementation.
func (svc *HubRouter) HandleInvokeAction(
	senderID string, dThingID string, actionName string, input any, reqID string, clcid string) (
	status string, output any, messageID string, err error) {

	// check if consumer or agent has the right permissions
	hasPerm := svc.authzAgent.HasPermission(senderID, vocab.MessageTypeAction, dThingID, true)
	if !hasPerm {
		err = fmt.Errorf("Client '%s' does not have permission to invoke action '%s' on Thing '%s'",
			senderID, actionName, dThingID)
		slog.Warn("HandleInvokeAction: " + err.Error())
		return vocab.ProgressStatusFailed, nil, messageID, err
	}

	// assign a messageID if none given
	messageID = reqID
	if messageID == "" {
		messageID = "action-" + shortid.MustGenerate()
	}

	// Forward the action to the built-in services
	agentID, thingID := tdd.SplitDigiTwinThingID(dThingID)
	// TODO: Consider injecting the internal services instead of having direct dependencies
	switch agentID {
	case digitwin.DirectoryAgentID:
		status, output, err = svc.dtwAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	case authn.AdminAgentID:
		status, output, err = svc.authnAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	case authz.AdminAgentID:
		status, output, err = svc.authzAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	case api.DigitwinServiceID:
		status, output, err = svc.dtwAgent.HandleAction(senderID, dThingID, actionName, input, messageID)
	default:
		// Forward the action to external agents
		slog.Info("HandleInvokeAction (to agent)",
			slog.String("dThingID", dThingID),
			slog.String("actionName", actionName),
			slog.String("messageID", reqID),
			slog.String("senderID", senderID),
		)

		// FIXME: Don't store RPC stateless actions for services. Only track stateful actions results.
		//  Is there is a use-case for reading the last known action input?
		//  Is there is a use-case for reading the last known action output or is
		//  it better to link to a property value as per Ben Francis recommendation?
		// Stateful actions do have a representation using properties although the
		// schema may be different from the output.
		err = svc.dtwStore.UpdateActionStart(dThingID, actionName, input, messageID, senderID)

		err = nil

		// store a new action progress by message ID to support sending replies to the sender
		actionRecord := &ActionFlowRecord{
			MessageType: vocab.MessageTypeAction,
			AgentID:     agentID,
			ThingID:     thingID,
			MessageID:   messageID,
			Name:        actionName,
			SenderID:    senderID,
			Progress:    status,
			Updated:     time.Now(),
			CLCID:       clcid,
		}
		svc.mux.Lock()
		svc.activeCache[messageID] = actionRecord
		svc.mux.Unlock()

		// forward to external services/things and return its response
		found := false

		// Note: agents should only have a single instance
		c := svc.cm.GetConnectionByClientID(agentID)
		if c != nil {
			found = true
			status, output, err = c.InvokeAction(thingID, actionName, input, messageID, senderID)
		}

		// Update the action status
		svc.mux.Lock()
		actionRecord.Progress = status
		svc.mux.Unlock()

		if !found {
			err = fmt.Errorf("HandleInvokeAction: Agent '%s' not reachable. Ignored", agentID)
			status = vocab.ProgressStatusFailed
		}
		// update action delivery status
		_, _ = svc.dtwStore.UpdateActionProgress(
			agentID, thingID, actionName, status, output)

		// remove the action when completed
		if status == vocab.ProgressStatusCompleted {
			svc.mux.Lock()
			delete(svc.activeCache, messageID)
			svc.mux.Unlock()

			slog.Info("HandleInvokeAction - finished",
				slog.String("dThingID", dThingID),
				slog.String("actionName", actionName),
				slog.String("messageID", reqID),
				slog.String("status", status),
			)
		} else if status == vocab.ProgressStatusFailed {
			svc.mux.Lock()
			delete(svc.activeCache, messageID)
			svc.mux.Unlock()

			errText := ""
			if err != nil {
				errText = err.Error()
			}
			slog.Warn("HandleInvokeAction - failed",
				slog.String("dThingID", dThingID),
				slog.String("actionName", actionName),
				slog.String("messageID", reqID),
				slog.String("err", errText))
		}
	}
	return status, output, messageID, err
}

// HandleInvokeActionProgress agent sends an action progress update.
// The message payload contains a ActionProgress object
//
// This:
// 1. Validates the request is still ongoing
// 2: Updates the status of the current digital twin action record.
// 3: Forwards the update to the sender, if this is an active request
// 4: Updates the status of the active request cache
//
// If the message is no longer in the active cache then it is ignored.
func (svc *HubRouter) HandleInvokeActionProgress(agentID string, data any) (err error) {
	var stat hubclient.ActionProgress
	err = utils.Decode(data, &stat)
	if err != nil {
		return err
	}

	// 1: Validate this is an active action
	svc.mux.Lock()
	actionRecord, found := svc.activeCache[stat.MessageID]
	svc.mux.Unlock()
	if !found {
		err = fmt.Errorf(
			"HandleInvokeActionProgress: Message '%s' from agent '%s' not in action cache. It is ignored",
			stat.MessageID, agentID)
		slog.Warn(err.Error())
		return err
	}
	svc.mux.Lock()
	arAgentID := actionRecord.AgentID
	thingID := actionRecord.ThingID
	actionName := actionRecord.Name
	clcid := actionRecord.CLCID
	senderID := actionRecord.SenderID
	svc.mux.Unlock()
	// Update the thingID to notify the sender with progress on the digital twin thing ID
	stat.ThingID = tdd.MakeDigiTwinThingID(agentID, thingID)
	stat.Name = actionName
	// the sender (agents) must be the thing agent
	if agentID != arAgentID {
		err = fmt.Errorf(
			"HandleInvokeActionProgress: progress update '%s' of thing '%s' does not come from agent '%s' but from '%s'. Update ignored.",
			stat.MessageID, thingID, arAgentID, agentID)
		slog.Warn(err.Error(), "agentID", agentID)
		return err
	}

	// 2: Update the action status in the digital twin action record and log errors
	//   (for use with query actions)
	_, _ = svc.dtwStore.UpdateActionProgress(agentID, thingID, actionName, stat.Progress, stat.Reply)

	if stat.Error != "" {
		slog.Warn("HandleInvokeActionProgress - with error",
			slog.String("AgentID", agentID),
			slog.String("ThingID", thingID),
			slog.String("Name", actionName),
			slog.String("Progress", stat.Progress),
			slog.String("MessageID", stat.MessageID),
			slog.String("Error", stat.Error),
		)
	} else if stat.Progress == vocab.ProgressStatusCompleted {
		slog.Info("HandleInvokeActionProgress - completed",
			slog.String("ThingID", thingID),
			slog.String("Name", actionName),
			slog.String("MessageID", stat.MessageID),
		)
	} else {
		slog.Info("HandleInvokeActionProgress - progress",
			slog.String("ThingID", thingID),
			slog.String("Name", actionName),
			slog.String("MessageID", stat.MessageID),
			slog.String("Progress", stat.Progress),
		)
	}

	// 3: Forward the progress update to the original sender
	c := svc.cm.GetConnectionByCLCID(clcid)
	if c != nil {
		err = c.PublishActionProgress(stat, agentID)
	} else {
		err = fmt.Errorf("client connection-id (clcid) '%s' not found for client '%s'", clcid, senderID)
		// try workaround

	}

	if err != nil {
		slog.Warn("HandleInvokeActionProgress. Forwarding to sender failed",
			slog.String("senderID", senderID),
			slog.String("thingID", thingID),
			slog.String("clcid", clcid),
			slog.String("err", err.Error()),
			slog.String("MessageID", stat.MessageID),
		)
		err = nil
	}

	// 4: Update the active action cache and remove the action when completed or failed
	svc.mux.Lock()
	defer svc.mux.Unlock()
	actionRecord.Progress = stat.Progress

	if stat.Progress == vocab.ProgressStatusCompleted || stat.Progress == vocab.ProgressStatusFailed {
		delete(svc.activeCache, stat.MessageID)
	}
	return nil
}

// HandleQueryAction returns the action status
func (svc *HubRouter) HandleQueryAction(clientID string, dThingID string, name string) (reply any, err error) {
	reply, err = svc.dtwService.ValuesSvc.QueryAction(clientID,
		digitwin.ValuesQueryActionArgs{ThingID: dThingID, Name: name})
	return reply, err
}

// HandleQueryAllAction returns the status of all actions of a Thing
func (svc *HubRouter) HandleQueryAllActions(clientID string, dThingID string) (reply any, err error) {
	reply, err = svc.dtwService.ValuesSvc.QueryAllActions(clientID, dThingID)
	return reply, err
}
