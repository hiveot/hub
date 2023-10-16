package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
)

const PropertiesBucketName = "properties"

// HistoryService provides storage for action and event history using the bucket store
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService interface
type HistoryService struct {

	// The history service bucket store with a bucket for each Thing
	bucketStore buckets.IBucketStore
	// Storage of the latest properties of a thing
	propsStore *LatestPropertiesStore
	// handling of events retention
	retentionMgr *HistoryRetention
	// Instance ID of this service
	readHistSvc *ReadHistoryService

	serviceID string
	// the pubsub service to subscribe to event
	hc hubclient.IHubClient
	// optional handling of pubsub events. nil if not used
	subEventHandler *PubSubEventHandler
}

// Start using the history service
// This will open the store and panic if the store cannot be opened.
func (svc *HistoryService) Start() (err error) {
	slog.Info("Start")
	propsbucket := svc.bucketStore.GetBucket(PropertiesBucketName)
	svc.propsStore = NewPropertiesStore(propsbucket)

	//err = svc.retentionMgr.Start()

	svc.readHistSvc, err = StartReadHistoryService(
		svc.hc, svc.bucketStore, svc.propsStore.GetProperties)
	//if err == nil {
	//	svc.updateHistSvc, err = StartUpdateHistoryService(svc.hc, tdBucket)
	//}

	// subscribe to events to add history
	if err == nil && svc.hc != nil {
		capAddEvent := NewAddHistory(
			svc.bucketStore, svc.retentionMgr, svc.propsStore.HandleAddValue)
		svc.subEventHandler = NewSubEventHandler(svc.hc, capAddEvent)
		err = svc.subEventHandler.Start()
	}

	return err
}

// Stop using the history service and release resources
func (svc *HistoryService) Stop() error {
	slog.Info("Stop")
	err := svc.propsStore.SaveChanges()
	if err != nil {
		slog.Error(err.Error())
	}
	//svc.retentionMgr.Stop()
	if svc.subEventHandler != nil {
		svc.subEventHandler.Stop()
	}
	return err
}

// NewHistoryService creates a new instance for the history service using the given
// storage bucket.
//
//	config optional configuration or nil to use defaults
//	store contains an opened bucket store to use.
//	hc connection with the hub
func NewHistoryService(
	hc hubclient.IHubClient, store buckets.IBucketStore) *HistoryService {

	//var retentionMgr *HistoryRetention
	//if config != nil {
	//	retentionMgr = NewManageRetention(config.Retention)
	//} else {
	//	retentionMgr = NewManageRetention(nil)
	//}
	svc := &HistoryService{
		bucketStore: store,
		propsStore:  nil,
		serviceID:   hc.ClientID(),
		//retentionMgr: retentionMgr,
		hc: hc,
	}
	return svc
}
