package transports

// ReplyToHandler is a server-side handler for sending replies to requests that
// expect one. E.g: InvokeAction and WriteProperty (tbd)
// The server side transport protocol implements the most efficient way to send that reply
// back to the client, either as an immediate result or asynchronously after the
// request has returned.
//
//	status is the progress status: RequestDelivered,RequestFailed,RequestCompleted
//	output is the result if status is RequestCompleted. Can be nil.
//	err is the error in case status is RequestFailed
type ReplyToHandler func(status string, output any, err error)

// ServerMessageHandler handles a request. The handler is responsible for
// sending a reply to the sender.
//
//	msg is the envelope that contains the request to process.
//	replyTo is the connection for sending a reply, or nil when no reply should be sent.
type ServerMessageHandler func(msg *ThingMessage, replyTo IServerConnection)

// IServerConnection is the interface of an incoming client connection on the server.
// Protocol servers must implement this interface to return information to the consumer.
//
// This provides a return channel for sending messages from the digital twin to
// agents or consumers.
//
// Subscription to events or properties can be made externally via this API,
// or handled internally by the protocol handler if the protocol defines the
// messages for subscription.
type IServerConnection interface {

	// Disconnect the client.
	Disconnect()

	// GetClientID returns the authentication ID of connected agent or consumer
	GetClientID() string

	// GetConnectionID returns the client's connection ID belonging to this endpoint
	GetConnectionID() string

	// GetProtocolType returns the name of the protocol binding of this connection.
	GetProtocolType() string

	// SendError returns an error to the client, send by an agent
	// Intended to send an error to the client instead of a response.
	SendError(thingID, name string, errResponse string, requestID string)

	// SendNotification sends a notification to the client without a response.
	// Intended for consumers to receive an update.
	//
	// operation is the operation to invoke
	// thingID of the thing the operation applies to
	// name of the affordance the operation applies to
	// data contains the notification data as described in the TD affordance.
	SendNotification(operation string, dThingID, name string, data any)

	// SendRequest sends a request (action, write property) to the client (agent).
	// The client MUST send a response message and include the provided requestID.
	// Intended for agents to receive a request.
	//
	// operation identifies the request to invoke.
	// thingID of the thing the operation applies to
	// name of the affordance the operation applies to
	// input data of the request as per affordance
	// requestID is the message requestID to use in the response.
	SendRequest(operation string, thingID, name string, input any, requestID string)

	// SendResponse send a response (action status) to the client for a previous sent request.
	//	output is the response data
	//	requestID contains the requestID provided in the request.
	SendResponse(thingID, name string, output any, requestID string)
}
