package things

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/lib/utils"
	"time"
)

// ThingMessage holds an event or action received from agents, services or consumers.
type ThingMessage struct {
	//--- required fields to be filled-in by the sender

	// ThingID of the thing this value applies to.
	// For messages from/to agents this is the agent ThingID
	// For messages to/from consumers this is the digitwin ThingID
	// This is required.
	ThingID string `json:"thingID"`

	// Key of the event, action or property as defined in the TD property/event/action map.
	// This is required.
	Key string `json:"key"`

	// Type of message this value was sent as: (MessageTypeEvent, MessageTypeAction...)
	// This is required
	MessageType string `json:"messageType"`

	// SenderID is the account ID of the agent, service or user sending the message
	// to the hub.
	// This is required and used in authorization of the sender and routing of messages.
	// The underlying protocol binding MUST set this to the authenticated client.
	SenderID string `json:"senderID"`

	//--- optional fields

	// Timestamp the value was created using RFC3339milli
	// Optional. This will be set to 'now' if omitted.
	Created string `json:"created,omitempty"`

	// Data in the native format as described in the TD affordance dataschema.
	Data any `json:"data,omitempty"`

	// Raw is the serialized message data
	//Raw []byte `json:"-"`

	// MessageID of the message. Intended to detect duplicates and send replies.
	// Optional. The hub will generate a unique messageID if omitted.
	MessageID string `json:"messageID,omitempty"`
}

// DataAsText return a text representation of the data that is independent of
// the message serialization used.
func (tm *ThingMessage) DataAsText() string {
	if tm.Data == nil {
		return ""
	}
	dataAsText := fmt.Sprintf("%v", tm.Data)
	return dataAsText
}

// GetUpdated is a helper function to return the formatted time the data was last updated.
// The default format is RFC822 ("02 Jan 06 15:04 MST")
// Optionally "WT" is weekday, time (Mon, 14:31:01 PDT)
// or, provide the time format directly, eg: "02 Jan 06 15:04 MST" for rfc822
func (tm *ThingMessage) GetUpdated(format ...string) (updated string) {
	createdTime, _ := dateparse.ParseAny(tm.Created)
	if format != nil && len(format) == 1 {
		if format[0] == "WT" {
			// Format weekday, time
			updated = createdTime.Format("Mon, 15:04:05 MST")
		} else {
			updated = createdTime.Format(format[0])
		}
	} else {
		updated = createdTime.Format(time.RFC822)
	}
	return updated
}

// Decode converts the any-type to the given interface type.
// This returns an error if unmarshalling fails.
func (tm *ThingMessage) Decode(arg interface{}) error {
	if tm.Data == nil {
		arg = nil
	}
	// the ugly workaround is to marshal/unmarshal using json.
	// TODO: more efficient method to convert the any type to the given type.
	jsonData, _ := json.Marshal(tm.Data)
	return json.Unmarshal(jsonData, arg)
}

// NewThingMessage creates a new ThingMessage object with the address of the things,
// the message action, event or rpc key, and the serialized value data.
// This copies the value buffer.
//
//	messageType is the type of value: action, event, config, rpc request
//	thingID is the thing the value applies to (destination of action or source of event)
//	key is the property, event or action key of the value as described in the thing TD
//	data is the native message data as defined in the corresponding TD dataschema.
//	senderID is the accountID of the creator of the value
func NewThingMessage(messageType, thingID, key string, data any, senderID string) *ThingMessage {
	return &ThingMessage{
		Created:     time.Now().Format(utils.RFC3339Milli),
		Data:        data,
		Key:         key,
		MessageType: messageType,
		SenderID:    senderID,
		ThingID:     thingID,
	}
}
