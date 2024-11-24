// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/wot/transports"
)

// HandleLogin converts the login operation into an authn service action
func (svc *DigitwinRouter) HandleLogin(msg *transports.ThingMessage) transports.RequestStatus {
	authnMsg := *msg
	authnMsg.ThingID = authn.UserDThingID
	authnMsg.Name = authn.UserLoginMethod

	stat := svc.authnAction(&authnMsg)
	return stat
}

// HandleLoginRefresh converts the token refresh operation into an authn service action
func (svc *DigitwinRouter) HandleLoginRefresh(msg *transports.ThingMessage) transports.RequestStatus {
	authnMsg := *msg
	authnMsg.ThingID = authn.UserDThingID
	authnMsg.Name = authn.UserRefreshTokenMethod

	stat := svc.authnAction(&authnMsg)
	return stat
}

// HandleLogout converts the logout operation into an authn service action
func (svc *DigitwinRouter) HandleLogout(msg *transports.ThingMessage) transports.RequestStatus {
	authnMsg := *msg
	authnMsg.ThingID = authn.UserDThingID
	authnMsg.Name = authn.UserLogoutMethod

	stat := svc.authnAction(&authnMsg)
	// logout and close all existing connections
	svc.cm.CloseAllClientConnections(msg.SenderID)
	return stat
}
