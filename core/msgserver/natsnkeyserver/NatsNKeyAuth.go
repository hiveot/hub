package natsnkeyserver

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/nats-io/nats-server/v2/server"
	"golang.org/x/exp/slog"
)

// viewers can subscribe to all things
var viewerPermissions = msgserver.RolePermissions{{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeEvent,
	AllowSub: true,
}}

// operators can also publish thing actions
var operatorPermissions = append(viewerPermissions, msgserver.RolePermissions{{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeAction,
	AllowPub: true,
}}...)

// managers can also publish configuration
var managerPermissions = append(operatorPermissions, msgserver.RolePermissions{{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeConfig,
	AllowPub: true,
}}...)

// administrators can do all and publish to services
var adminPermissions = append(managerPermissions, msgserver.RolePermissions{{
	Prefix:   "svc",
	MsgType:  natshubclient.MessageTypeAction,
	AllowPub: true,
	AllowSub: true,
}}...)

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[string]msgserver.RolePermissions{
	auth.ClientRoleViewer:   viewerPermissions,
	auth.ClientRoleOperator: operatorPermissions,
	auth.ClientRoleManager:  managerPermissions,
	auth.ClientRoleAdmin:    adminPermissions,
	auth.ClientRoleNone:     nil,
}

// ApplyAuth reconfigures the server for the given clients and authorization
// if rolePermissions is nil, use the default role permissions
func (srv *NatsNKeyServer) ApplyAuth(clients []msgserver.AuthClient) error {

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

// MakePermissions constructs a permissions object for a client
// Clients that are sources (device,service) receive hard-coded permissions, while users (user,service) permissions
// are based on their role.
func (srv *NatsNKeyServer) MakePermissions(
	clientInfo msgserver.AuthClient,
	authzRoles map[string]msgserver.RolePermissions) *server.Permissions {

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

	// services can also receive action requests on the svc prefix
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
func (srv *NatsNKeyServer) SetRolePermissions(rolePerms map[string]msgserver.RolePermissions) {
	srv.rolePermissions = rolePerms
}
