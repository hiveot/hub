package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
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
	propsStore *LatestPropertiesStore
	// handling of events retention
	retentionMgr *ManageHistory
	// Instance ID of this service
	readHistSvc *ReadHistoryService

	serviceID string
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
// This will open the store and panic if the store cannot be opened.
func (svc *HistoryService) Start(hc hubclient.IHubClient) (err error) {
	slog.Warn("Starting HistoryService", "clientID", hc.ClientID())

	// setup
	svc.hc = hc
	svc.serviceID = hc.ClientID()
	svc.retentionMgr = NewManageHistory(hc, nil)

	propsbucket := svc.bucketStore.GetBucket(PropertiesBucketName)
	svc.propsStore = NewPropertiesStore(propsbucket)

	err = svc.retentionMgr.Start()

	svc.readHistSvc, err = StartReadHistoryService(
		svc.hc, svc.bucketStore, svc.propsStore.GetProperties)
	//if err == nil {
	//	svc.updateHistSvc, err = StartUpdateHistoryService(svc.hc, tdBucket)
	//}

	// Set the required permissions for using this service
	// any user roles can view the directory
	myProfile := NewProfileClient(svc.hc)
	err = myProfile.SetServicePermissions(historyapi.ReadHistoryCap, []string{
		api.ClientRoleViewer,
		api.ClientRoleOperator,
		api.ClientRoleManager,
		api.ClientRoleAdmin,
		api.ClientRoleService})
	if err == nil {
		// only admin role can manage the history
		err = myProfile.SetServicePermissions(historyapi.ManageHistoryThingID, []string{api.ClientRoleAdmin})
	}

	// subscribe to events to add to the history store
	if err == nil && svc.hc != nil {
		// the onAddedValue callback is used to update the 'latest' properties
		svc.addHistory = NewAddHistory(
			svc.bucketStore, svc.retentionMgr, svc.propsStore.HandleAddValue)

		// add events to the history filtered through the retention manager
		err = svc.hc.SubEvents("", "", "")
		svc.hc.SetEventHandler(func(msg *things.ThingMessage) {
			slog.Debug("received event",
				slog.String("senderID", msg.SenderID),
				slog.String("thingID", msg.ThingID),
				slog.String("key", msg.Key),
				slog.Int64("createdMSec", msg.CreatedMSec))
			_ = svc.addHistory.AddMessage(msg)
		})
		// TODO: capture all actions
		//svc.hc.Subscribe("","","","")
		//	slog.Debug("received event",
		//		slog.String("agentID", msg.AgentID),
		//		slog.String("thingID", msg.ThingID),
		//		slog.String("name", msg.Name),
		//		slog.Int64("createdMSec", msg.CreatedMSec))
		//	_ = svc.addHistory.AddMessage(msg)
		//})

		// add actions to the history, filtered through retention manager
		// FIXME: this needs the ability to subscribe to actions for other agents
		//svc.actionSub, err = svc.hc.SubActions("", "", "",
		//	func(msg *things.ThingMessage) {
		//		slog.Info("received action", slog.String("name", msg.Name))
		//		_ = svc.addHistory.AddAction(msg)
		//	})
	}

	return err
}

// Stop using the history service and release resources
func (svc *HistoryService) Stop() {
	slog.Warn("Stopping HistoryService")
	err := svc.propsStore.SaveChanges()
	if err != nil {
		slog.Error(err.Error())
	}
	if svc.readHistSvc != nil {
		svc.readHistSvc.Stop()
		svc.readHistSvc = nil
	}
	if svc.retentionMgr != nil {
		svc.retentionMgr.Stop()
		svc.retentionMgr = nil
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
		propsStore:  nil,
	}
	return svc
}