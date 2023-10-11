package launcherclient

import (
	"fmt"
	"github.com/hiveot/hub/core/launcher"
	"github.com/hiveot/hub/lib/hubclient"
)

// LauncherClient is a marshaller for service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type LauncherClient struct {
	// ID of the launcher service that handles the requests
	agentID string
	capID   string
	hc      hubclient.IHubClient
}

// List services
func (cl *LauncherClient) List(onlyRunning bool) ([]launcher.PluginInfo, error) {

	req := launcher.LauncherListArgs{
		OnlyRunning: onlyRunning,
	}
	resp := launcher.LauncherListResp{}
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, launcher.LauncherListReq, req, &resp)
	return resp.ServiceInfoList, err
}

// Start cannot start remotely
func (cl *LauncherClient) Start() error {
	return fmt.Errorf("cannot start launcher remotely")
}

// StartPlugin requests to start a plugin
func (cl *LauncherClient) StartPlugin(name string) (launcher.PluginInfo, error) {

	req := launcher.LauncherStartPluginArgs{
		Name: name,
	}
	resp := launcher.LauncherStartPluginResp{}
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, launcher.LauncherStartPluginReq, req, &resp)
	return resp.ServiceInfo, err
}

// StartAllPlugins starts all enabled plugins
// This returns the error from the last service that could not be started
func (cl *LauncherClient) StartAllPlugins() error {
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, launcher.LauncherStartAllPluginsReq, nil, nil)
	return err
}

// Stop cannot stop remotely
func (cl *LauncherClient) Stop() error {
	return fmt.Errorf("cannot stop launcher remotely")
}

// StopPlugin stops a running plugin
func (cl *LauncherClient) StopPlugin(name string) (launcher.PluginInfo, error) {
	req := launcher.LauncherStopPluginArgs{
		Name: name,
	}
	resp := launcher.LauncherStopPluginResp{}
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, launcher.LauncherStopPluginReq, req, &resp)
	return resp.ServiceInfo, err
}

// StopAllPlugins stops running plugins
func (cl *LauncherClient) StopAllPlugins() error {
	_, err := cl.hc.PubRPCRequest(cl.agentID, cl.capID, launcher.LauncherStopAllPluginsReq, nil, nil)
	return err
}

// NewLauncherClient returns a launcher service client
//
//	launcherID is the optional ID of the launcher to use. Default is 'launcher'
//	hc is the hub client connection to use.
func NewLauncherClient(launcherID string, hc hubclient.IHubClient) *LauncherClient {
	if launcherID == "" {
		launcherID = launcher.ServiceName
	}
	cl := LauncherClient{
		hc:      hc,
		agentID: launcherID,
		capID:   launcher.LauncherManageCapability,
	}
	return &cl
}
