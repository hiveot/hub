package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/runtime/api"
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

	mux sync.RWMutex
	// The transport binding (manager) to communicate with agents and consumers
	tb api.ITransportBinding
}

// ReadAction returns the last known action invocation status of the given name
func (svc *DigitwinService) ReadAction(
	consumerID string, dThingID string, actionName string) (v digitwin.DigitalTwinActionValue, err error) {

	return svc.dtwStore.ReadAction(dThingID, actionName)
}

// ReadAllActions returns the map of the latest actions on the thing
func (svc *DigitwinService) ReadAllActions(
	consumerID string, dThingID string) (map[string]digitwin.DigitalTwinActionValue, error) {

	return svc.dtwStore.ReadAllActions(dThingID)
}

// ReadAllEvents returns a list of known digitwin instance event values
func (svc *DigitwinService) ReadAllEvents(
	consumerID string, dThingID string) (map[string]digitwin.DigitalTwinEventValue, error) {
	return svc.dtwStore.ReadAllEvents(dThingID)
}

// ReadAllProperties returns a map of known digitwin instance property values
func (svc *DigitwinService) ReadAllProperties(
	consumerID string, dThingID string) (map[string]digitwin.DigitalTwinPropertyValue, error) {

	return svc.dtwStore.ReadAllProperties(dThingID)
}

// ReadAllThings returns a list digitwin Things
func (svc *DigitwinService) ReadAllThings(
	consumerID string, offset int, limit int) ([]tdd.TD, error) {
	dtlist, err := svc.dtwStore.ReadDThingList(offset, limit)
	return dtlist, err
}

// ReadEvent returns the latest event of a digitwin instance
func (svc *DigitwinService) ReadEvent(
	consumerID string, dThingID string, name string) (digitwin.DigitalTwinEventValue, error) {

	return svc.dtwStore.ReadEvent(dThingID, name)
}

// ReadProperty returns the last known value of a thing property
func (svc *DigitwinService) ReadProperty(
	consumerID string, dThingID string, name string) (p digitwin.DigitalTwinPropertyValue, err error) {

	return svc.dtwStore.ReadProperty(dThingID, name)
}

// ReadThing returns the digitwin TD of a Thing
func (svc *DigitwinService) ReadThing(
	consumerID string, dThingID string) (tdd.TD, error) {
	dtd, err := svc.dtwStore.ReadDThing(dThingID)
	return dtd, err
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
	if err == nil {
		svc = &DigitwinService{
			bucketStore: bucketStore,
			dtwStore:    dtwStore,
			mux:         sync.RWMutex{},
		}
		slog.Info("Started DigitwinService")
	}
	return svc, dtwStore, err
}
