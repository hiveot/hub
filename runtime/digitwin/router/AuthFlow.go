// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/transports"
)

// HandleLogin converts the login operation into an authn service action
func (svc *DigitwinRouter) HandleLogin(
	msg *transports.ThingMessage) (completed bool, output any, err error) {

	authnMsg := *msg
	authnMsg.ThingID = authn.UserDThingID
	authnMsg.Name = authn.UserLoginMethod

	output, err = svc.authnAction(&authnMsg)
	return true, output, err
}

// HandleLoginRefresh converts the token refresh operation into an authn service action
func (svc *DigitwinRouter) HandleLoginRefresh(
	msg *transports.ThingMessage) (completed bool, output any, err error) {

	authnMsg := *msg
	authnMsg.ThingID = authn.UserDThingID
	authnMsg.Name = authn.UserRefreshTokenMethod

	output, err = svc.authnAction(&authnMsg)
	return true, output, err
}

// HandleLogout converts the logout operation into an authn service action
// and closes all connections from the sender.
func (svc *DigitwinRouter) HandleLogout(
	msg *transports.ThingMessage) (completed bool, output any, err error) {

	authnMsg := *msg
	authnMsg.ThingID = authn.UserDThingID
	authnMsg.Name = authn.UserLogoutMethod

	_, err = svc.authnAction(&authnMsg)
	svc.cm.CloseAllClientConnections(msg.SenderID)
	return true, nil, err
}
