package digitwinclient

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/digitwin/digitwinagent"
	"log/slog"
)

// DirectoryClient is a convenience client for reading the Thing Directory.
// It is similar to the generated client except it returns the native things.TD objects
// instead of serialized TD documents.
type DirectoryClient struct {
	hc       hubclient.IHubClient
	dThingID string
}

// ReadTD returns a single TD document
func (cl *DirectoryClient) ReadTD(thingID string) (td *things.TD, err error) {
	resp, err := directory.ReadTD(cl.hc, directory.ReadTDArgs{ThingID: thingID})
	if err != nil {
		return nil, err
	}
	td = &things.TD{}
	err = json.Unmarshal([]byte(resp.Output), td)
	return td, err
}

// ReadTDs returns a batch of TD documents
func (cl *DirectoryClient) ReadTDs(offset int, limit int) (tdList []*things.TD, err error) {
	resp, err := directory.ReadTDs(cl.hc, directory.ReadTDsArgs{Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	tdList = make([]*things.TD, 0, len(resp.Output))
	for _, tdJSON := range resp.Output {
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

// NewDirectoryClient creates a new instance of a client to read the hub directory
func NewDirectoryClient(hc hubclient.IHubClient) *DirectoryClient {
	cl := DirectoryClient{
		hc: hc,
		dThingID: things.MakeDigiTwinThingID(
			digitwinagent.DigiTwinAgentID, directory.ServiceID),
	}
	return &cl
}
