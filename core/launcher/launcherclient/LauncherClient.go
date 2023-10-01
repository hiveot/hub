package launcherclient

import (
	"fmt"
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
func (cl *LauncherClient) List(onlyRunning bool) ([]launcher.PluginInfo, error) {

	req := launcher.LauncherListReq{
		OnlyRunning: onlyRunning,
	}
	resp := launcher.LauncherListResp{}
	err := cl.pubReq(launcher.LauncherListRPC, req, &resp)
	return resp.ServiceInfoList, err
}

// Start cannot start remotely
func (cl *LauncherClient) Start() error {
	return fmt.Errorf("cannot start launcher remotely")
}

// StartPlugin requests to start a plugin
func (cl *LauncherClient) StartPlugin(name string) (launcher.PluginInfo, error) {

	req := launcher.LauncherStartPluginReq{
		Name: name,
	}
	resp := launcher.LauncherStartPluginResp{}
	err := cl.pubReq(launcher.LauncherStartPluginRPC, req, &resp)
	return resp.ServiceInfo, err
}

// StartAllPlugins starts all enabled plugins
// This returns the error from the last service that could not be started
func (cl *LauncherClient) StartAllPlugins() error {
	err := cl.pubReq(launcher.LauncherStartAllPluginsRPC, nil, nil)
	return err
}

// Stop cannot stop remotely
func (cl *LauncherClient) Stop() error {
	return fmt.Errorf("cannot stop launcher remotely")
}

// StopPlugin stops a running plugin
func (cl *LauncherClient) StopPlugin(name string) (launcher.PluginInfo, error) {
	req := launcher.LauncherStopPluginReq{
		Name: name,
	}
	resp := launcher.LauncherStopPluginResp{}
	err := cl.pubReq(launcher.LauncherStopPluginRPC, req, &resp)
	return resp.ServiceInfo, err
}

// StopAllPlugins stops running plugins
func (cl *LauncherClient) StopAllPlugins() error {
	err := cl.pubReq(launcher.LauncherStopAllPluginsRPC, nil, nil)
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
