package launcherclient

import (
	"github.com/hiveot/hub/api/go/certs"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/launcher"
	"github.com/hiveot/hub/lib/ser"
)

// LauncherClient is a marshaller for service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type LauncherClient struct {
	// ID of the certs service that handles the requests
	serviceID string
	hc        hubclient.IHubClient
}

// helper for publishing an rpc request to the launcher service
func (cl *LauncherClient) pubReq(action string, req interface{}, resp interface{}) error {
	var msg []byte
	if req != nil {
		msg, _ = ser.Marshal(req)
	}

	data, err := cl.hc.PubServiceRPC(cl.serviceID, certs.CertsManageCertsCapability, action, msg)
	if err != nil {
		return err
	}
	if data.ErrorReply != nil {
		return data.ErrorReply
	}
	err = cl.hc.ParseResponse(data.Payload, resp)
	return err
}

// List services
func (cl *LauncherClient) List(onlyRunning bool) ([]launcher.ServiceInfo, error) {

	req := launcher.LauncherListReq{
		OnlyRunning: onlyRunning,
	}
	resp := launcher.LauncherListResp{}
	err := cl.pubReq(launcher.LauncherListRPC, req, &resp)
	return resp.ServiceInfoList, err
}

// StartService start a service
func (cl *LauncherClient) StartService(name string) (launcher.ServiceInfo, error) {

	req := launcher.LauncherStartServiceReq{
		Name: name,
	}
	resp := launcher.LauncherStartServiceResp{}
	err := cl.pubReq(launcher.LauncherStartServiceRPC, req, &resp)
	return resp.ServiceInfo, err
}

// StartAll starts all enabled services
// This returns the error from the last service that could not be started
func (cl *LauncherClient) StartAll() error {
	err := cl.pubReq(launcher.LauncherStartAllRPC, nil, nil)
	return err
}

// StopService stops a running service
func (cl *LauncherClient) StopService(name string) (launcher.ServiceInfo, error) {
	req := launcher.LauncherStopServiceReq{
		Name: name,
	}
	resp := launcher.LauncherStopServiceResp{}
	err := cl.pubReq(launcher.LauncherStopServiceRPC, req, &resp)
	return resp.ServiceInfo, err
}

// StopAll running services
func (cl *LauncherClient) StopAll() error {
	err := cl.pubReq(launcher.LauncherStopAllRPC, nil, nil)
	return err
}

// NewLauncherClient returns a launcher service client
//
//	hc is the hub client connection to use
func NewLauncherClient(hc hubclient.IHubClient) *LauncherClient {
	serviceID := launcher.ServiceName
	cl := LauncherClient{
		hc:        hc,
		serviceID: serviceID,
	}
	return &cl
}
