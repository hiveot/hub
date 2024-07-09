package historyapi

import (
	"github.com/hiveot/hub/lib/things"
)

// ReadHistoryServiceID is the ID of the service exposed by the agent
const ReadHistoryServiceID = "read"

// DefaultLimit nr items of none provided
const DefaultLimit = 100

// Read history methods
const (
	// CursorNextNMethod returns a batch of next N historical values
	CursorNextNMethod = "cursorNextN"

	// CursorNextUntilMethod reads until the given end time or limit is reached
	CursorNextUntilMethod = "cursorNextUntil"

	// CursorPrevNMethod returns a batch of prior N historical values
	CursorPrevNMethod = "cursorPrevN"

	// CursorPrevUntilMethod reads until the given start time or limit is reached
	CursorPrevUntilMethod = "cursorPrevUntil"

	// CursorReleaseMethod releases the cursor and resources
	// This MUST be called after the cursor is not longer used.
	CursorReleaseMethod = "cursorRelease"

	// CursorSeekMethod seeks the starting point in time for iterating the history
	// This returns a single value response with the value at timestamp or next closest
	// if it doesn't exist.
	// Returns empty value when there are no values at or past the given timestamp
	CursorSeekMethod = "cursorSeek"

	// GetCursorMethod returns a cursor to iterate the history of a things
	// The cursor MUST be released after use.
	// The cursor will expire after not being used for the default expiry period.
	GetCursorMethod = "getCursor"
)

// cursor methods that take the key as arg and returns a single value
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
	Value *things.ThingMessage `json:"value"`
	// The current position holds a valid value
	Valid bool `json:"valid"`
}

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
	Values []*things.ThingMessage `json:"values"`
	// There are still items remaining.
	ItemsRemaining bool `json:"itemsRemaining"`
}

// CursorUntilArgs arguments for next-until or prev-until cursor
// TimeStamp is the start time to reach for prev-until or end-time for next-until
type CursorUntilArgs struct {
	// Cursor identifier obtained with GetCursor
	CursorKey string `json:"cursorKey"`
	// timestamp in rfc8601 format or 'now' for default
	TimeStamp string `json:"timeStamp"`
	// limit nr of results or 0 for default
	Limit int `json:"limit"`
}

type CursorReleaseArgs struct {
	// Cursor identifier obtained with GetCursor
	CursorKey string `json:"cursorKey"`
}

type CursorSeekArgs struct {
	// Cursor identifier obtained with GetCursor
	CursorKey string `json:"cursorKey"`
	// timestamp in rfc8601 format
	TimeStamp string `json:"timeStamp"`
}

// returns CursorSingleResp

// GetCursorArgs request arguments:
type GetCursorArgs struct {
	// Digitwin thing providing the event to get (required)
	ThingID string `json:"thingID"`
	// Optional filter value to search for a specific key
	FilterOnKey string `json:"filterOnKey,omitempty"`
}
type GetCursorResp struct {
	// Cursor identifier
	// The cursor MUST be released after use.
	CursorKey string `json:"cursorKey"`
}
