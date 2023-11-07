package things

import (
	"time"
)

// ThingValue contains a Thing event, action or property value
//
//	{
//	   "agentID": {string},
//	   "thingID": {string},
//	   "name": {string},
//	   "data": [{byte array}],
//	   "created": {int64},   // msec since epoc
//	}
type ThingValue struct {
	// AgentID is the ID of the device or service that owns the Thing
	AgentID string `json:"agentID"`

	// ThingID or capabilityID of the things itself
	ThingID string `json:"thingID"`

	// Name of event, action or property as defined in the TD event/action map.
	Name string `json:"name"`

	// Data with serialized value payload, as defined by the TD affordance DataSchema
	Data []byte `json:"data,omitempty"`

	// Timestamp the value was created in unix time, msec since Epoch Jan 1st,1970 00:00 utc
	CreatedMSec int64 `json:"created,omitempty"`

	// Expiry time of the value in msec since epoc.
	// Events expire based on their update interval.
	// Actions expiry is used for queueing. 0 means the action expires immediately after receiving it and is not queued.
	//Expiry int64

	// Sequence of the message from its creator. Intended to prevent replay attacks.
	//Sequence int64

	// ID of the publisher of the value
	// For events this is the agentID
	// For actions,config and rpc this is the remote user sending the request
	SenderID string `json:"senderID"`

	// Type of value, event, action, config, rpc
	ValueType string `json:"valueType"`
}

// NewThingValue creates a new ThingValue object with the address of the things, the action or event id and the serialized value data
// This copies the value buffer.
//
//	valueType is the type of value: action, event, config, rpc request
func NewThingValue(valueType, agentID, thingID, name string, data []byte, senderID string) *ThingValue {
	return &ThingValue{
		ValueType:   valueType,
		AgentID:     agentID,
		ThingID:     thingID,
		Name:        name,
		SenderID:    senderID,
		CreatedMSec: time.Now().UnixMilli(),
		// DO NOT REMOVE THE TYPE CONVERSION
		// this clones the data so the its buffer can be reused
		Data: []byte(string(data)),
	}
}
