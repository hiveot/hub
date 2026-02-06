package converters

import (
	"github.com/hiveot/hub/lib/messaging"
	jsoniter "github.com/json-iterator/go"
)

// Passthrough message converter simply passes request, response and notification
// messages as-is. Intended to be used when no WoT exists.
// This implements the IMessageConverter interface
type PassthroughMessageConverter struct {
}

// DecodeNotification passes the notification message as-is
// Raw is the json serialized encoded message
func (svc *PassthroughMessageConverter) DecodeNotification(raw []byte) *messaging.NotificationMessage {

	var notif messaging.NotificationMessage
	err := jsoniter.Unmarshal(raw, &notif)
	//err := tputils.DecodeAsObject(msg, &notif)
	if err != nil || notif.MessageType != messaging.MessageTypeNotification {
		return nil
	}
	return &notif
}

// DecodeRequest passes the request message as-is
// Raw is the json serialized encoded message
func (svc *PassthroughMessageConverter) DecodeRequest(raw []byte) *messaging.RequestMessage {

	var req messaging.RequestMessage
	err := jsoniter.Unmarshal(raw, &req)
	//err := tputils.DecodeAsObject(msg, &req)
	if err != nil || req.MessageType != messaging.MessageTypeRequest {
		return nil
	}
	return &req
}

// DecodeResponse passes the response message as-is
// Raw is the json serialized encoded message
func (svc *PassthroughMessageConverter) DecodeResponse(
	raw []byte) *messaging.ResponseMessage {

	var resp messaging.ResponseMessage
	err := jsoniter.Unmarshal(raw, &resp)
	if err != nil || resp.MessageType != messaging.MessageTypeResponse {
		return nil
	}
	return &resp
}

// EncodeNotification passes the notification message as-is
func (svc *PassthroughMessageConverter) EncodeNotification(req *messaging.NotificationMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = messaging.MessageTypeNotification
	return req, nil
}

// EncodeRequest passes the request message as-is
func (svc *PassthroughMessageConverter) EncodeRequest(req *messaging.RequestMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = messaging.MessageTypeRequest
	return req, nil
}

// EncodeResponse passes the response message as-is
func (svc *PassthroughMessageConverter) EncodeResponse(resp *messaging.ResponseMessage) any {
	// ensure this field is present as it is needed for decoding
	resp.MessageType = messaging.MessageTypeResponse
	return resp
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *PassthroughMessageConverter) GetProtocolType() string {
	return messaging.ProtocolTypeWSS
}

// Create a new instance of the hiveot passthrough message converter
func NewPassthroughMessageConverter() *PassthroughMessageConverter {
	return &PassthroughMessageConverter{}
}
