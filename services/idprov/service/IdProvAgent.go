package service

import (
	"github.com/hiveot/hivehub/lib/hubagent"
	"github.com/hiveot/hivehub/services/idprov/idprovapi"
	"github.com/hiveot/hivekitgo/messaging"
)

// StartIdProvAgent registers the idprov messaging agent for the idprov service.
// This uses the given hub client for subscribing to actions.
//
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the service whose capabilities to expose
//	ag is the optional message client connected to the server protocol
func StartIdProvAgent(svc *ManageIdProvService, ag *messaging.Agent) *hubagent.AgentHandler {

	methods := map[string]interface{}{
		idprovapi.ApproveRequestMethod:    svc.ApproveRequest,
		idprovapi.GetRequestsMethod:       svc.GetRequests,
		idprovapi.PreApproveClientsMethod: svc.PreApproveClients,
		idprovapi.RejectRequestMethod:     svc.RejectRequest,
		idprovapi.SubmitRequestMethod:     svc.SubmitRequest,
	}
	agentHandler := hubagent.NewAgentHandler(idprovapi.ManageServiceID, methods)
	ag.SetRequestHandler(agentHandler.HandleRequest)
	// TODO: publish service TD
	return agentHandler
}
