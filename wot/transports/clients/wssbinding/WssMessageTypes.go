package wssbinding

import (
	"github.com/hiveot/hub/wot"
)

// Action status message values
const (
	ActionStatusCompleted = "completed"
	ActionStatusDelivered = "delivered"
	ActionStatusFailed    = "error"
	ActionStatusPending   = "pending"
)

// websocket binding message types
const (
	MsgTypeActionStatus            = "actionStatus"
	MsgTypeActionStatuses          = "actionStatuses"
	MsgTypeCancelAction            = "cancelAction"
	MsgTypeInvokeAction            = "invokeAction"
	MsgTypeLogin                   = "login"
	MsgTypeLogout                  = "logout"
	MsgTypeObserveAllProperties    = "observeAllProperties"
	MsgTypeObserveProperty         = "observeProperty"
	MsgTypePing                    = "ping"
	MsgTypePong                    = "pong"
	MsgTypePublishError            = "error"
	MsgTypePublishEvent            = "event"
	MsgTypeQueryAction             = "queryAction"
	MsgTypeQueryAllActions         = "queryAllActions"
	MsgTypeReadAllEvents           = "readAllEvents"
	MsgTypeReadAllProperties       = "readAllProperties"
	MsgTypeReadAllTDs              = "readAllTDs"
	MsgTypeReadEvent               = "readEvent"
	MsgTypeReadMultipleEvents      = "readMultipleEvents"
	MsgTypeReadMultipleProperties  = "readMultipleProperties"
	MsgTypeReadProperty            = "readProperty"
	MsgTypeReadTD                  = "readTD"
	MsgTypeSubscribeAllEvents      = "subscribeAllEvents"
	MsgTypeSubscribeEvent          = "subscribeEvent"
	MsgTypeUnobserveAllProperties  = "unobserveAllProperties"
	MsgTypeUnobserveProperty       = "unobserveProperty"
	MsgTypeUnsubscribeAllEvents    = "unsubscribeAllEvents"
	MsgTypeUnsubscribeEvent        = "unsubscribeEvent"
	MsgTypePropertyReadings        = "propertyReadings"
	MsgTypePropertyReading         = "propertyReading"
	MsgTypeUpdateTD                = "updateTD"
	MsgTypeWriteAllProperties      = "writeAllProperties"
	MsgTypeWriteMultipleProperties = "writeMultipleProperties"
	MsgTypeWriteProperty           = "writeProperty"
)

// MsgTypeToOp converts websocket message types to a WoT operation
var MsgTypeToOp = map[string]string{
	MsgTypeActionStatus:            wot.HTOpUpdateActionStatus,
	MsgTypeActionStatuses:          wot.HTOpUpdateActionStatuses,
	MsgTypeCancelAction:            wot.OpCancelAction,
	MsgTypeInvokeAction:            wot.OpInvokeAction,
	MsgTypeLogin:                   wot.HTOpLogin,
	MsgTypeLogout:                  wot.HTOpLogout,
	MsgTypeObserveAllProperties:    wot.OpObserveAllProperties,
	MsgTypeObserveProperty:         wot.OpObserveProperty,
	MsgTypePublishError:            wot.HTOpPublishError,
	MsgTypePublishEvent:            wot.HTOpPublishEvent,
	MsgTypeQueryAction:             wot.OpQueryAction,
	MsgTypeQueryAllActions:         wot.OpQueryAllActions,
	MsgTypeReadAllEvents:           wot.HTOpReadAllEvents,
	MsgTypeReadAllProperties:       wot.OpReadAllProperties,
	MsgTypeReadAllTDs:              wot.HTOpReadAllTDs,
	MsgTypeReadEvent:               wot.HTOpReadEvent,
	MsgTypeReadMultipleProperties:  wot.OpReadMultipleProperties,
	MsgTypeReadProperty:            wot.OpReadProperty,
	MsgTypeReadTD:                  wot.HTOpReadTD,
	MsgTypeSubscribeAllEvents:      wot.OpSubscribeAllEvents,
	MsgTypeSubscribeEvent:          wot.OpSubscribeEvent,
	MsgTypeUnobserveAllProperties:  wot.OpUnobserveAllProperties,
	MsgTypeUnobserveProperty:       wot.OpUnobserveProperty,
	MsgTypeUnsubscribeAllEvents:    wot.OpUnsubscribeAllEvents,
	MsgTypeUnsubscribeEvent:        wot.OpUnsubscribeEvent,
	MsgTypePropertyReadings:        wot.HTOpUpdateMultipleProperties,
	MsgTypePropertyReading:         wot.HTOpUpdateProperty,
	MsgTypeUpdateTD:                wot.HTOpUpdateTD,
	MsgTypeWriteAllProperties:      wot.OpWriteAllProperties,
	MsgTypeWriteMultipleProperties: wot.OpWriteMultipleProperties,
	MsgTypeWriteProperty:           wot.OpWriteProperty,
}

// Base message struct for common field. Used to partially parse the message
// before knowing the operation and full type.
type BaseMessage struct {
	ThingID       string `json:"thingId"`
	MessageType   string `json:"messageType"`
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

type PropertyMessage struct {
	ThingID       string   `json:"thingId"`
	MessageType   string   `json:"messageType"`
	Name          string   `json:"property"`
	Names         []string `json:"properties,omitempty"`
	Data          any      `json:"data,omitempty"`
	LastTimestamp string   `json:"lastPropertyReading,omitempty"`
	Timestamp     string   `json:"timestamp,omitempty"`
	//
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

type ActionMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`
	Name        string `json:"action"`
	Data        any    `json:"input,omitempty"`

	// FIXME: under discussions. href has nothing to do with tracking actions
	HRef string `json:"href,omitempty"`
	//
	Timestamp     string `json:"timestamp"`
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`

	// hiveot only: SenderID is used for logging and storage (mainly history)
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
	Timestamp     string `json:"timestamp"` // timestamp of this update
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// See also https://www.rfc-editor.org/rfc/rfc9457
type ErrorMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`
	// Error message short text
	Title string `json:"title"`
	// Detailed error description if available
	Detail string `json:"detail"`
	// Error code, eg 404, 405, 500, ... (yes http codes)
	Status string `json:"status"`
	// Link to request that is in error
	CorrelationID string `json:"correlationId,omitempty"`
}

type EventMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`

	Name  string   `json:"event"`
	Names []string `json:"events"`
	Data  any      `json:"data,omitempty"`

	// subscription only
	LastEvent string `json:"lastEvent,omitempty"` // OpSubscribe...

	Timestamp     string `json:"timestamp"`
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

type TDMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`

	Name          string `json:"event"`
	Data          any    `json:"data,omitempty"` // JSON TD or list of JSON TDs
	Timestamp     string `json:"timestamp"`
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}
