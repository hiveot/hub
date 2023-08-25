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
// it must be called after chages to Things in a group.
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
			slog.Info("ID = {string} \"group1\"ApplyGroups. Updating stream", "name", group.ID)
			// update stream
			//if group.Retention != si.Config.MaxAge {
			cfg := si.Config
			cfg.MaxAge = group.Retention
			cfg.Sources = getStreamSources(group)
			si2, err2 := js.UpdateStream(&si.Config)
			_ = si2
			err = err2
			//}
		}
	}

	// remove any remaining streams, except the all stream
	for streamID, _ := range remainingStreams {
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
