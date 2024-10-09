package hubrouter

import "github.com/hiveot/hub/lib/hubclient"

// IHubRouter is the interface of the handler of action,event and property update
// messages received from consumers and agents.
type IHubRouter interface {
	// HandleActionFlow consumer sends request to invoke an action on the digital twin.
	//
	// The router will assign a message-ID if no request ID is given, update the digital
	// twin with the pending request and forward it the agent if the agent is online.
	// If the agent is not reachable this returns with the status failed.
	//
	// This returns a message ID for tracking the action flow
	// The status response is 'delivered' if the action was handed over to the agent without error.
	// The status response is 'failed' if the action cannot be passed to the agent
	// The status response is 'pending' if the agent is offline and the action is queued for delivery - when supported.
	HandleActionFlow(
		consumerID string, dThingID string, actionName string, input any, reqID string) (
		status string, output any, messageID string, err error)

	// HandleActionProgress agent publishes an action progress update
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
		consumerID string, dThingID string, name string, newValue any) (
		status string, messageID string, err error)
}
