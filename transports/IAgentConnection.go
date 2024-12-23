// Package transports with the interface of a client transport connection
package transports

// IAgentConnection extends the consumer interface with methods for agents.
type IAgentConnection interface {
	// agents have consumer capabilities
	IConsumerConnection

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
