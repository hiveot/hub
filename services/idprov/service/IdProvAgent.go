package service

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/services/idprov/idprovapi"
)

// StartIdProvAgent registers the idprov messaging agent for the idprov service.
// This uses the given hub client for subscribing to actions.
//
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the service whose capabilities to expose
//	hc is the optional message client connected to the server protocol
func StartIdProvAgent(svc *ManageIdProvService, hc hubclient.IAgentClient) *hubclient.AgentHandler {

	methods := map[string]interface{}{
		idprovapi.ApproveRequestMethod:    svc.ApproveRequest,
		idprovapi.GetRequestsMethod:       svc.GetRequests,
		idprovapi.PreApproveClientsMethod: svc.PreApproveClients,
		idprovapi.RejectRequestMethod:     svc.RejectRequest,
		idprovapi.SubmitRequestMethod:     svc.SubmitRequest,
	}
	ah := hubclient.NewAgentHandler(idprovapi.ManageServiceID, methods)
	hc.SetRequestHandler(ah.HandleRequest)

	return ah
}
