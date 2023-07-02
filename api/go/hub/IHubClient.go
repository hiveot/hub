package hub

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/api/go/thing"
)

// predefined event IDs start with '$'
const (
	// EventIDProperties is an event that contains a map of all Thing properties that have changed
	EventIDProperties = "$properties"

	// EventIDTD is the event under which a TD document is sent
	EventIDTD = "$td"
)

// EventMessage for subscribers
type EventMessage struct {
	// Authenticated ClientID of the device or service publishing the event
	PublisherID string `yaml:"publisherID"`
	// Optional ThingID of the Thing that generated the event
	ThingID string `yaml:"thingID,omitempty"`
	// EventID of the event as defined in the TD document
	EventID string `yaml:"eventID"`
	// Optional event payload as defined in the TD document
	Payload []byte `yaml:"payload,omitempty"`
	// Timestamp the event was created
	Timestamp int64 `yaml:"timestamp"`
}

// ActionMessage for subscribers
type ActionMessage struct {
	// Authenticated ClientID of the user or service publishing the action
	PublisherID string `yaml:"senderID"`
	// ClientID of the device or service receiving the message
	DestinationID string `yaml:"destinationID"`
	// Optional ThingID of the Thing handling the action
	ThingID string `yaml:"thingID,omitempty"`
	// ActionID of the action as defined in the TD document
	ActionID string `yaml:"actionID"`
	// Optional action payload as defined in the TD document
	Payload []byte `yaml:"payload,omitempty"`
	// Timestamp the action was issued
	Timestamp int64 `yaml:"timestamp"`
}

// IHubClient interface of the golang hub messaging client
type IHubClient interface {

	// ConnectWithCert to the Hub with a client certificate.
	//
	// The client certificate must be signed by the hub CA.
	// The url supports various formats.
	//  "" for the default address and port, eg localhost:port
	//  ":port" when using a specific port
	//  "address"  when using an address with the default port
	//  "schema://address:port/path for the full url
	//
	//	url of the Hub. Use "" for the default url.
	//	clientID to connect as
	//	clientCert for certificate based authentication
	//	caCert of the server
	ConnectWithCert(url string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate) (err error)

	// ConnectWithPassword connects to the Hub server using a login ID and password.
	//
	// The url supports various formats.
	//  "" for the default address and port, eg localhost:defaultport
	//  ":port" when using a specific port on localhost
	//  "address"  when using an address with the default port
	//  "schema://address:port/path for the full url
	//
	// Provide a CA certificate if available. If nil then the connection will still
	// use TLS but no server verification will be used (InsecureSkipVerify=true)
	ConnectWithPassword(url string, loginID string, password string, caCert *x509.Certificate) (err error)

	// ConnectWithJWT connects to the Hub server using a user JWT credentials token
	//
	// The url supports various formats. Use "schema://address:port/path for the full url.
	//
	// Provide a CA certificate if available. If nil then the connection will still
	// use TLS but no server verification will be used (InsecureSkipVerify=true)
	ConnectWithJWT(url string, token string, caCert *x509.Certificate) (err error)

	// ConnectUnauthenticated connects to the Hub server as an unauthenticated user.
	// Unauthenticated users can only use methods that explicitly describe they are for unauthorized users,
	// and can only subscribe to the "unauthenticated" group.
	//
	// The url supports various formats. Use "schema://address:port/path for the full url.
	//
	// Provide a CA certificate if available. If nil then the connection will still
	// use TLS but no server verification will be used (InsecureSkipVerify=true)
	ConnectUnauthenticated(url string, caCert *x509.Certificate) (err error)

	// Disconnect from the hub server
	Disconnect()

	// PubAction publishes a request for action.
	//
	// The client's authentication ID will be included as the publisher ID of the action.
	//
	//	destinationID is the deviceID or serviceID that handles the action
	//	thingID is the destination thingID that handles the action
	//  actionID is the ID of the action as described in the Thing's TD
	//  payload is the optional payload of the action as described in the Thing's TD
	// This returns a response payload if successful and a response is given.
	PubAction(destinationID string, thingID string, actionID string, payload []byte) ([]byte, error)

	// PubEvent publishes the given event. The payload is an event value as per TD document.
	//
	// The client's authentication ID will be included as the publisher ID of the event.
	//
	// thingID is the ID of the 'thing' whose event to publish. This is the ID under which the
	// TD document is published that describes the thing. It can be the ID of the sensor, actuator
	// or service.
	//
	// eventID is the key of the event described in the TD document 'events' section,
	// or one of the predefined events listed above as EventIDXyz
	//
	//  thingID of the Thing whose event is published
	//  eventID is one of the predefined events as described in the Thing TD
	//  value is the serialized event value, or nil if the event has no value
	PubEvent(thingID string, eventID string, value []byte) (err error)

	// PubTD publishes ann event with a Thing TD document.
	// The client's authentication ID will be included as the publisher ID of the event.
	PubTD(td *thing.TD) error

	// SubActions subscribes to actions directed requested of this client.
	// The supported actions are defined in the TD document of the things this client
	// has published.
	// Returns optional response payload that is sent as a reply
	SubActions(cb func(msg *ActionMessage) ([]byte, error)) error

	// SubEvent subscribes to events send by a publisher and/or thing
	// At least one publisherID, or thingID or eventID must be provided.
	//
	//  publisherID optional ID of the device or service publishing the event, or "" for any
	//  thingID ID of the Thing generating the event, or "" for any
	//  eventID optional ID of the event to subscribe to, or "" for any.
	//SubEvent(publisherID string, thingID string, eventID string, cb func(msg EventMessage)) error

	// SubGroup subscribes to events from things in a group the client is a member of.
	//
	// Groups are backed by a store that retains messages for a limited duration.
	// This is a JetStream stream in NATS.
	//
	//  groupName is the group to subscribe to.
	//  thingID optional ID of the Thing generating the event, or "" for any
	//  eventID optional ID of the event to subscribe to, or "" for any.
	SubGroup(groupName string, thingID string, eventID string, cb func(msg *EventMessage)) error
}
