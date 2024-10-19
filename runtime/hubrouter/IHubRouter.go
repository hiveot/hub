package hubrouter

import (
	"github.com/hiveot/hub/lib/hubclient"
)

// ActionProgressHandler is the handler for return an action progress to sender.
// Used by router HandleActionProgress to send the action result to the sender.
type ActionProgressHandler func(stat hubclient.DeliveryStatus, agentID string) error

// IHubRouter is the interface of the handler of action,event and property update
// messages received from consumers and agents.
type IHubRouter interface {

	// HandleActionFlow client (consumer or agent) sends request to invoke an action on the digital twin.
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
	HandleActionFlow(
		dThingID string, actionName string, input any, reqID string, senderID string, cid string) (
		status string, output any, messageID string, err error)

	// HandleActionProgress agent publishes a progress update message
	// This updates the corresponding digital twin action status
	HandleActionProgress(agentID string, stat hubclient.DeliveryStatus) error

	// HandleEventFlow agent publishes an event
	// This can contains a messageID if the event is a response to an action
	// or property write, depending on the agent implementation.
	HandleEventFlow(agentID string, thingID string, name string, value any, messageID string) error

	// HandleUpdateTDFlow agent updates a TD
	HandleUpdateTDFlow(agentID string, tdJSON string) error

	// HandleUpdatePropertyFlow agent publishes a property change
	// If the update is the result of a WriteProperty request then the messageID
	// contains the ID returned to the consumer by WriteProperty.
	//
	// Whether a messageID is included depends on the agent implementation.
	HandleUpdatePropertyFlow(agentID string, thingID string, name string, value any, messageID string) error

	// HandleWritePropertyFlow consumer sends request to write a property.
	// This returns the delivery status and an optional messageID to link related updates.
	HandleWritePropertyFlow(
		dThingID string, name string, newValue any, consumerID string) (
		status string, messageID string, err error)
}
