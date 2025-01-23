package launcherclient

import (
	"fmt"
	"github.com/hiveot/hub/services/launcher/launcherapi"
	"github.com/hiveot/hub/transports/messaging"
	"github.com/hiveot/hub/wot/td"
)

// LauncherClient is a marshaller for service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type LauncherClient struct {
	// ID of the launcher service that handles the requests
	dThingID string // capability
	//serviceID string
	co *messaging.Consumer
}

// List services
func (cl *LauncherClient) List(onlyRunning bool) ([]launcherapi.PluginInfo, error) {

	req := launcherapi.ListArgs{
		OnlyRunning: onlyRunning,
	}
	resp := launcherapi.ListResp{}
	err := cl.co.InvokeAction(cl.dThingID, launcherapi.ListMethod, req, &resp)
	return resp.PluginInfoList, err
}

// Start cannot start remotely
func (cl *LauncherClient) Start() error {
	return fmt.Errorf("cannot start launcher remotely")
}

// StartPlugin requests to start a plugin
func (cl *LauncherClient) StartPlugin(name string) (launcherapi.PluginInfo, error) {

	req := launcherapi.StartPluginArgs{
		Name: name,
	}
	resp := launcherapi.StartPluginResp{}
	err := cl.co.InvokeAction(cl.dThingID, launcherapi.StartPluginMethod, req, &resp)
	return resp.PluginInfo, err
}

// StartAllPlugins starts all enabled plugins
// This returns the error from the last service that could not be started
func (cl *LauncherClient) StartAllPlugins() error {
	err := cl.co.InvokeAction(cl.dThingID, launcherapi.StartAllPluginsMethod, nil, nil)
	return err
}

// Stop cannot stop remotely
func (cl *LauncherClient) Stop() error {
	return fmt.Errorf("cannot stop launcher remotely")
}

// StopPlugin stops a running plugin
func (cl *LauncherClient) StopPlugin(name string) (launcherapi.PluginInfo, error) {
	req := launcherapi.StopPluginArgs{
		Name: name,
	}
	resp := launcherapi.StopPluginResp{}
	err := cl.co.InvokeAction(cl.dThingID, launcherapi.StopPluginMethod, req, &resp)
	return resp.PluginInfo, err
}

// StopAllPlugins stops running plugins
func (cl *LauncherClient) StopAllPlugins() error {
	req := launcherapi.StopAllPluginsArgs{
		IncludingRuntime: false,
	}
	err := cl.co.InvokeAction(cl.dThingID, launcherapi.StopAllPluginsMethod, &req, nil)
	return err
}

// NewLauncherClient returns a launcher service client
//
//	launcherID is the optional ID of the launcher to use. Default is 'launcher'
//	co is the hub client connection to use.
func NewLauncherClient(agentID string, hc *messaging.Consumer) *LauncherClient {
	if agentID == "" {
		agentID = launcherapi.AgentID
	}
	cl := LauncherClient{
		co:       hc,
		dThingID: td.MakeDigiTwinThingID(agentID, launcherapi.ManageServiceID),
	}
	return &cl
}
