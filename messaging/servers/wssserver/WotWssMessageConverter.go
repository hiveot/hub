package wssserver

import (
	"fmt"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// WoT websocket binding message type names
const (
	MsgTypeAck = "ack" // used to replace missing confirmation messages

	MsgTypeActionStatus           = "actionStatus"
	MsgTypeActionStatuses         = "actionStatuses"
	MsgTypeCancelAction           = "cancelAction"
	MsgTypeInvokeAction           = "invokeAction"
	MsgTypeObserveAllProperties   = "observeAllProperties"
	MsgTypeObserveProperty        = "observeProperty"
	MsgTypePing                   = "ping"
	MsgTypePong                   = "pong"
	MsgTypeError                  = "error"
	MsgTypeEvent                  = "event"
	MsgTypeQueryAction            = "queryAction"
	MsgTypeQueryAllActions        = "queryAllActions"
	MsgTypeReadAllProperties      = "readAllProperties"
	MsgTypeReadAllTDs             = "readAllTDs"
	MsgTypeReadProperty           = "readProperty"
	MsgTypeSubscribeAllEvents     = "subscribeAllEvents"
	MsgTypeSubscribeEvent         = "subscribeEvent"
	MsgTypeUnobserveAllProperties = "unobserveAllProperties"
	MsgTypeUnobserveProperty      = "unobserveProperty"
	MsgTypeUnsubscribeAllEvents   = "unsubscribeAllEvents"
	MsgTypeUnsubscribeEvent       = "unsubscribeEvent"
	MsgTypePropertyReadings       = "propertyReadings"
	MsgTypePropertyReading        = "propertyReading"
	MsgTypeUpdateTD               = "updateTD"
	MsgTypeWriteProperty          = "writeProperty"
)

// MsgTypeToOp converts websocket message types to a WoT operation
var MsgTypeToOp = map[string]string{
	// FIXME: actionStatus can be a response to query action and invoke action
	// Workaround: include operation in the response
	MsgTypeActionStatus:   wot.OpQueryAction,
	MsgTypeActionStatuses: wot.OpQueryAllActions,
	MsgTypeCancelAction:   wot.OpCancelAction,
	MsgTypeInvokeAction:   wot.OpInvokeAction,
	//MsgTypeLogin:                  wot.HTOpLogin,
	//MsgTypeLogout:                 wot.HTOpLogout,
	MsgTypeObserveAllProperties: wot.OpObserveAllProperties,
	MsgTypeObserveProperty:      wot.OpObserveProperty,
	MsgTypeError:                "error",
	MsgTypePing:                 wot.HTOpPing,
	MsgTypePong:                 wot.HTOpPing,
	//FIXME: propertyReading can be a response to read property and subscription
	// Workaround: include operation in the response
	MsgTypePropertyReadings:       wot.OpObserveAllProperties,
	MsgTypePropertyReading:        wot.OpObserveProperty,
	MsgTypeEvent:                  wot.OpSubscribeEvent,
	MsgTypeQueryAction:            wot.OpQueryAction,
	MsgTypeQueryAllActions:        wot.OpQueryAllActions,
	MsgTypeReadAllProperties:      wot.OpReadAllProperties,
	MsgTypeReadProperty:           wot.OpReadProperty,
	MsgTypeSubscribeAllEvents:     wot.OpSubscribeAllEvents,
	MsgTypeSubscribeEvent:         wot.OpSubscribeEvent,
	MsgTypeUnobserveAllProperties: wot.OpUnobserveAllProperties,
	MsgTypeUnobserveProperty:      wot.OpUnobserveProperty,
	MsgTypeUnsubscribeAllEvents:   wot.OpUnsubscribeAllEvents,
	MsgTypeUnsubscribeEvent:       wot.OpUnsubscribeEvent,
	MsgTypeWriteProperty:          wot.OpWriteProperty,
}

// req2MsgType converts a request operation to a WoT websocket message type
var req2MsgType = map[string]string{
	wot.OpCancelAction:           MsgTypeCancelAction,
	wot.OpInvokeAction:           MsgTypeInvokeAction,
	wot.OpObserveAllProperties:   MsgTypeObserveAllProperties,
	wot.OpObserveProperty:        MsgTypeObserveProperty,
	"error":                      MsgTypeError,
	wot.HTOpPing:                 MsgTypePing,
	wot.OpQueryAction:            MsgTypeQueryAction,
	wot.OpQueryAllActions:        MsgTypeQueryAllActions,
	wot.OpReadAllProperties:      MsgTypeReadAllProperties,
	wot.OpReadProperty:           MsgTypeReadProperty,
	wot.OpSubscribeAllEvents:     MsgTypeSubscribeAllEvents,
	wot.OpSubscribeEvent:         MsgTypeSubscribeEvent,
	wot.OpUnobserveAllProperties: MsgTypeUnobserveAllProperties,
	wot.OpUnobserveProperty:      MsgTypeUnobserveProperty,
	wot.OpUnsubscribeAllEvents:   MsgTypeUnsubscribeAllEvents,
	wot.OpUnsubscribeEvent:       MsgTypeUnsubscribeEvent,
	wot.OpWriteProperty:          MsgTypeWriteProperty,
}

// respop2MsgType converts a response operation to a WoT websocket message type
var resp2MsgType = map[string]string{
	// FIXME: maybe these should simply be an 'ack' message type
	//wot.OpCancelAction:            MsgTypeCancelAction,
	wot.OpInvokeAction:         MsgTypeActionStatus,
	wot.OpObserveAllProperties: MsgTypePropertyReadings,
	wot.OpObserveProperty:      MsgTypePropertyReading,
	"error":                    MsgTypeError,
	wot.HTOpPing:               MsgTypePong,
	wot.OpQueryAction:          MsgTypeActionStatus,
	wot.OpQueryAllActions:      MsgTypeActionStatuses,
	wot.OpReadAllProperties:    MsgTypePropertyReadings,
	wot.OpReadProperty:         MsgTypePropertyReading,
	wot.OpSubscribeAllEvents:   MsgTypeEvent,
	wot.OpSubscribeEvent:       MsgTypeEvent,

	wot.OpUnobserveAllProperties: MsgTypeAck,
	wot.OpUnobserveProperty:      MsgTypeAck,
	wot.OpUnsubscribeAllEvents:   MsgTypeAck,
	wot.OpUnsubscribeEvent:       MsgTypeAck,
	wot.OpWriteProperty:          MsgTypeAck,
}

// BaseMessage struct for common field. Used to partially parse the message
// before knowing the operation and full type.
type BaseMessage struct {
	// The correlationID is not in strawman but will likely be added as optional.
	CorrelationID string `json:"correlationID,omitempty"`
	// MessageID is not in strawman but will likely be added as mandatory
	MessageID string `json:"messageID"`

	MessageType string `json:"messageType"`
	ThingID     string `json:"thingId"`
	// operation is included as strawman does have deterministic responses
	// to requests (unless the correlationID is used).
	// there is some discussion on including it in the spec.
	// this will need some cleanup once WoT WSS is finalized.
	Operation string `json:"operation"`
}

type ActionMessage struct {
	BaseMessage
	//ThingID     string `json:"thingId"`
	//MessageType string `json:"messageType"`
	Name      string `json:"action"`
	Input     any    `json:"input,omitempty"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`

	// FIXME: under discussions. href has nothing to do with tracking actions
	HRef string `json:"href,omitempty"`

	// The correlationID is not in strawman but will likely be added as optional.
	//CorrelationID string `json:"correlationID,omitempty"`
	// MessageID is not in strawman but will likely be added as mandatory
	//MessageID string `json:"messageID"`
	// to be removed. Agents as clients are not supported in WoT protocols
	SenderID string `json:"senderID"`
}

// ActionStatusMessage containing progress of an action or property write request
type ActionStatusMessage struct {
	BaseMessage
	Name string `json:"action"`

	// FIXME: under discussions. href has nothing to do with tracking actions
	HRef string `json:"href,omitempty"` // queryAction

	// progress value: RequestDelivered, RequestCompleted, ...
	Status        string `json:"status"`           //
	Error         string `json:"error,omitempty"`  // only when status is failed
	Output        any    `json:"output,omitempty"` // only when completed
	TimeRequested string `json:"timeRequested"`
	TimeEnded     string `json:"timeEnded,omitempty"` // only when status is completed

	//
	Timestamp string `json:"timestamp"` // timestamp of this update
}

// See also https://www.rfc-editor.org/rfc/rfc9457
// The problem with this is that these fields don't exist at the application level
// or in the other protocols. RFC9457 also defines an message format with
// an errors array.
type ErrorMessage struct {
	// The thingID reporting the error
	ThingID string `json:"thingId"`
	// The action or property name the error applies to
	Name string `json:"name"`
	// this should be MsgTypError
	MessageType string `json:"messageType"`
	// Error message short text
	Title string `json:"title"`
	// Detailed error description if available
	Detail string `json:"detail"`
	// Error code, eg 404, 405, 500, ... (yes http codes)
	Status string `json:"status"`
	// Link to request that is in error
	CorrelationID string `json:"correlationID,omitempty"`
	// MessageID is not in strawman but will likely be added as mandatory
	MessageID string `json:"messageID"`
	// Time of the error
	Timestamp string `json:"timestamp"`
}

type EventMessage struct {
	BaseMessage

	Name string `json:"event"`
	//Names []string `json:"events,omitempty"`
	Data any `json:"data,omitempty"`
	// subscription only
	LastEvent string `json:"lastEvent,omitempty"` // OpSubscribe...
	Timestamp string `json:"timestamp"`
}

type PropertyMessage struct {
	BaseMessage
	Data          any    `json:"data,omitempty"`
	LastTimestamp string `json:"lastPropertyReading,omitempty"`
	Name          string `json:"property"`
	//Names         []string `json:"properties,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// Directory operations are not supported in WoT protocols
//type TDMessage struct {
//	ThingID     string `json:"thingId"`
//	MessageType string `json:"messageType"`
//
//	Name      string `json:"event"`
//	Data      any    `json:"data,omitempty"` // JSON TD or list of JSON TDs
//	Timestamp string `json:"timestamp"`
//	// The correlationID is not in strawman but will likely be added as optional.
//	CorrelationID string `json:"correlationID,omitempty"`
//	// MessageID is not in strawman but will likely be added as mandatory
//	MessageID string `json:"messageID"`
//}

type WotWssMessageConverter struct {
}

// DecodeRequest converts a native protocol request to a standard request message.
// Requests are received by decoded by agents.
func (svc *WotWssMessageConverter) DecodeRequest(raw []byte) (req *messaging.RequestMessage) {

	// the messageType is needed to determine the type of request
	// unfortunately this needs double unmarshalling :(
	baseMsg := BaseMessage{}
	err := jsoniter.Unmarshal(raw, &baseMsg)
	if err != nil {
		err = fmt.Errorf("DecodeRequest: unmarshalling request failed. Message ignored.")
		slog.Warn(err.Error())
		return nil
	}
	// determine the operation, fall back to message type if operation isn't present
	op := baseMsg.Operation
	if op == "" {
		op, _ = MsgTypeToOp[baseMsg.MessageType]
	}
	switch baseMsg.MessageType {

	// request for invoke action and query action
	case MsgTypeInvokeAction,
		MsgTypeQueryAction, MsgTypeQueryAllActions:
		wssMsg := ActionMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = messaging.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Input, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp
		req.MessageID = wssMsg.MessageID
		req.SenderID = wssMsg.SenderID

	// request to read/write properties
	case // property requests. Forward as requests and return the response.
		MsgTypeReadAllProperties,
		MsgTypeReadProperty,
		MsgTypeWriteProperty:
		wssMsg := PropertyMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = messaging.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp
		req.MessageID = wssMsg.MessageID

	// subscriptions are handled inside this binding
	case MsgTypeObserveProperty, MsgTypeObserveAllProperties,
		MsgTypeSubscribeEvent, MsgTypeSubscribeAllEvents,
		MsgTypeUnobserveProperty, MsgTypeUnobserveAllProperties,
		MsgTypeUnsubscribeEvent, MsgTypeUnsubscribeAllEvents:
		wssMsg := PropertyMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = messaging.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, nil, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp
		req.MessageID = wssMsg.MessageID

	case MsgTypePing:
		wssMsg := BaseMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = messaging.NewRequestMessage(
			op, wssMsg.ThingID, "", nil, wssMsg.CorrelationID)
		req.MessageID = wssMsg.MessageID

	default:
		slog.Warn("_receive: unknown operation",
			"messageType", baseMsg.MessageType,
			"operation", baseMsg.Operation,
		)
	}
	return req
}

// DecodeResponse convert a native protocol received message to a standard response message.
// raw is the raw json serialized message
func (svc *WotWssMessageConverter) DecodeResponse(raw []byte) (resp *messaging.ResponseMessage) {

	// the operation is needed to determine whether this is a request or send and forget message
	// unfortunately this needs double unmarshalling :(
	baseMsg := BaseMessage{}
	err := jsoniter.Unmarshal(raw, &baseMsg)
	if err != nil {
		slog.Warn("DecodeMessage: unmarshalling message failed. Message ignored.", "err", err.Error())
		return nil
	}
	// determine the operation, fall back to message type if operation is not present
	// This is necessary to get around missing message types to construct a proper response
	op := baseMsg.Operation
	if op == "" {
		op, _ = MsgTypeToOp[baseMsg.MessageType]
	}
	switch baseMsg.MessageType {

	case MsgTypeAck:
		wssMsg := BaseMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = messaging.NewResponseMessage(
			op, wssMsg.ThingID, "", "", nil, wssMsg.CorrelationID)
		resp.MessageID = wssMsg.MessageID

	// response to invoke or query action(s)
	case MsgTypeActionStatus, MsgTypeActionStatuses:
		wssMsg := ActionStatusMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = messaging.NewResponseMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Output, nil, wssMsg.CorrelationID)
		resp.Error = wssMsg.Error
		resp.Status = wssMsg.Status // todo: convert from wss to global names
		resp.Updated = wssMsg.TimeEnded
		resp.MessageID = wssMsg.MessageID
		resp.Output = wssMsg.Output

	// response to subscribe events
	case MsgTypeEvent: // Message type for event?
		wssMsg := EventMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = messaging.NewResponseMessage(wot.OpSubscribeEvent,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Data, nil, wssMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp
		resp.MessageID = wssMsg.MessageID

	// response with observed properties (long running request)
	// this can be a result of readproperty or observeproperty
	case                         // agent response
		MsgTypePropertyReadings, // agent response
		MsgTypePropertyReading:  // agent response
		// map the message to a ThingMessage
		wssMsg := PropertyMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = messaging.NewResponseMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, nil, wssMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp
		resp.MessageID = wssMsg.MessageID

	// other messages handled inside this binding
	case MsgTypeError: // agent returned an error
		wssMsg := ErrorMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = messaging.NewResponseMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Detail, nil, wssMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp
		resp.Error = wssMsg.Title
		resp.MessageID = wssMsg.MessageID

	case MsgTypePong:
		wssMsg := BaseMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = messaging.NewResponseMessage(
			op, wssMsg.ThingID, "", "", nil, wssMsg.CorrelationID)
		resp.MessageID = wssMsg.MessageID
		resp.Output = wssMsg.MessageType // FIXME: pong has output?

	default:
		// This is not a response
		//slog.Warn("_receive: unknown operation",
		//	"messageType", baseMsg.MessageType,
		//	"operation", baseMsg.Operation,
		//)
	}
	return resp
}

// EncodeRequest converts a hiveot RequestMessage to websocket equivalent message
// Requests are always sent to agents
func (svc *WotWssMessageConverter) EncodeRequest(
	req *messaging.RequestMessage) (msg any, err error) {

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	msgType := req2MsgType[req.Operation]
	timestamp := time.Now().Format(wot.RFC3339Milli)

	switch req.Operation {
	case wot.OpInvokeAction, wot.OpQueryAllActions, wot.OpQueryAction:
		msg = ActionMessage{
			BaseMessage: BaseMessage{
				CorrelationID: req.CorrelationID,
				MessageID:     req.MessageID,
				MessageType:   msgType,
				Operation:     req.Operation,
				ThingID:       req.ThingID,
			},
			Input:     req.Input,
			Name:      req.Name,
			SenderID:  req.SenderID,
			Timestamp: timestamp,
		}

	case wot.OpObserveAllProperties, wot.OpObserveProperty,
		wot.OpUnobserveAllProperties, wot.OpUnobserveProperty,
		wot.OpReadAllProperties, wot.OpReadProperty,
		wot.OpWriteProperty:
		msg = PropertyMessage{
			BaseMessage: BaseMessage{
				CorrelationID: req.CorrelationID,
				MessageID:     req.MessageID,
				MessageType:   msgType,
				Operation:     req.Operation,
				ThingID:       req.ThingID,
			},
			Data:      req.Input,
			Name:      req.Name,
			Timestamp: timestamp,
		}
	default:
		err = fmt.Errorf("unknown operation for WoT WSS: %s", req.Operation)
		slog.Error(err.Error())
	}

	return msg, err
}

// EncodeResponse converts a hiveot ResponseMessage to protocol equivalent message
// Responses are always sent by agents to consumers.
func (svc *WotWssMessageConverter) EncodeResponse(
	resp *messaging.ResponseMessage) (msg any, err error) {

	msgType := resp2MsgType[resp.Operation]
	timestamp := time.Now().Format(wot.RFC3339Milli)

	switch resp.Operation {
	case wot.OpInvokeAction, wot.OpQueryAllActions, wot.OpQueryAction:
		msg = ActionStatusMessage{
			BaseMessage: BaseMessage{
				CorrelationID: resp.CorrelationID,
				MessageID:     resp.MessageID,
				MessageType:   msgType,
				Operation:     resp.Operation,
				ThingID:       resp.ThingID,
			},
			Output:    resp.Output,
			Name:      resp.Name,
			Status:    resp.Status,
			Timestamp: timestamp,
		}
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		msg = EventMessage{
			BaseMessage: BaseMessage{
				MessageType:   msgType,
				MessageID:     resp.MessageID,
				CorrelationID: resp.CorrelationID,
				Operation:     resp.Operation,
			},
			Data:      resp.Output,
			Name:      resp.Name,
			Timestamp: resp.Updated,
		}

	case wot.OpReadProperty, wot.OpReadAllProperties,
		wot.OpObserveProperty, wot.OpObserveAllProperties:

		msg = PropertyMessage{
			BaseMessage: BaseMessage{
				MessageType:   msgType,
				MessageID:     resp.MessageID,
				CorrelationID: resp.CorrelationID,
				Operation:     resp.Operation,
			},
			Data:      resp.Output,
			Name:      resp.Name,
			Timestamp: resp.Updated,
		}

	case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents,
		wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
		msg = BaseMessage{
			MessageType:   msgType,
			MessageID:     resp.MessageID,
			CorrelationID: resp.CorrelationID,
			Operation:     resp.Operation,
		}

	case wot.HTOpPing:
		msg = BaseMessage{
			MessageType:   msgType,
			MessageID:     resp.MessageID,
			CorrelationID: resp.CorrelationID,
			Operation:     resp.Operation,
		}
	default:
		err = fmt.Errorf("EncodeResponse: Unknown operation '%s'", resp.Operation)
	}
	return msg, err
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *WotWssMessageConverter) GetProtocolType() string {
	return messaging.ProtocolTypeWotWSS
}

func (svc *WotWssMessageConverter) Marshal(in interface{}) (string, error) {
	return jsoniter.MarshalToString(in)
}
func (svc *WotWssMessageConverter) Unmarshal(raw []byte, out interface{}) error {
	return jsoniter.Unmarshal(raw, out)
}
