// Package transports with the interface of a client transport connection
package transports

import "github.com/hiveot/hub/wot/td"

// IAgentConnection extends the consumer interface with methods for agents.
type IAgentConnection interface {
	// agents have consumer capabilities
	IConsumerConnection

	// PubEvent helper for agents to publish an event
	// This is short for SendNotification( ... wot.OpEvent ...)
	PubEvent(thingID string, name string, value any) error

	// PubProperty helper for agents to publish a property value update
	// This is short for SendNotification( ... wot.OpProperty ...)
	PubProperty(thingID string, name string, value any) error

	// PubProperties helper for agents to publish a map of property values
	// This is short for SendNotification( ... wot.OpProperties ...)
	PubProperties(thingID string, propMap map[string]any) error

	// PubTD helper for agents to publish a TD update
	// This is short for SendNotification( ... wot.HTOpTD ...)
	PubTD(td *td.TD) error

	// SendNotification [agent] sends a notification to subscribers.
	// Notifications do not receive a response.
	//
	// This returns an error if the notification could not be delivered to the server.
	SendNotification(notification NotificationMessage) error

	// SendResponse [agent] sends a response to a request.
	// This returns an error if the response could not be delivered to the server
	SendResponse(response ResponseMessage) error

	// SetRequestHandler [agent] sets the handler for operations that return a response.
	// This replaces any previously set handler.
	SetRequestHandler(request RequestHandler)
}
