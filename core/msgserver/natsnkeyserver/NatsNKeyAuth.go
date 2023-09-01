package natsnkeyserver

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/nats-io/nats-server/v2/server"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
	"time"
)

// viewers can subscribe to all things
var viewerPermissions = []msgserver.RolePermission{{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeEvent,
	AllowSub: true,
}}

// operators can also publish thing actions
var operatorPermissions = append(viewerPermissions, msgserver.RolePermission{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeAction,
	AllowPub: true,
})

// managers can also publish configuration
var managerPermissions = append(operatorPermissions, msgserver.RolePermission{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeConfig,
	AllowPub: true,
})

// administrators can do all and publish to services
var adminPermissions = append(managerPermissions, msgserver.RolePermission{
	Prefix:   "svc",
	MsgType:  natshubclient.MessageTypeAction,
	AllowPub: true,
	AllowSub: true,
})

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[string][]msgserver.RolePermission{
	auth.ClientRoleViewer:   viewerPermissions,
	auth.ClientRoleOperator: operatorPermissions,
	auth.ClientRoleManager:  managerPermissions,
	auth.ClientRoleAdmin:    adminPermissions,
	auth.ClientRoleNone:     nil,
}

// ServicePermissions defines for each role the service capability that can be used
var ServicePermissions = map[string][]msgserver.RolePermission{}

// ApplyAuth reconfigures the server for authentication and authorization.
// For each client this applies the permissions associated with the client type and role.
//
//	Role permissions can be changed with 'SetRolePermissions'.
//	Service permissions can be set with 'SetServicePermissions'
func (srv *NatsNKeyServer) ApplyAuth(clients []msgserver.AuthClient) error {

	// password users authenticate with password while nkey users authenticate with key-pairs.
	// clients can use both.
	pwUsers := []*server.User{}
	nkeyUsers := []*server.NkeyUser{}

	// keep the core service that was added on server start
	coreServicePub, _ := srv.cfg.CoreServiceKP.PublicKey()
	nkeyUsers = append(nkeyUsers, &server.NkeyUser{
		Nkey:        coreServicePub,
		Permissions: nil, // unlimited access
		Account:     srv.cfg.appAcct,
	})

	// apply authn all clients
	for _, clientInfo := range clients {
		userPermissions := srv.MakePermissions(clientInfo, srv.rolePermissions)

		if clientInfo.PasswordHash != "" {
			pwUsers = append(pwUsers, &server.User{
				Username:    clientInfo.ClientID,
				Password:    clientInfo.PasswordHash,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
			})
		}

		if clientInfo.PubKey != "" {
			// add an nkey entry
			nkeyUsers = append(nkeyUsers, &server.NkeyUser{
				Nkey:        clientInfo.PubKey,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
			})
		}
	}
	srv.natsOpts.Users = pwUsers
	srv.natsOpts.Nkeys = nkeyUsers

	err := srv.ns.ReloadOptions(&srv.natsOpts)
	return err
}

// CreateToken uses the public key as token when using nkeys
func (srv *NatsNKeyServer) CreateToken(
	clientID string, clientType string, pubKey string, tokenValidity time.Duration) (token string, err error) {
	//
	if srv.natsOpts.AuthCallout != nil {
		token, err = srv.tokenizer.CreateToken(clientID, clientType, pubKey, tokenValidity)
	} else {
		// not using callout sso use public key as token
		token = pubKey
	}
	return token, err
}

