package transports

// IMessageConverter converts between the standardized hiveot request and response
// messages and the underlying protocol encoded message format, ready for delivery.
//
// This is used for the WoT websocket protocol, HttpBasic/SSE-SC protocol, MQTT
// protocol, the native Hiveot transfer and others.
//
// Intended for use by consumers and agents on the client and server side.
type IMessageConverter interface {
	// DecodeMessage converts a protocol message to a hiveot request or response message
	// provide the serialized data to avoid multiple unmarshalls
	DecodeMessage(raw []byte) (*RequestMessage, *ResponseMessage, error)

	// EncodeRequest converts a hiveot RequestMessage to a native protocol message
	// return an error if the message cannot be converted.
	EncodeRequest(req *RequestMessage) (any, error)

	// EncodeResponse converts a hiveot ResponseMessage to a native protocol message
	// returns an error if the message cannot be converted
	EncodeResponse(resp *ResponseMessage) (any, error)
}
