package service

import (
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

//
//const (
//	ActionAddUser        = "addUser"
//	ActionGetProfile     = "getProfile"
//	ActionListClients    = "listClients"
//	ActionLogin          = "login"
//	ActionLogout         = "logout"
//	ActionRefresh        = "refresh"
//	ActionRemoveClient   = "removeClient"
//	ActionResetPassword  = "resetPassword"
//	ActionUpdateName     = "updateName"
//	ActionUpdatePassword = "updatePassword"
//)

// AuthnNatsServer is a NATS binding for authn service
// Subjects: things.authn.*.{action}
type AuthnNatsServer struct {
	service *AuthnService
	hc      hub.IHubClient
}

func (natsrv *AuthnNatsServer) handleClientActions(action *hub.ActionMessage) error {
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case authn.NewTokenAction:
		req := &authn.NewTokenReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := natsrv.service.NewToken(
			action.BindingID, req.Password, req.PubKey)
		if err == nil {
			resp := authn.NewTokenResp{JwtToken: newToken}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err

	case authn.RefreshAction:
		req := &authn.RefreshReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := natsrv.service.Refresh(action.BindingID, req.OldToken)
		if err == nil {
			resp := authn.RefreshResp{JwtToken: newToken}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case authn.UpdateNameAction:
		req := &authn.UpdateNameReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.UpdateName(req.ClientID, req.NewName)
		if err == nil {
			action.SendAck()
		}
		return err
	case authn.UpdatePasswordAction:
		req := &authn.UpdatePasswordReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.ResetPassword(req.ClientID, req.NewPassword)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		return nil
	}
}

func (natsrv *AuthnNatsServer) handleManageActions(action *hub.ActionMessage) error {
	slog.Info("handleManageActions", slog.String("actionID", action.ActionID))
	// TODO: doublecheck the caller is an admin or service
	switch action.ActionID {
	case authn.AddUserAction:
		req := authn.AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.AddUser(req.UserID, req.Name, req.Password)
		if err == nil {
			action.SendAck()
		}
		return err
	case authn.GetProfileAction:
		req := authn.GetProfileReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		profile, err := natsrv.service.GetProfile(req.ClientID)
		if err == nil {
			resp := authn.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.ListClientsAction:
		clientList, err := natsrv.service.ListClients()
		if err == nil {
			resp := authn.ListClientsResp{Profiles: clientList}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case authn.RemoveClientAction:
		req := &authn.RemoveClientReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = natsrv.service.RemoveClient(req.ClientID)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		//err := errors.New("invalid action '" + action.ActionID + "'")
		return nil
	}
}

// Start subscribes to the actions
func (natsrv *AuthnNatsServer) Start() {
	_ = natsrv.hc.SubActions("", natsrv.handleManageActions)
	_ = natsrv.hc.SubActions("", natsrv.handleClientActions)
}

// Stop removes subscriptions
func (natsrv *AuthnNatsServer) Stop() {

}

// NewAuthnNatsServer create a nats binding for the authn service
//
//	svc is the authn service to bind to.
//	hc is the hub client, connected using the service credentials
func NewAuthnNatsServer(svc *AuthnService, hc hub.IHubClient) *AuthnNatsServer {
	an := &AuthnNatsServer{
		service: svc,
		hc:      hc,
	}
	return an
}
