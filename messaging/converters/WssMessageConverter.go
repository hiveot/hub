package converters

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	jsoniter "github.com/json-iterator/go"
)

// Websocket notification message with all possible fields for all operations
type WssNotificationMessage struct {
	messaging.NotificationMessage
}

// Websocket requests message with all possible fields for all operations
type WssRequestMessage struct {
	messaging.RequestMessage
	// queryaction
	ActionID string `json:"actionID,omitempty"` // input for operation
	// readmultipleproperties: array of property names
	Names []string `json:"names,omitempty"`
	// writeallproperties, writemultipleproperties input
	Values any `json:"values,omitempty"`
}
type WssActionStatus struct {
	ActionID      string                `json:"actionID"`
	Error         *messaging.ErrorValue `json:"error,omitempty"`
	Output        any                   `json:"output,omitempty"` // when completed
	State         string                `json:"state"`
	TimeRequested string                `json:"timeRequested"`
	TimeEnded     string                `json:"timeEnded,omitempty"` // when completed
}

// Websocket response message with all possible fields for all operations
type WssResponseMessage struct {
	messaging.ResponseMessage

	// invokeaction (async), queryaction response contains status
	Status *WssActionStatus `json:"status,omitempty"`

	// queryallactions response:
	Statuses map[string]WssActionStatus `json:"statuses,omitempty"`

	// readallproperties,readmultipleproperties,
	// writeallproperties, writemultipleproperties:
	// object with property name-value pairs
	Values any `json:"values,omitempty"`

	// invokeaction output value (synchronous)
	// for hiveot clients: readallproperties, readmultipleproperties ThingValue map
	Output any `json:"output,omitempty"`
}

// Websocket message converter converts requests, responses and notifications
// between hiveot standardiz envelope and the WoT websocket protocol (draft) messages.
// Websocket messages vary based on the operation.
type WssMessageConverter struct {
}

// DecodeNotification converts a websocket notification to a hiveot notification message.
// Raw is the json serialized encoded message
func (svc *WssMessageConverter) DecodeNotification(raw []byte) *messaging.NotificationMessage {

	var wssnotif WssNotificationMessage
	err := jsoniter.Unmarshal(raw, &wssnotif)
	//err := tputils.DecodeAsObject(msg, &notif)
	if err != nil || wssnotif.MessageType != messaging.MessageTypeNotification {
		return nil
	}
	notifmsg := &wssnotif.NotificationMessage
	switch wssnotif.Operation {
	}
	return notifmsg
}

// DecodeRequest converts a websocket request message to a hiveot request message.
// Raw is the json serialized encoded message.
// Websocket request messages are nearly identical to hiveot, so use passthrough.
// Conversion by operation:
// - cancelaction: copy wss actionID field to input
// - invokeaction: none
// - queryaction: copy wss actionID field to input
// - queryallactions: none
// - writeproperty:
func (svc *WssMessageConverter) DecodeRequest(raw []byte) *messaging.RequestMessage {

	var wssreq WssRequestMessage
	err := jsoniter.Unmarshal(raw, &wssreq)

	//err := tputils.DecodeAsObject(msg, &req)
	if err != nil || wssreq.MessageType != messaging.MessageTypeRequest {
		return nil
	}
	// query/cancel action messages carry an actionID in the request
	// (tentative: https://github.com/w3c/web-thing-protocol/issues/43)
	reqmsg := &wssreq.RequestMessage
	switch wssreq.Operation {

	case vocab.OpQueryAction, vocab.OpCancelAction:
		// input is actionID
		reqmsg.Input = wssreq.ActionID
	}
	return reqmsg
}

