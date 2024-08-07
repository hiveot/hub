package historyclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/history/historyapi"
	"time"
)

// ReadHistoryClient for talking to the history service
type ReadHistoryClient struct {
	// ThingID of the service providing the read history capability
	dThingID string
	hc       hubclient.IHubClient
}

// GetCursor returns an iterator for ThingMessage objects containing historical events,tds or actions
// This returns a release function that MUST be called after completion.
//
//	thingID the event or action belongs to
//	filterOnKey option filter on a specific event or action name
func (cl *ReadHistoryClient) GetCursor(thingID string, filterOnKey string) (
	cursor *HistoryCursorClient, releaseFn func(), err error) {

	args := historyapi.GetCursorArgs{
		ThingID:     thingID,
		FilterOnKey: filterOnKey,
	}
	resp := historyapi.GetCursorResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.GetCursorMethod, &args, &resp)
	cursor = NewHistoryCursorClient(cl.hc, resp.CursorKey)
	return cursor, cursor.Release, err
}

// ReadHistory returns a list of historical messages in time order.
//
//	thingID the event or action belongs to
//	filterOnKey option filter on a specific event or action name
//	timestamp to start/end
//	duration number of seconds to return. Use negative number to go back in time.
//	limit max nr of items to return. Use 0 for max limit
//
// This returns a list of messages and a flag indicating of all duration was returned
// or whether items were remaining. If items were remaining them use the last entry
// to continue reading the next page.
func (cl *ReadHistoryClient) ReadHistory(thingID string, filterOnKey string,
	timestamp time.Time, duration int, limit int) (
	batch []*things.ThingMessage, itemsRemaining bool, err error) {

	args := historyapi.ReadHistoryArgs{
		ThingID:     thingID,
		FilterOnKey: filterOnKey,
		Timestamp:   timestamp.Format(time.RFC3339),
		Duration:    duration,
		Limit:       limit,
	}
	resp := historyapi.ReadHistoryResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.ReadHistoryMethod, &args, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// NewReadHistoryClient returns an instance of the read history client using the given connection
func NewReadHistoryClient(hc hubclient.IHubClient) *ReadHistoryClient {
	agentID := historyapi.AgentID
	histCl := ReadHistoryClient{
		hc:       hc,
		dThingID: things.MakeDigiTwinThingID(agentID, historyapi.ReadHistoryServiceID),
	}
	return &histCl
}
