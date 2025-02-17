package wssserver

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// Wot websocket message converter is just a pass-through
// This implements the IMessageConverter interface
const WSSOpConnect = "wss-connect"

// WoT websocket binding message type names
const (
	MsgTypeActionStatus   = "actionStatus"
	MsgTypeActionStatuses = "actionStatuses"
	MsgTypeCancelAction   = "cancelAction"
	MsgTypeInvokeAction   = "invokeAction"
	//MsgTypeLogin                  = "login"
	//MsgTypeLogout                 = "logout"
	MsgTypeObserveAllProperties   = "observeAllProperties"
	MsgTypeObserveProperty        = "observeProperty"
	MsgTypePing                   = "ping"
	MsgTypePong                   = "pong"
	MsgTypeError                  = "error"
	MsgTypePublishEvent           = "event"
	MsgTypeQueryAction            = "queryAction"
	MsgTypeQueryAllActions        = "queryAllActions"
	MsgTypeReadAllEvents          = "readAllEvents"
	MsgTypeReadAllProperties      = "readAllProperties"
	MsgTypeReadAllTDs             = "readAllTDs"
	MsgTypeReadEvent              = "readEvent"
	MsgTypeReadProperty           = "readProperty"
	MsgTypeReadTD                 = "readTD"
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

	MsgTypeActionStatus:   "actionstatus",
	MsgTypeActionStatuses: "actionstatuses",
	MsgTypeCancelAction:   wot.OpCancelAction,
	MsgTypeInvokeAction:   wot.OpInvokeAction,
	//MsgTypeLogin:                  wot.HTOpLogin,
	//MsgTypeLogout:                 wot.HTOpLogout,
	MsgTypeObserveAllProperties:   wot.OpObserveAllProperties,
	MsgTypeObserveProperty:        wot.OpObserveProperty,
	MsgTypeError:                  "error",
	MsgTypePing:                   wot.HTOpPing,
	MsgTypePong:                   wot.HTOpPing,
	MsgTypePublishEvent:           wot.OpSubscribeEvent,
	MsgTypeQueryAction:            wot.OpQueryAction,
	MsgTypeQueryAllActions:        wot.OpQueryAllActions,
	MsgTypeReadAllEvents:          wot.HTOpReadAllEvents,
	MsgTypeReadAllProperties:      wot.OpReadAllProperties,
	MsgTypeReadAllTDs:             wot.HTOpReadAllTDs,
	MsgTypeReadEvent:              wot.HTOpReadEvent,
	MsgTypeReadProperty:           wot.OpReadProperty,
	MsgTypeReadTD:                 wot.HTOpReadTD,
	MsgTypeSubscribeAllEvents:     wot.OpSubscribeAllEvents,
	MsgTypeSubscribeEvent:         wot.OpSubscribeEvent,
	MsgTypeUnobserveAllProperties: wot.OpUnobserveAllProperties,
	MsgTypeUnobserveProperty:      wot.OpUnobserveProperty,
	MsgTypeUnsubscribeAllEvents:   wot.OpUnsubscribeAllEvents,
	MsgTypeUnsubscribeEvent:       wot.OpUnsubscribeEvent,
	MsgTypePropertyReadings:       wot.OpObserveAllProperties,
	MsgTypePropertyReading:        wot.OpObserveProperty,
	MsgTypeUpdateTD:               wot.HTOpUpdateTD,
	MsgTypeWriteProperty:          wot.OpWriteProperty,
}

