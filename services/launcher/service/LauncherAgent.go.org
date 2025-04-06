package service

import (
	"github.com/hiveot/hub/lib/hubagent"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/services/launcher/launcherapi"
)

// StartLauncherAgent returns a new instance of the agent for the launcher service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the service whose capabilities to expose
//	ag is the optional message client connected to the server protocol
func StartLauncherAgent(svc *LauncherService, ag *messaging.Agent) *hubagent.AgentHandler {

	methods := map[string]interface{}{
		launcherapi.ListMethod:            svc.List,
		launcherapi.StartPluginMethod:     svc.StartPlugin,
		launcherapi.StopPluginMethod:      svc.StopPlugin,
		launcherapi.StartAllPluginsMethod: svc.StartAllPlugins,
		launcherapi.StopAllPluginsMethod:  svc.StopAllPlugins,
	}

	agHandler := hubagent.NewAgentHandler(launcherapi.ManageServiceID, methods)
	// todo: publish service TD
	ag.SetRequestHandler(agHandler.HandleRequest)
	return agHandler
}
