package things

import (
	"time"
)

// ThingMessage holds an event or action received from agents, services or end-users.
type ThingMessage struct {
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

	// SenderID is the account ID of the agent, service or user sending the message.
	// This is used in authorization of the sender and routing of messages.
	SenderID string `json:"senderID"`

	// Type of message this value was sent as: (MessageTypeEvent, MessageTypeAction...)
	MessageType string `json:"messageType"`
}

// GetUpdated is a helper function to return the formatted time the data was last updated.
// This uses the time format RFC822 ("02 Jan 06 15:04 MST")
func (tv *ThingMessage) GetUpdated() string {
	created := time.Unix(tv.CreatedMSec/1000, 0).Local()
	return created.Format(time.RFC822)
}

// NewThingMessage creates a new ThingMessage object with the address of the things,
// the message action, event or rpc key, and the serialized value data.
// This copies the value buffer.
//
//	messageType is the type of value: action, event, config, rpc request
//	thingID is the thing the value applies to (destination of action or source of event)
//	key is the property, event or action key of the value as described in the thing TD
//	data is the message serialized payload as defined in the corresponding TD dataschema.
//	senderID is the accountID of the creator of the value
func NewThingMessage(messageType, thingID, key string, data []byte, senderID string) *ThingMessage {
	return &ThingMessage{
		MessageType: messageType,
		ThingID:     thingID,
		Key:         key,
		SenderID:    senderID,
		CreatedMSec: time.Now().UnixMilli(),
		Data:        data,
	}
}