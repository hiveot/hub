package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/runtime/sessions"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"os"
	"path"
	"sync"
)

const DTWBucketName = "dtw"

// The DigitwinService orchestrates the flow of properties, events and actions
// between Thing agents and consumers.
// It stores digital twin things, property values and the latest event and action.
type DigitwinService struct {
	// underlying store for the digital twin objects
	bucketStore buckets.IBucketStore
	DtwStore    *DigitwinStore

	// Directory service for reading and updating TDs
	DirSvc *DigitwinDirectoryService

	// service for reading latest values
	ValuesSvc *DigitwinValuesService

	mux sync.RWMutex
}

// ReadAllDTDs returns a list digitwin TDs
func (svc *DigitwinService) ReadAllDTDs(
	consumerID string, offset int, limit int) ([]*tdd.TD, error) {
	dtlist, err := svc.DtwStore.ReadDTDs(offset, limit)
	return dtlist, err
}

//// ReadThing returns the digitwin TD of a Thing
//func (svc *DigitwinService) ReadThing(
//	consumerID string, dThingID string) (tdd.TD, error) {
//	dtd, err := svc.DtwStore.ReadDThing(dThingID)
//	return dtd, err
//}

// SetFormsHook sets the transport hook for reading forms and publishing
// service events.
func (svc *DigitwinService) SetFormsHook(addFormsHandler func(*tdd.TD) error) {
	svc.DirSvc.addFormsHandler = addFormsHandler
}

// Stop the service
func (svc *DigitwinService) Stop() {
	slog.Info("Stopping DigitwinService")
	svc.DtwStore.Close()
	svc.bucketStore.Close()
}

// StartDigitwinService creates and start the digitwin administration service.
// This creates a bucket store for the directory, inbox, and outbox.
//
// Use SetFormsHook to set the outgoing transport protocol handler for use
// by this service.
//
// storesDir is the directory where to create the digitwin storage
// cm is the connection manager used to send messages to clients
func StartDigitwinService(storesDir string, cm *sessions.ConnectionManager) (
	svc *DigitwinService, store *DigitwinStore, err error) {

	sPath := path.Join(storesDir, "digitwin")
	err = os.MkdirAll(sPath, 0700)
	storePath := path.Join(sPath, "digitwin.store")

	bucketStore := kvbtree.NewKVStore(storePath)
	var dtwStore *DigitwinStore
	err = bucketStore.Open()
	if err == nil {
		dtwStore, err = OpenDigitwinStore(bucketStore, false)
	}
	dirSvc := NewDigitwinDirectoryService(dtwStore, cm)
	valuesSvc := NewDigitwinValuesService(dtwStore)
	if err == nil {
		svc = &DigitwinService{
			bucketStore: bucketStore,
			DtwStore:    dtwStore,
			mux:         sync.RWMutex{},
			DirSvc:      dirSvc,
			ValuesSvc:   valuesSvc,
		}
		slog.Info("Started DigitwinService")
	}
	return svc, dtwStore, err
}
