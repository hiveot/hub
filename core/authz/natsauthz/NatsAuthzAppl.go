package natsauthz

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
)

// GroupSourceStreamName all group streams use this stream as their source
const GroupSourceStreamName = "$events"

// NatsAuthzAppl applies authz groups to NATS JetStream streams
// This implements the IAuthz API to support dependency injection for other messaging systems.
// This ensures that an event ingress stream exists to can be used as a source.
// Each group has its own stream with a source for events of each thingID. The subject
// of the source is "things.*.{thingID}.event.>
// Users that are group members are added to the stream.
type NatsAuthzAppl struct {
	//nc *nats.Conn
	js nats.JetStreamContext
}

// AddGroup adds a new group and creates a stream for it.
//
// publish to the connected stream.
func (svc *NatsAuthzAppl) AddGroup(groupName string, retention uint64) error {
	slog.Info("AddGroup", "groupName", groupName)

	// return if the stream exists
	_, err := svc.js.StreamInfo(groupName)
	if err == nil {
		return nil
	}

	// TODO: add retention
	// sources that produce events and are a member of the group
	sources := make([]*nats.StreamSource, 0)

	cfg := &nats.StreamConfig{
		Name:        groupName,
		Description: "HiveOT Group",
		Retention:   nats.LimitsPolicy,
		Template:    "",
		Sources:     sources,
		// TODO: change things.> to groupName.>  ???
		SubjectTransform: nil,
	}
	strmInfo, err := svc.js.AddStream(cfg)
	if err != nil {
		return err
	}
	_ = strmInfo
	return err
}
func (svc *NatsAuthzAppl) AddService(serviceID string, groupName string) error {
	//return fmt.Errorf("not yet implemented")
	// todo: access control for the service. Not sure how.
	return nil
}

// AddThing adds the thing's subject as a stream source
func (svc *NatsAuthzAppl) AddThing(thingID string, groupName string) error {
	slog.Info("AddThing",
		slog.String("thingID", thingID),
		slog.String("groupName", groupName),
	)
	streamInfo, err := svc.js.StreamInfo(groupName)
	if err != nil {
		return fmt.Errorf("AddThing adding thing '%s' to group '%s' failed: %w",
			thingID, groupName, err)
	}
	thingSubject := natshubclient.MakeSubject("", thingID, "event", "")
	// the 'GroupSourceStreamName' stream receives all events
	newSource := &nats.StreamSource{
		Name:          GroupSourceStreamName,
		FilterSubject: thingSubject,
	}
	streamInfo.Config.Sources = append(streamInfo.Config.Sources, newSource)
	newInfo, err := svc.js.UpdateStream(&streamInfo.Config)
	_ = newInfo
	return err
}
func (svc *NatsAuthzAppl) AddUser(userID string, role string, groupName string) (err error) {
	// todo: access control for the service. Not sure how.
	// allow user to create a stream consumer
	eventSubj := "$JS.API.CONSUMER.CREATE." + groupName
	_ = eventSubj
	// FIXME: where to set this user permission?
	return fmt.Errorf("not yet implemented")
}

func (svc *NatsAuthzAppl) DeleteGroup(groupName string) error {
	slog.Info("DeleteGroup", "groupName", groupName)
	err := svc.js.DeleteStream(groupName)
	return err
}

func (svc *NatsAuthzAppl) GetGroup(groupName string) (grp authz.Group, err error) {
	// intended for verifying during testing
	sInfo, err := svc.js.StreamInfo(groupName)
	if err != nil {
		return grp, err
	}
	grp.Name = groupName
	// each source is a thing
	for _, s := range sInfo.Sources {
		_, thingID, _, _, err := natshubclient.SplitSubject(s.FilterSubject)
		if err != nil {
			grp.MemberRoles[thingID] = authz.GroupRoleThing
		}
	}
	// sorry, the stream doesn't contain other permissions
	// TODO: determine other members roles from the stream. Not sure how.
	return grp, nil
}

func (svc *NatsAuthzAppl) GetRole(clientID string, thingID string) (role string, err error) {
	// intended for verifying during testing
	err = fmt.Errorf("not implemented here")
	return
}
func (svc *NatsAuthzAppl) GetClientRoles(clientID string) (roles authz.RoleMap, err error) {
	err = fmt.Errorf("not implemented here")
	return
}
func (svc *NatsAuthzAppl) GetPermissions(clientID string, thingIDs []string) (permissions map[string][]string, err error) {
	err = fmt.Errorf("not implemented here")
	return
}
func (svc *NatsAuthzAppl) ListGroups(clientID string) (groups []authz.Group, err error) {
	names := svc.js.StreamNames()
	groups = make([]authz.Group, 0, len(names))
	for name := range names {
		groups = append(groups, authz.Group{Name: name})
	}
	return groups, err
}

func (svc *NatsAuthzAppl) RemoveClient(clientID string, groupName string) error {
	// remove things
	slog.Info("RemoveClient",
		slog.String("clientID", clientID),
		slog.String("groupName", groupName),
	)
	sInfo, err := svc.js.StreamInfo(groupName)
	if err != nil {
		return err
	}
	// remove clientID as a source of a thing
	// each source is a thing
	sources := sInfo.Config.Sources
	for i, s := range sources {
		// thing IDs are in the filter subject of the source
		_, thingID, _, _, _ := natshubclient.SplitSubject(s.FilterSubject)
		if thingID == clientID {
			sources[i] = sources[len(sources)-1]
			sources = sources[:len(sources)-1]
			break
		}
	}
	_, err = svc.js.UpdateStream(&sInfo.Config)
	return err
}
func (svc *NatsAuthzAppl) RemoveClientAll(clientID string) error {
	names := svc.js.StreamNames()
	for name := range names {
		err := svc.RemoveClient(clientID, name)
		if err != nil {
			return err
		}
	}
	slog.Info("RemoveClientAll",
		slog.String("clientID", clientID),
	)
	return nil
}

func (svc *NatsAuthzAppl) SetUserRole(userID string, role string, groupName string) (err error) {
	return fmt.Errorf("not implemented here")
}

//// Start synchronizes the authorization groups with the JetStream configuraiton
//func (svc *NatsAuthzAppl) Start() error {
//	return fmt.Errorf("not yet implemented")
//}
//
//func (svc *NatsAuthzAppl) Stop() {
//}

// ensure the events stream exists to receive all events
// this stream subscribes to all events
func (svc *NatsAuthzAppl) init() error {
	// return if the stream exists
	_, err := svc.js.StreamInfo(GroupSourceStreamName)
	if err != nil {
		return err
	}

	// TODO: add retention
	// sources that produce events and are a member of the group
	subj := natshubclient.MakeSubject("", "", natshubclient.SubjectTypeEvent, ">")
	cfg := &nats.StreamConfig{
		Name:        GroupSourceStreamName,
		Description: "HiveOT Events Intake Stream",
		Retention:   nats.LimitsPolicy,
		Subjects:    []string{subj},
	}
	_, err = svc.js.AddStream(cfg)
	return err
}

// NewNatsAuthzAppl applies authz to NATS JetStream streams
// This implements the IAuthz interface
func NewNatsAuthzAppl(js nats.JetStreamContext) (*NatsAuthzAppl, error) {
	svc := &NatsAuthzAppl{
		js: js,
	}
	err := svc.init()

	return svc, err
}
