package service

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/transports"
	"github.com/hiveot/hub/services/launcher/launcherapi"
)

// StartLauncherAgent returns a new instance of the agent for the launcher service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the service whose capabilities to expose
//	hc is the optional message client connected to the server protocol
func StartLauncherAgent(svc *LauncherService, hc hubclient.IConsumerClient) *transports.AgentHandler {

	methods := map[string]interface{}{
		launcherapi.ListMethod:            svc.List,
		launcherapi.StartPluginMethod:     svc.StartPlugin,
		launcherapi.StopPluginMethod:      svc.StopPlugin,
		launcherapi.StartAllPluginsMethod: svc.StartAllPlugins,
		launcherapi.StopAllPluginsMethod:  svc.StopAllPlugins,
	}

	ah := transports.NewAgentHandler(launcherapi.ManageServiceID, methods)

	hc.SetMessageHandler(ah.HandleMessage)
	return ah
}
