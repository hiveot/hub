// Package service with digital twin event handling functions
package hubrouter

import (
	"github.com/hiveot/hub/api/go/authn"
	"log/slog"
)

func (svc *HubRouter) HandleLogin(data any) (reply any, err error) {
	slog.Info("HandleLogin")

	//token, sid, err := svc.authenticator.Login(args.ClientID, args.Password)

	// first, verify the
	_, reply, err = svc.authnAgent.HandleAction(
		"", authn.UserDThingID, authn.UserLoginMethod, data, "")
	return reply, err
}

func (svc *HubRouter) HandleLoginRefresh(senderID string, data any) (reply any, err error) {
	slog.Info("HandleLoginRefresh", slog.String("clientID", senderID))
	_, reply, err = svc.authnAgent.HandleAction(
		senderID, authn.UserDThingID, authn.UserRefreshTokenMethod, data, "")
	return reply, err
}

func (svc *HubRouter) HandleLogout(senderID string) {
	slog.Info("HandleLogout", slog.String("senderID", senderID))
	// authn will invalidate the client session
	_, _, err := svc.authnAgent.HandleAction(
		senderID, authn.UserDThingID, authn.UserLogoutMethod, nil, "")

	if err != nil {
		// close all active client connections
		svc.cm.CloseAllClientConnections(senderID)
	}
}