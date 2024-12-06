package service

import (
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/services/history/historyapi"
	"github.com/hiveot/hub/wot/transports"
	"log/slog"
)

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
	hc transports.IClientConnection
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
func (svc *HistoryService) Start(hc transports.IClientConnection) (err error) {
	slog.Info("Starting HistoryService", "clientID", hc.GetClientID())

	// setup
	svc.hc = hc
	svc.agentID = hc.GetClientID()
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
		AgentID: hc.GetClientID(),
		ThingID: historyapi.ReadHistoryServiceID,
		Deny:    []authz.ClientRole{authz.ClientRoleNone},
	})

	//if err == nil {
	//	// only admin role can manage the history
	//	err = myProfile.SetServicePermissions(historyapi.ManageHistoryThingID, []string{api.ClientRoleAdmin})
	//}

	// subscribe to events to add to the history store
	if err == nil && svc.hc != nil {

		// handler of adding events to the history
		svc.addHistory = NewAddHistory(svc.bucketStore, svc.manageHistSvc)

		// register the history service methods
		StartHistoryAgent(svc, svc.hc)

		// TODO: add actions to the history, filtered through retention manager
		// subscribe to receive the events to add to the history, filtered through the retention manager
		err = svc.hc.Subscribe("", "")
		err = svc.hc.ObserveProperty("", "")
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
