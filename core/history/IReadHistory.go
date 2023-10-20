// Package history with message definitions for reading the history store.
package history

import (
	"github.com/hiveot/hub/lib/thing"
)

// rpc requests are published on the bus using the address:
//  rpc/{serviceID}/{capability}/{method}/{senderID}
// the request contains a json document with arguments as described below.
// Responses are sent to the replyTo address using nats or mqtt5 headers.

// ServiceName is the agent name of the default instance of the service
const ServiceName = "history"

// ReadHistoryCap is the capability ID to read the history
const ReadHistoryCap string = "readHistory"

// cursor methods that take the key as arg and returns a value
const (
	// CursorFirstMethod return the oldest value in the history
	CursorFirstMethod = "cursorFirst"
	// CursorLastMethod return the newest value in the history
	CursorLastMethod = "cursorLast"
	// CursorNextMethod returns the next value in the history
	CursorNextMethod = "cursorNext"
	// CursorPrevMethod returns the previous value in the history
	CursorPrevMethod = "cursorPrev"
)

// CursorArgs contain the cursor request for use in First,Last,Next,Prev
type CursorArgs struct {
	// Cursor identifier obtained with GetCursor
	CursorKey string `json:"cursorKey"`
}

// CursorSingleResp contains a single response value to a cursor request
type CursorSingleResp struct {
	// The value at the new cursor position or nil if not valid
	Value *thing.ThingValue `json:"value"`
	// The current position holds a valid value
	Valid bool `json:"valid"`
}

// CursorNextNMethod returns a batch of next N historical values
const CursorNextNMethod = "cursorNextN"

// CursorPrevNMethod returns a batch of prior N historical values
const CursorPrevNMethod = "cursorPrevN"

// CursorNArgs contains the request for use in NextN and PrevN
type CursorNArgs struct {
	// Cursor identifier obtained with GetCursor
	CursorKey string `json:"cursorKey"`
	// Maximum number of results to return
	Limit int `json:"limit"`
}

// CursorNResp contains the batch response to a cursor request
type CursorNResp struct {
	// Returns up to 'Limit' iterated values.
	// This will be an empty list when trying to read past the last value.
	Values []*thing.ThingValue `json:"values"`
	// There are still items remaining.
	ItemsRemaining bool `json:"itemsRemaining"`
}

// CursorReleaseMethod releases the cursor and resources
// This MUST be called after the cursor is not longer used.
const CursorReleaseMethod = "cursorRelease"

type CursorReleaseArgs struct {
	// Cursor identifier obtained with GetCursor
	CursorKey string `json:"cursorKey"`
}

// CursorSeekMethod seeks the starting point in time for iterating the history
// This returns a single value response with the value at timestamp or next closest
// if it doesn't exist.
// Returns empty value when there are no values at or past the given timestamp
const CursorSeekMethod = "cursorSeek"

type CursorSeekArgs struct {
	// Cursor identifier obtained with GetCursor
	CursorKey string `json:"cursorKey"`
	// timestamp in msec since epoc to find
	TimeStampMSec int64 `json:"timeStampMSec"`
}

// returns CursorSingleResp

// GetCursorMethod returns a cursor to iterate the history of a thing
// The cursor MUST be released after use.
// The cursor will expire after not being used for the default expiry period.
const GetCursorMethod = "getCursor"

// GetCursorArgs request arguments:
//
//	JSON:
//	{
//		   "agentID": {string},
//		   "thingID": {string},
//		   "name": {string}
//	}
type GetCursorArgs struct {
	// Agent providing the Thing (required)
	AgentID string `json:"agentID"`
	// Thing providing the event to get (required)
	ThingID string `json:"thingID"`
	// Name of the event or action whose history to get
	// Use "" to iterate all events/action of the Thing.
	Name string `json:"name"`
}

// GetCursor returns a cursor key
//
//	{
//	  "cursorKey": {string}
//	}
type GetCursorResp struct {
	// Cursor identifier
	// The cursor MUST be released after use.
	CursorKey string `json:"cursorKey"`
}

// GetLatestMethod returns the latest values of a Thing.
const GetLatestMethod = "getLatest"

type GetLatestArgs struct {
	//	agentID is the ID of the agent that published the Thing values
	AgentID string `json:"agentID"`
	//	thingID is the ID of the thing whose history to read
	ThingID string `json:"thingID"`
	//	names is the list of properties or events to return. Use nil for all known properties.
	Names []string `json:"names"`
}

// GetLatestResp returns the latest thing values
//
//	{
//	  "values": [ {thingValue}
//	}
type GetLatestResp struct {
	Values []*thing.ThingValue `json:"values"`
}
