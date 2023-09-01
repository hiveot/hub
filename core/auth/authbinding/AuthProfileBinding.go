package authbinding

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthProfileBinding binds the service to a message based API
// This unmarshal requests and marshals responses
type AuthProfileBinding struct {
	svc   auth.IAuthManageProfile
	clSub hubclient.ISubscription
	hc    hubclient.IHubClient
}

// handle action requests published by hub clients
func (binding *AuthProfileBinding) handleClientActions(action *hubclient.ActionMessage) error {
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case auth.GetProfileAction:
		// use the current client
		profile, err := binding.svc.GetProfile(action.ClientID)
		if err == nil {
			resp := auth.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.NewTokenAction:
		req := &auth.NewTokenReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		// extra check, the sender's clientID must match the requested token client
		if action.ClientID != req.ClientID {
			err = fmt.Errorf("Client '%s' cannot request token for user '%s'", action.ClientID, req.ClientID)
			return err
		}
		newToken, err := binding.svc.NewToken(action.ClientID, req.Password)
		if err == nil {
			resp := auth.NewTokenResp{Token: newToken}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case auth.RefreshAction:
		req := &auth.RefreshReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		newToken, err := binding.svc.Refresh(action.ClientID, req.OldToken)
		if err == nil {
			resp := auth.RefreshResp{NewToken: newToken}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case auth.UpdateNameAction:
		req := &auth.UpdateNameReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.UpdateName(req.ClientID, req.NewName)
		if err == nil {
			action.SendAck()
		}
		return err
	case auth.UpdatePasswordAction:
		req := &auth.UpdatePasswordReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.UpdatePassword(req.ClientID, req.NewPassword)
		if err == nil {
			action.SendAck()
		}
		return err
	case auth.UpdatePubKeyAction:
		req := &auth.UpdatePubKeyReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.UpdatePubKey(req.ClientID, req.NewPubKey)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		return fmt.Errorf("Unknown user action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// Start subscribes to the actions for management and client capabilities
// Register the binding subscription using the given connection
func (binding *AuthProfileBinding) Start() (err error) {
	// if the first succeeds then 2nd will succeed as well
	binding.clSub, _ = binding.hc.SubServiceCapability(auth.AuthProfileCapability, binding.handleClientActions)
	return err
}

// Stop removes subscriptions
func (binding *AuthProfileBinding) Stop() {
	binding.clSub.Unsubscribe()
}

// NewAuthProfileBinding create a messaging binding for the auth profile management
//
//	svc is the client service to bind to.
//	hc is the hub client, connected using the svc credentials
func NewAuthProfileBinding(svc auth.IAuthManageProfile, hc hubclient.IHubClient) *AuthProfileBinding {
	an := &AuthProfileBinding{
		svc: svc,
		hc:  hc,
	}
	return an
}
