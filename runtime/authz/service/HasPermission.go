package service

import (
	"github.com/hiveot/hub/lib/things"
	"golang.org/x/exp/slices"
)

// HasRolePermission returns whether the client has pub/sub permission based on their role.
//
// This returns true if the client has permission, false if the client does not have the permission
// See also HasServicePermissions to check if clients can invoke specific services
func (svc *AuthzService) HasRolePermission(msg *things.ThingMessage, isPub bool) bool {
	role, err := svc.GetClientRole(msg.SenderID)
	if err != nil || role == "" {
		// unknown client or missing role
		return false
	}

	// configured role permissions
	rolePerms, found := svc.cfg.RolePermissions[role]
	if !found {
		return false
	}
	// pick the first match. This doesn't check for agent, thing/interface, or key/method
	for _, perm := range rolePerms {
		if isPub && perm.AllowPub &&
			msg.MessageType == perm.MsgType {
			return true
		} else if !isPub && perm.AllowSub &&
			msg.MessageType == perm.MsgType {
			return true
		}
	}
	return false
}

// HasThingPermission returns whether the client has permission to use a requested thing.
// This only applies if permissions for the Thing (service or device) is set.
//
//	 msg contains the sender, thingID and key to validate
//		isPub true to check for publish permissions, false for subscribe permission
//
// This returns true if the client has permission, false if the client does not have the permission
func (svc *AuthzService) HasThingPermission(msg *things.ThingMessage, isPub bool) bool {
	sp, found := svc.cfg.GetPermissions(msg.ThingID)
	if !found {
		return false
	}
	clientRole, err := svc.GetClientRole(msg.SenderID)
	if err != nil {
		return false
	}
	// if allow is set then the default is denied
	if sp.Allow != nil && len(sp.Allow) > 0 {
		if slices.Contains(sp.Allow, clientRole) {
			return true
		}
		return false
	}
	// if deny list is set then the default is allowed
	if sp.Deny != nil && len(sp.Deny) > 0 {
		if slices.Contains(sp.Deny, clientRole) {
			return false
		}
		return true
	}
	return false
}

// HasPermission returns whether the client has permission to pub or sub a message type
// This returns true if the client has permission, false if the client does not have the permission
func (svc *AuthzService) HasPermission(msg *things.ThingMessage, isPub bool) (hasPerm bool) {
	//If a thing permission exists then it has priority
	_, found := svc.cfg.GetPermissions(msg.ThingID)
	if found {
		hasPerm = svc.HasThingPermission(msg, isPub)
	} else {
		hasPerm = svc.HasRolePermission(msg, isPub)
	}
	return hasPerm
}