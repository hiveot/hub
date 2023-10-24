package directoryapi

import "github.com/hiveot/hub/lib/thing"

// ServiceName is the agent name of the default instance of the service
const ServiceName = "directory"

// ReadDirectoryCap is the capability ID to read the directory
const ReadDirectoryCap = "readDirectory"

const CursorFirstMethod = "cursorFirst"

type CursorFirstArgs struct {
	// Iterator identifier obtained with GetReadCursorReq
	CursorKey string `json:"cursorKey"`
}
type CursorFirstResp struct {
	Value thing.ThingValue `json:"value"`
	Valid bool             `json:"valid"`
	// CursorKey with iteration location after the first
	CursorKey string `json:"cursorKey"`
}

const CursorNextMethod = "cursorNext"

type CursorNextArgs struct {
	// CursorKey with current iteration location
	CursorKey string `json:"cursorKey"`
}
type CursorNextResp struct {
	Value thing.ThingValue `json:"value"`
	Valid bool             `json:"valid"`
	// CursorKey with new iteration location
	CursorKey string `json:"cursorKey"`
}

const CursorNextNMethod = "cursorNextN"

type CursorNextNArgs struct {
	// CursorKey with current iteration location
	CursorKey string `json:"cursorKey"`
	Limit     uint   `json:"limit"`
}
type CursorNextNResp struct {
	Values         []thing.ThingValue `json:"values"`
	ItemsRemaining bool               `json:"itemsRemaining"`
	// CursorKey with new iteration location
	CursorKey string `json:"cursorKey"`
}

const CursorReleaseMethod = "cursorRelease"

type CursorReleaseArgs struct {
	CursorKey string `json:"cursorKey"`
}

const GetCursorMethod = "getCursor"

// GetCursorResp returns a read cursor
type GetCursorResp struct {
	// Iterator identifier
	CursorKey string `json:"cursorKey"`
}

const GetTDMethod = "getTD"

type GetTDArgs struct {
	AgentID string `json:"agentID"`
	ThingID string `json:"thingID"`
}
type GetTDResp struct {
	Value thing.ThingValue `json:"value"`
}

const GetTDsMethod = "getTDs"

type GetTDsArgs struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
type GetTDsResp struct {
	Values []thing.ThingValue `json:"values"`
}

//--- Interface

// IDirectoryCursor is a cursor to iterate the directory
type IDirectoryCursor interface {
	// First return the first directory entry.
	//  tdDoc contains the serialized TD document
	// Returns nil if the store is empty
	First() (value thing.ThingValue, valid bool, err error)

	// Next returns the next directory entry
	// Returns nil when trying to read past the last value
	Next() (value thing.ThingValue, valid bool, err error)

	// NextN returns a batch of next directory entries
	// Returns empty list when trying to read past the last value
	// itemsRemaining is true as long as more items can be retrieved
	// limit provides the maximum number of items to obtain.
	NextN(limit uint) (batch []thing.ThingValue, itemsRemaining bool, err error)

	// Release the cursor after use
	Release()
}

//// IReadDirectory defines the capability of reading the Thing directory
//type IReadDirectory interface {
//	// GetCursor returns an iterator for ThingValue objects containing TD documents
//	GetCursor() (cursor IDirectoryCursor, err error)
//
//	// GetTD returns the TD document for the given Device/Thing ID in JSON format.
//	// Returns the value containing the JSON serialized TD document
//	// or nil if the agent/thing doesn't exist, and an error if the store is not reachable.
//	GetTD(agentID, thingID string) (value thing.ThingValue, err error)
//
//	// GetTDs returns a batch of TD values
//	GetTDs(offset int, limit int) (value []thing.ThingValue, err error)
//}
