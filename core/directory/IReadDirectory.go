// Package directory with POGS capability definitions of the directory store.
// Unfortunately capnp does generate POGS types so we need to duplicate them
package directory

import "github.com/hiveot/hub/lib/thing"

// ServiceName is the instance name of the service to connect to
const ServiceName = "directory"

// ReadDirectoryCapability is the capability ID to read the directory
const ReadDirectoryCapability = "readDirectory"

const GetCursorMethod = "getCursor"

// Returns a read cursor
type GetCursorResp struct {
	// Iterator identifier
	CursorKey string
}

const CursorFirstMethod = "first"

type CursorFirstArgs struct {
	// Iterator identifier obtained with GetReadCursorReq
	CursorKey string
}
type CursorFirstResp struct {
	Value thing.ThingValue
	Valid bool
	// CursorKey with iteration location after the first
	CursorKey string
}

const CursorNextMethod = "cursorNext"

type CursorNextArgs struct {
	// CursorKey with current iteration location
	CursorKey string
}
type CursorNextResp struct {
	Value thing.ThingValue
	Valid bool
	// CursorKey with new iteration location
	CursorKey string
}

const CursorNextNMethod = "cursorNextN"

type CursorNextNArgs struct {
	// CursorKey with current iteration location
	CursorKey string
	Limit     int
}
type CursorNextNResp struct {
	Values         []thing.ThingValue
	ItemsRemaining bool
	// CursorKey with new iteration location
	CursorKey string
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
	NextN(limit int) (batch []thing.ThingValue, itemsRemaining bool, err error)

	// Release the cursor after use
	Release()
}

// IReadDirectory defines the capability of reading the Thing directory
type IReadDirectory interface {
	// GetCursor returns an iterator for ThingValue objects containing TD documents
	GetCursor() (cursor IDirectoryCursor, err error)

	// GetTD returns the TD document for the given Device/Thing ID in JSON format.
	// Returns the value containing the JSON serialized TD document
	// or nil if the agent/thing doesn't exist, and an error if the store is not reachable.
	GetTD(agentID, thingID string) (value thing.ThingValue, err error)

	// GetTDs returns a batch of TD values
	GetTDs(offset int, limit int) (value []thing.ThingValue, err error)
}
