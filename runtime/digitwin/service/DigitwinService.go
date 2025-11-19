package service

import (
	"log/slog"
	"os"
	"path"
	"sync"

	"github.com/hiveot/hivekit/go/buckets"
	"github.com/hiveot/hivekit/go/buckets/kvbtree"
	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/runtime/digitwin/store"
)

// The DigitwinService stores digital twin things, property values and provide
// the latest event and action values.
type DigitwinService struct {
	// persistent store for the digital twin objects
	bucketStore buckets.IBucketStore

	// in-memory store with digital twin instances
	DtwStore *store.DigitwinStore

	// Directory service for reading and updating TDs
	DirSvc *DirectoryService

	// service for reading latest values
	ValuesSvc *ValuesService

	mux sync.RWMutex
}

// ReadAllTDs returns a list of digital twin TDs in the store
func (svc *DigitwinService) ReadAllTDs(
	clientID string, offset int64, limit int64) ([]*td.TD, error) {
	dtlist, err := svc.DtwStore.ReadTDs(offset, limit)
	return dtlist, err
}

//// ReadThing returns the digitwin TD of a Thing
//func (svc *DigitwinService) ReadThing(
//	consumerID string, dThingID string) (tdd.TD, error) {
//	dtd, err := svc.DtwStore.ReadDThing(dThingID)
//	return dtd, err
//}

// SetFormsHook sets the transport hook for adding forms and securityScheme entries to TDs
func (svc *DigitwinService) SetFormsHook(addFormsHandler func(*td.TD, bool)) {
	svc.DirSvc.addFormsHandler = addFormsHandler
}

// Stop the service
func (svc *DigitwinService) Stop() {
	slog.Info("Stopping DigitwinService")
	svc.DtwStore.Close()
	svc.bucketStore.Close()
}

// Start starts the digitwin services.
// This creates a bucket store for the directory, inbox, and outbox.
//
// storesDir is the directory where to create the digitwin storage
// notifHandler is notifies of changes to digital twin state.
// includeAffordanceForms to include forms for affordances in digital twin TDs.
func (svc *DigitwinService) Start(
	storesDir string, notifHandler messaging.NotificationHandler, includeAffordanceForms bool) (
	digitwinStore *store.DigitwinStore, err error) {

	sPath := path.Join(storesDir, "digitwin")
	err = os.MkdirAll(sPath, 0700)
	storePath := path.Join(sPath, "digitwinStore")

	bucketStore := kvbtree.NewKVStore(storePath)
	err = bucketStore.Open()
	if err != nil {
		slog.Error("Unable to open digital twin storage bucket", "err", err.Error())
		return nil, err
	}

	digitwinStore, err = store.OpenDigitwinStore(bucketStore, false)
	if err != nil {
		slog.Error("Unable to open digital twin store itself", "err", err.Error())
		return nil, err
	}
	svc.bucketStore = bucketStore
	svc.DtwStore = digitwinStore
	svc.DirSvc = NewDigitwinDirectoryService(digitwinStore, notifHandler, includeAffordanceForms)
	svc.ValuesSvc = NewDigitwinValuesService(digitwinStore)
	return digitwinStore, err
}

// StartDigitwinService creates and start the digitwin services.
// This creates a bucket store for the directory, inbox, and outbox.
//
// storesDir is the directory where to create the digitwin storage
// notifHandler is notifies of changes to digital twin state.
// includeAffordanceForms to include forms for affordances in digital twin TDs.
func StartDigitwinService(
	storesDir string, notifHandler messaging.NotificationHandler, includeAffordanceForms bool) (
	svc *DigitwinService, digitwinStore *store.DigitwinStore, err error) {

	svc = &DigitwinService{}
	digitwinStore, err = svc.Start(storesDir, notifHandler, includeAffordanceForms)
	return svc, digitwinStore, err
}
