package service

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/transports"
	"github.com/hiveot/hub/services/launcher/launcherapi"
)

//
//// LauncherAgent agent for the authentication services:
//type LauncherAgent struct {
//	hc  hubclient.IHubClient
//	svc *service.LauncherService
//}
//type AgentMethods map[string]interface{}
//
//
//func HandleMessage(msg *things.ThingMessage) (stat api.DeliveryStatus) {
//	if msg.ThingID == launcherapi.LauncherThingID {
//		argsType, found := h[msg.Key]
//		if found {
//			if argsType == nil
//		}
//	}
//	stat.Failed(msg, fmt.Errorf(
//		"Agent for service '%s' does not have method '%s'", launcherapi.LauncherThingID, msg.Key))
//	return stat
//}
//
//
//func (agent *LauncherAgent) List(msg *things.ThingMessage) (stat api.DeliveryStatus) {
//	args := launcherapi.ListArgs{}
//	resp := launcherapi.ListResp{}
//	err := msg.Decode( &args)
//	if err == nil {
//		resp, err = agent.svc.List(args)
//	}
//	if err == nil {
//		stat.Reply, err = json.Marshal(resp)
//	}
//	stat.Completed(msg, err)
//	return stat
//}
//
//func (agent *LauncherAgent) StartPlugin(msg *things.ThingMessage) (stat api.DeliveryStatus) {
//	args := launcherapi.StartPluginArgs{}
//	resp := launcherapi.StartPluginResp{}
//	err := msg.Decode( &args)
//	if err == nil {
//		resp, err = agent.svc.StartPlugin(args)
//	}
//	if err == nil {
//		stat.Reply, err = json.Marshal(resp)
//	}
//	stat.Completed(msg, err)
//	return stat
//}
//
//func (agent *LauncherAgent) StopPlugin(msg *things.ThingMessage) (stat api.DeliveryStatus) {
//	args := launcherapi.StopPluginArgs{}
//	resp := launcherapi.StopPluginResp{}
//	err := msg.Decode( &args)
//	if err == nil {
//		resp, err = agent.svc.StopPlugin(args)
//	}
//	if err == nil {
//		stat.Reply, err = json.Marshal(resp)
//	}
//	stat.Completed(msg, err)
//	return stat
//}

// StartLauncherAgent returns a new instance of the agent for the launcher service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the service whose capabilities to expose
//	hc is the optional message client connected to the server protocol
func StartLauncherAgent(svc *LauncherService, hc hubclient.IHubClient) *transports.AgentHandler {

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
