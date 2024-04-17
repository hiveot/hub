package api

// IMessageTransport defines the function for posting action messages to
// the hub and services by clients.
// This encodes the request arguments and decodes the response into the reply struct.
type IMessageTransport func(thingID string, method string, args interface{}, reply interface{}) error
