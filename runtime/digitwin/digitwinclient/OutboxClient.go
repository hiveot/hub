package digitwinclient

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
)

// ReadOutbox returns a last known event values of a thing from the outbox
func ReadOutbox(hc hubclient.IHubClient, dThingID string) (values things.ThingMessageMap, err error) {
	args := digitwin.OutboxReadLatestArgs{ThingID: dThingID}
	resp := ""
	err = hc.Rpc(digitwin.OutboxDThingID, digitwin.OutboxReadLatestMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	eventValues := things.ThingMessageMap{}
	err = json.Unmarshal([]byte(resp), &eventValues)
	return eventValues, err
}
