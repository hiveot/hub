package transports

const DefaultHttpsPort = 8444
const DefaultMqttTcpPort = 8883
const DefaultMqttWssPort = 8884

// HTTP protoocol constants
const (
	// StatusHeader contains the result of the request, eg Pending, Completed or Failed
	StatusHeader = "status"
	// RequestIDHeader for transports that support headers can include a message-ID
	RequestIDHeader = "request-id"
	// ConnectionIDHeader identifies the client's connection in case of multiple
	// connections from the same client.
	ConnectionIDHeader = "connection-id"
	// DataSchemaHeader to indicate which  'additionalresults' dataschema being returned.
	DataSchemaHeader = "dataschema"

	// HTTP Paths for auth.
	// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
	// The hub client will need the TD (ConsumedThing) to determine the paths.
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"

	// paths for the various clients
	SSESCPathPrefix = "/ssesc"

	// Generic form href that maps to all operations for the http client, using URI variables
	GenericHttpHRef = "/digitwin/{operation}/{thingID}/{name}"
)

// ReplyToHandler is a server-side handler for sending replies to requests that
// expect one. E.g: InvokeAction and WriteProperty (tbd)
// The server side transport protocol implements the most efficient way to send that reply
// back to the client, either as an immediate result or asynchronously after the
// request has returned.
//
//	status is the progress status: RequestDelivered,RequestFailed,RequestCompleted
//	output is the result if status is RequestCompleted. Can be nil.
//	err is the error in case status is RequestFailed
//type ReplyToHandler func(status string, output any, err error)

// ServerMessageHandler handles a request. The handler is responsible for
// sending a reply to the sender.
//
//	msg is the envelope that contains the request to process.
//	replyTo is the connection for sending a reply, or nil when no reply should be sent.
//
// type ServerMessageHandler func(msg *ThingMessage, replyTo IServerConnection)

// ServerMessageHandler handles a request. The handler either returns a result
// immediately, if available, or sends it asynchronously to the replyTo address.
//
//	msg is the envelope that contains the request to process.
//	replyTo is the connection-ID for sending a reply to the sender, or nil when
//	no reply should be sent.
//
// This returns a flag whether the message handling is completed, potential output or an error
// if completed is false then an async response on the replyTo client is expected.
// Use cm.GetConnectionByConnectionID to obtain the connection to send a response.
type ServerMessageHandler func(msg *ThingMessage, replyTo string) (completed bool, output any, err error)

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

	// SendNotification sends a notification to the client without expecting a response.
	// Intended to send updates to consumers.
	//
	// operation is the operation to invoke
	// thingID of the thing the operation applies to
	// name of the affordance the operation applies to
	// data contains the notification data as described in the TD affordance.
	SendNotification(operation string, dThingID, name string, data any)

	// SendRequest sends a request (action, write property) to the client (agent).
	//
	// Unlike the client's SendRequest, this expects a result asynchronously to prevent
	// resources to be depleted if no reply comes.
	// The client MUST send a response message and include the provided requestID.
	// Intended to send requests to agents.
	//
	// operation identifies the request to invoke.
	// thingID of the thing the operation applies to
	// name of the affordance the operation applies to
	// input data of the request as per affordance
	// requestID is the message requestID to use in the response.
	//
	// This returns an error if the agent isn't reachable
	SendRequest(operation string, thingID, name string, input any, requestID string) error

	// SendResponse send a response to the client for a previous sent request.
	// Typically used in sending a reply to a invokeaction request.
	//	output is the response data
	//	requestID contains the requestID provided in the request.
	SendResponse(thingID, name string, output any, err error, requestID string) error
}
