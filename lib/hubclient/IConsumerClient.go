package hubclient

import (
	"crypto/x509"
	"github.com/hiveot/hub/lib/keys"
)

// PingMessage can be used by the server to ping the client that the connection is ready
const PingMessage = "ping"

// StatusHeader for transports that support headers can include a progress status field
const StatusHeader = "status"

// MessageIDHeader for transports that support headers can include a message-ID
const MessageIDHeader = "message-id"

// ConnectionIDHeader identifies the client's connection in case of multiple
// connections in the same session. Used to identify the connection for subscriptions.
const ConnectionIDHeader = "cid"

// DataSchemaHeader for transports that support headers can include a dataschema
// header to indicate an 'additionalresults' dataschema being returned.
const DataSchemaHeader = "dataschema"

type ConnectionStatus string

const (
	// Connecting attempting a (re)connection
	Connecting ConnectionStatus = "connecting"
	// Connected and authenticated successful
	Connected ConnectionStatus = "connected"
	// ConnectFailed after failure to connect
	// Only used if retry has given up
	ConnectFailed ConnectionStatus = "connectFailed"
	// Disconnected by client or not yet connected
	Disconnected ConnectionStatus = "disconnected"
	// Expired authentication token
	Expired ConnectionStatus = "expired"
	// Unauthorized login name or password
	Unauthorized ConnectionStatus = "unauthorized"
)

// TransportStatus connection status of a hub client transport
type TransportStatus struct {
	// URL of the hub
	HubURL string
	// CA used to connect
	CaCert *x509.Certificate
	// the client ID to identify as
	//ClientID string

	// The current connection status
	ConnectionStatus ConnectionStatus
	// The last connection error message, if any
	LastError error

	// flags indicating the supported protocols
	SupportsCertAuth     bool
	SupportsPasswordAuth bool
	SupportsKeysAuth     bool
	SupportsTokenAuth    bool
}

// EventHandler processes an event without return value
type EventHandler func(msg *ThingMessage) error

// MessageHandler defines the method that processes an action or event message and
// returns a delivery status.
//
// As actions are targeted to an agent, the delivery status is that of delivery	to the agent.
// As events are broadcast, the delivery status is that of delivery to at least one subscriber.
type MessageHandler func(msg *ThingMessage) DeliveryStatus

// IConsumerClient defines the interface of the client that connects to a messaging server.
//
// TODO split this up in pure transport, consumed thing and exposed thing apis
type IConsumerClient interface {
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
	CreateKeyPair() (kp keys.IHiveKey)

	// Disconnect from the messaging server.
	// This retains the session and allows a reconnect with the session token.
	// See Logout for invalidating existing sessions
	Disconnect()

	// GetClientID returns the agent or user clientID for this hub client
	GetClientID() string

	// GetStatus returns the current transport connection status
	GetStatus() TransportStatus

	// GetProtocolType returns the protocol type supported by this transport
	// "https", "mqtt", "coap", ...
	// See also: https://www.w3.org/TR/wot-binding-templates/#protocol-bindings-table
	GetProtocolType() string

	// InvokeAction [consumer] invokes an action request and returns as soon as the
	// request is delivered to the Hub.
	//
	// There are two main use-cases for this.
	// 1. Invoke an action on an IoT device and receive confirmation of delivery and
	//    whether it was successful. The initial delivery 'delivered' status will be returned
	//    immediately if the device agent is connected. The 'completed' status will be
	//    sent asynchronously. This can take a while if the device is asleep.
	// 2. Invoke a method on a service and retrieve a result.
	//    In most cases it is recommended to use the 'Rpc' method for this as it waits
	//    and correlates the delivery status event with the request and converts
	//    the result to the appropriate type.
	//
	// Embedded services respond with a completed status and the unmarshalled
	// result in the Reply field.
	// Actions aimed to IoT devices and non embedded services will return a delivery status
	// update separately through a delivery event.
	//
	//	url to invoke
	//  data with action input as defined in its TD
	//
	// This returns a delivery status with response data if delivered
	InvokeAction(thingID string, name string, data any, messageID string) DeliveryStatus

	// Logout of the hub and invalidate the connected session token
	Logout() error

	// Observe adds a subscription for properties from the given ThingID.
	// Use SetMessageHandler to receive property update messages.
	//
	//  dThingID is the digital twin Thing ID of the Thing to observe.
	//	name of the property to observe as described in the TD or "" for all properties
	Observe(dThingID string, name string) error

	// RefreshToken refreshes the authentication token
	// The resulting token can be used with 'SetAuthToken'
	RefreshToken(oldToken string) (newToken string, err error)

	// Rpc [consumer] makes a RPC call using an action and waits for a delivery
	// confirmation event.
	//
	// This is equivalent to use PubAction to send the request, use SetMessageHandler
	// to receive the delivery confirmation event and match the 'messageID' from the
	// delivery status event with the status returned by the action request.
	//
	// The arguments and responses are defined in structs (same approach as gRPC) which are
	// defined in the service api. This struct can also be generated from the TD document
	// if available at build time. See cmd/genapi for the CLI.
	//
	//	dThingID is the digital twin ID of the service providing the RPC method
	//	name of the RPC method as described in the service TD action affordance
	//	args is the struct or type containing the arguments to marshal
	//	resp is the address of a struct or type receiving the result values
	//
	// This returns an error if delivery failed or an error was returned
	Rpc(dThingID string, name string, args interface{}, resp interface{}) error

	// SendOperation [consumer] is form-based method of invoking an operation
	// This is under development.
	//SendOperation(href string, op tdd.Form, data any, messageID string) DeliveryStatus

	// SetMessageHandler adds a handler for messages from the hub.
	// This replaces any previously set handler.
	// The handler should return a DeliveryStatus response for action and
	// property messages. This response is ignored for events.
	//
	// To receive events use the 'Subscribe' method to set the events to listen for.
	// To receive property updates use 'Observe'.
	// For agents to receive actions, no subscription is necessary.
	SetMessageHandler(cb MessageHandler)

	// SetConnectHandler sets the notification handler of connection status changes
	SetConnectHandler(cb func(status TransportStatus))

	// Subscribe adds a subscription for events from the given ThingID.
	// Use SetMessageHandler to receive the event messages
	//
	//  dThingID is the digital twin Thing ID of the Thing to subscribe to.
	//	name of the event to subscribe as described in the TD or "" for all events
	Subscribe(dThingID string, name string) error

	// Unsubscribe [consumer] removes a previous event subscription.
	// dThingID and key must match that of Subscribe
	Unsubscribe(dThingID string, name string) error

	// Unobserve [consumer] removes a previous property subscription.
	// dThingID and key must match that of Observe
	Unobserve(dThingID string, name string) error

	// WriteProperty [consumer] publishes a property change request
	//
	//	dThingID is the digital twin thingID whose property to write
	//	name is the name of the property to write
	//	Value is a value based on the PropertyAffordances in the TD
	// This returns the delivery status and an error code if delivery fails
	WriteProperty(dThingID string, name string, value any) DeliveryStatus
}
