package authbinding

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthClientsBinding binds message api to the management service
// This unmarshal requests and marshals responses
type AuthClientsBinding struct {
	svc    auth.IAuthnManageClients
	mngSub hubclient.ISubscription
	hc     hubclient.IHubClient
}

// handle authn management requests published by a hub manager
func (binding *AuthClientsBinding) handleManageActions(action *hubclient.ActionMessage) error {
	slog.Info("handleManageActions",
		slog.String("actionID", action.ActionID))

	// TODO: doublecheck the caller is an admin or svc
	switch action.ActionID {
	case auth.AddDeviceAction:
		req := auth.AddDeviceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := binding.svc.AddDevice(
			req.DeviceID, req.DisplayName, req.PubKey, req.TokenValidity)
		if err == nil {
			resp := auth.AddDeviceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.AddServiceAction:
		req := auth.AddServiceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := binding.svc.AddService(
			req.ServiceID, req.DisplayName, req.PubKey, req.TokenValidity)
		if err == nil {
			resp := auth.AddServiceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.AddUserAction:
		req := auth.AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := binding.svc.AddUser(
			req.UserID, req.DisplayName, req.Password, req.PubKey, req.Role)
		if err == nil {
			resp := auth.AddUserResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.GetCountAction:
		n, err := binding.svc.GetCount()
		resp := auth.GetCountResp{N: n}
		reply, _ := ser.Marshal(&resp)
		action.SendReply(reply)
		return err
	case auth.GetProfileAction:
		req := auth.GetProfileReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		profile, err := binding.svc.GetProfile(req.ClientID)
		if err == nil {
			resp := auth.GetProfileResp{Profile: profile}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case auth.GetProfilesAction:
		clientList, err := binding.svc.GetProfiles()
		if err == nil {
			resp := auth.GetProfilesResp{Profiles: clientList}
			reply, _ := ser.Marshal(resp)
			action.SendReply(reply)
		}
		return err
	case auth.RemoveClientAction:
		req := &auth.RemoveClientReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.RemoveClient(req.ClientID)
		if err == nil {
			action.SendAck()
		}
		return err
	case auth.UpdateClientAction:
		req := &auth.UpdateClientReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.UpdateClient(req.ClientID, req.Profile)
		if err == nil {
			action.SendAck()
		}
		return err
	default:
		return fmt.Errorf("Unknown manage action '%s' for client '%s'", action.ActionID, action.ClientID)
	}
}

// Start subscribes to the actions for management and client capabilities
// Register the binding subscription using the given connection
func (binding *AuthClientsBinding) Start() (err error) {
	// if the first succeeds then 2nd will succeed as well
	binding.mngSub, err = binding.hc.SubServiceCapability(auth.AuthManageClientsCapability, binding.handleManageActions)
	return err
}

// Stop removes subscriptions
func (binding *AuthClientsBinding) Stop() {
	binding.mngSub.Unsubscribe()
}

// NewAuthnClientsBinding create a messaging binding to manage clients
//
//	svc is the manage authn svc to bind to.
//	hc is the hub client, connected using the svc credentials
func NewAuthnClientsBinding(svc auth.IAuthnManageClients, hc hubclient.IHubClient) *AuthClientsBinding {
	an := &AuthClientsBinding{
		svc: svc,
		hc:  hc,
	}
	return an
}