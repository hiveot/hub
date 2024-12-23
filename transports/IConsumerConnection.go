// Package transports with the interface of a client transport connection
package transports

// IConsumerConnection defines the client interface of a consumer client connection
type IConsumerConnection interface {
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

	// Ping the server and receive confirmation
	// Intended to check if the connection is alive.
	Ping() error

	// RefreshToken refreshes the authentication token
	// The resulting token can be used with 'ConnectWithToken'
	RefreshToken(oldToken string) (newToken string, err error)

	// SendRequest sends a request message.
	// wait is true to wait for a response or false to return immediately
	SendRequest(msg RequestMessage, wait bool) (ResponseMessage, error)

	// SetConnectHandler sets the callback for connection status changes
	// This replaces any previously set handler.
	SetConnectHandler(cb func(connected bool, err error))

	// SetNotificationHandler [consumer] sets the callback for receiving notifications.
	// This replaces any previously set handler.
	SetNotificationHandler(cb NotificationHandler)

	// SetResponseHandler [consumer] sets the callback for receiving unhandled responses
	// to requests. If a request is sent with 'sync' set to true then SendRequest
	// will handle the response instead.
	//
	// This replaces any previously set handler.
	SetResponseHandler(cb ResponseHandler)

	// ObserveProperty [consumer] observes changes to a property.
	//
	// The protocol binding handles this as per its specification
	// This is a convenience method and short for
	//  SendRequest(wot.ObserveProperty,thingID,name,nil)
	// name is the property to subscribe to or "" for all properties
	ObserveProperty(thingID string, name string) error
	// UnobserveProperty removes a previous observe of a property
	UnobserveProperty(thingID string, name string) error
	// Subscribe to one or all events of a thing
	// name is the event to subscribe to or "" for all events
	Subscribe(thingID string, name string) error
	// Unsubscribe from previous subscription
	Unsubscribe(thingID string, name string) error

	// WriteProperty submits a request to modify a property
	// This is short for SendRequest(wot.OpWriteProperty, ...)
	//	thingID is the thing whose property to write
	//	name is the property affordance name
	//	input is the value to write as per affordance dataschema
	//	async is true to return after submitting the request or false to wait for a response
	WriteProperty(thingID, name string, input any, async bool) error
}
