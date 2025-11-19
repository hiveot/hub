package service

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/agent"
	"github.com/hiveot/hivekit/go/buckets"
	authz "github.com/hiveot/hub/runtime/authz/api"
	"github.com/hiveot/hub/services/history/historyapi"
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
	// the messaging agent used to pubsub service to subscribe to event
	ag *agent.Agent
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

// Start the history service
func (svc *HistoryService) Start(ag *agent.Agent) (err error) {

	slog.Info("Starting HistoryService", "clientID", ag.GetClientID())

	// setup
	svc.ag = ag
	svc.agentID = ag.GetClientID()
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
	permissions := authz.ThingPermissions{
		AgentID: ag.GetClientID(),
		ThingID: historyapi.ReadHistoryServiceID,
		Deny:    []authz.ClientRole{authz.ClientRoleNone},
	}
	err = authz.UserSetPermissions(ag.Consumer, permissions)

	//if err == nil {
	//	// only admin role can manage the history
	//	err = myProfile.SetServicePermissions(historyapi.ManageHistoryThingID, []string{api.ClientRoleAdmin})
	//}

	// subscribe to events to add to the history store
	if err == nil && svc.ag != nil {

		// handler of adding events to the history
		svc.addHistory = NewAddHistory(svc.bucketStore, svc.manageHistSvc)

		// register the history service methods and listen for requests
		StartHistoryAgent(svc, svc.ag)

		// TODO: add actions to the history, filtered through retention manager
		// subscribe to receive the events to add to the history, filtered through the retention manager
		err = svc.ag.Subscribe("", "")
		err = svc.ag.ObserveProperty("", "")
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
	_ = svc.bucketStore.Close()
}

// NewHistoryService creates a new instance for the history service using the given
// storage bucket.
//
//	config optional configuration or nil to use defaults
//	store contains an opened bucket store to use. This will be closed on Stop.
//	hc connection with the hub
func NewHistoryService(store buckets.IBucketStore) *HistoryService {

	svc := &HistoryService{
		bucketStore: store,
	}
	return svc
}
