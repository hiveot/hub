package historyclient

import (
	"github.com/hiveot/hub/services/history/historyapi"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/messaging"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"time"
)

// ReadHistoryClient for talking to the history service
type ReadHistoryClient struct {
	// ThingID of the service providing the read history capability
	dThingID string
	co       *messaging.Consumer
}

// GetCursor returns an iterator for ThingMessage objects containing historical events,tds or actions
// This returns a release function that MUST be called after completion.
//
//	thingID the event or action belongs to
//	filterOnName option filter on a specific event or action name
func (cl *ReadHistoryClient) GetCursor(thingID string, filterOnName string) (
	cursor *HistoryCursorClient, releaseFn func(), err error) {

	args := historyapi.GetCursorArgs{
		ThingID:      thingID,
		FilterOnName: filterOnName,
	}
	resp := historyapi.GetCursorResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, historyapi.GetCursorMethod, &args, &resp)
	cursor = NewHistoryCursorClient(cl.co, resp.CursorKey)
	return cursor, cursor.Release, err
}

// ReadHistory returns a list of historical messages in time order.
//
//	thingID the event or action belongs to
//	FilterOnName option filter on a specific event or action name
//	timestamp to start/end
//	duration number of seconds to return. Use negative number to go back in time.
//	limit max nr of items to return. Use 0 for max limit
//
// This returns a list of messages and a flag indicating of all duration was returned
// or whether items were remaining. If items were remaining them use the last entry
// to continue reading the next page.
func (cl *ReadHistoryClient) ReadHistory(thingID string, filterOnName string,
	timestamp time.Time, duration time.Duration, limit int) (
	batch []*transports.ResponseMessage, itemsRemaining bool, err error) {

	args := historyapi.ReadHistoryArgs{
		ThingID:      thingID,
		FilterOnName: filterOnName,
		Timestamp:    timestamp.Format(time.RFC3339),
		Duration:     int(duration.Seconds()),
		Limit:        limit,
	}
	resp := historyapi.ReadHistoryResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, historyapi.ReadHistoryMethod, &args, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// NewReadHistoryClient returns an instance of the read history client using the given connection
//
//	invokeAction is the TD invokeAction for the invoke-action operation of the history service
func NewReadHistoryClient(co *messaging.Consumer) *ReadHistoryClient {
	agentID := historyapi.AgentID
	histCl := ReadHistoryClient{
		co:       co,
		dThingID: td.MakeDigiTwinThingID(agentID, historyapi.ReadHistoryServiceID),
	}
	return &histCl
}
