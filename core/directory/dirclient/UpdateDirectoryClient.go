// Package capnpclient that wraps the capnp generated client with a POGS API
package dirclient

import (
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/lib/hubclient"
)

// UpdateDirectoryClient is the client to updating a directory
// It can only be obtained from the DirectoryCapnpClient
type UpdateDirectoryClient struct {
	// agent handling the request
	agentID string
	// directory capability to use
	capID string
	hc    *hubclient.HubClient
}

// RemoveTD removes a TD document from the directory
func (cl *UpdateDirectoryClient) RemoveTD(agentID, thingID string) (err error) {
	req := &directoryapi.RemoveTDArgs{
		AgentID: agentID,
		ThingID: thingID,
	}
	err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.RemoveTDMethod, &req, nil)
	return err
}

// UpdateTD updates the TD document in the directory
// If the TD with the given ID doesn't exist it will be added.
func (cl *UpdateDirectoryClient) UpdateTD(agentID, thingID string, tdDoc []byte) (err error) {
	req := &directoryapi.UpdateTDArgs{
		AgentID: agentID,
		ThingID: thingID,
		TDDoc:   tdDoc,
	}
	err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directoryapi.UpdateTDMethod, &req, nil)
	return err
}

// NewUpdateDirectoryClient returns a directory update client for the directory service.
// This connects to the service with the default directory service name.
func NewUpdateDirectoryClient(hc *hubclient.HubClient) *UpdateDirectoryClient {
	cl := &UpdateDirectoryClient{
		agentID: directoryapi.ServiceName,
		capID:   directoryapi.UpdateDirectoryCap,
		hc:      hc,
	}
	return cl
}
