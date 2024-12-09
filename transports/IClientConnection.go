// Package transports with the interface of a client transport connection
package transports

// Supported transport protocol bindings types
const (
	ProtocolTypeHTTPS    = "https"
	ProtocolTypeSSE      = "sse"   // subprotocol of https
	ProtocolTypeSSESC    = "ssesc" // subprotocol of https
	ProtocolTypeWSS      = "wss"   // subprotocol of https
	ProtocolTypeMQTTS    = "mqtts"
	ProtocolTypeEmbedded = "embedded" // for testing
)

// NotificationHandler processes a message without a response
type NotificationHandler func(msg *ThingMessage)

// RequestHandler processes a request and returns a response.
//
//	msg is the envelope that contains the request to process
type RequestHandler func(msg *ThingMessage) (output any, err error)

// TransportStatus connection status of a hub client transport
//type TransportStatus struct {
//	// URL of the hub
//	HubURL string
//	// CA used to connect
//	CaCert *x509.Certificate
//	// the client ID to identify as
//	//SenderID string
//
//	// The current connection status
//	isConnected bool
//
//	// The last connection error message, if any
//	LastError error
//
//	// flags indicating the supported protocols
//	SupportsCertAuth     bool
//	SupportsPasswordAuth bool
//	SupportsKeysAuth     bool
//	SupportsTokenAuth    bool
//}

// IClientConnection defines the client interface of a transport protocol binding
type IClientConnection interface {
	// ConnectWithClientCert connects to the server using a client certificate.
	// This authentication method is optional
	//ConnectWithClientCert(kp keys.IHiveKey, cert *tls.Certificate) (err error)

	// ConnectWithPassword connects to the messaging server using password authentication.
	// If a connection already exists it will be closed first.
	//
	// This returns a connection token that can be used with ConnectWithToken.
	//
	//  password is created when registering the user with the auth service.
	//
	// This authentication method must be supported by all transport implementations.
	ConnectWithPassword(password string) (newToken string, err error)

	// ConnectWithToken connects to the messaging server using an authentication token.
	//
	// If a connection already exists it will be closed first.
	//
	// and pub/private keys provided when creating an instance of the hub client.
	//  token is created by the auth service.
	//
	// This authentication method must be supported by all transport implementations.
	ConnectWithToken(token string) (newToken string, err error)

	// CreateKeyPair returns a new set of serialized public/private key pair.
	//  serializedKP contains the serialized public/private key pair
	//  pubKey contains the serialized public key to be shared
	//CreateKeyPair() (kp keys.IHiveKey)

	// Disconnect from the messaging server.
	// This retains the session and allows a reconnect with the session token.
	// See Logout for invalidating existing sessions
	Disconnect()

	// GetClientID returns the agent or user clientID for this hub client
	GetClientID() string

	// GetConnectionID returns the client's connection ID belonging to this endpoint
	GetConnectionID() string

	// GetProtocolType returns the (sub)protocol type implemented by this client
	// See the  ProtocolType constants defined above.
	// See also: https://www.w3.org/TR/wot-binding-templates/#protocol-bindings-table
	GetProtocolType() string

	// GetServerURL returns the schema://address:port of the server
	GetServerURL() string

	// InvokeAction sends an action request and waits for a response or until timeout.
	// This is short for SendRequest(wot.OpInvokeAction, ...)
	InvokeAction(dThingID, name string, input any, output any) error

	// IsConnected returns true when the connection is active
	IsConnected() bool

	// Logout of the hub and invalidate its session.
	// This will disconnect all existing connections from the hub and invalidate
	// all authentication tokens.
	// The client has to use login with password to reauthenticate.
	Logout() error

	// RefreshToken refreshes the authentication token
	// The resulting token can be used with 'ConnectWithToken'
	RefreshToken(oldToken string) (newToken string, err error)

	// SendNotification sends a notification to subscribers.
	// Notifications do not receive a response.
	//
	// operation is the operation to invoke
	// thingID of the thing the operation applies to
	// name of the affordance the operation applies to
	// data contains the notification data as described in the TD affordance.
	//
	// This returns an error if the notification could not be delivered to the server.
	SendNotification(operation string, thingID, name string, data any) error

	// SendRequest sends an operation and waits for a response or until timeout.
	SendRequest(operation string, thingID, name string, input any, output any) error

	// SendResponse agent sends a response to a request.
	//
	// Intended for agents to send the response to a request.
	//	thingID of the thing the operation applies to
	//	name of the affordance the operation applies to
	//	output is the response data
	//	err in case the response is an error. Output can contain additional details.
	//	requestID contains the requestID provided in the request.
	SendResponse(thingID, name string, output any, err error, requestID string)

	// SetNotificationHandler sets the handler for notification message from the server.
	// This replaces any previously set handler.
	SetNotificationHandler(cb NotificationHandler)

	// SetRequestHandler sets the handler for operations that return a response.
	// This replaces any previously set handler.
	SetRequestHandler(cb RequestHandler)

	// SetConnectHandler sets the notification handler of connection status changes
	SetConnectHandler(cb func(connected bool, err error))

	// ObserveProperty changes to a property.
	//
	// The protocol binding handles this as per its specification
	// This is a convenience method and short for
	//  SendNotification(wot.ObserveProperty,thingID,name,nil)
	// name is the property to subscribe to or "" for all properties
	ObserveProperty(thingID string, name string) error
	// UnobserveProperty removes a previous observe of a property
	UnobserveProperty(thingID string, name string) error
	// Subscribe to one or all events of a thing
	// name is the event to subscribe to or "" for all events
	Subscribe(thingID string, name string) error
	// Unsubscribe from previous subscription
	Unsubscribe(thingID string, name string) error
}
