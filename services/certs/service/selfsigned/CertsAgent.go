package selfsigned

import (
	"github.com/hiveot/hivehub/lib/hubagent"
	"github.com/hiveot/hivehub/services/certs/certsapi"
	"github.com/hiveot/hivekitgo/messaging"
)

// StartCertsAgent returns a new instance of the agent for the certificate management service.
// This uses the given connected hubclient for publishing events and subscribing to actions.
// The client must be closed by the caller after use.
//
//	svc is the service whose capabilities to expose
//	hc is the hub client connected to the server protocol
func StartCertsAgent(svc *SelfSignedCertsService, ag *messaging.Agent) *hubagent.AgentHandler {

	methods := map[string]interface{}{
		certsapi.CreateDeviceCertMethod:  svc.CreateDeviceCert,
		certsapi.CreateServiceCertMethod: svc.CreateServiceCert,
		certsapi.CreateUserCertMethod:    svc.CreateUserCert,
		certsapi.VerifyCertMethod:        svc.VerifyCert,
	}

	ah := hubagent.NewAgentHandler(certsapi.CertsAdminThingID, methods)
	ag.SetRequestHandler(ah.HandleRequest)
	return ah
}
