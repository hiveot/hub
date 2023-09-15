package hubclient

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

// ISubscription interface to underlying subscription mechanism
type ISubscription interface {
	Unsubscribe()
}

// EventMessage for subscribers
type EventMessage struct {
	// ClientID of the device or service publishing the event
	DeviceID string `yaml:"deviceID"`
	// Optional ThingID of the Thing that generated the event
	ThingID string `yaml:"thingID,omitempty"`
	// EventID of the event as defined in the TD document
	EventID string `yaml:"eventID"`
	// Optional event payload as defined in the TD document
	Payload []byte `yaml:"payload,omitempty"`
	// Timestamp the event was created
	Timestamp int64 `yaml:"timestamp"`
}

// ActionRequest message for thing or service subscribers
type ActionRequest struct {
	// ClientID of the authenticated client publishing the request
	ClientID string `yaml:"clientID"`
	// Authenticated ClientID of the device or service that handles the action
	DeviceID string `yaml:"deviceID"`
	// ThingID of the Thing handling the action.
	// For services this is the name of the capability that handles the action.
	ThingID string `yaml:"thingID"`
	// ActionID of the action as defined in the TD document
	ActionID string `yaml:"actionID"`
	// Optional action payload as defined in the TD document
	Payload []byte `yaml:"payload,omitempty"`
	// Timestamp the action was issued
	Timestamp int64 `yaml:"timestamp"`

	// Reply to the received action with optional payload or error
	// This can be called multiple times to send multiple batches.
	SendReply func(payload []byte, err error) error
	// Acknowledge completion of the action
	SendAck func() error
}

// ErrorMessage payload
// Embed this in response messages as it will be used to respond with an error
type ErrorMessage struct {
	Error string `json:"error,omitempty"`
}

// IHubClient interface of the golang hub messaging client
type IHubClient interface {

	// ClientID of the current connection
	ClientID() string

	// ConnectWithCert connects to the messaging server using client certificate authentication
	// Support for client cert auth depends on the server setup.
	ConnectWithCert(brokerURL string,
		clientCert *tls.Certificate, caCert *x509.Certificate) error

	// ConnectWithToken connects to the messaging server using an authentication token obtained with login or refresh
	// The token type depends on the underlying messaging server.
	// The MQTT and NATS callout server uses short lived JWT tokens.
	// If Nats nkey setup is used, this authenticates using nkeys (effectively ignoring the token)
	ConnectWithToken(brokerURL string, token string, caCert *x509.Certificate) error

	// ConnectWithPassword connects to the messaging server using password authentication
	ConnectWithPassword(connectURL string, password string, caCert *x509.Certificate) error

	// Disconnect from the hub server
	Disconnect()

	// ParseResponse parses response message
	// This is a convenience function
	ParseResponse(data []byte, resp interface{}) error

	// Pub allows publication on any topic.
	// Intended for testing or for publishing to special topics.
	Pub(topic string, payload []byte) error

	// PubServiceAction publishes a request for action from a service.
	//
	// The client's ID is used as the publisher ID of the action.
	//
	//	destinationID is the serviceID that handles the action for the thing or service capability
	//	capability is the capability to invoke
	//  actionID is the name of capability action to invoke
	//  payload is the optional payload of the action
	// This returns a response payload if successful and a response is given.
	PubServiceAction(serviceID string, capability string, actionID string, payload []byte) ([]byte, error)

	// PubThingAction publishes a request for action from a thing.
	//
	// The client's ID is used as the publisher ID of the action.
	//
	//	deviceID is the deviceID that handles the action for the thing or service capability
	//	thingID is the destination thingID that handles the action
	//  actionID is the ID of the action as described in the Thing's TD
	//  payload is the optional payload of the action as described in the Thing's TD
	// This returns a response payload if successful and a response is given.
	PubThingAction(deviceID string, thingID string, actionID string, payload []byte) ([]byte, error)

	// PubEvent publishes the given things event. The payload is an event value as per TD document.
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

	// Sub allows subscribing to any topic/subject address that the client is authorized to.
	// Intended for testing or special topics.
	Sub(addr string, cb func(addr string, data []byte)) (ISubscription, error)

	// SubThingEvents subscribes to events sent by a Thing's device.
	// Intended for use by devices to receive requests for its things.
	//
	// The handler receives an action request message with request payload and returns
	// an optional reply or an error when the request wasn't accepted.
	//
	// The supported actions are defined in the TD document of the things this binding has published.
	//  thingID is the device thing or service capability to subscribe to, or "" for wildcard
	//  cb is the callback to invoke
	//
	// The handler receives an action request message with request payload and
	// must reply withwith msg.Reply or msg.Ack, or return an error
	SubThingEvents(deviceID string, thingID string,
		handler func(msg *EventMessage)) (ISubscription, error)

	// SubThingActions subscribes to actions requested of a Thing.
	// Intended for use by devices to receive requests for its things.
	//
	// The handler receives an action request message with request payload and returns
	// an optional reply or an error when the request wasn't accepted.
	//
	// The supported actions are defined in the TD document of the things this binding has published.
	//  thingID is the device thing or service capability to subscribe to, or "" for wildcard
	//  cb is the callback to invoke
	//
	// The handler receives an action request message with request payload and
	// must reply withwith msg.Reply or msg.Ack, or return an error
	SubThingActions(thingID string,
		handler func(msg *ActionRequest) error) (ISubscription, error)

	// SubServiceActions subscribes a service to a requested action.
	// Intended for use by services to receive requests for its capabilities.
	//
	// The handler receives an action request message with request payload and
	// must reply withwith msg.Reply or msg.Ack, or return an error
	//
	// The supported requests are defined in the TD document that the service has published.
	SubServiceActions(capability string,
		handler func(msg *ActionRequest) (err error)) (ISubscription, error)

	// SubStream subscribes to events from things
	//
	// The events stream is backed by a store that retains messages for a limited duration.
	// This is a JetStream stream in NATS.
	//
	// ReceiveLatest is handy to be up to date on all event instead of quering them separately. Only use this if
	// you're going to retrieve them anyways.
	//
	//  name is the stream to subscribe to or "" for the default events stream
	//	receiveLatest to immediately receive the latest event for each event instance
	SubStream(name string, receiveLatest bool, cb func(msg *EventMessage)) (ISubscription, error)
}
