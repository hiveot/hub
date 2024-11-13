// Package service with digital twin action flow handling functions
package router

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
	"log/slog"
	"time"
)

// ActionFlowRecord holds in progress actions
type ActionFlowRecord struct {
	Operation string    // The operation being tracked, invoke action or write property
	RequestID string    // RequestID of the ongoing action
	AgentID   string    // Agent that is handling the action request
	ThingID   string    // thingID as provided by the agent the action is for
	Name      string    // name of the action as described in the TD
	SenderID  string    // action sender that will receive progress messages
	Progress  string    // current progress of the action:
	Updated   time.Time // timestamp to handle expiry
	ReplyTo   string    // Action reply address. Typically the sender's connection-id
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
func (svc *DigitwinRouter) HandleInvokeAction(
	msg *hubclient.ThingMessage, replyTo string) (stat hubclient.RequestStatus) {
	//status string, output any, requestID string, err error) {

	// check if consumer or agent has the right permissions
	hasPerm := svc.hasPermission(msg.SenderID, msg.Operation, msg.ThingID)
	if !hasPerm {
		err := fmt.Errorf("Client '%s' does not have permission to invoke action '%s' on Thing '%s'",
			msg.SenderID, msg.Name, msg.ThingID)
		stat.Failed(msg, err)
		return stat
	}

	// Forward the action to the built-in services
	agentID, thingID := tdd.SplitDigiTwinThingID(msg.ThingID)
	// TODO: Consider injecting the internal services instead of having direct dependencies
	switch agentID {
	case digitwin.DirectoryAgentID:
		stat = svc.digitwinAction(msg)
	case authn.AdminAgentID:
		stat = svc.authnAction(msg)
	case authz.AdminAgentID:
		stat = svc.authzAction(msg)
	case api.DigitwinServiceID:
		stat = svc.digitwinAction(msg)
	default:
		// Forward the action to external agents
		slog.Info("HandleInvokeAction (to agent)",
			slog.String("dThingID", msg.ThingID),
			slog.String("actionName", msg.Name),
			slog.String("requestID", msg.RequestID),
			slog.String("senderID", msg.SenderID),
		)

		// Store the action progress to be able to respond to queryAction.
		// FIXME: don't store 'safe' actions as their results are meaningless in queryAction
		// TODO: Maybe this shouldn't be stored at all and use properties instead.
		_ = svc.dtwStore.UpdateActionStart(
			msg.ThingID, msg.Name, msg.Data, msg.RequestID, msg.SenderID)

		// store a new action progress by message ID to support sending replies to the sender
		actionRecord := ActionFlowRecord{
			Operation: msg.Operation,
			AgentID:   agentID,
			ThingID:   thingID,
			RequestID: msg.RequestID,
			Name:      msg.Name,
			SenderID:  msg.SenderID,
			Progress:  vocab.RequestPending,
			Updated:   time.Now(),
			ReplyTo:   replyTo,
		}
		svc.mux.Lock()
		svc.activeCache[msg.RequestID] = actionRecord
		svc.mux.Unlock()

		// forward to external services/things and return its response
		found := false

		// Note: agents should only have a single instance
		c := svc.cm.GetConnectionByClientID(agentID)
		if c != nil {
			found = true
			status, output, err := c.InvokeAction(thingID, msg.Name, msg.Data, msg.RequestID, msg.SenderID)
			stat.Delivered(msg)
			stat.Progress = status
			stat.Output = output
			if err != nil {
				stat.Error = err.Error()
			}
		}

		// Update the action status
		svc.mux.Lock()
		actionRecord.Progress = stat.Progress
		svc.activeCache[msg.RequestID] = actionRecord
		svc.mux.Unlock()

		if !found {
			err := fmt.Errorf("HandleInvokeAction: Agent '%s' not reachable. Ignored", agentID)
			stat.Failed(msg, err)
		}
		// update action status
		_, _ = svc.dtwStore.UpdateRequestProgress(
			agentID, thingID, msg.Name, stat.Progress, stat.Output)

		// remove the action when completed
		if stat.Progress == vocab.RequestCompleted {
			svc.mux.Lock()
			delete(svc.activeCache, msg.RequestID)
			svc.mux.Unlock()

			slog.Info("HandleInvokeAction - finished",
				slog.String("dThingID", msg.ThingID),
				slog.String("actionName", msg.Name),
				slog.String("requestID", msg.RequestID),
			)
		} else if stat.Progress == vocab.RequestFailed {
			svc.mux.Lock()
			delete(svc.activeCache, msg.RequestID)
			svc.mux.Unlock()

			slog.Warn("HandleInvokeAction - failed",
				slog.String("dThingID", msg.ThingID),
				slog.String("actionName", msg.Name),
				slog.String("requestID", msg.RequestID),
				slog.String("err", stat.Error))
		}
	}
	return stat
}

// HandlePublishRequestProgress agent sends an action progress update.
// The message payload contains a RequestStatus object
//
// This:
// 1. Validates the request is still ongoing
// 2: Updates the status of the current digital twin action record.
// 3: Forwards the update to the sender, if this is an active request
// 4: Updates the status of the active request cache
//
// If the message is no longer in the active cache then it is ignored.
func (svc *DigitwinRouter) HandlePublishRequestProgress(msg *hubclient.ThingMessage) {

	var stat hubclient.RequestStatus
	var err error

	agentID := msg.SenderID
	err = utils.DecodeAsObject(msg.Data, &stat)
	if err != nil {
		slog.Warn("HandlePublishRequestProgress. Invalid payload", "err", err.Error())
		return
	}

	// 1: Validate this is an active action
	svc.mux.Lock()
	actionRecord, found := svc.activeCache[stat.RequestID]
	svc.mux.Unlock()
	if !found {
		err = fmt.Errorf(
			"HandlePublishRequestProgress: Message '%s' from agent '%s' not in action cache. It is ignored",
			stat.RequestID, agentID)
		slog.Warn(err.Error())
		return
	}
	arAgentID := actionRecord.AgentID // this should match the msg.SenderID
	if arAgentID != agentID {
		slog.Error("HandlePublishRequestProgress. AgentID's don't match",
			"senderID", agentID, "requestRecord agent", arAgentID)
		return
	}
	thingID := actionRecord.ThingID
	actionName := actionRecord.Name
	replyTo := actionRecord.ReplyTo
	senderID := actionRecord.SenderID

	// Update the thingID to notify the sender with progress on the digital twin thing ID
	stat.ThingID = tdd.MakeDigiTwinThingID(agentID, thingID)
	stat.Name = actionName
	// the sender (agents) must be the thing agent
	if agentID != arAgentID {
		err = fmt.Errorf(
			"HandlePublishRequestProgress: progress update '%s' of thing '%s' does not come from agent '%s' but from '%s'. Update ignored.",
			stat.RequestID, thingID, arAgentID, agentID)
		slog.Warn(err.Error(), "agentID", agentID)
		return
	}

	// 2: Update the action status in the digital twin action record and log errors
	//   (for use with query actions)
	_, err = svc.dtwStore.UpdateRequestProgress(agentID, thingID, actionName, stat.Progress, stat.Output)

	if stat.Error != "" {
		slog.Warn("HandlePublishRequestProgress - with error",
			slog.String("AgentID", agentID),
			slog.String("ThingID", thingID),
			slog.String("Name", actionName),
			slog.String("Progress", stat.Progress),
			slog.String("RequestID", stat.RequestID),
			slog.String("Error", stat.Error),
		)
	} else if stat.Progress == vocab.RequestCompleted {
		slog.Info("HandlePublishRequestProgress - completed",
			slog.String("ThingID", thingID),
			slog.String("Name", actionName),
			slog.String("RequestID", stat.RequestID),
		)
	} else {
		slog.Info("HandlePublishRequestProgress - progress",
			slog.String("ThingID", thingID),
			slog.String("Name", actionName),
			slog.String("RequestID", stat.RequestID),
			slog.String("Progress", stat.Progress),
		)
	}

	// 3: Forward the progress update to the original sender
	c := svc.cm.GetConnectionByCLCID(replyTo)
	if c != nil {
		err = c.PublishActionStatus(stat, agentID)
	} else {
		err = fmt.Errorf("client connection-id (replyTo) '%s' not found for client '%s'", replyTo, senderID)
		// try workaround

	}

	if err != nil {
		slog.Warn("HandlePublishRequestProgress. Forwarding to sender failed",
			slog.String("senderID", senderID),
			slog.String("thingID", thingID),
			slog.String("replyTo", replyTo),
			slog.String("err", err.Error()),
			slog.String("RequestID", stat.RequestID),
		)
		err = nil
	}

	// 4: Update the active action cache and remove the action when completed or failed
	svc.mux.Lock()
	defer svc.mux.Unlock()
	actionRecord.Progress = stat.Progress
	svc.activeCache[stat.RequestID] = actionRecord

	if stat.Progress == vocab.RequestCompleted || stat.Progress == vocab.RequestFailed {
		delete(svc.activeCache, stat.RequestID)
	}
	return
}

// HandleQueryAction returns the action status
func (svc *DigitwinRouter) HandleQueryAction(clientID string, dThingID string, name string) (reply any, err error) {
	reply, err = svc.dtwService.ValuesSvc.QueryAction(clientID,
		digitwin.ValuesQueryActionArgs{ThingID: dThingID, Name: name})
	return reply, err
}

// HandleQueryAllAction returns the status of all actions of a Thing
func (svc *DigitwinRouter) HandleQueryAllActions(clientID string, dThingID string) (reply any, err error) {
	reply, err = svc.dtwService.ValuesSvc.QueryAllActions(clientID, dThingID)
	return reply, err
}
