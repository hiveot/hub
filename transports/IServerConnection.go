package transports

const DefaultHttpsPort = 8444
const DefaultMqttTcpPort = 8883
const DefaultMqttWssPort = 8884

// Supported transport protocol bindings types
const (
	ProtocolTypeHTTPS    = "https"
	ProtocolTypeSSE      = "sse"   // subprotocol of https
	ProtocolTypeSSESC    = "ssesc" // subprotocol of https
	ProtocolTypeWSS      = "wss"   // subprotocol of https
	ProtocolTypeMQTTS    = "mqtts"
	ProtocolTypeEmbedded = "embedded" // for testing
)

// ServerNotificationHandler handles an incoming notification from an agent.
// The handler delivers the notification to subscribers.
//
// Note that the ThingID in the notification is that of the agent, not the digital
// twin ThingID.
//
//	senderID is the authenticated ID of the agent sending the notification
//	msg is the notification message envelope
//
// There is no error result as this is a broadcast.
type ServerNotificationHandler func(senderID string, msg NotificationMessage)

// ServerRequestHandler handles an incoming request. The handler either returns a result
// immediately, if available, or sends it asynchronously to the replyTo address.
//
//	msg is the envelope that contains the request to process.
//	replyTo is the connection-ID for sending an asynchronous response back to
//	the sender, or nil when no reply should be sent.
//
// This returns a response message with a Status field indicating whether the message
// handling is in pending, running, completed, or failed.
// Use cm.GetConnectionByConnectionID(replyTo) to obtain the connection to send
// an async response.
type ServerRequestHandler func(msg RequestMessage, replyTo string) ResponseMessage

// ServerResponseHandler handles an incoming response to a request from an agent.
// The handler delivers the response to the client that sent the original request.
// Note that the ThingID in the response is that of the agent, not the digital
// twin ThingID.
//
//	senderID is the authenticated ID of the agent sending the response
//
// This returns an error if the client is not reachable. This can be used to
// retry sending the response or dispose of it altogether.
type ServerResponseHandler func(senderID string, msg ResponseMessage) error

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

	// SendNotification sends a notification to the client without a response.
	// Intended to send updates to consumers.
	//
	// operation is the operation to invoke
	// thingID of the thing the operation applies to
	// name of the affordance the operation applies to
	// data contains the notification data as described in the TD affordance.
	SendNotification(msg NotificationMessage)

	// SendRequest sends a request (action, write property) to the connecting agent.
	//
	// A ResponseMessage MUST be sent by the client when the request is handled,
	// including the provided requestID.
	//
	// msg contains the request information
	// This returns an error if the agent isn't reachable
	SendRequest(msg RequestMessage) error

	// SendResponse send a response message to the client as a reply to a previous request.
	//
	//	 msg contains the response information
	// This returns an error if the agent isn't reachable
	SendResponse(msg ResponseMessage) error

	// SetRequestHandler [agent] sets the handler for operations that return a response.
	// This replaces any previously set handler.
	//SetRequestHandler(request ServerRequestHandler)

	// SetNotificationHandler [consumer] sets the callback for receiving notifications.
	// This replaces any previously set handler.
	//SetNotificationHandler(cb ServerNotificationHandler)

	// SetResponseHandler [consumer] sets the callback for receiving unhandled responses
	// to requests. If a request is sent with 'sync' set to true then SendRequest
	// will handle the response instead.
	//
	// This replaces any previously set handler.
	//SetResponseHandler(cb ServerResponseHandler)
}
