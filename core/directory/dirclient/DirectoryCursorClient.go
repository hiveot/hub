// Package capnpclient that wraps the capnp generated client with a POGS API
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
	return resp.Value, false, err
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
func (cl *DirectoryCursorClient) NextN(limit int) (batch []thing.ThingValue, itemsRemaining bool, err error) {
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
}

// NewDirectoryCursorClient returns a read cursor client
// Intended for internal use.
//
//	cursorKey is the iterator key obtain when requesting the cursor
//	hc connection to the Hub
func NewDirectoryCursorClient(cursorKey string, hc hubclient.IHubClient) *DirectoryCursorClient {
	cl := &DirectoryCursorClient{
		cursorKey: cursorKey,
		agentID:   directory.ServiceName,
		capID:     directory.ReadDirectoryCapability,
		hc:        hc,
	}
	return cl
}