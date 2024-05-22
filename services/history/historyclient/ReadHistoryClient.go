package historyclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/history/historyapi"
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
//	agentID of the publisher of the event or action
//	thingID the event or action belongs to
//	filterOnKey option filter on a specific event or action name
func (cl *ReadHistoryClient) GetCursor(thingID string, filterOnKey string) (
	cursor *HistoryCursorClient, releaseFn func(), err error) {

	req := historyapi.GetCursorArgs{
		ThingID:     thingID,
		FilterOnKey: filterOnKey,
	}
	resp := historyapi.GetCursorResp{}
	err = cl.hc.Rpc(cl.dThingID, historyapi.GetCursorMethod, &req, &resp)
	cursor = NewHistoryCursorClient(cl.hc, resp.CursorKey)
	return cursor, cursor.Release, err
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
