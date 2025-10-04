// Package router with digital twin action flow handling functions
package router

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	authn "github.com/hiveot/hub/runtime/authn/api"
	authz "github.com/hiveot/hub/runtime/authz/api"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
)

// ActiveRequestRecord holds active requests
type ActiveRequestRecord struct {
	Operation     string    // The operation being tracked, invoke action or write property
	CorrelationID string    // CorrelationID of the ongoing action
	AgentID       string    // Agent that is handling the action request
	ThingID       string    // thingID as provided by the agent the action is for
	Name          string    // name of the action as described in the TD
	SenderID      string    // action sender that will receive progress messages
	Progress      string    // current progress of the action:
	Updated       time.Time // timestamp to handle expiry
	ReplyTo       string    // Action reply address. Typically, the sender's connection-id
}

// ForwardRequestToRemoteAgent forwards the request to an external agent.
//
// This tracks the action in the digitwin store, locates the agent connection,
// forwards the request to the agent.
//
// This sends a notification with an ActionStatus message to the sender (sc), or with an error if
// something went wrong.
//
// req is the request to forward
// sc is the server connection endpoint of the client sending the request
//
// If the agent is not connected status is failed.
func (r *DigitwinRouter) ForwardRequestToRemoteAgent(
	req *messaging.RequestMessage, sc messaging.IConnection) (
	resp *messaging.ResponseMessage) {

	agentID, agThingID := td.SplitDigiTwinThingID(req.ThingID)

	// Determine the agent to forward the request to.
	// Agents only have a single connection instance so the agentID can be used.
	agentConn := r.transportServer.GetConnectionByClientID(agentID)

	if agentConn == nil {
		// The request cannot be delivered as the agent is not reachable
		// For now return an error.
		// TODO: determine the rules and use-cases for queuing a request
		err := fmt.Errorf("ForwardRequestToRemoteAgent: Agent '%s' not reachable. Ignored", agentID)
		return req.CreateResponse(nil, err)
	}
	replyTo := ""
	if sc != nil {
		replyTo = sc.GetConnectionInfo().ConnectionID
	}

	// track the request progress so async responses can be returned to the client (replyTo)
	requestRecord := ActiveRequestRecord{
		AgentID:       agentID,
		Name:          req.Name,
		Operation:     req.Operation,
		ReplyTo:       replyTo,
		Updated:       time.Now().UTC(),
		CorrelationID: req.CorrelationID,
		Progress:      messaging.StatusPending,
		SenderID:      req.SenderID,
		ThingID:       agThingID,
	}
	r.mux.Lock()
	r.activeCache[requestRecord.CorrelationID] = requestRecord
	r.mux.Unlock()

	// forward the request to the agent using the ThingID of the agent, not the
	// digital twin agThingID.
	req2 := *req
	req2.ThingID = agThingID // agent uses the local message ID
	err := agentConn.SendRequest(&req2)

	// if forwarding the request to the agent failed, then remove the tracking,
	// update the action status, and return an error response
	if err != nil {
		slog.Warn("ForwardRequestToRemoteAgent - failed",
			slog.String("dThingID", req.ThingID),
			slog.String("actionName", req.Name),
			slog.String("correlationID", req.CorrelationID),
			slog.String("err", err.Error()))

		// cleanup as the record is no longer needed
		r.mux.Lock()
		delete(r.activeCache, req.CorrelationID)
		r.mux.Unlock()

		resp = req.CreateResponse(nil, err)
		//if stored {
		//	_, _ = svc.dtwStore.UpdateActionWithResponse(resp)
		//}
		return resp
	}

	// no immediate result so return nil
	return nil
}

