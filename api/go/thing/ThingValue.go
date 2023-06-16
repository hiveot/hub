package thing

import (
	"time"

	"github.com/hiveot/hub/api/go/vocab"
)

// ThingValue contains an event, action value or TD of a thing
type ThingValue struct {
	// ID of event, action or property as defined in the TD event/action map.
	// For the type of event/action see the TD event/action @type field.
	ID string `json:"id"`

	// PublisherID of the thing
	PublisherID string `json:"publisherID,omitempty"`

	// ThingID of the thing itself
	ThingID string `json:"thingID,omitempty"`

	// Data with serialized value payload, as defined by the TD affordance DataSchema
	Data []byte `json:"data,omitempty"`

	// Timestamp the value was created, in ISO8601 UTC format. Default "" is now()
	Created string `json:"created,omitempty"`
	// Timestamp in unix time, msec since Epoch.
	//CreatedMsec int64

	// Expiry time of the value in seconds since epoc.
	// Events expire based on their update interval.
	// Actions expiry is used for queueing. 0 means the action expires immediately after receiving it and is not queued.
	//Expiry int64

	// Sequence of the message from its creator. Intended to prevent replay attacks.
	//Sequence int64
}

// NewThingValue creates a new ThingValue object with the address of the thing, the action or event id and the serialized value data
// This copies the value buffer.
func NewThingValue(publisherID, thingID, id string, data []byte) ThingValue {
	return ThingValue{
		PublisherID: publisherID,
		ThingID:     thingID,
		ID:          id,
		Created:     time.Now().Format(vocab.ISO8601Format),
		// DO NOT REMOVE THE TYPE CONVERSION
		// this clones the valueJSON so the valueJSON buffer can be reused
		Data: []byte(string(data)),
	}
}
