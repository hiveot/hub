package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/hubclient"
	"golang.org/x/exp/slices"
	"log/slog"
)

// HasRolePermission returns whether the client has pub/sub permission based on their role.
//
// This returns true if the client has permission, false if the client does not have the permission
// See also HasServicePermissions to check if clients can invoke specific services
func (svc *AuthzService) HasRolePermission(senderID, messageType string, isPub bool) bool {
	role, err := svc.GetClientRole(senderID, senderID)
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
			messageType == perm.MsgType {
			return true
		} else if !isPub && perm.AllowSub &&
			messageType == perm.MsgType {
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
func (svc *AuthzService) HasThingPermission(senderID string, thingID string, isPub bool) bool {
	sp, found := svc.cfg.GetPermissions(thingID)
	if !found {
		return false
	}
	clientRole, err := svc.GetClientRole(senderID, senderID)
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

// HasPermission returns whether the sender has permission to pub or sub a
// message type for a IoT or a Thing.
//
// This returns true if the client has permission, false if the client does not
// have the permission.
//
// If the Thing is a service and has its permission set then use the service
// set permissions rather than the sender's role permissions.
//
//	senderID the login ID of the sender
//	messageType MessageTypeAction/Event/Property,
func (svc *AuthzService) HasPermission(senderID, messageType, dThingID string, isPub bool) (hasPerm bool) {
	//If a permission record is set for a service Thing then it has priority.
	_, found := svc.cfg.GetPermissions(dThingID)
	if found {
		hasPerm = svc.HasThingPermission(senderID, dThingID, isPub)
	} else {
		hasPerm = svc.HasRolePermission(senderID, messageType, isPub)
	}
	return hasPerm
}

// HasPubPermission returns whether a sender can publish the given message
//
// This returns an error if the client doesn't have permission
func (svc *AuthzService) HasPubPermission(msg *hubclient.ThingMessage) (*hubclient.ThingMessage, error) {
	var hasPerm bool
	//If a thing permission exists then it has priority
	_, found := svc.cfg.GetPermissions(msg.ThingID)
	if found {
		// the publish is for known thing, set with SetPermission()
		hasPerm = svc.HasThingPermission(msg.SenderID, msg.ThingID, true)
	} else {
		hasPerm = svc.HasRolePermission(msg.SenderID, msg.MessageType, true)
	}
	if !hasPerm {
		slog.Warn("Sender has no permissions to publish",
			slog.String("senderID", msg.SenderID),
			slog.String("messageType", msg.MessageType),
			slog.String("thingID", msg.ThingID),
			slog.String("name", msg.Name),
		)
		return msg, fmt.Errorf("Permission denied")
	}
	return msg, nil
}
