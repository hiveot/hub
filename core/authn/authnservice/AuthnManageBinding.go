package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthnManageBinding binds message api to the management service
// This unmarshal requests and marshals responses
type AuthnManageBinding struct {
	svc    authn.IAuthnManage
	mngSub hubclient.ISubscription
	hc     hubclient.IHubClient
}

// handle authn management requests published by a hub manager
func (binding *AuthnManageBinding) handleManageActions(action *hubclient.ActionMessage) error {
	slog.Info("handleManageActions",
		slog.String("actionID", action.ActionID))

	// TODO: doublecheck the caller is an admin or svc
	switch action.ActionID {
	case authn.AddDeviceAction:
		req := authn.AddDeviceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := binding.svc.AddDevice(req.DeviceID, req.DisplayName, req.PubKey, req.ValiditySec)
		if err == nil {
			resp := authn.AddDeviceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.AddServiceAction:
		req := authn.AddServiceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := binding.svc.AddService(req.ServiceID, req.DisplayName, req.PubKey, req.ValiditySec)
		if err == nil {
			resp := authn.AddServiceResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.AddUserAction:
		req := authn.AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		token, err := binding.svc.AddUser(req.UserID, req.DisplayName, req.Password, req.PubKey)
		if err == nil {
			resp := authn.AddUserResp{Token: token}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case authn.GetCountAction:
		n, err := binding.svc.GetCount()
		resp := authn.GetCountResp{N: n}
		reply, _ := ser.Marshal(&resp)
		action.SendReply(reply)
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
	case authn.UpdateClientAction:
		req := &authn.UpdateClientReq{}
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
func (binding *AuthnManageBinding) Start() (err error) {
	// if the first succeeds then 2nd will succeed as well
	binding.mngSub, err = binding.hc.SubActions(authn.ManageAuthnCapability, binding.handleManageActions)
	return err
}

// Stop removes subscriptions
func (binding *AuthnManageBinding) Stop() {
	binding.mngSub.Unsubscribe()
}

// NewAuthnManageBinding create a messaging binding for the authn management service
//
//	svc is the manage authn svc to bind to.
//	hc is the hub client, connected using the svc credentials
func NewAuthnManageBinding(svc authn.IAuthnManage, hc hubclient.IHubClient) *AuthnManageBinding {
	an := &AuthnManageBinding{
		svc: svc,
		hc:  hc,
	}
	return an
}
