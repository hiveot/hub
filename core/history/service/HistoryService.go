package service

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/pkg/bucketstore"
	"github.com/hiveot/hub/pkg/history"
	"github.com/hiveot/hub/pkg/history/config"
	"github.com/hiveot/hub/pkg/pubsub"
)

const PropertiesBucketName = "properties"

// HistoryService provides storage for action and event history using the bucket store
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService interface
type HistoryService struct {

	// The history service bucket store with a bucket for each Thing
	bucketStore bucketstore.IBucketStore
	// Storage of the latest properties of a thing
	propsStore *LastPropertiesStore
	// handling of retention of pubsub events
	retentionMgr *ManageRetention
	// Instance ID of this service
	serviceID string
	// the pubsub service to subscribe to event
	servicePubSub pubsub.IServicePubSub
	// optional handling of pubsub events. nil if not used
	subEventHandler *PubSubEventHandler
}

// CapAddHistory provides the capability to add to the history of any Thing.
// This capability should only be provided to trusted services that capture events from multiple sources
// and can verify their authenticity.
func (svc *HistoryService) CapAddHistory(
	_ context.Context, clientID string, ignoreRetention bool) (history.IAddHistory, error) {

	logrus.Infof("clientID=%s", clientID)

	var retentionMgr *ManageRetention
	if !ignoreRetention {
		retentionMgr = svc.retentionMgr
	}

	historyUpdater := NewAddHistory(
		clientID, svc.bucketStore, retentionMgr, svc.propsStore.HandleAddValue)
	return historyUpdater, nil
}

// CapManageRetention returns the capability to manage the retention of events
func (svc *HistoryService) CapManageRetention(
	_ context.Context, clientID string) (history.IManageRetention, error) {

	logrus.Infof("clientID=%s", clientID)
	evRet := svc.retentionMgr
	_ = clientID
	return evRet, nil
}

// CapReadHistory provides the capability to read history from a publisher Thing
//
//	clientID of the ID remote client doing the reading
//	publisherID of the publisher whose Thing to read
//	thingID of the Thing whose history to read
func (svc *HistoryService) CapReadHistory(_ context.Context, clientID string) (
	history.IReadHistory, error) {

	logrus.Infof("clientID=%s", clientID)
	//thingAddr := publisherID + "/" + thingID
	//bucket := svc.bucketStore.GetBucket(thingAddr)
	readHistory := NewReadHistory(clientID, svc.bucketStore, svc.propsStore.GetProperties)
	return readHistory, nil
}

// Start using the history service
// This will open the store and panic if the store cannot be opened.
func (svc *HistoryService) Start() (err error) {
	logrus.Infof("")
	propsbucket := svc.bucketStore.GetBucket(PropertiesBucketName)
	svc.propsStore = NewPropertiesStore(propsbucket)

	err = svc.retentionMgr.Start()

	// subscribe to events to add history
	if err == nil && svc.servicePubSub != nil {
		capAddEvent := NewAddHistory(svc.serviceID, svc.bucketStore, svc.retentionMgr, svc.propsStore.HandleAddValue)
		svc.subEventHandler = NewSubEventHandler(svc.servicePubSub, capAddEvent)
		err = svc.subEventHandler.Start()
	}

	return err
}

// Stop using the history service and release resources
func (svc *HistoryService) Stop() error {
	logrus.Infof("")
	err := svc.propsStore.SaveChanges()
	if err != nil {
		logrus.Error(err)
	}
	svc.retentionMgr.Stop()
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
//	sub optional pubsub client used to subscribe to events to store. nil to not subscribe to events. Will be released on Stop().
func NewHistoryService(
	config *config.HistoryConfig, store bucketstore.IBucketStore, sub pubsub.IServicePubSub) *HistoryService {

	var retentionMgr *ManageRetention
	serviceID := history.ServiceName
	if config != nil && config.ServiceID == "" {
		config.ServiceID = history.ServiceName
	}
	if config != nil {
		retentionMgr = NewManageRetention(config.Retention)
	} else {
		retentionMgr = NewManageRetention(nil)
	}
	svc := &HistoryService{
		bucketStore:   store,
		propsStore:    nil,
		serviceID:     serviceID,
		retentionMgr:  retentionMgr,
		servicePubSub: sub,
	}
	return svc
}