// HandleRequest routes requests from clients to agents.
//
// This handles all requests for the digital twin and forwards other request
// to the connected agent.
//
// This returns a response if available. replyTo is used to store the sender's
// reply-to address for handling responses to pending requests.
func (r *DigitwinRouter) HandleRequest(
	req *messaging.RequestMessage, c messaging.IConnection) (resp *messaging.ResponseMessage) {

	if req.Created == "" {
		req.Created = utils.FormatNowUTCMilli()
	}

	r.requestLogger.Info("-> REQ:",
		slog.String("correlationID", req.CorrelationID),
		slog.String("operation", req.Operation),
		slog.String("dThingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("Input", req.ToString(20)),
		slog.String("senderID", req.SenderID),
	)

	// middleware: authorize the request. (TODO: use a middleware chain)
	if !r.hasPermission(req.SenderID, req.Operation, req.ThingID) {
		err := fmt.Errorf("unauthorized. client '%s' does not have permission"+
			" to invoke operation '%s' on Thing '%s'", req.SenderID, req.Operation, req.ThingID)
		r.requestLogger.Warn(err.Error())
		return req.CreateResponse(nil, err)
	}
	switch req.Operation {
	// Thing actions status are tracked and stored.
	// Responses are send asynchronously to the replyTo address.
	case vocab.OpInvokeAction:
		resp = r.HandleInvokeAction(req, c)
	case vocab.OpWriteProperty:
		resp = r.HandleWriteProperty(req, c)

	// digital twin requests are handled immediately and return a response
	case vocab.OpQueryAction:
		resp = r.HandleQueryAction(req, c)
	case vocab.OpQueryAllActions:
		resp = r.HandleQueryAllActions(req, c)
	case vocab.OpReadProperty:
		resp = r.HandleReadProperty(req, c)
	case vocab.OpReadAllProperties:
		resp = r.HandleReadAllProperties(req, c)

	default:
		err := fmt.Errorf("unknown request operation '%s' from client '%s'",
			req.Operation, req.SenderID)
		slog.Warn(err.Error())
		resp = req.CreateResponse(nil, err)
	}
	// direct responses are optional
	if resp != nil {
		errMsg := ""
		if resp.Error != nil {
			errMsg = resp.Error.String()
		}
		r.requestLogger.Info("<- RESP",
			slog.String("correlationID", resp.CorrelationID),
			slog.String("operation", resp.Operation),
			slog.String("dThingID", resp.ThingID),
			slog.String("name", resp.Name),
			// slog.String("value", resp.ValueAsString(20)),
			slog.String("err", errMsg),
		)
	}
	return resp
}

// HandleInvokeAction handles the request to invoke an action on a Thing.
//
// This tracks the progress in the digital twin store and passes the action request
// to the thing agent.
//
// The digitwin acts as a proxy for the action and forwards the request to the agent
// identified by the dThingID. This can lead to one of these flows:
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
func (r *DigitwinRouter) HandleInvokeAction(
	req *messaging.RequestMessage, c messaging.IConnection) (resp *messaging.ResponseMessage) {

	// Forward the action to the built-in services
	agentID, thingID := td.SplitDigiTwinThingID(req.ThingID)
	_ = thingID

	// internal services return instant result.
	// There is no good use-case to record these actions.
	switch agentID {
	case digitwin.ThingDirectoryAgentID:
		resp = r.digitwinAction(req, c)
	case authn.AdminAgentID:
		resp = r.authnAction(req, c)
	case authz.AdminAgentID:
		resp = r.authzAction(req, c)
	default:
		// forward action to external service
		// Store the request progress to be able to respond to queryAction. Only
		// unsafe (stateful) actions are stored.
		actionStatus, stored, err := r.dtwStore.NewActionStart(req)
		_ = stored
		if err != nil {
			return req.CreateResponse(nil, err)
		}

		// Forward the action to external agents
		// Depending on how the agent is connection this can provide an immediate response
		// or no immediate response, in which case a notification is returned.
		resp = r.ForwardRequestToRemoteAgent(req, c)

		// the request has been sent successfully
		// actions return a response or a notification with ActionStatus record with status pending.
		// other requests simply don't return anything until an async response is received.
		if stored && resp != nil {
			// in case of immediately available response and the action was stored.
			_, _ = r.dtwStore.UpdateActionWithResponse(resp)
		} else if resp == nil && c != nil {
			// send an async notification if no response is available yet
			notif := req.CreateNotification()
			notif.Value = actionStatus
			_ = c.SendNotification(notif)
		}
	}
	return resp
}

// HandleQueryAction returns the action status
func (r *DigitwinRouter) HandleQueryAction(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
	av, err := r.dtwService.ValuesSvc.QueryAction(req.SenderID,
		digitwin.ThingValuesQueryActionArgs{ThingID: req.ThingID, Name: req.Name})
	return req.CreateResponse(av, err)
}
func (r *DigitwinRouter) HandleQueryAllActions(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
	av, err := r.dtwService.ValuesSvc.QueryAllActions(req.SenderID, req.ThingID)
	return req.CreateResponse(av, err)
}

// HandleReadEvent consumer requests a digital twin thing's event value
func (r *DigitwinRouter) HandleReadEvent(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	output, err := r.dtwService.ValuesSvc.ReadEvent(req.SenderID,
		digitwin.ThingValuesReadEventArgs{ThingID: req.ThingID, Name: req.Name})
	return req.CreateResponse(output, err)
}

// HandleReadAllEvents consumer requests all digital twin thing event values
func (r *DigitwinRouter) HandleReadAllEvents(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	output, err := r.dtwService.ValuesSvc.ReadAllEvents(req.SenderID, req.ThingID)
	return req.CreateResponse(output, err)
}

// HandleReadProperty consumer requests a digital twin thing's property value
func (r *DigitwinRouter) HandleReadProperty(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	output, err := r.dtwService.ValuesSvc.ReadProperty(req.SenderID,
		digitwin.ThingValuesReadPropertyArgs{ThingID: req.ThingID, Name: req.Name})
	return req.CreateResponse(output, err)
}

// HandleReadAllProperties consumer requests reading all digital twin's property values
func (r *DigitwinRouter) HandleReadAllProperties(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	output, err := r.dtwService.ValuesSvc.ReadAllProperties(req.SenderID, req.ThingID)
	return req.CreateResponse(output, err)
}

// HandleReadTD consumer reads a TD
// This converts the operation in an action for the directory service.
//func (svc *DigitwinRouter) HandleReadTD(
//	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
//
//	// the thingID in the request becomes the argument for the directory service, if any
//	req2 := *req
//	req2.Input = req.ThingID
//	req2.ThingID = digitwin.ThingDirectoryDThingID
//	req2.Name = digitwin.ThingDirectoryRetrieveThingMethod
//	resp := svc.digitwinAction(&req2, c)
//	return resp
//}

// HandleReadAllTDs consumer reads all TDs
// This converts the operation in an action for the directory service.
//func (svc *DigitwinRouter) HandleReadAllTDs(
//	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
//
//	req2 := *req
//	req2.ThingID = digitwin.ThingDirectoryDThingID
//	req2.Name = digitwin.ThingDirectoryRetrieveAllThingsMethod
//	resp := svc.digitwinAction(&req2, c)
//	return resp
//}

// HandleUpdateTD agent updates a TD
// This converts the operation in an action for the directory service.
//func (svc *DigitwinRouter) HandleUpdateTD(
//	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
//
//	// the thingID in the request becomes the argument for the directory service, if any
//	//rework the request
//	req2 := *req
//	req2.Input = req.Input
//	req2.ThingID = digitwin.ThingDirectoryDThingID
//	req2.Name = digitwin.ThingDirectoryUpdateThingMethod
//	resp := svc.digitwinAction(&req2, c)
//	return resp
//}

// HandleWriteProperty A consumer requests to write a new value to a property.
//
// This follows the same process as invoking an action but without tracking progress.
// The request is forwarded to the agent which is expected to send a response.
func (r *DigitwinRouter) HandleWriteProperty(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	// Note: if internal services have writable properties (currently they don't)
	// then add forwarding it here similar to invoking actions.

	return r.ForwardRequestToRemoteAgent(req, c)
}
