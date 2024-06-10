package service

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/history/historyapi"
	"log/slog"
)

const PropertiesBucketName = "properties"

// HistoryService provides storage for action and event history using the bucket store
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService interface
type HistoryService struct {

	// The history service bucket store with a bucket for each Thing
	bucketStore buckets.IBucketStore
	// Storage of the latest properties of a things
	//propsStore *LatestPropertiesStore
	// the manage history sub-service
	manageHistSvc *ManageHistory
	// the read-history sub-service
	readHistSvc *ReadHistory

	agentID string
	// the pubsub service to subscribe to event
	hc hubclient.IHubClient
	// optional handling of pubsub events. nil if not used
	//subEventHandler *PubSubEventHandler
	// handler that adds history to the store
	addHistory *AddHistory
}

// GetAddHistory returns the handler for adding history.
// Intended for testing.
func (svc *HistoryService) GetAddHistory() *AddHistory {
	return svc.addHistory
}

// Start using the history service
func (svc *HistoryService) Start(hc hubclient.IHubClient) (err error) {
	slog.Info("Starting HistoryService", "clientID", hc.ClientID())

	// setup
	svc.hc = hc
	svc.agentID = hc.ClientID()
	svc.manageHistSvc = NewManageHistory(nil)
	err = svc.manageHistSvc.Start()
	if err == nil {
		svc.readHistSvc = NewReadHistory(svc.bucketStore)
		err = svc.readHistSvc.Start()
	}
	if err != nil {
		return err
	}

	// Set the required permissions for using this service
	// any user roles can view the history
	err = authz.UserSetPermissions(hc, authz.ThingPermissions{
		AgentID: hc.ClientID(),
		ThingID: historyapi.ReadHistoryServiceID,
		Deny:    []string{authn.ClientRoleNone},
	})

	//if err == nil {
	//	// only admin role can manage the history
	//	err = myProfile.SetServicePermissions(historyapi.ManageHistoryThingID, []string{api.ClientRoleAdmin})
	//}

	// subscribe to events to add to the history store
	if err == nil && svc.hc != nil {
		// the onAddedValue callback is used to update the 'latest' properties
		svc.addHistory = NewAddHistory(svc.bucketStore, svc.manageHistSvc, nil)

		// add events to the history filtered through the retention manager
		err = svc.hc.Subscribe("", "")
		svc.hc.SetEventHandler(func(msg *things.ThingMessage) (err error) {
			slog.Debug("received event",
				slog.String("senderID", msg.SenderID),
				slog.String("thingID", msg.ThingID),
				slog.String("key", msg.Key),
				slog.Int64("createdMSec", msg.CreatedMSec))
			err = svc.addHistory.AddEvent(msg)
			return err
		})

		// register the history service methods
		StartHistoryAgent(svc, svc.hc)

		// add actions to the history, filtered through retention manager
		// FIXME: this needs the ability to subscribe to actions for other agents

		//svc.actionSub, err = svc.hc.SubActions("", "", "",
		//	func(msg *things.ThingMessage) {
		//		slog.Info("received action", slog.String("name", msg.Name))
		//		_ = svc.addHistory.AddAction(msg)
		//	})

		// TODO: add action history. Note that the agent also subscribes to actions,
		// so if we subscribe here we must invoke the agent from here

		err = svc.hc.Subscribe("", "")
	}

	return err
}

// Stop using the history service and release resources
func (svc *HistoryService) Stop() {
	slog.Info("Stopping HistoryService")
	if svc.readHistSvc != nil {
		svc.readHistSvc.Stop()
		svc.readHistSvc = nil
	}
	if svc.manageHistSvc != nil {
		svc.manageHistSvc.Stop()
		svc.manageHistSvc = nil
	}
}

// NewHistoryService creates a new instance for the history service using the given
// storage bucket.
//
//	config optional configuration or nil to use defaults
//	store contains an opened bucket store to use.
//	hc connection with the hub
func NewHistoryService(store buckets.IBucketStore) *HistoryService {

	svc := &HistoryService{
		bucketStore: store,
	}
	return svc
}
