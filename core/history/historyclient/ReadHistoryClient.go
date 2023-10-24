package historyclient

import (
	"github.com/hiveot/hub/core/history/historyapi"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
)

// ReadHistoryClient for talking to the history service
type ReadHistoryClient struct {
	// service providing the history capability
	serviceID string
	// capability to use
	capID string
	hc    hubclient.IHubClient
}

// GetCursor returns an iterator for ThingValue objects containing historical events,tds or actions
// This returns a release function that MUST be called after completion.
//
//	agentID of the publisher of the event or action
//	thingID the event or action belongs to
//	name option filter on a specific event or action name
func (cl *ReadHistoryClient) GetCursor(
	agentID string, thingID string, name string) (cursor *HistoryCursorClient, releaseFn func(), err error) {
	req := historyapi.GetCursorArgs{
		AgentID: agentID,
		ThingID: thingID,
		Name:    name,
	}
	resp := historyapi.GetCursorResp{}
	_, err = cl.hc.PubRPCRequest(cl.serviceID, cl.capID, historyapi.GetCursorMethod, &req, &resp)
	cursor = NewHistoryCursorClient(cl.hc, cl.serviceID, cl.capID, resp.CursorKey)
	return cursor, cursor.Release, err
}

// GetLatest returns the latest values of a Thing
//
//	agentID of the publisher of the event or action
//	thingID the event or action belongs to
//	names optionally filter on specific property, event or action names. nil for all values
func (cl *ReadHistoryClient) GetLatest(
	agentID string, thingID string, names []string) ([]*thing.ThingValue, error) {
	args := historyapi.GetLatestArgs{
		AgentID: agentID,
		ThingID: thingID,
		Names:   names,
	}
	resp := historyapi.GetLatestResp{}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, historyapi.GetLatestMethod, &args, &resp)
	return resp.Values, err
}

// NewReadHistoryClient returns an instance of the read history client using the given connection
func NewReadHistoryClient(hc hubclient.IHubClient) *ReadHistoryClient {
	histCl := ReadHistoryClient{
		hc:        hc,
		serviceID: historyapi.ServiceName,
		capID:     historyapi.ReadHistoryCap,
	}
	return &histCl
}
