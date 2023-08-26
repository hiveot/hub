package natsnkeyserver

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
)

// apply update authn/z settings
func (srv *NatsNKeyServer) applyAuth() error {

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
	for _, entry := range srv.clients {
		clientRoles := srv.userGroupRoles[entry.ClientID]
		userPermissions := srv.makePermissions(&entry.ClientProfile, clientRoles)

		if entry.PasswordHash != "" {
			pwUsers = append(pwUsers, &server.User{
				Username:    entry.ClientID,
				Password:    entry.PasswordHash,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
			})
		}

		if entry.PubKey != "" {
			// add an nkey entry
			nkeyUsers = append(nkeyUsers, &server.NkeyUser{
				Nkey:        entry.PubKey,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
				//	//InboxPrefix: "_INBOX." + NoAuthUserID,

			})
		}
	}
	srv.natsOpts.Users = pwUsers
	srv.natsOpts.Nkeys = nkeyUsers
	err := srv.ns.ReloadOptions(&srv.natsOpts)
	return err
}

// ApplyAuthn applies the service authentication clients.
//
//	clients is a list of user, device and service identities
func (srv *NatsNKeyServer) ApplyAuthn(clients []authn.AuthnEntry) error {
	srv.clients = clients
	return srv.applyAuth()
}

// ApplyAuthz applies the authn and authz from the stores to the static server config.
// If userGroupRoles is nil then no permission limits are set. Intended for testing
//
//	clients is a list of user, device and service identities
//	userGroupRoles is a map of users and their group roles
func (srv *NatsNKeyServer) ApplyAuthz(userGroupRoles map[string]authz.RoleMap) error {
	srv.userGroupRoles = userGroupRoles
	return srv.applyAuth()
}

// ApplyGroups synchronizes the groups with jetstream streams
// this adds all 'things' group members as event sources to the stream, so
// it must be called after changes to Things in a group.
func (srv *NatsNKeyServer) ApplyGroups(groups []authz.Group) error {
	remainingStreams := map[string]string{}
	// FIXME: use a dedicated connection, don't reconnect all the time
	nc, err := srv.ConnectInProcNC("applygroups", srv.cfg.CoreServiceKP)
	if err != nil {
		return err
	}
	js, err := nc.JetStream()
	if err != nil {
		return err
	}
	// track deleted groups
	for streamName := range js.StreamNames() {
		remainingStreams[streamName] = streamName
	}

	// add missing streams
	for _, group := range groups {
		delete(remainingStreams, group.ID)
		// create stream
		si, err := js.StreamInfo(group.ID)
		if err != nil {
			// group doesn't exist
			slog.Info("ApplyGroups. Adding stream", "name", group.ID)
			cfg := nats.StreamConfig{
				Name:        group.ID,
				Description: group.DisplayName,
				Retention:   nats.LimitsPolicy,
				MaxAge:      group.Retention,
				Sources:     getStreamSources(group),
			}
			_, err = js.AddStream(&cfg)
		} else {
			slog.Info("ApplyGroups. Updating stream", "name", group.ID)
			// update stream
			//if group.Retention != si.Config.MaxAge {
			cfg := si.Config
			cfg.MaxAge = group.Retention
			cfg.Subjects = nil // nats sets subjects to stream name if no sources were defined
			cfg.Sources = getStreamSources(group)
			si2, err2 := js.UpdateStream(&cfg)
			_ = si2
			err = err2
			//}
		}
		if err != nil {
			slog.Error("failed updating stream", "err", err.Error(), "streamID", group.ID)
		}
	}

	// remove any remaining streams, except the all stream
	for streamID := range remainingStreams {
		if streamID == EventsIntakeStreamName {
			// do nothing
		} else {
			slog.Warn("deleting stream", "name", streamID)
			err = js.DeleteStream(streamID)
			if err != nil {
				slog.Error("ApplyGroups", "err", err.Error())
			}
		}
	}
	nc.Close()
	return nil
}

