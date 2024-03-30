package dirclient

import (
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/lib/hubclient"

	"github.com/hiveot/hub/lib/things"
)

// DirectoryCursorClient provides iterator client for iterating the directory
// This implements the IDirectoryCursor interface
type DirectoryCursorClient struct {
	// the key identifying this cursor
	cursorKey string

	// agent handling the request
	agentID string
	// directory capability to use
	capID string
	hc    *hubclient.HubClient
}

// First positions the cursor at the first key in the ordered list
func (cl *DirectoryCursorClient) First() (thingValue things.ThingValue, valid bool, err error) {
	req := directoryapi.CursorFirstArgs{
		CursorKey: cl.cursorKey,
	}
	resp := directoryapi.CursorFirstResp{}
	err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.CursorFirstMethod, &req, &resp)
	cl.cursorKey = resp.CursorKey
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
func (cl *DirectoryCursorClient) Next() (thingValue things.ThingValue, valid bool, err error) {
	req := directoryapi.CursorNextArgs{
		CursorKey: cl.cursorKey,
	}
	resp := directoryapi.CursorFirstResp{}
	err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.CursorNextMethod, &req, &resp)
	cl.cursorKey = resp.CursorKey
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
func (cl *DirectoryCursorClient) NextN(limit uint) (batch []things.ThingValue, itemsRemaining bool, err error) {
	req := directoryapi.CursorNextNArgs{
		CursorKey: cl.cursorKey,
		Limit:     limit,
	}
	resp := directoryapi.CursorNextNResp{}
	err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.CursorNextNMethod, &req, &resp)
	cl.cursorKey = resp.CursorKey
	return resp.Values, resp.ItemsRemaining, err
}

// Release the cursor capability
func (cl *DirectoryCursorClient) Release() {
	req := directoryapi.CursorReleaseArgs{
		CursorKey: cl.cursorKey,
	}
	err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.CursorReleaseMethod, &req, nil)
	_ = err
	return
}

// NewDirectoryCursorClient returns a read cursor client
// Intended for internal use.
//
//	hc connection to the Hub
//	agentID of the directory service
//	capID of the read capability
//	cursorKey is the iterator key obtain when requesting the cursor
func NewDirectoryCursorClient(hc *hubclient.HubClient, agentID, capID string, cursorKey string) *DirectoryCursorClient {
	cl := &DirectoryCursorClient{
		cursorKey: cursorKey,
		agentID:   agentID,
		capID:     capID,
		hc:        hc,
	}
	return cl
}
