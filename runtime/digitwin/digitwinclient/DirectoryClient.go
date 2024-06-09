package digitwinclient

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// ReadTD is a convenience client for reading a single TD document
// It is similar to the generated client except it returns the native things.TD object
// instead of the serialized TD.
func ReadTD(hc hubclient.IHubClient, thingID string) (td *things.TD, err error) {
	args := thingID
	resp := ""
	err = hc.Rpc(digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, &args, &resp)
	if err != nil {
		return nil, err
	}
	td = &things.TD{}
	err = json.Unmarshal([]byte(resp), td)
	return td, err
}

// ReadTDs returns a batch of TD documents
// It is similar to the generated client except it returns the native things.TD object
// instead of the serialized TD.
func ReadTDs(hc hubclient.IHubClient, offset int, limit int) (tdList []*things.TD, err error) {
	args := digitwin.DirectoryReadTDsArgs{Limit: limit, Offset: offset}
	resp := make([]string, 0)
	err = hc.Rpc(digitwin.DirectoryDThingID, digitwin.DirectoryReadTDsMethod, &args, &resp)

	if err != nil {
		return nil, err
	}
	tdList = make([]*things.TD, 0, len(resp))
	for _, tdJSON := range resp {
		td := things.TD{}
		err = json.Unmarshal([]byte(tdJSON), &td)
		if err != nil {
			slog.Error("ReadTDs: unable to unmarshal server response", "err", err.Error)
		} else {
			tdList = append(tdList, &td)
		}
	}
	return tdList, err
}
