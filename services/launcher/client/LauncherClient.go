package launcher_client

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hub/services/launcher"
)

// LauncherClient is a marshaller for service messages using a provided hub connection.
// This uses the default serializer to marshal and unmarshal messages.
type LauncherClient struct {
	*consumer.Consumer

	// ID of the launcher service that handles the requests
	serviceID string
}

// List services
//
//	onlyRunning includes only the running plugins
//
// This returns the list of available or running plugins.
// An error is returned if the list could not be obtained.
func (cl *LauncherClient) ListPlugins(onlyRunning bool) ([]launcher.PluginInfo, error) {

	var output []launcher.PluginInfo
	err := cl.InvokeAction(cl.serviceID,
		launcher.ListPluginsAction, onlyRunning, &output)
	return output, err
}

// StartPlugin requests to start a plugin
//
//	name of the plugin to start as provided by ListAllPlugins
//
// This returns n error if the plugin could not be started
func (cl *LauncherClient) StartPlugin(name string) (launcher.PluginInfo, error) {
	var output launcher.PluginInfo
	err := cl.InvokeAction(cl.serviceID, launcher.StartPluginAction, name, &output)
	return output, err
}

// StartAllPlugins starts all enabled plugins in the order from the config
//
// # Use listPlugins() to determine which plugins started.
//
// This returns the error from the last service that could not be started
func (cl *LauncherClient) StartAllPlugins() error {
	err := cl.InvokeAction(cl.serviceID, launcher.StartAllPluginsAction, nil, nil)
	return err
}

// StopPlugin stops a running plugin
//
//	name of the plugin to stop
//	returns the updated plugin status
func (cl *LauncherClient) StopPlugin(name string) (launcher.PluginInfo, error) {
	var output launcher.PluginInfo
	err := cl.InvokeAction(cl.serviceID, launcher.StopPluginAction, name, &output)
	return output, err
}

// StopAllPlugins stops running plugins
//
//	fullStop also stop the runtime
func (cl *LauncherClient) StopAllPlugins(fullStop bool) error {
	err := cl.InvokeAction(cl.serviceID, launcher.StopAllPluginsAction, fullStop, nil)
	return err
}

// NewLauncherClient returns a launcher service client
//
// The provided sink must have a path to the launcher service.
//
//	serviceID is the thingID of the launcher service. "" for default.
//	sink is the transport module sink.
func NewLauncherClient(serviceID string, sink api.IHiveCell) *LauncherClient {
	if serviceID == "" {
		serviceID = launcher.LauncherServiceID
	}
	cl := &LauncherClient{
		serviceID: serviceID,
	}
	cl.SetRequestSink(sink)
	sink.SetNotificationSink(cl, serviceID)

	var _ api.IHiveCell = cl

	return cl
}
