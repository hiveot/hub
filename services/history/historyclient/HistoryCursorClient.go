package historyclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/history/historyapi"
	"time"
)

// HistoryCursorClient provides iterator client for iterating the history
type HistoryCursorClient struct {
	// the key identifying this cursor
	cursorKey string

	// history cursor service ID
	dThingID string
	hc       hubclient.IHubClient
}

// First positions the cursor at the first key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) First() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.CursorFirstMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Last positions the cursor at the last key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Last() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.CursorLastMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Next() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.CursorNextMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) NextN(limit int, until string) (batch []*things.ThingMessage, itemsRemaining bool, err error) {
	req := historyapi.CursorNArgs{
		CursorKey: cl.cursorKey,
		Until:     until,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.CursorNextNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Prev moves the cursor to the previous key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Prev() (thingValue *things.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.CursorPrevMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// PrevN moves the cursor to the previous N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) PrevN(limit int, until string) (batch []*things.ThingMessage, itemsRemaining bool, err error) {
	req := historyapi.CursorNArgs{
		CursorKey: cl.cursorKey,
		Until:     until,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.CursorPrevNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Release the cursor capability
func (cl *HistoryCursorClient) Release() {
	req := historyapi.CursorReleaseArgs{
		CursorKey: cl.cursorKey,
	}
	err := cl.hc.Rpc(cl.dThingID, historyapi.CursorReleaseMethod, &req, nil)
	_ = err
	return
}

// Seek the starting point for iterating the history
// timeStamp in ISO8106 format
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Seek(timeStamp time.Time) (
	thingValue *things.ThingMessage, valid bool, err error) {
	timeStampStr := timeStamp.Format(utils.RFC3339Milli)
	req := historyapi.CursorSeekArgs{
		CursorKey: cl.cursorKey,
		TimeStamp: timeStampStr,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.CursorSeekMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NewHistoryCursorClient returns a read cursor client
// Intended for internal use.
//
//	hc connection to the Hub
//	serviceID of the read capability
//	cursorKey is the iterator key obtain when requesting the cursor
func NewHistoryCursorClient(hc hubclient.IHubClient, cursorKey string) *HistoryCursorClient {
	agentID := historyapi.AgentID
	serviceID := historyapi.ReadHistoryServiceID
	cl := &HistoryCursorClient{
		cursorKey: cursorKey,
		// history cursor serviceID
		dThingID: things.MakeDigiTwinThingID(agentID, serviceID),
		hc:       hc,
	}
	return cl
}
