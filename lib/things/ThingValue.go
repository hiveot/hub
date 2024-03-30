package things

import (
	"time"
)

// ThingValue holds a value and metadata of an event, action, config or rpc message.
//
//	{
//	   "agentID": {string},
//	   "thingID": {string},
//	   "name": {string},
//	   "data": [{byte array}],
//	   "created": {int64},   // msec since epoc
//	}
type ThingValue struct {
	// AgentID is the ID of the device or service that owns the Thing.
	// This is required.
	AgentID string `json:"agentID"`

	// ThingID of the thing this value applies to.
	// This is required.
	ThingID string `json:"thingID"`

	// Key of the event, action or property as defined in the TD property/event/action map.
	// This is required.
	Key string `json:"key"`

	// Data converted to text from the type defined by the TD affordance DataSchema.
	// This can be omitted if no data is associated with the event or action.
	Data []byte `json:"data,omitempty"`

	// Timestamp the value was created in msec since Epoch Jan 1st,1970 00:00 utc
	CreatedMSec int64 `json:"created,omitempty"`

	// SequenceNr nr of the message from its sender. Intended to detect duplicates and prevent replay attacks.
	SequenceNr int64 `json:"sequenceNr,omitempty"`

	// SenderID is the account ID of the sender of the value.
	// This is used in authorization of the sender.
	// TODO: For security reasons this field is empty if the user reading this value has insufficient permissions.
	SenderID string `json:"senderID"`

	// Type of message this value was sent as: (MessageTypeEvent, MessageTypeAction, MessageTypeConfig, MessageTypeRPC...)
	MessageType string `json:"messageType"`
}

// GetUpdated is a helper function to return the formatted time the data was last updated.
// This uses the time format RFC822 ("02 Jan 06 15:04 MST")
func (tv *ThingValue) GetUpdated() string {
	created := time.Unix(tv.CreatedMSec/1000, 0).Local()
	return created.Format(time.RFC822)
}

// NewThingValue creates a new ThingValue object with the address of the things, the action or event id and the serialized value data
// This copies the value buffer.
//
//	valueType is the type of value: action, event, config, rpc request
//	agentID is the agent of the thing
//	thingID is the thing the value applies to (destination of action or source of event)
//	key is the property, event or action key of the value as described in the thing TD
//	value is the stringified value from the type defined in the value's TD dataschema
//	senderID is the accountID of the creator of the value
func NewThingValue(messageType, agentID, thingID, key string, data []byte, senderID string) *ThingValue {
	return &ThingValue{
		MessageType: messageType,
		AgentID:     agentID,
		ThingID:     thingID,
		Key:         key,
		SenderID:    senderID,
		CreatedMSec: time.Now().UnixMilli(),
		Data:        data,
	}
}
