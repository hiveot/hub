package authz

import (
	"github.com/hiveot/hub/core/hubclient"
	"github.com/hiveot/hub/lib/ser"
)

// AuthzClient is a marshaller for messaging with the authz service using the hub client connection
// This uses the default serializer to marshal and unmarshal messages.
type AuthzClient struct {
	// ID of the authz service
	hc hubclient.IHubClient
}

// helper for publishing an action request to the authz service
func (authzClient *AuthzClient) pubReq(action string, msg []byte) ([]byte, error) {
	// FIXME: identify the calling client in the request
	return authzClient.hc.PubAction(AuthzServiceName, ManageAuthzCapability, action, msg)
}

func (authzClient *AuthzClient) AddGroup(groupName string, retentionSec uint64) error {
	req := AddGroupReq{
		GroupName: groupName,
		Retention: retentionSec,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(AddGroupAction, msg)
	return err
}
func (authzClient *AuthzClient) AddService(serviceID string, groupName string) error {
	req := AddServiceReq{
		GroupName: groupName,
		ServiceID: serviceID,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(AddServiceAction, msg)
	return err
}

func (authzClient *AuthzClient) AddThing(thingID string, groupName string) error {
	req := AddThingReq{
		GroupName: groupName,
		ThingID:   thingID,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(AddThingAction, msg)
	return err
}

// AddUser adds a user to a group
func (authzClient *AuthzClient) AddUser(userID string, role string, groupName string) error {
	req := AddUserReq{
		GroupName: groupName,
		UserID:    userID,
		Role:      role,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(AddUserAction, msg)
	return err
}

// DeleteGroup deletes a group stream from the default account
//
//	name of the group
func (authzClient *AuthzClient) DeleteGroup(groupName string) error {
	req := DeleteGroupReq{
		GroupName: groupName,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(DeleteGroupAction, msg)
	return err
}

func (authzClient *AuthzClient) GetGroup(groupName string) (grp Group, err error) {
	req := GetGroupReq{
		GroupName: groupName,
	}
	msg, _ := ser.Marshal(req)
	data, err := authzClient.pubReq(GetGroupAction, msg)
	if err != nil {
		return grp, err
	}
	resp := &GetGroupResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.Group, err
}

func (authzClient *AuthzClient) GetClientRoles(clientID string) (grp RoleMap, err error) {
	req := GetClientRolesReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	data, err := authzClient.pubReq(GetClientRolesAction, msg)
	if err != nil {
		return grp, err
	}
	resp := &GetClientRolesResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.Roles, err
}

// GetPermissions of the client for the given things
// Viewers and operators can only get permissions of their own clientID
func (authzClient *AuthzClient) GetPermissions(clientID string, thingIDs []string) (
	permissions map[string][]string, err error) {

	if clientID == "" {
		clientID = authzClient.hc.ClientID()
	}

	req := GetPermissionsReq{
		ClientID: clientID,
		ThingIDs: thingIDs,
	}
	msg, _ := ser.Marshal(req)
	data, err := authzClient.pubReq(GetPermissionsAction, msg)
	if err != nil {
		return nil, err
	}
	resp := &GetPermissionsResp{}
	err = ser.Unmarshal(data, resp)
	return resp.Permissions, err
}

// GetRole determines the highest role a client has for a thing
func (authzClient *AuthzClient) GetRole(clientID string, thingID string) (string, error) {

	req := GetRoleReq{
		ClientID: clientID,
		ThingID:  thingID,
	}
	msg, _ := ser.Marshal(req)
	data, err := authzClient.pubReq(GetRoleAction, msg)
	if err != nil {
		return "", err
	}
	resp := GetRoleResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.Role, err
}

// ListGroups returns a list of groups available to the client
func (authzClient *AuthzClient) ListGroups(clientID string) (groups []Group, err error) {

	req := ListGroupsReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	data, err := authzClient.pubReq(ListGroupsAction, msg)
	if err != nil {
		return nil, err
	}
	resp := ListGroupsResp{}
	err = ser.Unmarshal(data, &resp)
	return resp.Groups, err
}

// RemoveClient removes a client from a group or all groups
// The caller must be an administrator or service
func (authzClient *AuthzClient) RemoveClient(clientID string, groupName string) error {

	req := RemoveClientReq{
		ClientID:  clientID,
		GroupName: groupName,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(RemoveClientAction, msg)
	return err
}

// RemoveClientAll removes a client from all groups
// The caller must be an administrator or service
func (authzClient *AuthzClient) RemoveClientAll(clientID string) error {

	req := RemoveClientReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(RemoveClientAllAction, msg)
	return err
}
func (authzClient *AuthzClient) SetUserRole(userID string, userRole string, groupName string) error {

	req := SetUserRoleReq{
		UserID:    userID,
		GroupName: groupName,
		UserRole:  userRole,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(SetUserRoleAction, msg)
	return err
}

// NewAuthzClient creates a new authz client for use with the hub
func NewAuthzClient(hc hubclient.IHubClient) IAuthz {
	authClient := &AuthzClient{
		hc: hc,
	}
	return authClient

}