// req2MsgType converts a request operation to a WoT websocket message type
var req2MsgType = map[string]string{
	"actionstatus":     MsgTypeActionStatus,
	"actionstatuses":   MsgTypeActionStatuses,
	wot.OpCancelAction: MsgTypeCancelAction,
	wot.OpInvokeAction: MsgTypeInvokeAction,
	//wot.HTOpLogin:              MsgTypeLogin,
	//wot.HTOpLogout:             MsgTypeLogout,
	wot.OpObserveAllProperties: MsgTypeObserveAllProperties,
	wot.OpObserveProperty:      MsgTypeObserveProperty,
	//"error":                       MsgTypeError,
	wot.HTOpPing:                 MsgTypePing,
	wot.OpQueryAction:            MsgTypeQueryAction,
	wot.OpQueryAllActions:        MsgTypeQueryAllActions,
	wot.HTOpReadAllEvents:        MsgTypeReadAllEvents,
	wot.OpReadAllProperties:      MsgTypeReadAllProperties,
	wot.HTOpReadAllTDs:           MsgTypeReadAllTDs,
	wot.HTOpReadEvent:            MsgTypeReadEvent,
	wot.OpReadProperty:           MsgTypeReadProperty,
	wot.HTOpReadTD:               MsgTypeReadTD,
	wot.OpSubscribeAllEvents:     MsgTypeSubscribeAllEvents,
	wot.OpSubscribeEvent:         MsgTypeSubscribeEvent,
	wot.OpUnobserveAllProperties: MsgTypeUnobserveAllProperties,
	wot.OpUnobserveProperty:      MsgTypeUnobserveProperty,
	wot.OpUnsubscribeAllEvents:   MsgTypeUnsubscribeAllEvents,
	wot.OpUnsubscribeEvent:       MsgTypeUnsubscribeEvent,
	wot.HTOpUpdateTD:             MsgTypeUpdateTD,
	wot.OpWriteProperty:          MsgTypeWriteProperty,
}

// respop2MsgType converts a response operation to a WoT websocket message type
// FIXME: mapping responses
var resp2MsgType = map[string]string{
	//"actionstatus":                MsgTypeActionStatus,
	//"actionstatuses":              MsgTypeActionStatuses,
	//wot.OpCancelAction:            MsgTypeCancelAction,
	wot.OpInvokeAction: MsgTypeActionStatus,
	//wot.HTOpLogin:                 MsgTypeLogin,
	//wot.HTOpLogout:                MsgTypeLogout,
	wot.OpObserveAllProperties: MsgTypePropertyReadings,
	wot.OpObserveProperty:      MsgTypePropertyReading,
	"error":                    MsgTypeError,
	wot.HTOpPing:               MsgTypePong,
	//wot.OpQueryAction:             MsgTypeQueryAction,
	//wot.OpQueryAllActions:         MsgTypeQueryAllActions,
	//wot.HTOpReadAllEvents:         MsgTypeReadAllEvents,
	//wot.OpReadAllProperties:       MsgTypeReadAllProperties,
	//wot.HTOpReadAllTDs:            MsgTypeReadAllTDs,
	wot.HTOpReadEvent: MsgTypePublishEvent,
	//wot.OpReadProperty:            MsgTypeReadProperty,
	//wot.HTOpReadTD:                MsgTypeReadTD,
	//wot.OpSubscribeAllEvents:      MsgTypeSubscribeAllEvents,
	//wot.OpSubscribeEvent:          MsgTypeSubscribeEvent,
	//wot.OpUnobserveAllProperties:  MsgTypeUnobserveAllProperties,
	//wot.OpUnobserveProperty:       MsgTypeUnobserveProperty,
	//wot.OpUnsubscribeAllEvents:    MsgTypeUnsubscribeAllEvents,
	//wot.OpUnsubscribeEvent:        MsgTypeUnsubscribeEvent,
	//wot.HTOpUpdateTD:              MsgTypeUpdateTD,
	//wot.OpWriteProperty:           MsgTypeWriteProperty,
}

// Base message struct for common field. Used to partially parse the message
// before knowing the operation and full type.
type BaseMessage struct {
	ThingID       string `json:"thingId"`
	MessageType   string `json:"messageType"`
	MessageID     string `json:"messageId"`
	CorrelationID string `json:"correlationID,omitempty"`
}

type ActionMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`
	Name        string `json:"action"`
	Data        any    `json:"input,omitempty"`
	Timestamp   string `json:"timestamp"`

	// FIXME: under discussions. href has nothing to do with tracking actions
	HRef string `json:"href,omitempty"`
	//
	// The correlationID is not in the spec but needed to be able to correlate a response
	// message.
	CorrelationID string `json:"correlationID,omitempty"`
	// to be removed. Agents as clients are not supported in WoT protocols
	SenderID string `json:"senderID"`
}

// ActionStatusMessage containing progress of an action or property write request
type ActionStatusMessage struct {
	// common base
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"` // OpUpdateActionStatus
	Name        string `json:"action"`

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
	//MessageID string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationID,omitempty"`
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
	// Time of the error
	Timestamp string `json:"timestamp"`
}

type EventMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`

	Name string `json:"event"`
	//Names []string `json:"events,omitempty"`
	Data any `json:"data,omitempty"`

	// subscription only
	LastEvent string `json:"lastEvent,omitempty"` // OpSubscribe...

	Timestamp     string `json:"timestamp"`
	CorrelationID string `json:"correlationID,omitempty"`
}

type PropertyMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`
	Name        string `json:"property"`
	//Names         []string `json:"properties,omitempty"`
	Data          any    `json:"data,omitempty"`
	LastTimestamp string `json:"lastPropertyReading,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
	//
	CorrelationID string `json:"correlationID,omitempty"`
	// to be removed. Agents as clients are not supported in WoT protocols
	SenderID string `json:"senderID"`
}

type TDMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`

	Name          string `json:"event"`
	Data          any    `json:"data,omitempty"` // JSON TD or list of JSON TDs
	Timestamp     string `json:"timestamp"`
	CorrelationID string `json:"correlationID,omitempty"`
}

type WotWssMessageConverter struct {
}

// DecodeMessage converts a native protocol received message to a standard
// request or response message.
// raw is the raw json serialized message
func (svc *WotWssMessageConverter) DecodeMessage(
	raw []byte) (req *transports.RequestMessage, resp *transports.ResponseMessage, err error) {

	// the operation is needed to determine whether this is a request or send and forget message
	// unfortunately this needs double unmarshalling :(
	baseMsg := BaseMessage{}
	err = jsoniter.Unmarshal(raw, &baseMsg)
	if err != nil {
		err = fmt.Errorf("DecodeMessage: unmarshalling message failed. Message ignored.")
		return nil, nil, err
	}
	op, _ := MsgTypeToOp[baseMsg.MessageType]
	switch baseMsg.MessageType {

	case MsgTypeActionStatus:
		wssMsg := ActionStatusMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = transports.NewResponseMessage(wot.OpInvokeAction,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Output, nil, wssMsg.CorrelationID)
		resp.Error = wssMsg.Error
		resp.Status = wssMsg.Status // todo: convert from wss to global names
		resp.Updated = wssMsg.TimeEnded

	case MsgTypeInvokeAction,
		MsgTypeQueryAction, MsgTypeQueryAllActions:
		wssMsg := ActionMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp

	case MsgTypeReadAllEvents, MsgTypeReadEvent:
		wssMsg := EventMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp

	case MsgTypePublishEvent:
		wssMsg := EventMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = transports.NewNotificationResponse(wot.OpSubscribeEvent,
			wssMsg.ThingID, wssMsg.Name, wssMsg.Data, nil)
		resp.Updated = wssMsg.Timestamp

	case // property requests. Forward as requests and return the response.
		MsgTypeReadAllProperties,
		MsgTypeReadProperty,
		MsgTypeWriteProperty:
		// map the message to a ThingMessage
		wssMsg := PropertyMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp

	case // agent response
		MsgTypePropertyReadings, // agent response
		MsgTypePropertyReading:  // agent response
		// map the message to a ThingMessage
		wssMsg := PropertyMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = transports.NewNotificationResponse(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, nil)
		resp.Updated = wssMsg.Timestamp

	case MsgTypeReadTD, MsgTypeUpdateTD: // td messages
		wssMsg := TDMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Data, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp

	// subscriptions are handled inside this binding
	case MsgTypeObserveProperty, MsgTypeObserveAllProperties,
		MsgTypeSubscribeEvent, MsgTypeSubscribeAllEvents,
		MsgTypeUnobserveProperty, MsgTypeUnobserveAllProperties,
		MsgTypeUnsubscribeEvent, MsgTypeUnsubscribeAllEvents:
		wssMsg := PropertyMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			op, wssMsg.ThingID, wssMsg.Name, nil, wssMsg.CorrelationID)
		req.Created = wssMsg.Timestamp

	// other messages handled inside this binding
	case MsgTypeError: // agent returned an error
		wssMsg := ErrorMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		resp = transports.NewResponseMessage(
			op, wssMsg.ThingID, wssMsg.Name, wssMsg.Detail, nil, wssMsg.CorrelationID)
		resp.Updated = wssMsg.Timestamp
		resp.Error = wssMsg.Title

	case MsgTypePing:
		wssMsg := BaseMessage{}
		_ = svc.Unmarshal(raw, &wssMsg)
		req = transports.NewRequestMessage(
			op, wssMsg.ThingID, "", nil, wssMsg.CorrelationID)

	default:
		// FIXME: a no-operation with correlationID is a response
		slog.Warn("_receive: unknown operation",
			"messageType", baseMsg.MessageType)
	}
	return req, resp, err
}

