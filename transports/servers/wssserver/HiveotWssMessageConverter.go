package wssserver

import (
	"github.com/hiveot/hub/transports"
	jsoniter "github.com/json-iterator/go"
)

// Hiveot Native message converter is just a pass-through
// This implements the IMessageConverter interface

type HiveotMessageConverter struct {
}

// DecodeRequest converts a native protocol received message to a hiveot request message.
// Raw is the json serialized encoded message
func (svc *HiveotMessageConverter) DecodeRequest(raw []byte) *transports.RequestMessage {

	var req transports.RequestMessage
	err := jsoniter.Unmarshal(raw, &req)
	//err := tputils.DecodeAsObject(msg, &req)
	if err != nil || req.MessageType != transports.MessageTypeRequest {
		return nil
	}
	return &req
}

// DecodeResponse converts a native protocol received message to a hiveot response message.
// Raw is the json serialized encoded message
func (svc *HiveotMessageConverter) DecodeResponse(
	raw []byte) *transports.ResponseMessage {

	var resp transports.ResponseMessage
	err := jsoniter.Unmarshal(raw, &resp)
	if err != nil || resp.MessageType != transports.MessageTypeResponse {
		return nil
	}
	return &resp
}

// EncodeRequest converts a hiveot RequestMessage to protocol equivalent message
func (svc *HiveotMessageConverter) EncodeRequest(req *transports.RequestMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = transports.MessageTypeRequest
	return req, nil
}

// EncodeResponse converts a hiveot ResponseMessage to protocol equivalent message
func (svc *HiveotMessageConverter) EncodeResponse(resp *transports.ResponseMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	resp.MessageType = transports.MessageTypeResponse
	return resp, nil
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *HiveotMessageConverter) GetProtocolType() string {
	return transports.ProtocolTypeHiveotWSS
}
