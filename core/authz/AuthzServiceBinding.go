package authz

import (
	"errors"
	"github.com/hiveot/hub/core/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"golang.org/x/exp/slog"
)

// AuthzServiceBinding is a messaging binding for marshalling Authz service messages.
type AuthzServiceBinding struct {
	svc    IAuthz
	hc     hubclient.IHubClient
	mngSub hubclient.ISubscription
}

// handle authz management requests published by a hub manager
func (binding *AuthzServiceBinding) handleManageActions(action *hubclient.ActionMessage) error {
	slog.Info("handleManageActions",
		slog.String("actionID", action.ActionID),
		"my addr", binding)

	// TODO: doublecheck the caller is an admin or svc
	switch action.ActionID {
	case AddGroupAction:
		req := AddGroupReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.AddGroup(req.GroupName, req.Retention)
		if err == nil {
			action.SendAck()
		}
		return err
	case AddServiceAction:
		req := AddServiceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.AddService(req.ServiceID, req.GroupName)
		if err == nil {
			action.SendAck()
		}
		return err
	case AddThingAction:
		req := AddThingReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.AddThing(req.ThingID, req.GroupName)
		if err == nil {
			action.SendAck()
		}
		return err
	case AddUserAction:
		req := AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.AddThing(req.UserID, req.GroupName)
		if err == nil {
			action.SendAck()
		}
		return err
	case DeleteGroupAction:
		req := DeleteGroupReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.DeleteGroup(req.GroupName)
		if err == nil {
			action.SendAck()
		}
		return err
	case GetClientRolesAction:
		req := GetClientRolesReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		roles, err := binding.svc.GetClientRoles(req.ClientID)
		if err == nil {
			resp := GetClientRolesResp{Roles: roles}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case GetGroupAction:
		req := GetGroupReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		group, err := binding.svc.GetGroup(req.GroupName)
		if err == nil {
			resp := GetGroupResp{Group: group}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case GetPermissionsAction:
		req := GetPermissionsReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		perms, err := binding.svc.GetPermissions(req.ClientID, req.ThingIDs)
		if err == nil {
			resp := GetPermissionsResp{Permissions: perms}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case ListGroupsAction:
		req := ListGroupsReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		groups, err := binding.svc.ListGroups(req.ClientID)
		if err == nil {
			resp := ListGroupsResp{Groups: groups}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err
	case RemoveClientAction:
		req := RemoveClientReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.RemoveClient(req.ClientID, req.GroupName)
		if err == nil {
			action.SendAck()
		}
		return err
	case RemoveClientAllAction:
		req := RemoveClientAllReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.RemoveClientAll(req.ClientID)
		if err == nil {
			action.SendAck()
		}
		return err
	case SetUserRoleAction:
		req := SetUserRoleReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.SetUserRole(req.UserID, req.UserRole, req.GroupName)
		if err == nil {
			action.SendAck()
		}
		return err
	}
	return errors.New("unknown action: " + action.ActionID)
}

// Start subscribes to authz message requests
func (binding *AuthzServiceBinding) Start() error {
	sub, err := binding.hc.SubActions(ManageAuthzCapability, binding.handleManageActions)
	binding.mngSub = sub
	return err
}

// Stop unsubscribes from authz message requests
func (binding *AuthzServiceBinding) Stop() {
	if binding.mngSub != nil {
		binding.mngSub.Unsubscribe()
	}
}

// NewAuthzMsgBinding creates a new instance of the messaging binding
// This uses an existing client connection to the server to subscribe and unsubscribe.
// opening and closing this connection is the responsibility of the caller.
//
//	svc is the authz service that handles the requests
//	hc is an existing client connection to the messaging server used to subscribe to actions
func NewAuthzMsgBinding(svc IAuthz, hc hubclient.IHubClient) *AuthzServiceBinding {
	binding := &AuthzServiceBinding{
		svc:    svc,
		hc:     hc,
		mngSub: nil,
	}
	return binding
}
