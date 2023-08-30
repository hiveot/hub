package authzclient

import (
	"github.com/hiveot/hub/api/go/auth"
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
	data, err := authzClient.hc.PubAction(auth.AuthzServiceName, auth.ManageAuthzCapability, action, msg)
	err = authzClient.hc.ParseResponse(data, err, resp)
	return err
}

func (authzClient *AuthzClient) AddSource(publisherID, thingID string, groupID string) error {
	req := authz.AddSourceReq{
		GroupID:     groupID,
		PublisherID: publisherID,
		ThingID:     thingID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.AddSourceAction, msg, nil)
	return err
}

// AddUser adds a user to a group
func (authzClient *AuthzClient) AddUser(userID string, role string, groupID string) error {
	req := authz.AddUserReq{
		GroupID: groupID,
		UserID:  userID,
		Role:    role,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.AddUserAction, msg, nil)
	return err
}
func (authzClient *AuthzClient) CreateGroup(
	groupID string, displayName string, retention time.Duration) error {
	req := authz.CreateGroupReq{
		GroupID:     groupID,
		DisplayName: displayName,
		Retention:   uint64(retention.Seconds()),
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.CreateGroupAction, msg, nil)
	return err
}

// DeleteGroup deletes a group stream from the default account
//
//	name of the group
func (authzClient *AuthzClient) DeleteGroup(groupID string) error {
	req := authz.DeleteGroupReq{
		GroupID: groupID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.DeleteGroupAction, msg, nil)
	return err
}

//func (authzClient *AuthzClient) GetClientRoles(clientID string) (grp authz.RoleMap, err error) {
//	req := authz.GetClientRolesReq{
//		ClientID: clientID,
//	}
//	msg, _ := ser.Marshal(req)
//	resp := authz.GetClientRolesResp{}
//	err = authzClient.pubReq(authz.GetClientRolesAction, msg, &resp)
//	if err != nil {
//		return grp, err
//	}
//	return resp.Roles, err
//}

func (authzClient *AuthzClient) GetGroup(groupID string) (grp authz.Group, err error) {
	req := authz.GetGroupReq{
		GroupID: groupID,
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
//func (authzClient *AuthzClient) GetPermissions(clientID string, thingIDs []string) (
//	permissions map[string][]string, err error) {
//
//	if clientID == "" {
//		clientID = authzClient.hc.ClientID()
//	}
//
//	req := authz.GetPermissionsReq{
//		ClientID: clientID,
//		ThingIDs: thingIDs,
//	}
//	msg, _ := ser.Marshal(req)
//	resp := &authz.GetPermissionsResp{}
//	err = authzClient.pubReq(authz.GetPermissionsAction, msg, &resp)
//	if err != nil {
//		return nil, err
//	}
//	return resp.Permissions, err
//}

//// GetRole determines the highest role a client has for a thing
//func (authzClient *AuthzClient) GetRole(clientID string, thingID string) (string, error) {
//
//	req := authz.GetRoleReq{
//		ClientID: clientID,
//		ThingID:  thingID,
//	}
//	msg, _ := ser.Marshal(req)
//	resp := authz.GetRoleResp{}
//	err := authzClient.pubReq(authz.GetRoleAction, msg, &resp)
//	if err != nil {
//		return "", err
//	}
//	return resp.Role, err
//}

// GetUserGroups returns a list of groups available to the client
func (authzClient *AuthzClient) GetUserGroups(userID string) (groups []authz.Group, err error) {

	req := authz.GetUserGroupsReq{
		UserID: userID,
	}
	msg, _ := ser.Marshal(req)
	resp := authz.GetUserGroupsResp{}
	err = authzClient.pubReq(authz.GetUserGroupsAction, msg, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Groups, err
}

// GetUserRoles returns a map of [groupID]role of the user
func (authzClient *AuthzClient) GetUserRoles(userID string) (roles authz.UserRoleMap, err error) {

	req := authz.GetUserRolesReq{
		UserID: userID,
	}
	msg, _ := ser.Marshal(req)
	resp := authz.GetUserRolesResp{}
	err = authzClient.pubReq(authz.GetUserRolesAction, msg, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Roles, err
}

// RemoveSource removes an event source from a group
// The caller must be an administrator or service
func (authzClient *AuthzClient) RemoveSource(publisherID, thingID string, groupID string) error {

	req := authz.RemoveSourceReq{
		PublisherID: publisherID,
		ThingID:     thingID,
		GroupID:     groupID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.RemoveSourceAction, msg, nil)
	return err
}

// RemoveUser removes a user from a group
// The caller must be an administrator or service
func (authzClient *AuthzClient) RemoveUser(userID string, groupID string) error {

	req := authz.RemoveUserReq{
		UserID:  userID,
		GroupID: groupID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.RemoveUserAction, msg, nil)
	return err
}

// RemoveUserAll removes a user from all groups
// The caller must be an administrator or service
func (authzClient *AuthzClient) RemoveUserAll(userID string) error {

	req := authz.RemoveUserAllReq{
		UserID: userID,
	}
	msg, _ := ser.Marshal(req)
	err := authzClient.pubReq(authz.RemoveUserAllAction, msg, nil)
	return err
}
func (authzClient *AuthzClient) SetUserRole(userID string, userRole string, groupID string) error {

	req := authz.SetUserRoleReq{
		UserID:   userID,
		GroupID:  groupID,
		UserRole: userRole,
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
func NewAuthzClient(hc hubclient.IHubClient) auth.IAuthz {
	authzClient := &AuthzClient{
		hc: hc,
	}
	return authzClient

}
