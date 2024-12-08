package transports

import (
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/wot"
	"time"
)

// ThingMessage is an internal-use envelope, for an event, action or property message,
// as received from agents, services or consumers.
// This is not intended for wire transfer as each transport protocol handles this
// its own way.
type ThingMessage struct {
	//--- required fields to be filled-in by the sender

	// ThingID of the thing this value applies to.
	// For messages from/to agents this is the agent ThingID
	// For messages to/from consumers this is the digitwin dThingID
	// This is required.
	ThingID string

	// Name of the event, action or property as defined in the TD property/event/action map.
	// This is required.
	Name string

	// The operation for this message as defined in TD-1.1
	Operation string

	// SenderID is the account ID of the agent, service or user sending the message
	// to the hub.
	// This is required and used in authorization of the sender and routing of messages.
	// The underlying protocol binding MUST set this to the authenticated client.
	SenderID string

	//--- optional fields

	// Timestamp the data was created using RFC3339milli
	// Optional. This will be set to 'now' if omitted.
	Timestamp string

	// Data in the native format as described in the TD affordance dataschema.
	Data any

	// RequestID of the message. Intended to track action progress.
	// Optional. The hub will generate a unique requestID if omitted.
	RequestID string

	// MessageID unique ID of the message. Intended to detect duplicates.
	// Generated on receiving a message.
	MessageID string
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
// Optionally "WT" is weekday, time (Mon, 14:31:01 PDT) when less than a week old
// or, provide the time format directly, eg: "02 Jan 06 15:04 MST" for rfc822
func (tm *ThingMessage) GetUpdated(format ...string) (updated string) {
	if tm.Timestamp == "" {
		return ""
	}
	createdTime, _ := dateparse.ParseAny(tm.Timestamp)
	if format != nil && len(format) == 1 {
		if format[0] == "WT" {
			// Format weekday, time if less than a week old
			age := time.Now().Sub(createdTime)
			if age < time.Hour*24*7 {
				updated = createdTime.Format("Mon, 15:04:05 MST")
			} else {
				updated = createdTime.Format(time.RFC822)
			}
		} else {
			updated = createdTime.Format(format[0])
		}
	} else {
		updated = createdTime.Format(time.RFC822)
	}
	return updated
}

// NewThingMessage creates a new ThingMessage object with the address of the things,
// the message action, event or rpc name, and the serialized value data.
// This copies the value buffer.
//
//	operation is the message operation WoTOp... or HTOp...
//	thingID is the thing the value applies to (destination of action or source of event)
//	name is the property, event or action name as described in the thing TD
//	data is the native message data as defined in the corresponding TD dataschema.
//	senderID is the accountID of the creator of the value
func NewThingMessage(operation string, thingID, name string, data any, senderID string) *ThingMessage {
	return &ThingMessage{
		Timestamp: time.Now().Format(wot.RFC3339Milli),
		Data:      data,
		Name:      name,
		Operation: operation,
		SenderID:  senderID,
		ThingID:   thingID,
	}
}
