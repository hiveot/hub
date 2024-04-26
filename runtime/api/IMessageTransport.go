package api

// IMessageTransport defines the function for posting action messages to
// the hub and services by clients.
// This encodes the request arguments and decodes the response into the reply struct.
type IMessageTransport func(
	thingID string,
	key string,
	args interface{}, reply interface{}) error

// IEventTransport defines the function for posting events messages to the hub
type IEventTransport func(
	thingID string,
	eventID string,
	args interface{}) error
