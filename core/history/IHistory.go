// Package history with POGS definitions of the history store.
// Unfortunately capnp does generate POGS types so we need to duplicate them
package history

import (
	"context"
	"github.com/hiveot/hub/lib/vocab"

	"github.com/hiveot/hub/api/go/hubapi"
	"github.com/hiveot/hub/lib/thing"
)

// ServiceName is the name of this service to connect to
const ServiceName = hubapi.HistoryServiceName

// EventNameProperties 'properties' is the name of the event that holds a JSON encoded map
// with one or more property values of a thing.
const EventNameProperties = vocab.WoTProperties

// EventRetention with a retention rule for an event (or action)
type EventRetention struct {
	// Name of the event to record
	Name string `yaml:"name"`

	// Optional, only accept the event from these publishers
	Publishers []string `yaml:"publishers"`

	// Optional, only accept the event from these things
	Things []string `yaml:"things"`

	// Optional, exclude the event from these things
	Exclude []string `yaml:"exclude"`

	// RetentionDays sets the age of the event after which it can be removed. 0 for indefinitely (default)
	RetentionDays int `yaml:"retentionDays"`
}

// IHistoryService defines the  capability to access the thing history service
type IHistoryService interface {

	// CapAddHistory provides the capability to add to the history of any Thing.
	// This capability should only be provided to trusted services that capture events from a
	// message bus or gateways to make them available. Events published on the internal pubsub
	// are already captured.
	//  clientID is the ID of the device or service requesting the capability
	//  ignoreRetention does not apply retention filters before adding the events
	CapAddHistory(ctx context.Context, clientID string, ignoreRetention bool) (IAddHistory, error)

	// CapManageRetention manages retention configuration for event storage
	CapManageRetention(ctx context.Context, clientID string) (IManageRetention, error)

	// CapReadHistory provides the capability to iterate history of a single Thing.
	// This returns an iterator for the history.
	// Values added after creating the cursor might not be included, depending on the
	// underlying store.
	// This capability can be provided to anyone who has read access to the thing.
	//
	//  clientID is the ID of the device or service requesting the capability
	CapReadHistory(ctx context.Context, clientID string) (IReadHistory, error)
}

// IAddHistory defines the capability to add to a Thing's history.
// If this capability was created with the thingAddr constraint then only values for this
// thingAddr will be accepted.
// Values stored are restricted to those that meet the filter criteria for event names,
// thingIDs and publisherIDs defined in the retention configuration.
type IAddHistory interface {

	// AddAction adds a Thing action with the given name and value to the action history
	// The given value object must not be modified after this call.
	AddAction(ctx context.Context, thingValue thing.ThingValue) error

	// AddEvent adds an event to the event history
	// The given value object must not be modified after this call.
	AddEvent(ctx context.Context, thingValue thing.ThingValue) error

	// AddEvents provides a bulk-add of events to the event history
	// The given value objects must not be modified after this call.
	AddEvents(ctx context.Context, eventValues []thing.ThingValue) error

	// Release the capability and its resources
	Release()
}

// IHistoryCursor is a cursor to iterate the Thing event and action history
// Use Seek to find the start of the range and NextN to read batches of values
type IHistoryCursor interface {
	// First return the oldest value in the history
	// Returns nil if the store is empty
	First() (thingValue thing.ThingValue, valid bool)

	// Last returns the latest value in the history
	// Returns nil if the store is empty
	Last() (thingValue thing.ThingValue, valid bool)

	// Next returns the next value in the history
	// Returns nil when trying to read past the last value
	Next() (thingValue thing.ThingValue, valid bool)

	// NextN returns a batch of next history values
	// Returns empty list when trying to read past the last value
	// itemsRemaining is true as long as more items can be retrieved
	NextN(steps uint) (batch []thing.ThingValue, itemsRemaining bool)

	// Prev returns the previous value in history
	// Returns nil when trying to read before the first value
	Prev() (thingValue thing.ThingValue, valid bool)

	// PrevN returns a batch of previous history values
	// Returns empty list when trying to read before the first value
	// itemsRemaining is true as long as more items can be retrieved
	PrevN(steps uint) (batch []thing.ThingValue, itemsRemaining bool)

	// Release the cursor and resources
	Release()

	// Seek the starting point for iterating the history
	// This returns the value at timestamp or next closest if it doesn't exist
	// Returns empty list when there are no values at or past the given timestamp
	Seek(isoTimestamp string) (thingValue thing.ThingValue, valid bool)
}

// IManageRetention defines the capability to manage the events that are recorded
type IManageRetention interface {

	// GetEvents returns the events with retention rules
	GetEvents(ctx context.Context) ([]EventRetention, error)

	// GetEventRetention returns the retention configuration of an event by name
	// This applies to events from any publishers and things
	// returns nil if there is no retention rule for the event
	//  eventName whose retention to return
	GetEventRetention(ctx context.Context, eventName string) (EventRetention, error)

	// RemoveEventRetention removes an existing event retention rule
	// If the rule doesn't exist this is considered successful and no error will be returned
	RemoveEventRetention(ctx context.Context, eventName string) error

	// SetEventRetention configures the retention of a Thing event
	SetEventRetention(ctx context.Context, eventRet EventRetention) error

	// TestEvent tests if the event will be retained
	TestEvent(ctx context.Context, eventValue thing.ThingValue) (bool, error)

	// Release the capability and its resources
	Release()
}

// IReadHistory defines the capability to read information from a thing
type IReadHistory interface {
	// GetEventHistory returns a cursor to iterate the history of the thing
	// name is the event or action to filter on. Use "" to iterate all events/action of the thing
	// The cursor MUST be released after use.
	//  publisherID is the ID of the Thing's publisher
	//  thingID is the ID of the thing whose history to read
	//  name is the event to read
	GetEventHistory(ctx context.Context, publisherID string, thingID string, name string) IHistoryCursor

	// GetProperties returns the latest values of a Thing.
	//  publisherID is the ID of the Thing's publisher
	//  thingID is the ID of the thing whose history to read
	//  names is the list of properties or events to return. Use nil for all known properties.
	GetProperties(ctx context.Context, publisherID string, thingID string, names []string) []thing.ThingValue

	// Info returns the history storage information of the thing
	//Info(ctx context.Context) *bucketstore.BucketStoreInfo

	// Release the capability and its resources
	Release()
}
