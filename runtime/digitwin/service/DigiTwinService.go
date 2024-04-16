package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"log/slog"
)

//const DefaultDigiTwinStoreFilename = "digitwin.kvbtree"

// DigiTwinService implements digital twin services for storing and reading the thing directory
// and its values.
type DigiTwinService struct {
	// digitwin data store
	dtwBucketStore buckets.IBucketStore

	// DirectoryService holding modified TD documents
	Directory *DirectoryService
	// ValueService holding most recent values of properties, events and queued actions
	Values *ValueService
	// TODO: HistoryStore holding historical values
	//History *HistoryService
}

// Start the service
func (svc *DigiTwinService) Start() (err error) {
	slog.Info("Starting DigiTwinService")
	err = svc.Directory.Start()
	if err != nil {
		return err
	}
	err = svc.Values.Start()
	return err
}

// Stop the service
func (svc *DigiTwinService) Stop() {
	slog.Info("Stopping DigiTwinService")
	svc.Directory.Stop()
	svc.Values.Stop()
}

// NewDigiTwinService creates a new instance of the digital twin service
//
//	dtwBucketStore is an opened store used to persist directory and value buckets
func NewDigiTwinService(dtwBucketStore buckets.IBucketStore) *DigiTwinService {
	svc := DigiTwinService{
		dtwBucketStore: dtwBucketStore,
		Directory:      NewDirectoryStore(dtwBucketStore),
		Values:         NewThingValueStore(dtwBucketStore),
	}
	return &svc
}
