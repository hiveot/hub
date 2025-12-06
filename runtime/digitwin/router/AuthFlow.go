// Package service with digital twin event handling functions
package router

import (
	"github.com/hiveot/hivekit/go/lib/messaging"
	authn "github.com/hiveot/hub/runtime/authn/api"
)

// HandleLogin passes the request as an action to the authn service
func (r *DigitwinRouter) HandleLogin(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	// Data {login,password}
	req.ThingID = authn.UserDThingID
	req.Name = authn.UserLoginMethod
	resp := r.authnAction(req, c)
	return resp
}

// HandleLoginRefresh passes the request as an action to the authn service
func (r *DigitwinRouter) HandleLoginRefresh(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	// Data {oldToken}
	req.ThingID = authn.UserDThingID
	req.Name = authn.UserRefreshTokenMethod

	resp := r.authnAction(req, c)
	return resp
}

// HandleLogout converts the logout operation into an authn service action
// and closes all connections from the sender.
func (r *DigitwinRouter) HandleLogout(
	req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {

	req.ThingID = authn.UserDThingID
	req.Name = authn.UserLogoutMethod
	resp := r.authnAction(req, c)
	r.transportServer.CloseAllClientConnections(req.SenderID)
	return resp
}
