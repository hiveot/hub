package transports

// IMessageConverter converts between the standardized hiveot request and response
// messages and the underlying protocol encoded message format, ready for delivery.
//
// This is used for the WoT websocket protocol, HttpBasic/SSE-SC protocol, MQTT
// protocol, the native Hiveot transfer and others.
//
// Intended for use by consumers and agents on the client and server side.
type IMessageConverter interface {
	// ProtocolToHiveot converts a protocol message to a hiveot request or response message
	// provide the serialized data to avoid multiple unmarshalls
	ProtocolToHiveot(raw []byte) (*RequestMessage, *ResponseMessage, error)

	// RequestToProtocol converts a hiveot RequestMessage to a native protocol message
	// return an error if the message cannot be converted.
	RequestToProtocol(req *RequestMessage) (any, error)

	// ResponseToProtocol converts a hiveot ResponseMessage to a native protocol message
	// returns an error if the message cannot be converted
	ResponseToProtocol(resp *ResponseMessage) (any, error)
}
