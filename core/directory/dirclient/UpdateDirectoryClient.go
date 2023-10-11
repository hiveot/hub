// Package capnpclient that wraps the capnp generated client with a POGS API
package dirclient

import (
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/lib/hubclient"
)

// UpdateDirectoryClient is the client to updating a directory
// It can only be obtained from the DirectoryCapnpClient
type UpdateDirectoryClient struct {
	// agent handling the request
	agentID string
	// directory capability to use
	capID string
	hc    hubclient.IHubClient
}

// RemoveTD removes a TD document from the directory
func (cl *UpdateDirectoryClient) RemoveTD(agentID, thingID string) (err error) {
	req := &directory.RemoveTDArgs{
		AgentID: agentID,
		ThingID: thingID,
	}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.RemoveTDMethod, &req, nil)
	return err
}

// UpdateTD updates the TD document in the directory
// If the TD with the given ID doesn't exist it will be added.
func (cl *UpdateDirectoryClient) UpdateTD(agentID, thingID string, tdDoc []byte) (err error) {
	req := &directory.UpdateTDArgs{
		AgentID: agentID,
		ThingID: thingID,
		TDDoc:   tdDoc,
	}
	_, err = cl.hc.PubRPCRequest(cl.agentID, cl.capID, directory.UpdateTDMethod, &req, nil)
	return err
}

// NewUpdateDirectoryClient returns a directory update client for the directory service
func NewUpdateDirectoryClient(hc hubclient.IHubClient) directory.IUpdateDirectory {
	cl := &UpdateDirectoryClient{
		agentID: directory.ServiceName,
		capID:   directory.UpdateDirectoryCapability,
		hc:      hc,
	}
	return cl
}