// EncodeRequest converts a hiveot RequestMessage to websocket equivalent message
func (svc *WotWssMessageConverter) EncodeRequest(
	req *transports.RequestMessage) (msg any, err error) {

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	msgType := req2MsgType[req.Operation]
	timestamp := time.Now().Format(wot.RFC3339Milli)

	switch req.Operation {
	case wot.OpInvokeAction, wot.OpQueryAllActions, wot.OpQueryAction:
		msg = ActionMessage{
			ThingID:       req.ThingID,
			MessageType:   msgType,
			Name:          req.Name,
			CorrelationID: req.CorrelationID,
			SenderID:      req.SenderID,
			Data:          req.Input,
			Timestamp:     timestamp,
		}

	case wot.OpObserveAllProperties, wot.OpObserveProperty,
		wot.OpUnobserveAllProperties, wot.OpUnobserveProperty,
		wot.OpReadAllProperties, wot.OpReadProperty,
		wot.OpWriteProperty:
		msg = PropertyMessage{
			ThingID:       req.ThingID,
			MessageType:   msgType,
			Name:          req.Name,
			Data:          req.Input,
			CorrelationID: req.CorrelationID,
			Timestamp:     timestamp,
		}
	case wot.HTOpReadAllEvents, wot.HTOpReadEvent,
		wot.OpSubscribeEvent, wot.OpSubscribeAllEvents,
		wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents,
		wot.HTOpPing:
		msg = EventMessage{
			ThingID:       req.ThingID,
			MessageType:   msgType,
			Name:          req.Name,
			Data:          req.Input,
			CorrelationID: req.CorrelationID,
			Timestamp:     timestamp,
		}
	case wot.HTOpReadTD, wot.HTOpReadAllTDs,
		wot.HTOpUpdateTD:
		msg = TDMessage{
			ThingID:       req.ThingID,
			MessageType:   msgType,
			Name:          req.Name,
			Data:          req.Input,
			CorrelationID: req.CorrelationID,
			Timestamp:     timestamp,
		}
	default:
		err = fmt.Errorf("unknown operation for WoT WSS: %s", req.Operation)
		slog.Error(err.Error())
	}

	return msg, err
}

// EncodeResponse converts a hiveot ResponseMessage to protocol equivalent message
func (svc *WotWssMessageConverter) EncodeResponse(
	resp *transports.ResponseMessage) (msg any, err error) {

	msgType := req2MsgType[resp.Operation]
	timestamp := time.Now().Format(wot.RFC3339Milli)

	switch resp.Operation {
	case wot.OpInvokeAction, wot.OpQueryAllActions, wot.OpQueryAction:
		msg = ActionMessage{
			ThingID:       resp.ThingID,
			MessageType:   msgType,
			Name:          resp.Name,
			CorrelationID: resp.CorrelationID,
			SenderID:      resp.SenderID,
			Data:          resp.Output,
			Timestamp:     timestamp,
		}
	}
	return resp, nil
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *WotWssMessageConverter) GetProtocolType() string {
	return transports.ProtocolTypeWotWSS
}

func (svc *WotWssMessageConverter) Marshal(in interface{}) (string, error) {
	return jsoniter.MarshalToString(in)
}
func (svc *WotWssMessageConverter) Unmarshal(raw []byte, out interface{}) error {
	return jsoniter.Unmarshal(raw, out)
}
