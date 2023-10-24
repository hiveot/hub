// Package capnpclient that wraps the capnp generated client with a POGS API
package dirclient

import (
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
)

// ReadDirectoryClient is the messenger client for reading the Thing Directory
// This implements the IReadDirectory interface
type ReadDirectoryClient struct {
	// agent handling the request
	agentID string
	// capability to use
	capID string
	hc    hubclient.IHubClient
}

// GetCursor returns an iterator for ThingValue objects containing TD documents
func (cl *ReadDirectoryClient) GetCursor() (directoryapi.IDirectoryCursor, error) {
	resp := directoryapi.GetCursorResp{}
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.GetCursorMethod, nil, &resp)
	cursor := NewDirectoryCursorClient(cl.hc, cl.agentID, cl.capID, resp.CursorKey)
	return cursor, err
}

// GetTD returns a thing value containing the TD document for the given Thing address
func (cl *ReadDirectoryClient) GetTD(
	agentID string, thingID string) (tv thing.ThingValue, err error) {

	req := &directoryapi.GetTDArgs{
		AgentID: agentID,
		ThingID: thingID,
	}
	resp := &directoryapi.GetTDResp{}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.GetTDMethod, &req, &resp)
	return resp.Value, err
}

// GetTDs returns a batch of TD documents.
// The order is undefined.
func (cl *ReadDirectoryClient) GetTDs(
	offset int, limit int) (tv []thing.ThingValue, err error) {

	req := &directoryapi.GetTDsArgs{
		Offset: offset,
		Limit:  limit,
	}
	resp := &directoryapi.GetTDsResp{}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.GetTDsMethod, &req, &resp)
	return resp.Values, err
}

// NewReadDirectoryClient creates a instance of a read-directory client
// This connects to the service with the default directory service name.
func NewReadDirectoryClient(hc hubclient.IHubClient) *ReadDirectoryClient {
	return &ReadDirectoryClient{
		agentID: directoryapi.ServiceName,
		capID:   directoryapi.ReadDirectoryCap,
		hc:      hc,
	}
}
