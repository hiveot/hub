// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hub/messaging"
	authn "github.com/hiveot/hub/runtime/authn/api"
)

// HandleLogin passes the request as an action to the authn service
func (svc *DigitwinRouter) HandleLogin(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	// Data {login,password}
	req.ThingID = authn.UserDThingID
	req.Name = authn.UserLoginMethod
	resp := svc.authnAction(req, c)
	return resp
}

// HandleLoginRefresh passes the request as an action to the authn service
func (svc *DigitwinRouter) HandleLoginRefresh(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	// Data {oldToken}
	req.ThingID = authn.UserDThingID
	req.Name = authn.UserRefreshTokenMethod

	resp := svc.authnAction(req, c)
	return resp
}

// HandleLogout converts the logout operation into an authn service action
// and closes all connections from the sender.
func (svc *DigitwinRouter) HandleLogout(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	req.ThingID = authn.UserDThingID
	req.Name = authn.UserLogoutMethod
	resp := svc.authnAction(req, c)
	svc.transportServer.CloseAllClientConnections(req.SenderID)
	return resp
}
