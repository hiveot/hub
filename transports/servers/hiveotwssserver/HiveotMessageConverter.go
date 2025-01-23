package hiveotwssserver

import (
	"github.com/hiveot/hub/transports"
	jsoniter "github.com/json-iterator/go"
)

// Hiveot Native message converter is just a pass-through
// This implements the IMessageConverter interface

type HiveotMessageConverter struct {
}

// ProtocolToHiveot converts a native protocol received message to a hiveot
// request or response message.
// Raw is the json serialized encoded message
func (svc *HiveotMessageConverter) ProtocolToHiveot(
	raw []byte) (*transports.RequestMessage, *transports.ResponseMessage, error) {

	// FIXME: worst case this is triple unmarshalling

	var req transports.RequestMessage
	err := jsoniter.Unmarshal(raw, &req)
	//err := tputils.DecodeAsObject(msg, &req)
	if err != nil {
		return nil, nil, err
	}
	if req.MessageType == transports.MessageTypeRequest {
		return &req, nil, err
	} else {
		var resp transports.ResponseMessage
		err := jsoniter.Unmarshal(raw, &resp)
		//err = tputils.DecodeAsObject(msg, &resp)
		return nil, &resp, err
	}
}

// RequestToProtocol converts a hiveot RequestMessage to protocol equivalent message
func (svc *HiveotMessageConverter) RequestToProtocol(req *transports.RequestMessage) (any, error) {
	return req, nil
}

// ResponseToProtocol converts a hiveot ResponseMessage to protocol equivalent message
func (svc *HiveotMessageConverter) ResponseToProtocol(resp *transports.ResponseMessage) (any, error) {
	return resp, nil
}
