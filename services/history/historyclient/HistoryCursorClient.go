package historyclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/services/history/historyapi"

	"github.com/hiveot/hub/lib/things"
)

// HistoryCursorClient provides iterator client for iterating the history
type HistoryCursorClient struct {
	// the key identifying this cursor
	cursorKey string

	// agent handling the request
	agentID string
	// history capability to use
	capThingID string
	hc         hubclient.IHubClient
}

// First positions the cursor at the first key in the ordered list
func (cl *HistoryCursorClient) First() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	_, err = cl.hc.Rpc(nil, cl.capThingID, historyapi.CursorFirstMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Last positions the cursor at the last key in the ordered list
func (cl *HistoryCursorClient) Last() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	_, err = cl.hc.Rpc(nil, cl.capThingID, historyapi.CursorLastMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
func (cl *HistoryCursorClient) Next() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	_, err = cl.hc.Rpc(nil, cl.capThingID, historyapi.CursorNextMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
func (cl *HistoryCursorClient) NextN(limit int) (batch []*things.ThingMessage, itemsRemaining bool, err error) {
	req := historyapi.CursorNArgs{
		CursorKey: cl.cursorKey,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	_, err = cl.hc.Rpc(cl.agentID, cl.capThingID, historyapi.CursorNextNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Prev moves the cursor to the previous key from the current cursor
func (cl *HistoryCursorClient) Prev() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	_, err = cl.hc.Rpc(cl.agentID, cl.capThingID, historyapi.CursorPrevMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// PrevN moves the cursor to the previous N steps from the current cursor
func (cl *HistoryCursorClient) PrevN(limit int) (batch []*things.ThingMessage, itemsRemaining bool, err error) {
	req := historyapi.CursorNArgs{
		CursorKey: cl.cursorKey,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	_, err = cl.hc.Rpc(cl.agentID, cl.capThingID, historyapi.CursorPrevNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Release the cursor capability
func (cl *HistoryCursorClient) Release() {
	req := historyapi.CursorReleaseArgs{
		CursorKey: cl.cursorKey,
	}
	err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, historyapi.CursorReleaseMethod, &req, nil)
	_ = err
	return
}

// Seek the starting point for iterating the history
func (cl *HistoryCursorClient) Seek(timeStampMSec int64) (
	thingValue *things.ThingMessage, valid bool, err error) {

	req := historyapi.CursorSeekArgs{
		CursorKey:     cl.cursorKey,
		TimeStampMSec: timeStampMSec,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.hc.Rpc(cl.agentID, cl.thingID, historyapi.CursorSeekMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NewHistoryCursorClient returns a read cursor client
// Intended for internal use.
//
//	agentID of the history service
//	capID of the read capability
//	cursorKey is the iterator key obtain when requesting the cursor
//	hc connection to the Hub
func NewHistoryCursorClient(
	hc hubclient.IHubClient, agentID, capID string, cursorKey string) *HistoryCursorClient {

	cl := &HistoryCursorClient{
		cursorKey: cursorKey,
		agentID:   agentID,
		capID:     capID,
		hc:        hc,
	}
	return cl
}
