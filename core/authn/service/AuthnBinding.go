package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthnMsgBinding is a binding for handling Authn messaging requests
// The purpose of the binding is to unmarshal request and marshal responses.
type AuthnMsgBinding struct {
	svc    *AuthnService
	mngSub hub.ISubscription
	clSub  hub.ISubscription
}

// handle action requests published by hub clients
func (binding *AuthnMsgBinding) handleClientActions(action *hub.ActionMessage) error {
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case authn.GetProfileAction:
		// use the current client
		profile, err := binding.svc.GetClientProfile(action.ClientID)
		if err == nil {
			resp := authn.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
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
		newToken, err := binding.svc.NewToken(
			action.ClientID, req.Password, req.PubKey)
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
		newToken, err := binding.svc.Refresh(action.ClientID, req.OldToken)
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
		err = binding.svc.UpdateName(req.ClientID, req.NewName)
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
		err = binding.svc.ResetPassword(req.ClientID, req.NewPassword)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		return nil
	}
}

// handle authn management requests published by a hub manager
func (binding *AuthnMsgBinding) handleManageActions(action *hub.ActionMessage) error {
	slog.Info("handleManageActions",
		slog.String("actionID", action.ActionID),
		"my addr", binding)

	// TODO: doublecheck the caller is an admin or svc
	switch action.ActionID {
	case authn.AddUserAction:
		req := authn.AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.AddUser(req.UserID, req.Name, req.Password)
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
		profile, err := binding.svc.GetClientProfile(req.ClientID)
		if err == nil {
			resp := authn.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.ListClientsAction:
		clientList, err := binding.svc.ListClients()
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
		err = binding.svc.RemoveClient(req.ClientID)
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
// Register the binding subscription using the given connection
func (binding *AuthnMsgBinding) Start(hc hub.IHubClient) (err error) {
	// if the first succeeds then 2nd will succeed as well
	binding.mngSub, err = hc.SubActions(authn.ManageAuthnCapability, binding.handleManageActions)
	binding.clSub, _ = hc.SubActions(authn.ClientAuthnCapability, binding.handleClientActions)
	return err
}

// Stop removes subscriptions
func (binding *AuthnMsgBinding) Stop() {
	binding.clSub.Unsubscribe()
	binding.mngSub.Unsubscribe()
}

// NewAuthnMsgBinding create a messaging binding for the authn service
//
//	svc is the authn svc to bind to.
//	hc is the hub client, connected using the svc credentials
func NewAuthnMsgBinding(svc *AuthnService) *AuthnMsgBinding {
	an := &AuthnMsgBinding{
		svc: svc,
	}
	return an
}