// DecodeResponse converts a websocket response message to a hiveot response message.
// Raw is the json serialized encoded message
func (svc *WssMessageConverter) DecodeResponse(
	raw []byte) *messaging.ResponseMessage {

	var wssResp WssResponseMessage
	err := jsoniter.Unmarshal(raw, &wssResp)
	if err != nil {
		slog.Warn("DecodeResponse: Can't unmarshal websocket response", "error", err, "raw", string(raw))
		return nil
	}
	if wssResp.MessageType != messaging.MessageTypeResponse {
		return nil
	}

	respMsg := &wssResp.ResponseMessage

	// if the response is an error response then no need to decode any further
	if respMsg.Error != nil {
		return respMsg
	}

	switch wssResp.Operation {

	case vocab.OpCancelAction:
		// hiveot response API doesnt contain the actionID. This is okay as the sender knows it.
	case vocab.OpInvokeAction:
		// hiveot always returns an ActionStatus object
		//
		// in websocket profile synchronous actions respond with output,
		// while async actions respond with actionID
		as := messaging.ActionStatus{
			Name:    wssResp.Name,
			Output:  wssResp.Output,
			State:   messaging.StatusCompleted,
			ThingID: wssResp.ThingID,
		}
		// if wss contains an actionID the request is pending
		if wssResp.Status != nil {
			as.State = wssResp.Status.State
			as.TimeUpdated = wssResp.Status.TimeRequested
			as.TimeUpdated = wssResp.Status.TimeEnded
		}
		respMsg.Value = as

	case vocab.OpQueryAction:
		// ResponseMessage should contain ActionStatus object
		var wssStatus WssActionStatus
		err = tputils.Decode(wssResp.Status, &wssStatus)
		if err != nil {
			return nil
		}
		if respMsg.Value == nil {
			// non hiveot server
			as := messaging.ActionStatus{
				ActionID:      wssStatus.ActionID,
				Name:          wssResp.Name,
				Output:        wssStatus.Output,
				State:         wssStatus.State,
				ThingID:       wssResp.ThingID,
				TimeRequested: wssStatus.TimeRequested,
				TimeUpdated:   wssStatus.TimeEnded,
			}
			respMsg.Value = as
		}

	case vocab.OpQueryAllActions:
		// ResponseMessage should contain ActionStatus list
		var wssStatusMap map[string]WssActionStatus
		actionStatusMap := make(map[string]messaging.ActionStatus)
		err = tputils.Decode(wssResp.Statuses, &wssStatusMap)
		if err != nil {
			return nil
		}
		for _, wssStatus := range wssStatusMap {
			actionStatusMap[wssResp.Name] = messaging.ActionStatus{
				ThingID:       wssResp.ThingID,
				Name:          wssResp.Name,
				ActionID:      wssStatus.ActionID,
				State:         wssStatus.State,
				TimeRequested: wssStatus.TimeRequested,
				TimeUpdated:   wssStatus.TimeEnded,
				Output:        wssStatus.Output,
			}
		}
		respMsg.Value = actionStatusMap

	case vocab.OpReadAllProperties, vocab.OpReadMultipleProperties,
		vocab.OpWriteMultipleProperties:

		// the 'Value' property from the messaging.ResponseMessage embedded struct
		// already contains the messaging.ThingValue map.
		// But, if the websocket response is from a non-hiveot device then convert
		// the websocket 'Values' field k-v map to ThingValue map
		tvMap := make(map[string]messaging.ThingValue)
		if respMsg.Value == nil {
			wssPropValues := make(map[string]any)
			tputils.DecodeAsObject(wssResp.Values, wssPropValues)
			for k, v := range wssPropValues {
				tv := messaging.ThingValue{
					AffordanceType: messaging.AffordanceTypeProperty,
					Name:           k,
					Data:           v,
					ThingID:        wssResp.ThingID,
					// Timestamp: n/a
				}
				tvMap[tv.Name] = tv
			}
			respMsg.Value = tvMap
		}
	}
	return respMsg
}

// EncodeNotification converts a hiveot RequestMessage to a websocket equivalent message
func (svc *WssMessageConverter) EncodeNotification(notif *messaging.NotificationMessage) (any, error) {
	wssNotif := WssNotificationMessage{
		NotificationMessage: *notif,
	}
	// ensure this field is present as it is needed for decoding
	wssNotif.MessageType = messaging.MessageTypeNotification
	return wssNotif, nil
}

// EncodeRequest converts a hiveot RequestMessage to websocket equivalent message
func (svc *WssMessageConverter) EncodeRequest(req *messaging.RequestMessage) (any, error) {
	wssReq := WssRequestMessage{
		RequestMessage: *req,
		ActionID:       req.CorrelationID,
	}
	// ensure this field is present as it is needed for decoding
	wssReq.MessageType = messaging.MessageTypeRequest
	switch req.Operation {
	case vocab.OpWriteMultipleProperties:
		wssReq.Values = req.Input
	case vocab.OpQueryAction:
		// correlationID is used as actionID
		wssReq.ActionID = req.CorrelationID
	}
	return wssReq, nil
}

