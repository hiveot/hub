// Package service with digital twin action flow handling functions
package router

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
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
	msg *transports.ThingMessage, replyTo string) (completed bool, output any, err error) {

	// Forward the action to the built-in services
	agentID, thingID := td.SplitDigiTwinThingID(msg.ThingID)
	_ = thingID

	// internal services return instant result
	completed = true
	switch agentID {
	case digitwin.DirectoryAgentID:
		output, err = svc.digitwinAction(msg)
	case authn.AdminAgentID:
		output, err = svc.authnAction(msg)
	case authz.AdminAgentID:
		output, err = svc.authzAction(msg)
	case api.DigitwinServiceID:
		output, err = svc.digitwinAction(msg)
	default:
		completed = false
		// Forward the action to external agents
		completed, output, err = svc.HandleInvokeRemoteAgent(msg, replyTo)
	}
	return completed, output, err
}

// HandleInvokeRemoteAgent forwards the action to external agents
func (svc *DigitwinRouter) HandleInvokeRemoteAgent(
	msg *transports.ThingMessage, replyTo string) (completed bool, output any, err error) {

	agentID, thingID := td.SplitDigiTwinThingID(msg.ThingID)

	slog.Info("HandleInvokeRemoteAgent (to agent)",
		slog.String("dThingID", msg.ThingID),
		slog.String("actionName", msg.Name),
		slog.String("requestID", msg.RequestID),
		slog.String("senderID", msg.SenderID),
	)

	// Store the action progress to be able to respond to queryAction.
	// The TD of a service must be known before its action can be invoked.
	// If we get here then the form to invoke the action is somehow known so
	// the TD isn't really neccesary.
	// Considerations where this isn't an error:
	// 1. 'secret' services where the TD is shared out of band and not in the directory
	// 2. 'safe' actions that don't change state.
	// FIXME: don't store 'safe' actions as their results are meaningless in queryAction
	// TODO: Maybe this shouldn't be stored at all and use properties instead.
	err = svc.dtwStore.UpdateActionStart(
		msg.ThingID, msg.Name, msg.Data, msg.RequestID, msg.SenderID)

	if err != nil {
		return false, nil, err
	}
	// forward to external services/things and return its response
	var progress string

	// Note: agents should only have a single instance
	c := svc.cm.GetConnectionByClientID(agentID)

	if c == nil {
		progress = vocab.RequestFailed
		err = fmt.Errorf("HandleInvokeRemoteAgent: Agent '%s' not reachable. Ignored", agentID)
	} else {
		progress = vocab.RequestDelivered

		// store a new action progress by message ID to support sending replies to the sender
		actionRecord := ActionFlowRecord{
			Operation: msg.Operation,
			AgentID:   agentID,
			ThingID:   thingID,
			RequestID: msg.RequestID,
			Name:      msg.Name,
			SenderID:  msg.SenderID,
			Progress:  progress,
			Updated:   time.Now(),
			ReplyTo:   replyTo,
		}

		// track in-progress actions
		svc.mux.Lock()
		actionRecord.Progress = progress
		svc.activeCache[msg.RequestID] = actionRecord
		svc.mux.Unlock()

		msg2 := *msg
		msg2.ThingID = thingID // agent uses the local message ID
		err = c.SendRequest(msg2)
	}
	// post processing by tracking action progress
	if err != nil {
		slog.Warn("HandleInvokeRemoteAgent - failed",
			slog.String("dThingID", msg.ThingID),
			slog.String("actionName", msg.Name),
			slog.String("requestID", msg.RequestID),
			slog.String("err", err.Error()))

		// cleanup as the record is no longer needed
		svc.mux.Lock()
		delete(svc.activeCache, msg.RequestID)
		svc.mux.Unlock()
	}

	// store the action status
	_, _ = svc.dtwStore.UpdateActionStatus(
		agentID, thingID, msg.Name, progress, nil)

	return false, output, err
}

// HandleActionResponse agent sent an action response.
// The message must contain the requestID previously used in SendRequest.
// The payload is the application output object. (action output)
// This completes the action successfully.
//
// This:
// 1. Validates the request is still ongoing.
// 2: Updates the status of the current digital twin action record to completed.
// 3: Forwards the update to the sender, if this is an active request.
// 4: Remove the active request from the cache.
//
// If the message is no longer in the active cache then it is ignored.
func (svc *DigitwinRouter) HandleActionResponse(msg *transports.ThingMessage) {
	var err error

	agentID := msg.SenderID
	slog.Info("HandleActionResponse",
		slog.String("ThingID", msg.ThingID),
		slog.String("Name", msg.Name),
		slog.String("RequestID", msg.RequestID),
	)

	// 1: Validate this is an active action
	svc.mux.Lock()
	actionRecord, found := svc.activeCache[msg.RequestID]
	svc.mux.Unlock()
	if !found {
		err = fmt.Errorf(
			"HandleActionResponse: Message '%s' from agent '%s' not in action cache. It is ignored",
			msg.RequestID, agentID)
		slog.Warn(err.Error())
		return
	}
	arAgentID := actionRecord.AgentID // this should match the msg.SenderID
	if arAgentID != msg.SenderID {
		slog.Error("HandleActionResponse. AgentID's don't match",
			"senderID", agentID, "requestRecord agent", arAgentID)
		return
	}
	thingID := actionRecord.ThingID
	actionName := actionRecord.Name
	replyTo := actionRecord.ReplyTo
	senderID := actionRecord.SenderID

	// the action response by sender uses the sender's thingID. Convert this to the
	// digital twin thingID.
	msg.ThingID = td.MakeDigiTwinThingID(agentID, thingID)
	msg.Name = actionName

	// the sender (agents) must be the thing agent
	if agentID != arAgentID {
		// TODO: update the action status?

		err = fmt.Errorf("HandleActionResponse: response ID '%s' of thing '%s' "+
			"does not come from agent '%s' but from '%s'. Response ignored",
			msg.RequestID, thingID, arAgentID, agentID)
		slog.Warn(err.Error(), "agentID", agentID)
		return
	}

	// 2: Update the action status in the digital twin action record and log errors
	//   (for use with query actions)
	_, _ = svc.dtwStore.UpdateActionStatus(
		agentID, thingID, actionName, transports.StatusCompleted, msg.Data)

	// 3: Forward the progress update to the sender of the request
	c := svc.cm.GetConnectionByConnectionID(replyTo)
	if c != nil {
		err = nil
		if msg.Operation == wot.HTOpPublishError {
			errText := fmt.Sprintf("%v", msg.Data)
			// send the response to the consumer
			err = errors.New(errText)
		}
		// send the response to the consumer
		err = c.SendResponse(msg.ThingID, msg.Name, msg.Data, err, msg.RequestID)

	} else {
		// can't reach the consumer
		err = fmt.Errorf("client connection-id (replyTo) '%s' not found for client '%s'",
			replyTo, senderID)
	}

	if err != nil {
		slog.Warn("HandleActionResponse. Forwarding to sender failed",
			slog.String("senderID", senderID),
			slog.String("thingID", thingID),
			slog.String("replyTo", replyTo),
			slog.String("err", err.Error()),
			slog.String("RequestID", msg.RequestID),
		)
		err = nil
	}

	// 4: Update the active action cache and remove the action when completed or failed
	svc.mux.Lock()
	defer svc.mux.Unlock()
	delete(svc.activeCache, msg.RequestID)
	return
}

// TODO
func (svc *DigitwinRouter) HandleUpdateMultipleActionStatuses(msg *transports.ThingMessage) {
	slog.Error("HandleUpdateMultipleActionStatuses: not yet implemented")
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
