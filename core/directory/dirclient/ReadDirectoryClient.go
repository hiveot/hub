// Package capnpclient that wraps the capnp generated client with a POGS API
package dirclient

import (
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
)

// ReadDirectoryClient is the messenger client for reading the Thing Directory
// This implements the IReadDirectory interface
type ReadDirectoryClient struct {
	// agent handling the request
	agentID string
	// directory capability to use
	capID string
	hc    hubclient.IHubClient
}

// GetCursor returns an iterator for ThingValue objects containing TD documents
func (cl *ReadDirectoryClient) GetCursor() (directory.IDirectoryCursor, error) {
	resp := directory.GetCursorResp{}
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.GetCursorMethod, nil, &resp)
	cursor := NewDirectoryCursorClient(resp.CursorKey, cl.hc)
	return cursor, err
}

// GetTD returns a thing value containing the TD document for the given Thing address
func (cl *ReadDirectoryClient) GetTD(
	agentID string, thingID string) (tv thing.ThingValue, err error) {

	req := &directory.GetTDArgs{
		AgentID: agentID,
		ThingID: thingID,
	}
	resp := &directory.GetTDResp{}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.GetTDMethod, &req, &resp)
	return resp.Value, err
}

// GetTDs returns a batch of TD documents.
// The order is undefined.
func (cl *ReadDirectoryClient) GetTDs(
	offset int, limit int) (tv []thing.ThingValue, err error) {

	req := &directory.GetTDsArgs{
		Offset: offset,
		Limit:  limit,
	}
	resp := &directory.GetTDsResp{}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.GetTDsMethod, &req, &resp)
	return resp.Values, err
}

func NewReadDirectoryClient(hc hubclient.IHubClient) directory.IReadDirectory {
	return &ReadDirectoryClient{
		agentID: directory.ServiceName,
		capID:   directory.ReadDirectoryCapability,
		hc:      hc,
	}
}
