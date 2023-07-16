package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/ser"
	"github.com/nats-io/nkeys"
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

// AuthnNatsBinding is a NATS binding for handling Authn requests
// Subjects: things.svc.*.{action}
type AuthnNatsBinding struct {
	service *AuthnService
	hc      hub.IHubClient
	//signingKey nkeys.KeyPair
}

// handle action requests published by hub clients
func (natsrv *AuthnNatsBinding) handleClientActions(action *hub.ActionMessage) error {
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case authn.NewTokenAction:
		req := &authn.NewTokenReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		// extra check, the sender's clientID must match the requested token client
		if action.ClientID != req.ClientID {
			err = fmt.Errorf("Client '%s' cannot request token for user '%s'", action.ClientID, req.ClientID)
			return err
		}
		newToken, err := natsrv.service.NewToken(
			action.ClientID, req.Password, req.PubKey)
		if err == nil {
			resp := authn.NewTokenResp{JwtToken: newToken}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case authn.GetProfileAction:
		// use the current client
		profile, err := natsrv.service.GetClientProfile(action.ClientID)
		if err == nil {
			resp := authn.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.RefreshAction:
		req := &authn.RefreshReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := natsrv.service.Refresh(action.ClientID, req.OldToken)
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

// handle authn management requests published by a hub manager
func (natsrv *AuthnNatsBinding) handleManageActions(action *hub.ActionMessage) error {
	slog.Info("handleManageActions",
		slog.String("actionID", action.ActionID),
		"my addr", natsrv)

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
	case authn.GetClientProfileAction:
		req := authn.GetClientProfileReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		profile, err := natsrv.service.GetClientProfile(req.ClientID)
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

// Start subscribes to the actions for management and client capabilities
func (natsrv *AuthnNatsBinding) Start() {
	_ = natsrv.hc.SubActions(authn.ManageAuthnCapability, natsrv.handleManageActions)
	_ = natsrv.hc.SubActions(authn.ClientAuthnCapability, natsrv.handleClientActions)
	//_ = natsrv.hc.Subscribe(server.AuthCalloutSubject, )
}

// Stop removes subscriptions
func (natsrv *AuthnNatsBinding) Stop() {
	//natsrv.hc.UnSubscribeAll()
}

// NewAuthnNatsBinding create a nats binding for the svc service
//
//	svc is the svc service to bind to.
//	hc is the hub client, connected using the service credentials
func NewAuthnNatsBinding(signingKey nkeys.KeyPair, svc *AuthnService, hc hub.IHubClient) *AuthnNatsBinding {
	an := &AuthnNatsBinding{
		service: svc,
		hc:      hc,
		//signingKey: signingKey,
	}
	return an
}