// getStreamSources returns a list of stream sources for the given group
func getStreamSources(group authz.Group) []*nats.StreamSource {
	sources := []*nats.StreamSource{}
	for clientID, memberRole := range group.MemberRoles {
		if memberRole == authz.GroupRoleThing {
			// each thing is a source
			// once multi-subscriptions is supported this can just be a list of subjects
			subject := natshubclient.MakeSubject("", clientID, natshubclient.SubjectTypeEvent, "")
			sources = append(sources, &nats.StreamSource{
				Name:          EventsIntakeStreamName,
				FilterSubject: subject,
			})
		} else if memberRole == authz.GroupRoleIotDevice {
			// each publisher is a source
			// once multi-subscriptions is supported this can just be a list of subjects
			subject := natshubclient.MakeSubject(clientID, "", natshubclient.SubjectTypeEvent, "")
			sources = append(sources, &nats.StreamSource{
				Name:          EventsIntakeStreamName,
				FilterSubject: subject,
			})
		}
		// ignore non-device/things
	}
	return sources
}

// construct a permissions object for a client and its group memberships
// if groupRoles is nil or empty then the user has no permissions for
// "$JS.API.CONSUMER.>".
func (srv *NatsNKeyServer) makePermissions(
	clientProf *authn.ClientProfile, groupsRole authz.RoleMap) *server.Permissions {

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
	subInbox := "_INBOX." + clientProf.ClientID + ".>"
	subPerm.Allow = append(subPerm.Allow, subInbox)

	// services can pub/sub actions and events
	if clientProf.ClientType == authn.ClientTypeService {
		// publish actions to any thing in the group
		// subscribe events from any thing in the group
		pubService := natshubclient.MakeSubject("", "", "action", ">")
		pubPerm.Allow = append(subPerm.Allow, pubService)
		subService := natshubclient.MakeSubject("", "", "event", ">")
		subPerm.Allow = append(subPerm.Allow, subService)
		// publish events from the service
		// subscribe to actions send to this service
		mySubject := natshubclient.MakeSubject("", clientProf.ClientID, "", ">")
		subPerm.Allow = append(subPerm.Allow, mySubject)
		pubPerm.Allow = append(subPerm.Allow, mySubject)
	} else if clientProf.ClientType == authn.ClientTypeDevice {
		// devices can pub/sub on their own address
		mySubject := natshubclient.MakeSubject(clientProf.ClientID, "", "", ">")
		pubPerm.Allow = append(subPerm.Allow, mySubject)
		subPerm.Allow = append(subPerm.Allow, mySubject)
	} else if clientProf.ClientType == authn.ClientTypeUser {
		// when users have no roles, they cannot use consumers or streams
		if groupsRole == nil || len(groupsRole) == 0 {
			pubPerm.Deny = append(pubPerm.Deny, "$JS.API.CONSUMER.>")
			pubPerm.Deny = append(pubPerm.Deny, "$JS.API.STREAM.>")
		}
		// FIXME: server shouldn't give access to services.
		// how to manage this access?
		subject := natshubclient.MakeSubject(
			authn.AuthnServiceName,
			authn.ClientAuthnCapability,
			natshubclient.SubjectTypeAction, ">")
		pubPerm.Allow = append(pubPerm.Allow, subject)
	}

	// users and services can subscribe to streams (groups) they are a member of.
	if groupsRole != nil &&
		(clientProf.ClientType == authn.ClientTypeUser || clientProf.ClientType == authn.ClientTypeService) {
		for groupName, role := range groupsRole {
			// group members can read from the stream
			// FIXME: is any of this needed?
			subPerm.Allow = append(subPerm.Allow, []string{
				"$JS.API.>", // todo: remove after things start to work
			}...)
			pubPerm.Allow = append(pubPerm.Allow, []string{
				"$JS.API.>", // todo: remove after things start to work
				"$JS.API.CONSUMER.CREATE." + groupName,
				"$JS.API.CONSUMER.LIST." + groupName,
				"$JS.API.CONSUMER.INFO." + groupName + ".>",     // to get consumer info?
				"$JS.API.CONSUMER.MSG.NEXT." + groupName + ".>", // to get consumer info?
			}...)

			// TODO: operators and managers can publish actions for all things in the group
			// Can we use a stream publish that mapped back to the thing?
			// eg: {groupName}.{publisher}.{thing}.action.>
			// maps to things.{publisher}.{thing}.action.>
			// where the stream has a filter on all things added to the stream?
			if role == authz.GroupRoleOperator || role == authz.GroupRoleManager {
				actionSubj := groupName + ".*.*.action.>"
				pubPerm.Allow = append(pubPerm.Allow, actionSubj)
			}
		}
	}

	return perm
}
