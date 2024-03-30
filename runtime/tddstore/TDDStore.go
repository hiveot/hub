package tddstore

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"sync"
)

const TDBucketName = "td"

// TDDStore manages storage of TDD's (thing description documents) using one of the bucket stores as backend.
type TDDStore struct {
	store        buckets.IBucketStore
	agentID      string // thingID of the service instance
	tdBucketName string
	tdBucket     buckets.IBucket

	// capabilities and subscriptions
	readDirSvc *ReadTDDStore
	//readSub      hubclient.ISubscription
	updateDirSvc *UpdateTDDStore
	//updateSub    hubclient.ISubscription
}

//// handleTDEvent stores a received Thing TD document
//func (svc *TDDStore) handleTDEvent(event *things.ThingValue) {
//	args := directoryapi.UpdateTDArgs{
//		AgentID: event.AgentID,
//		ThingID: event.ThingID,
//		TDDoc:   event.Data,
//	}
//	ctx := hubclient.ServiceContext{SenderID: event.AgentID}
//	err := svc.updateDirSvc.UpdateTD(ctx, args)
//	if err != nil {
//		slog.Error("handleTDEvent failed", "err", err)
//	}
//}

// Start the directory service and open the directory stored TD bucket
func (svc *TDDStore) Start() (err error) {
	slog.Warn("Starting TDDStore")
	// listen for requests
	tdBucket := svc.store.GetBucket(svc.tdBucketName)
	svc.tdBucket = tdBucket

	svc.readDirSvc = StartReadTDDStore(tdBucket)
	svc.updateDirSvc = StartUpdateTDDStore(tdBucket)

	return err
}

// Stop the service
func (svc *TDDStore) Stop() {
	slog.Warn("Stopping TDDStore")
	if svc.updateDirSvc != nil {
		svc.updateDirSvc.Stop()
	}
	if svc.readDirSvc != nil {
		svc.readDirSvc.Stop()
	}
	if svc.tdBucket != nil {
		_ = svc.tdBucket.Close()
	}
}

func NewTDDStore(storage buckets.IBucket) *TDDStore {

	svc := &TDDStore{
		store:         storage,
		cache:         make(map[string]things.ThingValueMap),
		cacheMux:      sync.RWMutex{},
		changedThings: make(map[string]bool),
	}
	return svc
}
