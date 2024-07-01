package hubclient

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/things"
)

//// ISubscription interface to underlying subscription mechanism
//type ISubscription interface {
//	Unsubscribe() error
//}

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
	ClientID string

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
type EventHandler func(msg *things.ThingMessage) error

// MessageHandler defines the method that processes an action or event message and
// returns a delivery status.
//
// As actions are targeted to an agent, the delivery status is that of delivery	to the agent.
// As events are broadcast, the delivery status is that of delivery to at least one subscriber.
type MessageHandler func(msg *things.ThingMessage) DeliveryStatus

// IHubClient defines the interface of the client that connects to a messaging server.
type IHubClient interface {
	// ClientID returns the agent or user clientID for this hub client
	ClientID() string

	// ConnectWithClientCert connects to the server using a client certificate.
	// This authentication method is optional
	ConnectWithClientCert(kp keys.IHiveKey, cert *tls.Certificate) (err error)

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

	// GetStatus returns the current transport connection status
	GetStatus() TransportStatus

	// Logout of the hub and invalidate the connected session token
	Logout() error

	// PubAction publishes an action request and returns as soon as the request is delivered
	// to the Hub inbox.
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
	//	dThingID the digital twin ID for whom the action is intended
	//	key is the action ID or method name of the action to invoke
	//  value with message data to publish in native format
	//
	// This returns a delivery status with serialized response message if delivered
	PubAction(dThingID string, key string, value any) DeliveryStatus

	// PubEvent publishes an event style message without a response.
	// It returns as soon as delivery to the hub is confirmed.
	// This is intended for agents, not for consumers.
	//
	// Events are published by agents using their native ID, not the digital twin ID.
	// The Hub outbox broadcasts this event using the digital twin ID.
	//
	//	thingID native ID of the thing as used by the agent
	//	key ID of the event
	//	value with native data to publish, as per TD dataschema
	//
	// This returns an error if the event cannot not be delivered to the hub
	PubEvent(thingID string, key string, value any) error

	// PubProperty publishes a property change request
	// Value is a value based on the PropertyAffordances in the TD
	PubProperty(dThingID string, key string, value any) DeliveryStatus

	// PubProps publishes a property values event.
	// It returns as soon as delivery to the hub is confirmed.
	// This is intended for agents, not for consumers.
	//
	//	thingID is the native ID of the device (not including the digital twin ID)
	//	props is the property key-value map to publish where value is the native value
	PubProps(thingID string, props map[string]any) error

	// PubTD publishes an TD document event.
	// It returns as soon as delivery to the hub is confirmed.
	// This is intended for agents, not for consumers.
	//	td is the Thing Description document describing the Thing
	PubTD(td *things.TD) error

	// RefreshToken refreshes the authentication token
	// The resulting token can be used with 'ConnectWithToken'
	RefreshToken(oldToken string) (newToken string, err error)

	// Rpc makes a RPC call using an action and waits for a delivery confirmation event.
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
	//	key is the ID of the RPC method as described in the service TD action affordance
	//	args is the address of a struct containing the arguments to marshal
	//	resp is the address of a struct receiving the result values
	//
	// This returns an error if delivery failed or an error was returned
	Rpc(dThingID string, key string, args interface{}, resp interface{}) error

	// SendDeliveryUpdate sends a delivery progress update to the hub.
	// The hub's inbox will update the status of the action and notify the original sender.
	//
	// Intended for agents that have processed an incoming action request asynchronously
	// and need to send an update on further progress.
	SendDeliveryUpdate(stat DeliveryStatus)

	// SetMessageHandler adds a handler for messages from the hub.
	// This replaces any previously set handler.
	// The handler should return a DeliveryStatus response for action and
	// property messages. This response is ignored for events.
	//
	// To receive events use the 'Subscribe' method to set the events to listen for.
	SetMessageHandler(cb MessageHandler)

	// SetConnectHandler sets the notification handler of connection status changes
	SetConnectHandler(cb func(status TransportStatus))

	// Subscribe adds a subscription for events from the given ThingID.
	//
	// This is for events only. Actions directed to this client are automatically passed
	// to this client's messageHandler.
	//
	//  dThingID is the digital twin ID of the Thing to subscribe to.
	//	key is the type of event to subscribe to or "" for all events
	Subscribe(dThingID string, key string) error

	// Unsubscribe removes a previous event subscription.
	// No more events or requests will be received after Unsubscribe.
	Unsubscribe(dThingID string) error
}
