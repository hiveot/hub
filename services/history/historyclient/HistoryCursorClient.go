package historyclient

import "C"
import (
	"github.com/hiveot/hub/services/history/historyapi"
	transports2 "github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"time"
)

// HistoryCursorClient provides iterator client for iterating the history
type HistoryCursorClient struct {
	// the key identifying this cursor
	cursorKey string

	// history cursor service ID
	dThingID string
	cc       transports2.IClientConnection
}

// First positions the cursor at the first key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) First() (thingValue *transports2.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorFirstMethod, &req, &resp)
	//err = cl.cc.SendRequest(cl.dThingID, historyapi.CursorFirstMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Last positions the cursor at the last key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Last() (thingValue *transports2.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorLastMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Next() (thingValue *transports2.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorNextMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) NextN(limit int, until string) (batch []*transports2.ThingMessage, itemsRemaining bool, err error) {
	req := historyapi.CursorNArgs{
		CursorKey: cl.cursorKey,
		Until:     until,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	err = cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorNextNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Prev moves the cursor to the previous key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Prev() (thingValue *transports2.ThingMessage, valid bool, err error) {
	req := historyapi.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorPrevMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// PrevN moves the cursor to the previous N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) PrevN(limit int, until string) (batch []*transports2.ThingMessage, itemsRemaining bool, err error) {
	req := historyapi.CursorNArgs{
		CursorKey: cl.cursorKey,
		Until:     until,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	err = cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorPrevNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Release the cursor capability
func (cl *HistoryCursorClient) Release() {
	req := historyapi.CursorReleaseArgs{
		CursorKey: cl.cursorKey,
	}
	err := cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorReleaseMethod, &req, nil)
	_ = err
	return
}

// Seek the starting point for iterating the history
// timeStamp in ISO8106 format
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Seek(timeStamp time.Time) (
	thingValue *transports2.ThingMessage, valid bool, err error) {
	timeStampStr := timeStamp.Format(wot.RFC3339Milli)
	req := historyapi.CursorSeekArgs{
		CursorKey: cl.cursorKey,
		TimeStamp: timeStampStr,
	}
	resp := historyapi.CursorSingleResp{}
	err = cl.cc.SendRequest(wot.OpInvokeAction, cl.dThingID, historyapi.CursorSeekMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NewHistoryCursorClient returns a read cursor client
// Intended for internal use.
//
//	cc client connection to the Hub
//	serviceID of the read capability
//	cursorKey is the iterator key obtain when requesting the cursor
func NewHistoryCursorClient(cc transports2.IClientConnection, cursorKey string) *HistoryCursorClient {
	agentID := historyapi.AgentID
	serviceID := historyapi.ReadHistoryServiceID
	cl := &HistoryCursorClient{
		cursorKey: cursorKey,
		// history cursor serviceID
		dThingID: td.MakeDigiTwinThingID(agentID, serviceID),
		cc:       cc,
	}
	return cl
}
