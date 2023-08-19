package authzclient

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"time"
)

// AuthzClient is a marshaller for messaging with the authz service using the hub client connection
// This uses the default serializer to marshal and unmarshal messages.
type AuthzClient struct {
	// ID of the authz service
	hc hubclient.IHubClient
}

// helper for publishing an action request to the authz service
func (authzClient *AuthzClient) pubReq(action string, msg []byte, resp interface{}) error {
	// FIXME: identify the calling client in the request
	data, err := authzClient.hc.PubAction(authz.AuthzServiceName, authz.ManageAuthzCapability, action, msg)
	err = authzClient.hc.ParseResponse(data, err, resp)
	return err
}

func (authzClient *AuthzClient) AddGroup(groupName string, retention time.Duration) error {
	req := authz.AddGroupReq{
		GroupName: groupName,
		Retention: uint64(retention.Seconds()),
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.AddGroupAction, msg, nil)
	return err
}
func (authzClient *AuthzClient) AddService(serviceID string, groupName string) error {
	req := authz.AddServiceReq{
		GroupName: groupName,
		ServiceID: serviceID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.AddServiceAction, msg, nil)
	return err
}

func (authzClient *AuthzClient) AddThing(thingID string, groupName string) error {
	req := authz.AddThingReq{
		GroupName: groupName,
		ThingID:   thingID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.AddThingAction, msg, nil)
	return err
}

// AddUser adds a user to a group
func (authzClient *AuthzClient) AddUser(userID string, role string, groupName string) error {
	req := authz.AddUserReq{
		GroupName: groupName,
		UserID:    userID,
		Role:      role,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.AddUserAction, msg, nil)
	return err
}

// DeleteGroup deletes a group stream from the default account
//
//	name of the group
func (authzClient *AuthzClient) DeleteGroup(groupName string) error {
	req := authz.DeleteGroupReq{
		GroupName: groupName,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.DeleteGroupAction, msg, nil)
	return err
}

func (authzClient *AuthzClient) GetClientRoles(clientID string) (grp authz.RoleMap, err error) {
	req := authz.GetClientRolesReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	resp := authz.GetClientRolesResp{}
	err = authzClient.pubReq(authz.GetClientRolesAction, msg, &resp)
	if err != nil {
		return grp, err
	}
	return resp.Roles, err
}

func (authzClient *AuthzClient) GetGroup(groupName string) (grp authz.Group, err error) {
	req := authz.GetGroupReq{
		GroupName: groupName,
	}
	msg, _ := ser.Marshal(req)
	resp := &authz.GetGroupResp{}
	err = authzClient.pubReq(authz.GetGroupAction, msg, &resp)
	if err != nil {
		return grp, err
	}
	return resp.Group, err
}

// GetPermissions of the client for the given things
// Viewers and operators can only get permissions of their own clientID
func (authzClient *AuthzClient) GetPermissions(clientID string, thingIDs []string) (
	permissions map[string][]string, err error) {

	if clientID == "" {
		clientID = authzClient.hc.ClientID()
	}

	req := authz.GetPermissionsReq{
		ClientID: clientID,
		ThingIDs: thingIDs,
	}
	msg, _ := ser.Marshal(req)
	resp := &authz.GetPermissionsResp{}
	err = authzClient.pubReq(authz.GetPermissionsAction, msg, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Permissions, err
}

// GetRole determines the highest role a client has for a thing
func (authzClient *AuthzClient) GetRole(clientID string, thingID string) (string, error) {

	req := authz.GetRoleReq{
		ClientID: clientID,
		ThingID:  thingID,
	}
	msg, _ := ser.Marshal(req)
	resp := authz.GetRoleResp{}
	err := authzClient.pubReq(authz.GetRoleAction, msg, &resp)
	if err != nil {
		return "", err
	}
	return resp.Role, err
}

// ListGroups returns a list of groups available to the client
func (authzClient *AuthzClient) ListGroups(clientID string) (groups []authz.Group, err error) {

	req := authz.ListGroupsReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	resp := authz.ListGroupsResp{}
	err = authzClient.pubReq(authz.ListGroupsAction, msg, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Groups, err
}

// RemoveClient removes a client from a group or all groups
// The caller must be an administrator or service
func (authzClient *AuthzClient) RemoveClient(clientID string, groupName string) error {

	req := authz.RemoveClientReq{
		ClientID:  clientID,
		GroupName: groupName,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.RemoveClientAction, msg, nil)
	return err
}

// RemoveClientAll removes a client from all groups
// The caller must be an administrator or service
func (authzClient *AuthzClient) RemoveClientAll(clientID string) error {

	req := authz.RemoveClientReq{
		ClientID: clientID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.RemoveClientAllAction, msg, nil)
	return err
}
func (authzClient *AuthzClient) SetUserRole(userID string, userRole string, groupName string) error {

	req := authz.SetUserRoleReq{
		UserID:    userID,
		GroupName: groupName,
		UserRole:  userRole,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.SetUserRoleAction, msg, nil)
	return err
}

// Start doesn't do anything here
func (authzClient *AuthzClient) Start() error {
	return nil
}
func (authzClient *AuthzClient) Stop() {}

// NewAuthzClient creates a new authz client for use with the hub
func NewAuthzClient(hc hubclient.IHubClient) authz.IAuthz {
	authzClient := &AuthzClient{
		hc: hc,
	}
	return authzClient

}
