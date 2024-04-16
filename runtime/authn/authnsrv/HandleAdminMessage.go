// Package authnsrv serves authn messages to the service API
package authnsrv

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/authn/service"
)

// AuthnAdminSrv serves the message based interface to the authn service API.
// This converts the request messages into API calls and converts the result
// back to a reply message, if any.
// The main entry point is the HandleMessage function.
type AuthnAdminSrv struct {
	svc *service.AuthnAdminService
}

// HandleMessage an event or action message for the authn admin service
// This handle action messages with the AuthnAdminServiceID ThingID.
func (rpc *AuthnAdminSrv) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	if msg.MessageType == vocab.MessageTypeEvent {
		// this service does not use events
	} else if msg.MessageType == vocab.MessageTypeAction {
		// handle authn admin actions
	}
	return nil, fmt.Errorf("Unknown action '%s' for service '%s'", msg.Key, msg.ThingID)
}

// NewAuthnAdminSrv creates a new instance of the messaging api server for the
// authentication admin service.
func NewAuthnAdminSrv(svc *service.AuthnAdminService) *AuthnAdminSrv {
	rpc := AuthnAdminSrv{svc: svc}
	return &rpc
}
