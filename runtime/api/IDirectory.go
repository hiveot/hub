package api

import "github.com/hiveot/hub/lib/things"

// DirectoryServiceID Directory service 'thingID'
const DirectoryServiceID = "directory"

// Directory crud method keys
const (
	DirectoryReadTDDMethod   = "readTDD"
	DirectoryReadTDDsMethod  = "readTDDs"
	DirectoryRemoveTDDMethod = "removeTDD"
)

// DirectoryReadTDDArgs
type DirectoryReadTDDArgs struct {
	ThingID string `json:"thingID"`
}

// DirectoryReadTDDResp response containing a single TD document
type DirectoryReadTDDResp struct {
	TDD *things.TD `json:"tdd"`
}

// DirectoryReadTDDsArgs
type DirectoryReadTDDsArgs struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

// DirectoryReadTDDsResp response containing a list of TD documents
type DirectoryReadTDDsResp struct {
	TDDs []*things.TD `json:"tdd"`
}

// DirectoryRemoveTDDArgs arguments for removing a TDD
type DirectoryRemoveTDDArgs struct {
	ThingID string `json:"thingID"`
}

// DirectoryTDEventArgs is a serialized TDD document sent as event
type DirectoryTDEventArgs *things.TD
