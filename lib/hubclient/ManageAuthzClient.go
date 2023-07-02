package hubclient

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
	"time"
)

// ManageAuthzClient is a marshaller for messaging with the authz service using the hub client
type ManageAuthzClient struct {
	serviceID string
	hc        *HubClient
}

// helper for publishing an action request to the authz service
func (authzClient *ManageAuthzClient) pubReq(action string, msg []byte) ([]byte, error) {
	return authzClient.hc.PubAction(authzClient.serviceID, "", action, msg)
}

func (authzClient *ManageAuthzClient) AddGroup(groupName string, retention time.Duration) error {
	req := hub.AddGroupReq{
		GroupName: groupName,
		Retention: uint64(retention),
	}
	msg, _ := json.Marshal(req)
	_, err := authzClient.pubReq(hub.AddGroupAction, msg)
	return err
}

func (authzClient *ManageAuthzClient) AddThing(groupName string, thingID string) error {
	req := hub.AddThingReq{
		GroupName: groupName,
		ThingID:   thingID,
	}
	msg, _ := json.Marshal(req)
	_, err := authzClient.pubReq(hub.AddThingAction, msg)
	return err
}
func (authzClient *ManageAuthzClient) AddService(groupName string, serviceID string) error {
	req := hub.AddServiceReq{
		GroupName: groupName,
		ServiceID: serviceID,
	}
	msg, _ := json.Marshal(req)
	_, err := authzClient.pubReq(hub.AddServiceAction, msg)
	return err
}

func (authzClient *ManageAuthzClient) AddUser(groupName string, userID string) error {
	req := hub.AddUserReq{
		GroupName: groupName,
		UserID:    userID,
	}
	msg, _ := json.Marshal(req)
	_, err := authzClient.pubReq(hub.AddUserAction, msg)
	return err
}

// DeleteGroup deletes a group stream from the default account
//
//	name of the group
func (authzClient *ManageAuthzClient) DeleteGroup(groupName string) error {
	req := hub.DeleteGroupReq{
		GroupName: groupName,
	}
	msg, _ := json.Marshal(req)
	_, err := authzClient.pubReq(hub.DeleteGroupAction, msg)
	return err
}

// SubGroup subscribes to all events in a group
func (authzClient *ManageAuthzClient) SubGroup(groupID string, cb func(tv *thing.ThingValue)) error {
	slog.Info("SubGroup", "groupID", groupID)
	subscription, err := authzClient.hc.js.Subscribe("", func(msg *nats.Msg) {
		md, _ := msg.Metadata()
		timeStamp := time.Now()
		if md != nil {
			timeStamp = md.Timestamp

		}
		pubID, thID, _, name, err := SplitSubject(msg.Subject)
		if err != nil {
			slog.Error("unable to handle subject", "err", err, "subject", msg.Subject)
			return
		}
		payload := msg.Data
		tv := &thing.ThingValue{
			ID:          name,
			PublisherID: pubID,
			ThingID:     thID,
			Data:        payload,
			Created:     timeStamp.Format(vocab.ISO8601Format),
		}
		cb(tv)
	}, nats.OrderedConsumer(),
		nats.BindStream(groupID),
	)
	_ = subscription
	return err
}

// NewAuthzClient creates a new authz client for use with the hub
func NewAuthzClient(hc *HubClient) *ManageAuthzClient {
	authClient := &ManageAuthzClient{
		hc:        hc,
		serviceID: "authz",
	}
	return authClient

}
