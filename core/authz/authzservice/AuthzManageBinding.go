package authzservice

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"time"
)

// AuthzBinding is a messaging binding for marshalling Authz service messages.
type AuthzBinding struct {
	svc    auth.IAuthz
	hc     hubclient.IHubClient
	mngSub hubclient.ISubscription
}

// handle authz requests published by a hub manager
func (binding *AuthzBinding) handleManageActions(action *hubclient.ActionMessage) error {
	//slog.Info("handleManageActions",
	//	slog.String("actionID", action.ActionID))

	// TODO: doublecheck the caller is an admin or svc
	switch action.ActionID {
	case authz.AddSourceAction:
		req := authz.AddSourceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.AddSource(req.PublisherID, req.ThingID, req.GroupID)
		if err == nil {
			action.SendAck()
		}
		return err

	case authz.AddUserAction:
		req := authz.AddUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.AddUser(req.UserID, req.Role, req.GroupID)
		if err == nil {
			action.SendAck()
		}
		return err

	case authz.CreateGroupAction:
		req := authz.CreateGroupReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.CreateGroup(req.GroupID, req.DisplayName, time.Second*time.Duration(req.Retention))
		if err == nil {
			action.SendAck()
		}
		return err

	case authz.DeleteGroupAction:
		req := authz.DeleteGroupReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.DeleteGroup(req.GroupID)
		if err == nil {
			action.SendAck()
		}
		return err

	//case authz.GetClientRolesAction:
	//	req := authz.GetClientRolesReq{}
	//	err := ser.Unmarshal(action.Payload, &req)
	//	if err != nil {
	//		return err
	//	}
	//	roles, err := binding.svc.GetClientRoles(req.ClientID)
	//	if err == nil {
	//		resp := authz.GetClientRolesResp{Roles: roles}
	//		reply, _ := ser.Marshal(&resp)
	//		action.SendReply(reply)
	//	}
	//	return err

	case authz.GetGroupAction:
		req := authz.GetGroupReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		group, err := binding.svc.GetGroup(req.GroupID)
		if err == nil {
			resp := authz.GetGroupResp{Group: group}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err

	//case authz.GetPermissionsAction:
	//	req := authz.GetPermissionsReq{}
	//	err := ser.Unmarshal(action.Payload, &req)
	//	if err != nil {
	//		return err
	//	}
	//	perms, err := binding.svc.GetPermissions(req.ClientID, req.ThingIDs)
	//	if err == nil {
	//		resp := authz.GetPermissionsResp{Permissions: perms}
	//		reply, _ := ser.Marshal(&resp)
	//		action.SendReply(reply)
	//	}
	//	return err

	case authz.GetUserGroupsAction:
		req := authz.GetUserGroupsReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		groups, err := binding.svc.GetUserGroups(req.UserID)
		if err == nil {
			resp := authz.GetUserGroupsResp{Groups: groups}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err

	case authz.GetUserRolesAction:
		req := authz.GetUserRolesReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		roles, err := binding.svc.GetUserRoles(req.UserID)
		if err == nil {
			resp := authz.GetUserRolesResp{Roles: roles}
			reply, _ := ser.Marshal(&resp)
			action.SendReply(reply)
		}
		return err

	case authz.RemoveSourceAction:
		req := authz.RemoveSourceReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.RemoveSource(req.PublisherID, req.ThingID, req.GroupID)
		if err == nil {
			action.SendAck()
		}
		return err

	case authz.RemoveUserAction:
		req := authz.RemoveUserReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.RemoveUser(req.UserID, req.GroupID)
		if err == nil {
			action.SendAck()
		}
		return err

	case authz.RemoveUserAllAction:
		req := authz.RemoveUserAllReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.RemoveUserAll(req.UserID)
		if err == nil {
			action.SendAck()
		}
		return err

	case authz.SetUserRoleAction:
		req := authz.SetUserRoleReq{}
		err := ser.Unmarshal(action.Payload, &req)
		if err != nil {
			return err
		}
		err = binding.svc.SetUserRole(req.UserID, req.UserRole, req.GroupID)
		if err == nil {
			action.SendAck()
		}
		return err
	}
	return errors.New("unknown action: " + action.ActionID)
}

// Start subscribes to authz message requests
func (binding *AuthzBinding) Start() error {
	if binding.hc == nil {
		return fmt.Errorf("HubClient is nil")
	} else if binding.svc == nil {
		return fmt.Errorf("authz service not provided to binding")
	}

	sub, err := binding.hc.SubActions(auth.ManageAuthzCapability, binding.handleManageActions)
	binding.mngSub = sub
	return err
}

// Stop unsubscribes from authz message requests
func (binding *AuthzBinding) Stop() {
	if binding.mngSub != nil {
		binding.mngSub.Unsubscribe()
	}
}

// NewAuthzBinding creates a new instance of the authz messaging binding
// This uses an existing client connection to the server to subscribe and unsubscribe.
// opening and closing this connection is the responsibility of the caller.
//
//	svc is the authz service that handles the requests
//	hc is an existing client connection to the messaging server used to subscribe to actions
func NewAuthzBinding(svc auth.IAuthz, hc hubclient.IHubClient) AuthzBinding {
	binding := AuthzBinding{
		svc:    svc,
		hc:     hc,
		mngSub: nil,
	}
	return binding
}
