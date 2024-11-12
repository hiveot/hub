package api

import (
	"github.com/hiveot/hub/lib/hubclient"
)

// RequestProgressHandler is the handler for return an action progress to sender.
// Used by router HandlePublishRequestProgress to send the action result to the sender.
type RequestProgressHandler func(stat hubclient.RequestProgress, agentID string) error

// IDigitwinRouter is the interface for routing the action,event and property messages
// received from consumers and agents. It handles the flow for TD level operations.
type IDigitwinRouter interface {

	// HandleInvokeAction client (consumer or agent) sends request to invoke an action on the digital twin.
	//
	// The router will assign a message-ID if no request ID is given, update the digital
	// twin with the pending request and forward it the agent if the agent is online.
	// If the agent is not reachable this returns with the status failed.
	//
	//	senderID is the ID of the agent or consumer invoking the action.
	//	dThingID is the digital twin ID of the Thing whose action to invoke.
	//	actionName is the name of the action to invoke
	//	input is the action input data as per schema defined in the Thing TD
	//	reqID is the requestID of the request. Used to send an async reply.
	//	cid is the sender's connection ID to be able to pass a progress update.
	//
	// This returns a message ID for tracking the action flow
	// The status response is 'pending' if the action was handed over to the agent without error.
	// The status response is 'failed' if the action cannot be passed to the agent
	// The status response is 'queued' if the agent is offline and the action is queued for delivery - when supported.
	HandleInvokeAction(
		senderID string, dThingID string, actionName string, input any, reqID string, cid string) (
		status string, output any, requestID string, err error)

	// HandleLogin routes the login request to the authentication service
	HandleLogin(data any) (reply any, err error)
	// HandleLoginRefresh routes the login token refresh request to the authentication service
	HandleLoginRefresh(clientID string, data any) (reply any, err error)
	// HandleLogout routes the session logout request to the authentication service
	HandleLogout(clientID string)

	// HandlePublishRequestProgress agent publishes an action progress update message
	// This updates the corresponding digital twin action status.
	HandlePublishRequestProgress(agentID string, progressMsg hubclient.RequestProgress) error

	// HandlePublishEvent agent publishes an event.
	// This can contain a requestID if the event is a response to an action
	// or property write, depending on the agent implementation.
	HandlePublishEvent(
		agentID string, thingID string, name string, value any, requestID string) error

	// HandlePublishMultipleProperties agent publishes a batch of property changes.
	HandlePublishMultipleProperties(
		agentID string, thingID string, propMap map[string]any, requestID string) error

	// HandlePublishProperty agent publishes a property change.
	// If the update is the result of a WriteProperty request then the requestID
	// contains the ID returned to the consumer by WriteProperty.
	//
	// Whether a requestID is included depends on the agent implementation.
	HandlePublishProperty(
		agentID string, thingID string, name string, value any, requestID string) error

	// HandlePublishTD agent publishes an updated TD
	HandlePublishTD(agentID string, args any) error

	// HandleQueryAction consumer requests the action status
	HandleQueryAction(consumerID string, dThingID string, name string) (reply any, err error)

	// HandleQueryAllActions consumer requests the status of all actions of a Thing
	HandleQueryAllActions(consumerID string, dThingID string) (reply any, err error)

	// HandleReadEvent consumer reads a digital twin thing's event value
	HandleReadEvent(consumerID string, dThingID string, name string) (reply any, err error)

	// HandleReadAllEvents consumer reads all digital twin thing's event values
	HandleReadAllEvents(consumerID string, dThingID string) (reply any, err error)

	// HandleReadProperty consumer reads a digital twin thing's property value
	HandleReadProperty(consumerID string, dThingID string, name string) (reply any, err error)

	// HandleReadAllProperties handles reading all digital twin thing's property values
	HandleReadAllProperties(consumerID string, dThingID string) (reply any, err error)

	// HandleReadTD consumer reads a digital twin thing's TD
	HandleReadTD(consumerID string, args any) (reply any, err error)

	// HandleReadAllTDs consumer reads all digital twin thing's TD
	HandleReadAllTDs(consumerID string) (reply any, err error)

	// HandleUpdateTDFlow agent updates a TD
	// Deprecated: use the directory API instead
	//HandleUpdateTDFlow(agentID string, tdJSON string) error

	// HandleWriteProperty consumer sends request to write a property.
	// This returns the delivery status and an optional requestID to link related updates.
	HandleWriteProperty(
		consumerID string, dThingID string, name string, newValue any) (
		status string, requestID string, err error)
}
