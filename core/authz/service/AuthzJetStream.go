package service

import (
	"fmt"
	"github.com/nats-io/nats.go"
)

// AuthzJetStream applies authz groups to NATS JetStream streams
// This ensures that an event ingress stream exists to can be used as a source.
// Each group has its own stream with a source for events of each thingID. The subject
// of the source is "things.*.{thingID}.event.>
// Users that are group members are added to the stream.
type AuthzJetStream struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// AddGroup adds a new group and creates a stream for it.
//
// publish to the connected stream.
func (svc *AuthzJetStream) AddGroup(groupName string, retention uint64) error {
	//slog.Info("Adding stream", "name", groupName, "source", sourceStream, "filters", subjects)

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
func (svc *AuthzJetStream) AddService(serviceID string, groupName string) error {
	return fmt.Errorf("not yet implemented")
}
func (svc *AuthzJetStream) AddThing(thingID string, groupName string) error {
	return fmt.Errorf("not yet implemented")
}
func (svc *AuthzJetStream) AddUser(userID string, role string, groupName string) (err error) {
	return fmt.Errorf("not yet implemented")
}

func (svc *AuthzJetStream) DeleteGroup(groupName string) error {
	return fmt.Errorf("not yet implemented")
}

func (svc *AuthzJetStream) RemoveClient(clientID string, groupName string) error {
	return fmt.Errorf("not yet implemented")
}
func (svc *AuthzJetStream) RemoveClientAll(clientID string) error {
	return fmt.Errorf("not yet implemented")
}

func (svc *AuthzJetStream) SetUserRole(userID string, role string, groupName string) (err error) {
	return fmt.Errorf("not yet implemented")
}

// Start synchronizes the authorization groups with the JetStream configuraiton
func (svc *AuthzJetStream) Start() error {
	return fmt.Errorf("not yet implemented")
}

func (svc *AuthzJetStream) Stop() {
}

// NewAuthzJetStream applies authz to NATS JetStream streams
func NewAuthzJetStream(nc *nats.Conn) *AuthzJetStream {
	js, err := nc.JetStream()
	if err != nil {
		panic("jetstream not available")
	}
	svc := &AuthzJetStream{
		nc: nc,
		js: js,
	}
	return svc
}