// MakePermissions constructs a permissions object for a client
// Clients that are sources (device,service) receive hard-coded permissions, while users (user,service) permissions
// are based on their role.
func (srv *NatsNKeyServer) MakePermissions(
	clientInfo msgserver.AuthClient,
	authzRoles map[string][]msgserver.RolePermission) *server.Permissions {

	subPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  []string{},
	}
	pubPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  []string{},
	}
	perm := &server.Permissions{
		Publish:   &pubPerm,
		Subscribe: &subPerm,
		Response:  nil,
	}
	// all clients can use their inbox, using inbox prefix
	subInbox := "_INBOX." + clientInfo.ClientID + ".>"
	subPerm.Allow = append(subPerm.Allow, subInbox)

	rolePerm, found := authzRoles[clientInfo.Role]
	if found && rolePerm == nil {
		// no permissions for this role
	} else if found {
		// apply role permissions
		for _, rp := range rolePerm {
			subj := natshubclient.MakeThingsSubject(rp.SourceID, rp.ThingID, rp.MsgType, rp.MsgName)
			if rp.AllowPub {
				pubPerm.Allow = append(pubPerm.Allow, subj)
			}
			if rp.AllowSub {
				subPerm.Allow = append(subPerm.Allow, subj)
			}
		}
		// allow event stream access
		streamName := EventsIntakeStreamName
		pubPerm.Allow = append(pubPerm.Allow, []string{
			//"$JS.API.>", // FIXME: remove after things start to work
			"$JS.API.CONSUMER.CREATE." + streamName,
			"$JS.API.CONSUMER.LIST." + streamName,
			"$JS.API.CONSUMER.INFO." + streamName + ".>",     // to get consumer info?
			"$JS.API.CONSUMER.MSG.NEXT." + streamName + ".>", // to get consumer info?
		}...)
	} else {
		// unknown role
		slog.Error("unknown role",
			"clientID", clientInfo.ClientID, "clientType", clientInfo.ClientType,
			"role", clientInfo.Role)

		// when clients have no roles, they cannot use consumers or streams
		pubPerm.Deny = append(pubPerm.Deny, "$JS.API.CONSUMER.>")
		pubPerm.Deny = append(pubPerm.Deny, "$JS.API.STREAM.>")
	}

	// user role might be allowed to use a service capability
	sp, found := ServicePermissions[clientInfo.Role]
	if found && sp != nil {
		for _, perm := range sp {
			subject := natshubclient.MakeServiceSubject(
				perm.SourceID, perm.ThingID, natshubclient.MessageTypeAction, "")
			pubPerm.Allow = append(pubPerm.Allow, subject)
		}
	}

	// devices and services are sources that can publish events and subscribe to actions and config requests
	if clientInfo.ClientType == auth.ClientTypeDevice || clientInfo.ClientType == auth.ClientTypeService {
		subject1 := natshubclient.MakeThingsSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeEvent, "")
		subject2 := natshubclient.MakeThingsSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeAction, "")
		subject3 := natshubclient.MakeThingsSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeConfig, "")
		pubPerm.Allow = append(pubPerm.Allow, subject1)
		subPerm.Allow = append(subPerm.Allow, subject2, subject3)
	}

	// services can also subscribe to actions on the svc prefix
	if clientInfo.ClientType == auth.ClientTypeService {
		subject1 := natshubclient.MakeServiceSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeEvent, "")
		subject2 := natshubclient.MakeServiceSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeAction, "")
		subject3 := natshubclient.MakeServiceSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeConfig, "")
		pubPerm.Allow = append(pubPerm.Allow, subject1)
		subPerm.Allow = append(subPerm.Allow, subject2, subject3)
	}

	return perm
}

// SetRolePermissions sets a custom map of user role->[]permissions
func (srv *NatsNKeyServer) SetRolePermissions(
	rolePerms map[string][]msgserver.RolePermission) {
	srv.rolePermissions = rolePerms
}

// SetServicePermissions adds the service permissions to the roles
func (srv *NatsNKeyServer) SetServicePermissions(
	serviceID string, capability string, roles []string) {

	for _, role := range roles {
		// add the role if needed
		rp := ServicePermissions[role]
		if rp == nil {
			rp = []msgserver.RolePermission{}
		}
		rp = append(rp, msgserver.RolePermission{
			Prefix:   "svc",
			SourceID: serviceID,
			ThingID:  capability,
			MsgType:  natshubclient.MessageTypeAction,
			MsgName:  "", // all methods of the capability can be used
			AllowPub: true,
			AllowSub: false,
		})
		ServicePermissions[role] = rp
	}

}

// ValidateToken checks if the given token belongs the clientID and is valid.
// When keys is used this returns success
// When nkeys is not used this validates the JWT token
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (srv *NatsNKeyServer) ValidateToken(
	clientID string, pubKey string, oldToken string, signedNonce string, nonce string) (err error) {
	if srv.natsOpts.AuthCallout == nil {
		// nkeys only
		if oldToken == "" || pubKey != oldToken {
			return fmt.Errorf("invalid old token for client '%s'", clientID)
		}
		return nil
	}
	return srv.tokenizer.ValidateToken(clientID, pubKey, oldToken, signedNonce, nonce)
}

// ValidatePassword checks if the given password matches the user
func (srv *NatsNKeyServer) ValidatePassword(loginID string, password string) error {
	if loginID == "" || password == "" {
		return fmt.Errorf("Password validation failed for user '%s'", loginID)
	}
	// don't expect many users so loop is okay
	for _, u := range srv.natsOpts.Users {
		if u.Username == loginID {
			err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
			return err
		}
	}
	return fmt.Errorf("unknown client '%s'", loginID)
}