// EncodeResponse converts a hiveot ResponseMessage to websocket equivalent message
// This always returns a response
func (svc *WssMessageConverter) EncodeResponse(resp *messaging.ResponseMessage) any {
	wssResp := WssResponseMessage{
		ResponseMessage: *resp,
	}

	// when the response contains an error instead of a reply
	// then there is no data to encode
	if resp.Error != nil {
		return wssResp
	}

	// ensure this field is present as it is needed for decoding
	wssResp.MessageType = messaging.MessageTypeResponse
	switch resp.Operation {
	case vocab.OpCancelAction:
		// actionID of cancelled action ?
		// wssResp.ActionID = resp.CorrelationID
	case vocab.OpInvokeAction:
		// hiveot invokeaction always contains an ActionStatus object in the response
		var as messaging.ActionStatus
		err := tputils.Decode(resp.Value, &as)
		if err != nil {
			wssResp.Error = messaging.ErrorValueFromError(err)
		}
		if as.State == messaging.StatusCompleted {
			// websocket synchronous response
			wssResp.Output = as.Output
		} else {
			// websocket asynchronous response returns ActionID
			wssResp.Status = &WssActionStatus{
				ActionID:      as.ActionID,
				Error:         as.Error, // error fields are identical
				State:         as.State,
				TimeRequested: as.TimeRequested,
			}
		}
	case vocab.OpQueryAction:
		// convert from messaging.ActionStatus to WssActionStatus
		var actionStatus messaging.ActionStatus
		err := tputils.Decode(resp.Value, &actionStatus)
		if err != nil {
			wssResp.Error = messaging.ErrorValueFromError(fmt.Errorf("Response does not contain ActionStatus object: %w", err))
		}
		wssResp.Status = &WssActionStatus{
			ActionID:      actionStatus.ActionID,
			Error:         actionStatus.Error, // error fields are identical
			Output:        actionStatus.Output,
			State:         actionStatus.State,
			TimeRequested: actionStatus.TimeRequested,
			TimeEnded:     actionStatus.TimeUpdated,
		}
	case vocab.OpQueryAllActions, vocab.OpQueryMultipleActions:
		// convert from messaging.ActionStatus map to WssActionStatuses map
		// FIXME: response is api.ActionStatus which differs from messaging.ActionStatus
		var actionStatusMap map[string]messaging.ActionStatus
		err := tputils.Decode(resp.Value, &actionStatusMap)
		if err != nil {
			err = fmt.Errorf("Can't convert ActionStatus map response to websocket type. "+
				"Response does not contain ActionStatus map. "+
				"thingID='%s'; name='%s'; operation='%s'; Received '%s'; Error='%s'",
				resp.ThingID, resp.Name, resp.Operation,
				tputils.DecodeAsString(resp.Value, 200), err.Error())
			wssResp.Error = messaging.ErrorValueFromError(err)
		}
		wssStatusMap := make(map[string]WssActionStatus)
		for _, actionStatus := range actionStatusMap {
			wssStatusMap[actionStatus.Name] = WssActionStatus{
				ActionID:      actionStatus.ActionID,
				Error:         actionStatus.Error, // error fields are identical
				State:         actionStatus.State,
				TimeRequested: actionStatus.TimeRequested,
				TimeEnded:     actionStatus.TimeUpdated,
				Output:        actionStatus.Output,
			}
		}
		wssResp.Statuses = wssStatusMap
	case vocab.OpReadAllProperties, vocab.OpReadMultipleProperties:
		// convert ThingValue map to map of name-value pairs
		// the last updated timestamp is lost.
		var thingValueList map[string]messaging.ThingValue
		err := tputils.DecodeAsObject(resp.Value, &thingValueList)
		if err != nil {
			err = fmt.Errorf("encodeResponse (%s). Not a ThingValue map; err: %w", resp.Operation, err)
			wssResp.Error = messaging.ErrorValueFromError(err)
		}
		wssPropValues := make(map[string]any)
		for _, thingValue := range thingValueList {
			wssPropValues[thingValue.Name] = thingValue.ToString(0)
		}
		wssResp.Values = wssPropValues
		// Note that wssResp also includes the ResponseMessage 'Value' property
		// which hiveot clients can use to obtain the ThingValue result.
		// non-hiveot clients will see the key-value map in 'Values'
	case vocab.OpReadProperty:
	}

	return wssResp
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *WssMessageConverter) GetProtocolType() string {
	return messaging.ProtocolTypeWSS
}

// Create a new instance of the WoT websocket to hiveot message converter
func NewWssMessageConverter() *WssMessageConverter {
	return &WssMessageConverter{}
}
