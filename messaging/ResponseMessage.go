// Package transports with the 3 flow messages: requests, response and  notifications
package messaging

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/teris-io/shortid"
	"time"
)

// ResponseMessage, ActionStatus and ThingValue define the standardized messaging
// envelopes for handling responses.
// Each transport protocol bindings map this format to this specific format.

type AffordanceType string

const AffordanceTypeEvent AffordanceType = "event"
const AffordanceTypeProperty AffordanceType = "property"
const AffordanceTypeAction AffordanceType = "action"

// MessageTypeResponse identify the message as a response.
const MessageTypeResponse = "response"

// Request status provided with the response.
// this aligns with action status values from WoT spec
const (
	// StatusPending - the request has not yet been delivered
	StatusPending = "pending"
	// StatusRunning - the request is being processed
	StatusRunning = "running"
	// StatusCompleted - the request processing was completed
	StatusCompleted = "completed"
	// StatusFailed - the request processing or delivery failed
	StatusFailed = "failed"
)

// ActionStatus is used in the response message payload for invokeaction and queryaction
// responses.
// Note: keep this in sync with the digital twin ActionStatus in the TD.
type ActionStatus struct {

	// ID that uniquely identifies the action
	// This can be an identifier or a URL.
	//
	// The action request identifier
	ActionID string `json:"actionID,omitempty"`

	// AgentID with Agent ID
	//
	// The agent handling the action
	AgentID string `json:"agentID,omitempty"`

	// Error with Error
	//
	// Action error info when failed
	Error string `json:"error,omitempty"`

	// Input with Action input
	//
	// Action input value
	Input any `json:"input,omitempty"`

	// Name with Action name
	//
	// name of the action or property
	Name string `json:"name,omitempty"`

	// Output with Action output
	Output any `json:"output,omitempty"`

	// SenderID of the client requesting the action
	SenderID string `json:"senderID,omitempty"`

	// Status with Action status
	//
	// Status of the action's progress: pending, running, completed, failed
	Status string `json:"status,omitempty"`

	// ThingID with Action Thing
	//
	// Digital twin ThingID the action applies to
	ThingID string `json:"thingID,omitempty"`

	// Requested time the action request was received
	//
	// Time the action was initially requested
	Requested string `json:"requested,omitempty"`

	// Updated time the action status was last updated
	//
	// Time the action status was last updated
	Updated string `json:"updated,omitempty"`
}

// ThingValue is the internal API response payload to subscribeevent, observeproperty,
// readevent and readproperty operations. The protocol binding maps between this
// and the protocol way of encoding values.
type ThingValue struct {
	// Type of affordance this is a value of: AffordanceTypeProperty|Event|Action
	AffordanceType AffordanceType `json:"affordanceType"`

	// Output with Payload
	//
	// Data in format as described by the thing's affordance
	Data any `json:"data,omitempty"`

	// Name with affordance name
	//
	// Name of the affordance holding the value
	Name string `json:"name,omitempty"`

	// ThingID with Thing ID
	//
	// Digital twin Thing ID
	ThingID string `json:"thingID,omitempty"`

	// Timestamp with Timestamp time
	//
	// Time the value was last updated
	Timestamp string `json:"timestamp,omitempty"`
}

// ToString is a helper to easily read the response output as a string
func (tv *ThingValue) ToString(maxlen int) string {
	return tputils.DecodeAsString(tv.Data, maxlen)
}
func NewThingValue(affordanceType AffordanceType, thingID, name string, data any, timestamp string) *ThingValue {
	tv := &ThingValue{
		AffordanceType: affordanceType,
		Data:           data,
		Name:           name,
		ThingID:        thingID,
		Timestamp:      timestamp,
	}
	if tv.Timestamp == "" {
		tv.Timestamp = utils.FormatUTCMilli(time.Now())
	}
	return tv
}

// ResponseMessage serves to notify a client of the result of a request.
//
// The Value field contains the message response data as defined by the operation
// Action related response output:
//   - invokeaction             action output as per TD, when status==completed
//   - queryaction              []ActionStatus object array
//   - queryallactions          map [name][]ActionStatus objects
//
// Property related response output:
//   - readproperty             ThingValue object
//   - readallproperties        map[name]ThingValue objects
//
// Event related response output
//   - readevent                ThingValue object
//   - readallevents            map[name]ThingValue objects
type ResponseMessage struct {

	// CorrelationID of the request this is a response to, if any.
	CorrelationID string `json:"correlationID,omitempty"`

	// Error contains the short error description when status is failed.
	Error string `json:"error"`

	// MessageID unique ID of the message. Intended to detect duplicates.
	// Generated by the protocol binding.
	MessageID string `json:"messageID,omitempty"`

	// MessageType identifies this message payload as a response
	// This is set to the value of MessageTypeResponse
	MessageType string `json:"messageType"`

	// Name of the action or property affordance this is a response from.
	Name string `json:"name"`

	// The operation this is a response to. This MUST be the operation provided in the request.
	Operation string `json:"operation"`

	// Authenticated ID of the agent sending the response, set by the server.
	//
	// This is non-wot and a feature of the hiveot Hub, to allow services to link requests
	// to authenticated users.
	//
	// The Hub protocol server MUST set this to the authenticated sender.
	SenderID string `json:"senderID"`

	// ThingID of the thing this is a response from.
	// For responses passed to consumers this is the digitwin dThingID
	// For responses sent by agents this is the agent ThingID
	// This field is optional and intended to help debugging and logging.
	ThingID string `json:"thingID,omitempty"`

	// Timestamp the response was created
	Timestamp string `json:"timestamp,omitempty"`

	// Value of the response as described in the TD affordance output or value dataschema.
	// If the operation is one of the Thing level operations, the value is specified
	// by the operation's dataschema.
	//
	// If an error is returned then value optionally contains a detailed error description.
	Value any `json:"value"`
}

// ToString is a helper to easily read the response output as a string
func (resp *ResponseMessage) ToString(maxlen int) string {
	return tputils.DecodeAsString(resp.Value, maxlen)
}

// NewResponseMessage creates a new ResponseMessage instance.
//
// This sets status to completed if err is nil or Failed if err is provided.
// If the status is not completed or failed then set it to the appropriate value after creation.
//
//	operation is the request operation WoTOp... or HTOp...
//	thingID is the thing the value applies to (destination of action or source of event)
//	name is the name of the property, event or action affordance as described in the thing TD
//	data is the response data as defined in the corresponding affordance dataschema or nil if not applicable
//	err is the optional error response which will set status to failed
//	correlationID  ID provided by the request
func NewResponseMessage(operation string, thingID, name string, data any, err error, correlationID string) *ResponseMessage {
	resp := &ResponseMessage{
		MessageType:   MessageTypeResponse,
		Operation:     operation,
		ThingID:       thingID,
		Name:          name,
		Value:         data,
		CorrelationID: correlationID,
		Timestamp:     utils.FormatUTCMilli(time.Now()),
		MessageID:     shortid.MustGenerate(),
	}
	if err != nil {
		resp.Error = err.Error()
	}
	return resp
}
