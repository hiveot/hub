package messaging

import (
	"crypto/x509"
	"time"
)

// Supported transport protocol bindings types
const (
	// WoT http basic protocol without return channel
	ProtocolTypeHTTPBasic = "http-basic"

	// websocket sub-protocol
	ProtocolTypeWSS = "wss"

	// WoT MQTT protocol over WSS
	ProtocolTypeWotMQTTWSS = "mqtt-wss"

	// HiveOT http SSE subprotocol return channel with direct messaging
	ProtocolTypeHiveotSSE = "hiveot-sse"

	// HiveOT message envelope passthrough
	ProtocolTypePassthrough = "passthrough"
)

var UnauthorizedError error = unauthorizedError{}

// UnauthorizedError for dealing with authorization problems
type unauthorizedError struct {
	Message string
}

func (e unauthorizedError) Error() string {
	return "Unauthorized: " + e.Message
}

// ConnectionInfo provides details of a connection
type ConnectionInfo struct {

	// Connection CA
	CaCert *x509.Certificate

	// GetClientID returns the authenticated clientID of this connection
	ClientID string

	// GetConnectionID returns the client's connection ID belonging to this endpoint
	ConnectionID string

	// GetConnectURL returns the full server URL used to establish this connection
	ConnectURL string

	// GetProtocolType returns the name of the protocol of this connection
	// See ProtocolType... constants above for valid values.
	//ProtocolType string

	// Connection timeout settings (clients only)
	Timeout time.Duration
}

// ConnectionHandler handles a change in connection status
//
//	connected is true when connected without errors
//	err details why connection failed
//	c is the connection instance being established or disconnected
type ConnectionHandler func(connected bool, err error, c IConnection)

// NotificationHandler handles a subscruption notification, send by an agent.
//
// retry sending the response at a later time.
type NotificationHandler func(msg *NotificationMessage)

// RequestHandler agent processes a request and returns a response.
//
//	req is the envelope that contains the request to process
//	c is the connection on which the request arrived and on which to send
//	asynchronous response(s).
type RequestHandler func(req *RequestMessage, c IConnection) (response *ResponseMessage)

// ResponseHandler handles a response to a request, send by an agent.
// The handler delivers the response to the client that sent the original request.
//
// This returns an error if the response cannot be delivered. This can be used to
// retry sending the response at a later time.
type ResponseHandler func(msg *ResponseMessage) error

// IConnection defines the interface of a server or client connection.
// Intended for exchanging messages between servients.
type IConnection interface {

	// Disconnect the client.
	Disconnect()

	// GetConnectionInfo return details of the connection
	GetConnectionInfo() ConnectionInfo

	// IsConnected returns the current connection status
	IsConnected() bool

	// SendNotification [agent] sends a notification to subscribers.
	// This returns an error if the notification could not be delivered
	SendNotification(notif *NotificationMessage) error

	// SendRequest client sends a request to an agent.
	// This returns an error if the request could not be delivered
	SendRequest(req *RequestMessage) error

	// SendResponse [agent] sends a response to a request.
	// This returns an error if the response could not be delivered
	SendResponse(response *ResponseMessage) error

	// SetConnectHandler sets the callback for connection status changes
	// This replaces any previously set handler.
	SetConnectHandler(handler ConnectionHandler)

	// SetNotificationHandler [client] sets the callback for receiving notifications.
	// This replaces any previously set handler.
	SetNotificationHandler(handler NotificationHandler)

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
