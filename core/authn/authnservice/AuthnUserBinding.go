package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthnUserBinding binds the service to a message based API
// This unmarshal requests and marshals responses
type AuthnUserBinding struct {
	svc   authn.IAuthnUser
	clSub hubclient.ISubscription
	hc    hubclient.IHubClient
}

// handle action requests published by hub clients
func (binding *AuthnUserBinding) handleClientActions(action *hubclient.ActionMessage) error {
	slog.Info("handleClientActions", slog.String("actionID", action.ActionID))
	switch action.ActionID {
	case authn.GetProfileAction:
		// use the current client
		profile, err := binding.svc.GetProfile(action.ClientID)
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
		newToken, err := binding.svc.NewToken(action.ClientID, req.Password)
		if err == nil {
			resp := authn.NewTokenResp{Token: newToken}
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
		err = binding.svc.UpdatePassword(req.ClientID, req.NewPassword)
		if err == nil {
			action.SendAck()
		}
		return err
	case authn.UpdatePubKeyAction:
		req := &authn.UpdatePubKeyReq{}
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
func (binding *AuthnUserBinding) Start() (err error) {
	// if the first succeeds then 2nd will succeed as well
	binding.clSub, _ = binding.hc.SubActions(authn.ClientAuthnCapability, binding.handleClientActions)
	return err
}

// Stop removes subscriptions
func (binding *AuthnUserBinding) Stop() {
	binding.clSub.Unsubscribe()
}

// NewAuthnMsgBinding create a messaging binding for the authn service
//
//	svc is the authn svc to bind to.
//	hc is the hub client, connected using the svc credentials
func NewAuthnUserBinding(svc authn.IAuthnUser, hc hubclient.IHubClient) *AuthnUserBinding {
	an := &AuthnUserBinding{
		svc: svc,
		hc:  hc,
	}
	return an
}
