package service

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
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
	dtwStore    *DigitwinStore

	// Directory service for reading and updating TDs
	DirSvc *DigitwinDirectoryService

	mux sync.RWMutex
	//// The transport binding (manager) to communicate with agents and consumers
	//tb api.ITransportBinding
}

// ReadAction returns the last known action invocation status of the given name
func (svc *DigitwinService) ReadAction(
	consumerID string, dThingID string, actionName string) (v digitwin.ActionValue, err error) {

	return svc.dtwStore.ReadAction(dThingID, actionName)
}

// ReadAllActions returns the map of the latest actions on the thing
func (svc *DigitwinService) ReadAllActions(
	consumerID string, dThingID string) (map[string]digitwin.ActionValue, error) {

	return svc.dtwStore.ReadAllActions(dThingID)
}

// ReadAllEvents returns a list of known digitwin instance event values
func (svc *DigitwinService) ReadAllEvents(
	consumerID string, dThingID string) (map[string]digitwin.EventValue, error) {
	return svc.dtwStore.ReadAllEvents(dThingID)
}

// ReadAllProperties returns a map of known digitwin instance property values
func (svc *DigitwinService) ReadAllProperties(
	consumerID string, dThingID string) (map[string]digitwin.PropertyValue, error) {

	return svc.dtwStore.ReadAllProperties(dThingID)
}

// ReadAllDTDs returns a list digitwin TDs
func (svc *DigitwinService) ReadAllDTDs(
	consumerID string, offset int, limit int) ([]*tdd.TD, error) {
	dtlist, err := svc.dtwStore.ReadDTDs(offset, limit)
	return dtlist, err
}

// ReadEvent returns the latest event of a digitwin instance
func (svc *DigitwinService) ReadEvent(
	consumerID string, dThingID string, name string) (digitwin.EventValue, error) {

	return svc.dtwStore.ReadEvent(dThingID, name)
}

// ReadProperty returns the last known property value of the given name,
// or an empty value if no value is known.
// This returns an error if the dThingID doesn't exist.
func (svc *DigitwinService) ReadProperty(
	consumerID string, dThingID string, name string) (p digitwin.PropertyValue, err error) {

	return svc.dtwStore.ReadProperty(dThingID, name)
}

//// ReadThing returns the digitwin TD of a Thing
//func (svc *DigitwinService) ReadThing(
//	consumerID string, dThingID string) (tdd.TD, error) {
//	dtd, err := svc.dtwStore.ReadDThing(dThingID)
//	return dtd, err
//}

// SetFormsHook sets the hook at add transport forms
func (svc *DigitwinService) SetFormsHook(addForms func(td *tdd.TD) error) {
	svc.DirSvc.addTDForms = addForms
}

// Stop the service
func (svc *DigitwinService) Stop() {
	slog.Info("Stopping DigitwinService")
	svc.dtwStore.Close()
	svc.bucketStore.Close()
}

// StartDigitwinService creates and start the digitwin administration service.
// This creates a bucket store for the directory, inbox, and outbox.
//
// Use SetFormsHook to set the outgoing transport protocol handler for use
// by this service.
//
// storesDir is the directory where to create the digitwin storage
// tb is the protocol binding or manager used to send messages to clients
func StartDigitwinService(storesDir string) (
	svc *DigitwinService, store *DigitwinStore, err error) {

	sPath := path.Join(storesDir, "digitwin")
	err = os.MkdirAll(sPath, 0700)
	storePath := path.Join(sPath, "digitwin.store")

	bucketStore := kvbtree.NewKVStore(storePath)
	var dtwStore *DigitwinStore
	err = bucketStore.Open()
	if err == nil {
		dtwStore, err = OpenDigitwinStore(bucketStore)
	}
	dirSvc := NewDigitwinDirectoryService(dtwStore, nil)
	if err == nil {
		svc = &DigitwinService{
			bucketStore: bucketStore,
			dtwStore:    dtwStore,
			mux:         sync.RWMutex{},
			DirSvc:      dirSvc,
		}
		slog.Info("Started DigitwinService")
	}
	return svc, dtwStore, err
}
