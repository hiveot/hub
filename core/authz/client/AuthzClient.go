package client

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/lib/ser"
	"time"
)

// AuthzClient is a marshaller for messaging with the authz service using the hub client connection
// This uses the default serializer to marshal and unmarshal messages.
type AuthzClient struct {
	// ID of the authz service
	serviceID string
	hc        *hubconn.HubConnNats
}

// helper for publishing an action request to the authz service
func (authzClient *AuthzClient) pubReq(action string, msg []byte) ([]byte, error) {
	return authzClient.hc.PubAction(authzClient.serviceID, "", action, msg)
}

func (authzClient *AuthzClient) AddGroup(groupName string, retention time.Duration) error {
	req := authz.AddGroupReq{
		GroupName: groupName,
		Retention: uint64(retention),
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(authz.AddGroupAction, msg)
	return err
}
func (authzClient *AuthzClient) AddService(groupName string, serviceID string) error {
	req := authz.AddServiceReq{
		GroupName: groupName,
		ServiceID: serviceID,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(authz.AddServiceAction, msg)
	return err
}

func (authzClient *AuthzClient) AddThing(groupName string, thingID string) error {
	req := authz.AddThingReq{
		GroupName: groupName,
		ThingID:   thingID,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(authz.AddThingAction, msg)
	return err
}

// AddUser adds a user to a group
func (authzClient *AuthzClient) AddUser(groupName string, userID string, role string) error {
	req := authz.AddUserReq{
		GroupName: groupName,
		UserID:    userID,
		Role:      role,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(authn.AddUserAction, msg)
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
	_, err := authzClient.pubReq(authz.DeleteGroupAction, msg)
	return err
}

func (authzClient *AuthzClient) GetGroup(groupName string) (grp *authz.Group, err error) {
	req := authz.GetGroupReq{
		GroupName: groupName,
	}
	msg, _ := ser.Marshal(req)
	data, err := authzClient.pubReq(authz.GetGroupAction, msg)
	if err != nil {
		return nil, err
	}
	resp := &authz.GetGroupResp{}
	err = ser.Unmarshal(data, &resp)
	return &resp.Group, err
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
	data, err := authzClient.pubReq(authz.GetPermissionsAction, msg)
	if err != nil {
		return nil, err
	}
	resp := &authz.GetPermissionsResp{}
	err = ser.Unmarshal(data, resp)
	return resp.Permissions, err
}

// ListGroups returns a list of groups available to the client
func (authzClient *AuthzClient) ListGroups() (groups []authz.Group, err error) {

	req := authz.ListGroupsReq{
		ClientID: authzClient.hc.ClientID(),
	}
	msg, _ := ser.Marshal(req)
	data, err := authzClient.pubReq(authz.ListGroupsAction, msg)
	if err != nil {
		return nil, err
	}
	resp := authz.ListGroupsResp{}
	err = ser.Unmarshal(data, &resp)
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
	_, err := authzClient.pubReq(authz.RemoveClientAction, msg)
	return err
}

func (authzClient *AuthzClient) SetUserRole(userID string, groupName string, userRole string) error {

	req := authz.SetUserRoleReq{
		UserID:    userID,
		GroupName: groupName,
		UserRole:  userRole,
	}
	msg, _ := ser.Marshal(req)
	_, err := authzClient.pubReq(authz.SetUserRoleAction, msg)
	return err
}

// NewAuthzClient creates a new authz client for use with the hub
func NewAuthzClient(hc *hubconn.HubConnNats) authz.IAuthz {
	authClient := &AuthzClient{
		hc:        hc,
		serviceID: "authz",
	}
	return authClient

}
