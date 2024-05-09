package transports

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
)

// inbox implementation depends on the underlying transport.
// this is the commonly used name for it.
const MessageTypeINBOX = "_INBOX"

// ISubscription interface to underlying subscription mechanism
type ISubscription interface {
	Unsubscribe() error
}

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

// MessageTypeINBOX special inbox prefix for RPCs
// reserved event and action names
const (
// MessageTypeAction = "action"
// MessageTypeConfig = "config"
// MessageTypeEvent  = "event"
// MessageTypeRPC    = "rpc"
// MessageTypeINBOX  = "_INBOX"
// EventTypeTD       = "$td"
// EventTypeProps    = "$properties"
)

var ErrorUnauthorized = errors.New(string(Unauthorized))

type HubTransportStatus struct {
	// URL of the hub
	HubURL string
	// CA used to connect
	CaCert *x509.Certificate
	// The transport core, eg mqtt or nats
	Core string
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

// IHubTransport defines the interface of the transport client that connects to a messaging server.
type IHubTransport interface {

	// ConnectWithCert connects to the server using a client certificate.
	// This authentication method is optional
	ConnectWithCert(kp keys.IHiveKey, cert *tls.Certificate) (token string, err error)

	// ConnectWithPassword connects to the messaging server using password authentication.
	//  loginID is the client's ID (typically consumers)
	//  password is created when registering the user with the auth service.
	//
	// This authentication method must be supported by all transports
	ConnectWithPassword(password string) error

	// ConnectWithJWT connects to the messaging server using an authentication token
	// and pub/private keys provided when creating an instance of the hub client.
	//  token is created by the auth service.
	//
	// This authentication method must be supported by all transports
	ConnectWithJWT(token string) error

	// CreateKeyPair returns a new set of serialized public/private key pair.
	//  serializedKP contains the serialized public/private key pair
	//  pubKey contains the serialized public key to be shared
	CreateKeyPair() (kp keys.IHiveKey)

	// Disconnect from the messaging server.
	// This removes all subscriptions.
	Disconnect()

	// GetStatus returns the current transport connection status
	GetStatus() HubTransportStatus

	// PubAction publishes an action request and waits for a response.
	//	thingID for whom the action is intended
	//	key ID or method name of the action
	//  payload with serialized message to publish
	//  returns a delivery status with serialized response message if delivered
	PubAction(thingID string, key string, payload []byte) api.DeliveryStatus

	// PubEvent publishes an event style message without waiting for a response.
	//	thingID whose event is published
	//	key ID of the event
	//	payload with serialized message to publish
	PubEvent(thingID string, key string, payload []byte) api.DeliveryStatus

	// SetConnectHandler sets the notification handler of connection status changes
	SetConnectHandler(cb func(status HubTransportStatus))

	// SetMessageHandler set the single handler that receives all subscribed messages
	// and messages directed at this client.
	//
	// Note that Agents receive actions with a thingID that does not have the agent
	// prefix as the agent prefix is consumer facing. Things from agents are physical
	// things while digitwin Things are virtual Things with a different ID.
	//
	// Consumers can only interact with the digital twin.
	//
	// See also 'Subscribe' to set the things this client receives messages for.
	SetMessageHandler(cb api.MessageHandler)

	// Subscribe adds a subscription for one or more events from the thingID.
	//
	// This is for events only. Actions directed to this client are automatically passed
	// to this client's messageHandler. The TD documents published by this agent have
	// their ThingID associated with the agent using this transport.
	//
	//
	// Events will be passed to the event handler.
	// This is pretty coarse grained.
	// Subscriptions remain in effect when the connection with the messaging server is interrupted.
	//  thingID is the ID of the Thing whose events to receive or "" for events from all things
	Subscribe(thingID string) error

	// Unsubscribe removes a previous event subscription.
	// No more events or requests will be received after Unsubscribe.
	Unsubscribe(thingID string)
}
