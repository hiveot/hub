package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"os"
	"path"
	"sync"
)

// The DigitwinService orchestrates the flow of properties, events and actions
// between Thing agents and consumers.
// It stores digital twin things, property values and the latest event and action.
type DigitwinService struct {
	// underlying store for the digital twin objects
	bucketStore buckets.IBucketStore
	DtwStore    *store.DigitwinStore

	// Directory service for reading and updating TDs
	DirSvc *DirectoryService

	// service for reading latest values
	ValuesSvc *ValuesService

	mux sync.RWMutex
}

// ReadAllTDs returns a list digitwin TDs
func (svc *DigitwinService) ReadAllTDs(
	clientID string, offset int, limit int) ([]*td.TD, error) {
	dtlist, err := svc.DtwStore.ReadTDs(offset, limit)
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
func (svc *DigitwinService) SetFormsHook(addFormsHandler func(*td.TD) error) {
	svc.DirSvc.addFormsHandler = addFormsHandler
}

// Stop the service
func (svc *DigitwinService) Stop() {
	slog.Info("Stopping DigitwinService")
	svc.DtwStore.Close()
	svc.bucketStore.Close()
}

// StartDigitwinService creates and start the digitwin services.
// This creates a bucket store for the directory, inbox, and outbox.
//
// storesDir is the directory where to create the digitwin storage
// cm is the connection manager used to send messages to clients
func StartDigitwinService(storesDir string, cm *connections.ConnectionManager) (
	svc *DigitwinService, digitwinStore *store.DigitwinStore, err error) {

	sPath := path.Join(storesDir, "digitwin")
	err = os.MkdirAll(sPath, 0700)
	storePath := path.Join(sPath, "digitwinStore")

	bucketStore := kvbtree.NewKVStore(storePath)
	err = bucketStore.Open()
	if err == nil {
		digitwinStore, err = store.OpenDigitwinStore(bucketStore, false)
	}
	dirSvc := NewDigitwinDirectoryService(digitwinStore, cm)
	valuesSvc := NewDigitwinValuesService(digitwinStore)
	if err == nil {
		svc = &DigitwinService{
			bucketStore: bucketStore,
			DtwStore:    digitwinStore,
			mux:         sync.RWMutex{},
			DirSvc:      dirSvc,
			ValuesSvc:   valuesSvc,
		}
		slog.Info("Started DigitwinService")
	}
	return svc, digitwinStore, err
}
