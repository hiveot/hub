package wssclient

import "github.com/hiveot/hub/api/go/vocab"

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
	MsgTypeReadMultipleProperties  = "readMultipleproperties"
	MsgTypeReadProperty            = "readProperty"
	MsgTypeReadTD                  = "readTD"
	MsgTypeRefresh                 = "refresh"
	MsgTypeSubscribeAllEvents      = "subscribeAllEvents"
	MsgTypeSubscribeEvent          = "subscribeEvent"
	MsgTypeUnobserveAllProperties  = "unobserveAllProperties"
	MsgTypeUnobserveProperty       = "unobserverProperty"
	MsgTypeUnsubscribeAllEvents    = "unsubscribeAllEvents"
	MsgTypeUnsubscribeEvent        = "unsubscribeEvent"
	MsgTypePropertyReadings        = "propertyReadings"
	MsgTypePropertyReading         = "propertyReading"
	MsgTypeUpdateTD                = "updatetd"
	MsgTypeWriteAllProperties      = "writeAllProperties"
	MsgTypeWriteMultipleProperties = "writeMultipleProperties"
	MsgTypeWriteProperty           = "writeProperty"
)

// MsgTypeToOp converts websocket message types to a WoT operation
var MsgTypeToOp = map[string]string{
	MsgTypeActionStatus:            vocab.HTOpUpdateActionStatus,
	MsgTypeActionStatuses:          vocab.HTOpUpdateActionStatuses,
	MsgTypeCancelAction:            vocab.OpCancelAction,
	MsgTypeInvokeAction:            vocab.OpInvokeAction,
	MsgTypeLogin:                   vocab.HTOpLogin,
	MsgTypeLogout:                  vocab.HTOpLogout,
	MsgTypeObserveAllProperties:    vocab.OpObserveAllProperties,
	MsgTypeObserveProperty:         vocab.OpObserveProperty,
	MsgTypePublishError:            vocab.HTOpPublishError,
	MsgTypePublishEvent:            vocab.HTOpPublishEvent,
	MsgTypeQueryAction:             vocab.OpQueryAction,
	MsgTypeQueryAllActions:         vocab.OpQueryAllActions,
	MsgTypeReadAllEvents:           vocab.HTOpReadAllEvents,
	MsgTypeReadAllProperties:       vocab.OpReadAllProperties,
	MsgTypeReadAllTDs:              vocab.HTOpReadAllTDs,
	MsgTypeReadEvent:               vocab.HTOpReadEvent,
	MsgTypeReadMultipleProperties:  vocab.OpReadMultipleProperties,
	MsgTypeReadProperty:            vocab.OpReadProperty,
	MsgTypeReadTD:                  vocab.HTOpReadTD,
	MsgTypeRefresh:                 vocab.HTOpRefresh,
	MsgTypeSubscribeAllEvents:      vocab.OpSubscribeAllEvents,
	MsgTypeSubscribeEvent:          vocab.OpSubscribeEvent,
	MsgTypeUnobserveAllProperties:  vocab.OpUnobserveAllProperties,
	MsgTypeUnobserveProperty:       vocab.OpUnobserveProperty,
	MsgTypeUnsubscribeAllEvents:    vocab.OpUnsubscribeAllEvents,
	MsgTypeUnsubscribeEvent:        vocab.OpUnsubscribeEvent,
	MsgTypePropertyReadings:        vocab.HTOpUpdateMultipleProperties,
	MsgTypePropertyReading:         vocab.HTOpUpdateProperty,
	MsgTypeUpdateTD:                vocab.HTOpUpdateTD,
	MsgTypeWriteAllProperties:      vocab.OpWriteAllProperties,
	MsgTypeWriteMultipleProperties: vocab.OpWriteMultipleProperties,
	MsgTypeWriteProperty:           vocab.OpWriteProperty,
}

// Base message struct for common field. Used to partially parse the message
// before knowing the operation and full type.
type BaseMessage struct {
	ThingID       string `json:"thingId"`
	MessageType   string `json:"messageType"`
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// messagetype for:
// OpReadProperty, OpReadMultipleProperties, OpReadAllProperties,
// OpWriteProperty, OpWriteMultipleProperties,
// OpObserveProperty, OpObserveAllProperties, OpUnobserveProperty, OpUnobserveAllProperties
// OpUpdateProperty, OpUpdateMultipleProperties
type PropertyMessage struct {
	ThingID       string   `json:"thingId"`
	MessageType   string   `json:"messageType"`
	Name          string   `json:"property"`
	Names         []string `json:"properties,omitempty"`          // OpReadMultipleProperties
	Data          any      `json:"data,omitempty"`                // OpWriteProperty
	LastTimestamp string   `json:"lastPropertyReading,omitempty"` // OpObserveProperty, OpObserveAllProperties
	Timestamp     string   `json:"timestamp,omitempty"`           // OpUpdateProperty
	//
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// OpInvokeAction, OpQueryAction, OpQueryAllActions, OpCancelAction
type ActionMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`
	Name        string `json:"action"`          // OpQueryAction, OpInvokeAction, OpCancelAction
	Data        any    `json:"input,omitempty"` // OpInvokeAction
	// FIXME: under discussions. href has nothing to do with tracking actions
	HRef string `json:"href,omitempty"` // queryAction
	//
	Timestamp     string `json:"timestamp"` // timestamp of this update
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
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

// OpSubscribeEvent, OpUnsubscribeEvent
// OpSubscribeMultipleEvents, OpUnsubscribeMultipleEvents,
// OpSubscribeAllEvents, OpUnsubscribeAllEvents
// OpPublishEvent
// OpUpdateTD
// OpReadEvent, OpReadMultipleEvents, OpReadAllEvents
type EventMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`

	Name string `json:"event"`          // OpPublishEvent,OpSubscribeEvent,OpUnsubscribeEvent
	Data any    `json:"data,omitempty"` // OpPublishEvent

	// subscription only
	LastEvent string `json:"lastEvent,omitempty"` // OpSubscribe...

	Timestamp     string `json:"timestamp"`
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// OpUpdateTD, opReadTD, opReadAllTDs
type TDMessage struct {
	ThingID     string `json:"thingId"`
	MessageType string `json:"messageType"`

	Name          string `json:"event"`
	Data          any    `json:"data,omitempty"` // JSON TD or list of JSON TDs
	Timestamp     string `json:"timestamp"`
	MessageID     string `json:"messageId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}
