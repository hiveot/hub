package dirclient

import (
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/lib/hubclient"

	"github.com/hiveot/hub/lib/thing"
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
	hc    hubclient.IHubClient
}

// First positions the cursor at the first key in the ordered list
func (cl *DirectoryCursorClient) First() (thingValue thing.ThingValue, valid bool, err error) {
	req := directory.CursorFirstArgs{
		CursorKey: cl.cursorKey,
	}
	resp := directory.CursorFirstResp{}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.CursorFirstMethod, &req, &resp)
	cl.cursorKey = resp.CursorKey
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
func (cl *DirectoryCursorClient) Next() (thingValue thing.ThingValue, valid bool, err error) {
	req := directory.CursorNextArgs{
		CursorKey: cl.cursorKey,
	}
	resp := directory.CursorFirstResp{}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.CursorNextMethod, &req, &resp)
	cl.cursorKey = resp.CursorKey
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
func (cl *DirectoryCursorClient) NextN(limit uint) (batch []thing.ThingValue, itemsRemaining bool, err error) {
	req := directory.CursorNextNArgs{
		CursorKey: cl.cursorKey,
		Limit:     limit,
	}
	resp := directory.CursorNextNResp{}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.CursorNextNMethod, &req, &resp)
	cl.cursorKey = resp.CursorKey
	return resp.Values, resp.ItemsRemaining, err
}

// Release the cursor capability
func (cl *DirectoryCursorClient) Release() {
	req := directory.CursorReleaseArgs{
		CursorKey: cl.cursorKey,
	}
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.CursorReleaseMethod, &req, nil)
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
func NewDirectoryCursorClient(hc hubclient.IHubClient, agentID, capID string, cursorKey string) *DirectoryCursorClient {
	cl := &DirectoryCursorClient{
		cursorKey: cursorKey,
		agentID:   agentID,
		capID:     capID,
		hc:        hc,
	}
	return cl
}
