package messaging

// IMessageConverter converts between the standardized hiveot request, response
// and notification messages, and the underlying protocol specific message format.
//
// Its purpose is to assist in decoupling the consumer from the messaging protocol used.
//
// This is used for the WoT websocket protocol, HttpBasic/SSE-SC protocol, MQTT
// protocol, the native Hiveot transfer and others.
//
// Intended for use by consumers and agents on the client and server side.
type IMessageConverter interface {
	// DecodeNotification converts a protocol message to a hiveot notification message
	// provide the serialized data to avoid multiple unmarshalls
	// This returns nil if this isn't a notification.
	DecodeNotification(raw []byte) *NotificationMessage

	// DecodeRequest converts a protocol message to a hiveot request message
	// provide the serialized data to avoid multiple unmarshalls
	// This returns nil if this isn't a request.
	DecodeRequest(raw []byte) *RequestMessage

	// DecodeResponse converts a protocol message to a hiveot response message.
	// This returns nil if this isn't a response
	DecodeResponse(raw []byte) *ResponseMessage

	// EncodeNotification converts a hiveot NotificationMessage to a native protocol message
	// return an error if the message cannot be converted.
	EncodeNotification(notif *NotificationMessage) (any, error)

	// EncodeRequest converts a hiveot RequestMessage to a native protocol message
	// return an error if the message cannot be converted.
	EncodeRequest(req *RequestMessage) (any, error)

	// EncodeResponse converts a hiveot ResponseMessage to a native protocol message
	// This returns an error response if the message cannot be converted
	EncodeResponse(resp *ResponseMessage) any

	// GetProtocolType provides the protocol type for these messages,
	// eg ProtocolTypeWSS
	GetProtocolType() string
}
