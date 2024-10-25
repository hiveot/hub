package hubrouter

import (
	"github.com/hiveot/hub/lib/hubclient"
)

// ActionProgressHandler is the handler for return an action progress to sender.
// Used by router HandleInvokeActionProgress to send the action result to the sender.
type ActionProgressHandler func(stat hubclient.ActionProgress, agentID string) error

// IHubRouter is the interface of the handler of action,event and property update
// messages received from consumers and agents.
// TODO: IHubRouter is effectively the digitwin agent service API for connecting
//
//	agents with the hub. Only agents are allowed to use it.
type IHubRouter interface {

	// HandleInvokeAction client (consumer or agent) sends request to invoke an action on the digital twin.
	//
	// The router will assign a message-ID if no request ID is given, update the digital
	// twin with the pending request and forward it the agent if the agent is online.
	// If the agent is not reachable this returns with the status failed.
	//
	//	dThingID is the digitwin ID of the thing whose action to invoke
	//	actionName is the name of the action to invoke
	//	input is the action input data as per schema defined in the Thing TD
	//	reqID is the messageID of the request. Used to send an async reply.
	//	senderID is the sender's account ID
	//	cid is the sender's connection ID to be able to pass a progress update
	//
	// This returns a message ID for tracking the action flow
	// The status response is 'pending' if the action was handed over to the agent without error.
	// The status response is 'failed' if the action cannot be passed to the agent
	// The status response is 'queued' if the agent is offline and the action is queued for delivery - when supported.
	HandleInvokeAction(
		clientID, dThingID string, actionName string, input any, reqID string, cid string) (
		status string, output any, messageID string, err error)

	// HandleInvokeActionProgress agent publishes a progress update message
	// This updates the corresponding digital twin action status
	HandleInvokeActionProgress(agentID string, data any) error

	HandleLogin(data any) (reply any, err error)
	HandleLoginRefresh(clientID string, data any) (reply any, err error)
	HandleLogout(clientID string)

	// HandlePublishEvent agent publishes an event
	// This can contain a messageID if the event is a response to an action
	// or property write, depending on the agent implementation.
	HandlePublishEvent(agentID string, thingID string, name string, value any, messageID string) error

	// HandlePublishProperty agent publishes a property change.
	// If the update is the result of a WriteProperty request then the messageID
	// contains the ID returned to the consumer by WriteProperty.
	//
	// Whether a messageID is included depends on the agent implementation.
	HandlePublishProperty(agentID string, thingID string, name string, value any, messageID string) error

	// HandlePublishTD agent publishes an updated TD
	HandlePublishTD(agentID string, args any) error

	// HandleQueryAction returns the action status
	HandleQueryAction(consumerID string, dThingID string, name string) (reply any, err error)

	// HandleQueryAllActions returns the status of all actions of a Thing
	HandleQueryAllActions(clientID string, dThingID string) (reply any, err error)

	// HandleReadEvent consumer reads a digital twin thing's event value
	HandleReadEvent(consumerID string, dThingID string, name string) (reply any, err error)

	// HandleReadAllEvents consumer reads all digital twin thing's event values
	HandleReadAllEvents(consumerID string, dThingID string) (reply any, err error)

	// HandleReadProperty consumer reads a digital twin thing's property value
	HandleReadProperty(consumerID string, dThingID string, name string) (reply any, err error)

	// HandleReadAllProperties handles reading all digital twin thing's property values
	HandleReadAllProperties(senderID string, dThingID string) (reply any, err error)

	// HandleReadTD consumer reads a digital twin thing's TD
	HandleReadTD(consumerID string, args any) (reply any, err error)

	// HandleReadAllTDs consumer reads all digital twin thing's TD
	HandleReadAllTDs(consumerID string) (reply any, err error)

	// HandleUpdateTDFlow agent updates a TD
	// Deprecated: use the directory API instead
	//HandleUpdateTDFlow(agentID string, tdJSON string) error

	// HandleWriteProperty consumer sends request to write a property.
	// This returns the delivery status and an optional messageID to link related updates.
	HandleWriteProperty(
		dThingID string, name string, newValue any, consumerID string) (
		status string, messageID string, err error)
}
