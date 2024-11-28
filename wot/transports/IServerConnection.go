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

// ServerMessageHandler processes a request and optionally returns a progress status.
//
// If a replyTo is provided then a result is expected. That result can be return
// with the RequestStatus immediately (status failer or completed), or returned
// asynchronously through the replyTo interface.
//
//	msg is the envelope that contains the request to process
//	replyTo is the connection to send an actionProgress result to.
type ServerMessageHandler func(msg *ThingMessage, replyTo IServerConnection) RequestStatus

// IServerConnection is the interface of an incoming consumer or agent connection on the server.
// Protocol servers must implement this interface to return information to the consumer.
// appbackend
// interact with
// the digitwin router.
//
// This provides a return channel for sending messages from the digital twin to
// agents or consumers.
//
// Subscription to events or properties can be made externally via this API,
// or handled internally by the protocol handler if the protocol defines the
// messages for subscription.
type IServerConnection interface {

	// Close the connection.
	// It is allowed to close an already closed connection.
	Close()

	// GetConnectionID returns the client's connection ID belonging to this endpoint
	// this is a combination of the clientid and header cid field
	GetConnectionID() string

	// GetClientID returns the authentication ID of connected agent or consumer
	GetClientID() string

	// GetProtocol returns the protocol binding of this connection.
	GetProtocol() string

	// InvokeAction invokes an action on a connected Thing's and return result
	// if available.
	//
	// Intended to invoke the action on a remote Thing, where the Thing connects
	// to this server as a client, as is the case with a Hub.
	//
	// Note that the WoT specifications currently do not define this behavior as
	// WoT is based on Things running servers.
	InvokeAction(thingID string, name string, input any, requestID string, senderID string) (
		status string, output any, err error)

	// PublishActionStatus sends an action progress update to the client.
	//
	// Intended for passing action progress or results asynchronously to consumers
	// or Thing agents.
	PublishActionStatus(stat RequestStatus, agentID string) error

	// PublishEvent publishes an event message to a consumer.
	// The protocol implementation will only pass the event if the consumer is
	// subscribed to the event (or all events).
	//
	//	dThingID is the digital twin thingID that publishes the event
	//	name is the event name as defined in the thing's TD
	//	value is the event value as per event affordance schema
	//	requestID if the event is associated with an action
	PublishEvent(dThingID string, name string, value any, requestID string, agentID string)

	// PublishProperty publishes a new property value to this client. If the client
	// does not observe this property then the connection will ignore the update..
	//
	//	dThingID is the digital twin thingID that publishes the property value
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	requestID if the property update is associated with an action
	PublishProperty(dThingID string, name string, value any, requestID string, agentID string)

	// WriteProperty requests a property value change from the connected agent.
	//
	// Intended for passing property write requests from the backend to the connected
	// Thing's agent, is as the case with a Hub.
	//
	// On success, the agent will publish a property update with the given requestID.
	//
	//	thingID is the thingID as known to the agent
	//	name is the property name as defined in the thing's TD
	//	value is the property value as per property affordance schema
	//	requestID if the associated property update request ID.
	//	senderID is the loginID of the consumer requesting the write
	WriteProperty(thingID, name string, value any, requestID string, senderID string) (status string, err error)
}
