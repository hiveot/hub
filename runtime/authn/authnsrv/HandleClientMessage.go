// Package authnmsg with server side messaging structs for use by clients
package authnsrv

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/authn/service"
)

// AuthnClientSrv serves the message based interface to the authn client service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthnClientSrv struct {
	svc *service.AuthnClientService
}

// HandleMessage an event or action message for the authn admin service
// This handle action messages with the AuthnAdminServiceID ThingID.
func (rpc *AuthnClientSrv) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	if msg.MessageType == vocab.MessageTypeEvent {
		// this service does not use events
	} else if msg.MessageType == vocab.MessageTypeAction {
		// handle authn client actions
	}
	return nil, fmt.Errorf("Unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
}

// NewAuthnClientSrv creates a new instance of the messaging api server for the
// authentication admin service.
func NewAuthnClientSrv(svc *service.AuthnClientService) *AuthnClientSrv {
	rpc := AuthnClientSrv{svc: svc}
	return &rpc
}
