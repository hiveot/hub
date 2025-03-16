package wssserver

import (
	"github.com/hiveot/hub/messaging"
	jsoniter "github.com/json-iterator/go"
)

// Hiveot Native message converter is just a pass-through
// This implements the IMessageConverter interface

type HiveotMessageConverter struct {
}

// DecodeNotification converts a native protocol received message to a hiveot notification message.
// Raw is the json serialized encoded message
func (svc *HiveotMessageConverter) DecodeNotification(raw []byte) *messaging.NotificationMessage {

	var notif messaging.NotificationMessage
	err := jsoniter.Unmarshal(raw, &notif)
	//err := tputils.DecodeAsObject(msg, &notif)
	if err != nil || notif.MessageType != messaging.MessageTypeNotification {
		return nil
	}
	return &notif
}

// DecodeRequest converts a native protocol received message to a hiveot request message.
// Raw is the json serialized encoded message
func (svc *HiveotMessageConverter) DecodeRequest(raw []byte) *messaging.RequestMessage {

	var req messaging.RequestMessage
	err := jsoniter.Unmarshal(raw, &req)
	//err := tputils.DecodeAsObject(msg, &req)
	if err != nil || req.MessageType != messaging.MessageTypeRequest {
		return nil
	}
	return &req
}

// DecodeResponse converts a native protocol received message to a hiveot response message.
// Raw is the json serialized encoded message
func (svc *HiveotMessageConverter) DecodeResponse(
	raw []byte) *messaging.ResponseMessage {

	var resp messaging.ResponseMessage
	err := jsoniter.Unmarshal(raw, &resp)
	if err != nil || resp.MessageType != messaging.MessageTypeResponse {
		return nil
	}
	return &resp
}

// EncodeNotification converts a hiveot RequestMessage to protocol equivalent message
func (svc *HiveotMessageConverter) EncodeNotification(req *messaging.NotificationMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = messaging.MessageTypeNotification
	return req, nil
}

// EncodeRequest converts a hiveot RequestMessage to protocol equivalent message
func (svc *HiveotMessageConverter) EncodeRequest(req *messaging.RequestMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = messaging.MessageTypeRequest
	return req, nil
}

// EncodeResponse converts a hiveot ResponseMessage to protocol equivalent message
func (svc *HiveotMessageConverter) EncodeResponse(resp *messaging.ResponseMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	resp.MessageType = messaging.MessageTypeResponse
	return resp, nil
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *HiveotMessageConverter) GetProtocolType() string {
	return messaging.ProtocolTypeWSS
}
