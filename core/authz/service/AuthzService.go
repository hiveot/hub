package service

import (
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/core/authz"
	"github.com/hiveot/hub/core/authz/service/aclstore"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
	"strings"
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
	// ca certificate for connection validation
	caCert *x509.Certificate
}

// GetPermissions returns a list of permissions a client has for a Thing
//func (authzService *AuthzService) GetPermissions(thingID string) (permissions []string, err error) {
//
//	return authzService.GetPermissions(clientID, thingID)
//}

// AddGroup adds a new group and creates a stream for it.
//
// publish to the connected stream.
func (svc *AuthzService) AddGroup(groupName string, retention uint64) error {
	//slog.Info("Adding stream", "name", groupName, "source", sourceStream, "filters", subjects)

	err := svc.aclStore.AddGroup(groupName)
	if err != nil {
		return err
	}

	// sources that produce events and are a member of the group
	sources := make([]*nats.StreamSource, 0)

	// TODO add a stream source per subject
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

// AddService adds a client with the service role to a group
func (svc *AuthzService) AddService(serviceID string, groupName string) error {

	err := svc.aclStore.SetRole(serviceID, authz.ClientRoleService, groupName)
	return err
}

// AddThing adds a client with the thing role to a group
func (svc *AuthzService) AddThing(thingID string, groupName string) error {

	err := svc.aclStore.SetRole(thingID, authz.ClientRoleThing, groupName)
	return err
}

// AddUser adds a client with the user role to a group
func (svc *AuthzService) AddUser(userID string, role string, groupName string) (err error) {

	if role == authz.ClientRoleViewer ||
		role == authz.ClientRoleManager ||
		role == authz.ClientRoleOperator {

		err = svc.aclStore.SetRole(userID, role, groupName)
	} else {
		err = fmt.Errorf("role '%s' doesn't apply to users", role)
	}
	return err
}

// DeleteGroup deletes the group and associated resources. Use with care
func (svc *AuthzService) DeleteGroup(groupName string) error {
	return svc.aclStore.DeleteGroup(groupName)
}

// GetClientRoles returns a map of [group]role for a client
func (svc *AuthzService) GetClientRoles(clientID string) (roles authz.RoleMap, err error) {

	// simple pass through
	roles = svc.aclStore.GetClientRoles(clientID)
	return roles, nil
}

// GetGroup returns the group with the given name, or an error if group is not found.
// GroupName must not be empty
func (svc *AuthzService) GetGroup(groupName string) (group authz.Group, err error) {

	group, err = svc.aclStore.GetGroup(groupName)
	return group, err
}

// GetPermissions returns a list of permissions a client has for Things
func (svc *AuthzService) GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error) {
	permissions = make(map[string][]string)
	for _, thingID := range thingIDs {
		var thingPerm []string
		clientRole := svc.aclStore.GetRole(clientID, thingID)
		switch clientRole {
		case authz.ClientRoleIotDevice:
		case authz.ClientRoleThing:
			thingPerm = []string{authz.PermPubEvents, authz.PermReadActions}
			break
		case authz.ClientRoleService:
			thingPerm = []string{authz.PermPubActions, authz.PermPubEvents, authz.PermReadActions, authz.PermReadEvents}
			break
		case authz.ClientRoleManager:
			// managers are operators but can also change configuration
			// TODO: is publishing configuration changes a separate permission?
			thingPerm = []string{authz.PermPubActions, authz.PermReadEvents}
			break
		case authz.ClientRoleOperator:
			thingPerm = []string{authz.PermPubActions, authz.PermReadEvents}
			break
		case authz.ClientRoleViewer:
			thingPerm = []string{authz.PermReadEvents}
			break
		default:
			thingPerm = []string{}
		}
		permissions[thingID] = thingPerm
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

// ListGroups returns the list of known groups available to the client
func (svc *AuthzService) ListGroups(clientID string) (groups []authz.Group, err error) {

	groups = svc.aclStore.ListGroups(clientID)
	return groups, nil
}

// RemoveClient from a group
func (svc *AuthzService) RemoveClient(clientID string, groupName string) error {
	err := svc.aclStore.RemoveClient(clientID, groupName)
	return err
}

// RemoveClientAll from all groups
func (svc *AuthzService) RemoveClientAll(clientID string) error {
	err := svc.aclStore.RemoveClientAll(clientID)
	return err
}

// SetUserRole sets the role for the user in a group
func (svc *AuthzService) SetUserRole(userID string, role string, groupName string) (err error) {
	if role == authz.ClientRoleViewer ||
		role == authz.ClientRoleManager ||
		role == authz.ClientRoleOperator {
		err = svc.aclStore.SetRole(userID, role, groupName)
	} else {
		err = fmt.Errorf("role '%s' doesn't apply to users", role)
	}
	return err
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

// Stop closes the service and release resources
func (svc *AuthzService) Stop() {
	slog.Info("stopping AuthzService")
	svc.aclStore.Close()
}

// NewAuthzService creates a new instance of the authorization service.
//
//	aclStore provides the functions to read and write authorization rules
func NewAuthzService(aclStorePath string, caCert *x509.Certificate) *AuthzService {
	aclStore := aclstore.NewAclFileStore(aclStorePath, authz.AuthzServiceName)

	authzService := AuthzService{
		aclStore: aclStore,
		caCert:   caCert,
	}
	return &authzService
}
