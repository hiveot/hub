package digitwinclient

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/digitwin/digitwinagent"
)

// OutboxClient is a convenience client for reading the latest event values in the outbox.
// It is similar to the generated client except it returns the native values.
type OutboxClient struct {
	hc       hubclient.IHubClient
	dThingID string
}

// ReadLatest returns a last known Thing values
func (cl *OutboxClient) ReadLatest(dThingID string) (values things.ThingMessageMap, err error) {
	args := outbox.ReadLatestArgs{ThingID: dThingID}
	resp := outbox.ReadLatestResp{}
	err = cl.hc.Rpc(cl.dThingID, outbox.ReadLatestMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	eventValues := things.ThingMessageMap{}
	err = json.Unmarshal([]byte(resp.Values), &eventValues)
	return eventValues, err
}

// NewOutboxClient creates a new instance of a client to read the hub directory
func NewOutboxClient(hc hubclient.IHubClient) *OutboxClient {
	cl := OutboxClient{
		hc: hc,
		dThingID: things.MakeDigiTwinThingID(
			digitwinagent.DigiTwinAgentID, outbox.ServiceID),
	}
	return &cl
}
