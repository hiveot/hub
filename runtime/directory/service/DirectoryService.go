package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
)

const TDBucketName = "td"

// DirectoryService is a wrapper around the internal store store
// This implements the IDirectory interface
type DirectoryService struct {
	store        buckets.IBucketStore
	agentID      string // thingID of the service instance
	tdBucketName string
	tdBucket     buckets.IBucket

	// capabilities and subscriptions
	readDirSvc *ReadDirectoryService
	//readSub      hubclient.ISubscription
	updateDirSvc *UpdateDirectoryService
	//updateSub    hubclient.ISubscription
}

//// handleTDEvent stores a received Thing TD document
//func (svc *DirectoryService) handleTDEvent(event *things.ThingValue) {
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
func (svc *DirectoryService) Start() (err error) {
	slog.Warn("Starting DirectoryService")
	// listen for requests
	tdBucket := svc.store.GetBucket(svc.tdBucketName)
	svc.tdBucket = tdBucket

	svc.readDirSvc = StartReadDirectoryService(tdBucket)
	svc.updateDirSvc = StartUpdateDirectoryService(tdBucket)

	return err
}

// Stop the service
func (svc *DirectoryService) Stop() {
	slog.Warn("Stopping DirectoryService")
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

// NewDirectoryService creates a new service instance for the directory of Thing TD documents.
//
//	store is an instance of the bucket store to store the directory data. This is opened by 'Start' and closed by 'Stop'
func NewDirectoryService(store buckets.IBucketStore) *DirectoryService {
	svc := &DirectoryService{
		store:        store,
		tdBucketName: TDBucketName,
	}
	return svc
}
