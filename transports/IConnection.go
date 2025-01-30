package transports

// Supported transport protocol bindings types
const (
	// WoT http basic protocol without return channel
	ProtocolTypeWotHTTPBasic = "wot-http-basic"
	// WoT http SSE subprotocol return channel (not implemented)
	//ProtocolTypeWotSSE = "wot-sse"
	// WoT http websocket subprotocol based on strawman proposal
	ProtocolTypeWotWSS = "wot-wss"
	// WoT MQTT protocol over WSS
	ProtocolTypeWotMQTTWSS = "mqtt-wss"

	// HiveOT http SSE subprotocol return channel with direct messaging
	ProtocolTypeHiveotSSE = "hiveot-sse"
	// HiveOT http WSS subprotocol with direct messaging
	ProtocolTypeHiveotWSS = "hiveot-wss"
	// Internal embedded direct call, for testing
	ProtocolTypeHTEmbedded = "embedded" // for testing
)

// ConnectionHandler handles a change in connection status
//
//	connected is true when connected without errors
//	err details why connection failed
//	c is the connection instance being established or disconnected
type ConnectionHandler func(connected bool, err error, c IConnection)

// RequestHandler agent processes a request and returns a response.
//
//	req is the envelope that contains the request to process
//	c is the connection on which the request arrived and on which to send
//	asynchronous response(s).
type RequestHandler func(req *RequestMessage, c IConnection) (response *ResponseMessage)

// ResponseHandler handles an response to a request, send by an agent.
// The handler delivers the response to the client that sent the original request.
// Note that the ThingID in the response is that of the agent, not the digital
// twin ThingID.
//
// This returns an error if the client is not reachable. This can be used to
// retry sending the response or dispose of it altogether.
type ResponseHandler func(msg *ResponseMessage) error

// IConnection defines the interface of a server or client connection.
// Intended for exchanging messages between servients.
type IConnection interface {

	// Disconnect the client.
	Disconnect()

	// GetClientID returns the authenticated clientID of this connection
	GetClientID() string

	// GetConnectionID returns the client's connection ID belonging to this endpoint
	GetConnectionID() string

	// GetProtocolType returns the name of the protocol of this connection
	// See ProtocolType... constants above for valid values.
	GetProtocolType() string

	// GetConnectURL returns the full URL used to establish this connection
	GetConnectURL() string

	// IsConnected returns the current connection status
	IsConnected() bool

	// SendRequest client sends a request to an agent.
	// This returns an error if the request could not be delivered
	SendRequest(req *RequestMessage) error

	// SendResponse [agent] sends a response to a request.
	// This returns an error if the response could not be delivered
	SendResponse(response *ResponseMessage) error

	// SetConnectHandler sets the callback for connection status changes
	// This replaces any previously set handler.
	SetConnectHandler(handler ConnectionHandler)

	// SetRequestHandler set the handler for receiving requests that return a response.
	// This replaces any previously set handler.
	SetRequestHandler(handler RequestHandler)

	// SetResponseHandler [consumer] sets the callback for receiving unhandled
	// asynchronous responses to requests.
	// If a request is sent with 'sync' set to true then SendRequest will handle
	// the response instead.
	//
	// This replaces any previously set handler.
	SetResponseHandler(handler ResponseHandler)
}
