package service

import (
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/core/authz/service/aclstore"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// AuthzService handles client management and authorization for access to Things.
// This implements the IAuthz interface
//
// Authorization uses access control lists with group membership and roles to determine if a client
// is authorized to receive or post a message. This applies to all users of the message bus,
// regardless of how they are authenticated.
type AuthzService struct {
	aclStore *aclstore.AclFileStore
	nc       *nats.Conn
}

// GetPermissions returns a list of permissions a client has for a Thing
//func (authzService *AuthzService) GetPermissions(thingID string) (permissions []string, err error) {
//
//	return authzService.GetPermissions(clientID, thingID)
//}

// AddGroup adds a new group and creates a stream for it.
//
// publish to the connected stream.
func (svc *AuthzService) AddGroup(groupName string, retention time.Duration) error {
	//slog.Info("Adding stream", "name", groupName, "source", sourceStream, "filters", subjects)

	// sources that produce events and are a member of the group
	sources := make([]*nats.StreamSource, 0)

	// add a stream source per subject
	//for i, subject := range subjects {
	//	streamSource := &nats.StreamSource{
	//		Name:          sourceStream,
	//		FilterSubject: subject,
	//	}
	//	sources[i] = streamSource
	//}
	cfg := &nats.StreamConfig{
		Name:      groupName,
		Retention: nats.LimitsPolicy,
		Sources:   sources,
		//Subjects:  subjects,
	}
	js, err := svc.nc.JetStream()
	if err != nil {
		return err
	}
	strmInfo, err := js.AddStream(cfg)
	if err != nil {
		return err
	}
	_ = strmInfo
	//
	//cfg := &nats.ConsumerConfig{
	//	Name:          name,
	//	FilterSubject: "",
	//	//Durable:
	//
	//}
	//cinfo, err := hc.js.AddConsumer(name, cfg)
	//_ = cinfo
	return err
}

// AddThing adds a Thing to a group
func (svc *AuthzService) AddThing(groupName string, thingID string) error {

	err := svc.aclStore.SetRole(thingID, groupName, authz.ClientRoleThing)
	return err
}

// GetPermissions returns a list of permissions a client has for a Thing
func (authzService *AuthzService) GetPermissions(clientID string, thingAddr string) (permissions []string, err error) {

	clientRole := authzService.aclStore.GetRole(clientID, thingAddr)
	switch clientRole {
	case authz.ClientRoleIotDevice:
	case authz.ClientRoleThing:
		permissions = []string{authz.PermPubEvents, authz.PermReadActions}
		break
	case authz.ClientRoleService:
		permissions = []string{authz.PermPubActions, authz.PermPubEvents, authz.PermReadActions, authz.PermReadEvents}
		break
	case authz.ClientRoleManager:
		permissions = []string{authz.PermPubActions, authz.PermReadEvents}
		break
	case authz.ClientRoleOperator:
		permissions = []string{authz.PermPubActions, authz.PermReadEvents}
		break
	case authz.ClientRoleViewer:
		permissions = []string{authz.PermReadEvents}
		break
	default:
		permissions = []string{}
	}
	return permissions, nil
}

// IsPublisher checks if the deviceID is the publisher of the thingAddr.
// This requires that the thingAddr is formatted as publisherID/thingID
// Returns true if the deviceID is the publisher of the thingID, false if not.
func (svc *AuthzService) IsPublisher(deviceID string, thingAddr string) (bool, error) {

	// FIXME use a helper for this so the domain knownledge is concentraged
	addrParts := strings.Split(thingAddr, "/")
	return addrParts[0] == deviceID, nil
}

// GetGroup returns the group with the given name, or an error if group is not found.
// GroupName must not be empty
func (svc *AuthzService) GetGroup(groupName string) (group authz.Group, err error) {

	group, err = svc.aclStore.GetGroup(groupName)
	return group, err
}

// GetGroupRoles returns a list of roles in groups the client is a member of.
func (svc *AuthzService) GetGroupRoles(clientID string) (roles authz.RoleMap, err error) {

	// simple pass through
	roles = svc.aclStore.GetGroupRoles(clientID)
	return roles, nil
}

// ListGroups returns the list of known groups
func (svc *AuthzService) ListGroups(limit int, offset int) (groups []authz.Group, err error) {

	groups = svc.aclStore.ListGroups(limit, offset)
	return groups, nil
}

// RemoveAll from all groups
func (svc *AuthzService) RemoveAll(clientID string) error {
	err := svc.aclStore.RemoveAll(clientID)
	return err
}

// RemoveClient from a group
func (svc *AuthzService) RemoveClient(clientID string, groupName string) error {
	err := svc.aclStore.Remove(clientID, groupName)
	return err
}

// RemoveThing removes a Thing from a group
func (svc *AuthzService) RemoveThing(thingID string, groupName string) error {

	err := svc.aclStore.Remove(thingID, groupName)
	return err
}

// SetClientRole sets the role for the client in a group
func (svc *AuthzService) SetClientRole(clientID string, groupName string, role string) error {
	err := svc.aclStore.SetRole(clientID, groupName, role)
	return err
}

// Stop closes the service and release resources
func (svc *AuthzService) Stop() {
	svc.aclStore.Close()
}

// Start the ACL store for reading
func (svc *AuthzService) Start() error {
	slog.Info("Opening ACL store")
	err := svc.aclStore.Open()
	if err != nil {
		return err
	}
	return nil
}

// NewAuthzService creates a new instance of the authorization service.
//
//	aclStore provides the functions to read and write authorization rules
func NewAuthzService(aclStorePath string) *AuthzService {
	aclStore := aclstore.NewAclFileStore(aclStorePath, authz.AuthzServiceName)

	authzService := AuthzService{
		aclStore: aclStore,
	}
	return &authzService
}
