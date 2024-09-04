package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"strings"
	"sync"
)

const TDBucketName = "td"

// DigitwinDirectoryService the Digitwin Directory stores, reads and queries TD documents
// This service keeps an in-memory Thing Description instance backed by a persistent bucket store.
//
// When a TDD is received, its ID is replaced by that of the digitwin thingID and its form entries
// are replaced to point to the Hub protocols, after which the new TDD is persisted in the bucket
// store.
//
// Read and query methods using the in-memory cache for fast performance.
type DigitwinDirectoryService struct {
	store    buckets.IBucketStore
	tdBucket buckets.IBucket
	tb       api.ITransportBinding

	// tdCache holds an in-memory cache version of stored TDs
	//tdCache map[string]*things.TD
	tdCache map[string]string
	// list of DTW thingIDs used as a consistent iterator for reading batches
	thingKeys []string
	// mutex for accessing the cache
	cachemux sync.RWMutex
}

// HandleTDEvent receives and stores a TD from an IoT agent or service after upgrading it
// to the digital twin version including Forms for protocol bindings.
//
//	msg is the thing message containing the JSON encoded TD.
//	tb is the transport binding whose protocols to add to the td, or nil when no protocols to add
func (svc *DigitwinDirectoryService) HandleTDEvent(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	err := svc.UpdateTD(msg.SenderID, msg.DataAsText())
	stat.Completed(msg, nil, err)
	return stat
}

// LoadCacheFromStore loads the cache from store into memory
func (svc *DigitwinDirectoryService) LoadCacheFromStore() error {
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	cursor, err := svc.tdBucket.Cursor()
	if err != nil {
		return err
	}
	svc.thingKeys = make([]string, 0, 1000)
	svc.tdCache = make(map[string]string)
	//k, v, valid := cursor.First()
	//_ = k
	//_ = v
	//_ = valid
	for {
		// read in batches of 300 TD documents
		tdmap, itemsRemaining := cursor.NextN(300)
		for thingID, tddjson := range tdmap {
			svc.thingKeys = append(svc.thingKeys, thingID)
			svc.tdCache[thingID] = string(tddjson)
		}
		if !itemsRemaining {
			break
		}
	}
	return nil
}

// QueryTDs query the collection of TD documents
func (svc *DigitwinDirectoryService) QueryTDs(senderID string,
	args digitwin.DirectoryQueryTDsArgs) (resp []string, err error) {

	// TBD: query based on what?
	return resp, fmt.Errorf("not yet implemented")
}

// ReadTD returns the TD document in json format for the given Thing ID
func (svc *DigitwinDirectoryService) ReadTD(
	senderID string, thingID string) (resp string, err error) {

	svc.cachemux.RLock()
	defer svc.cachemux.RUnlock()
	tddjson, found := svc.tdCache[thingID]
	if !found {
		err = fmt.Errorf("Thing with ID '%s' not found", thingID)
		return resp, err
	}
	return tddjson, err
}

// ReadTDs returns a list of TD documents
//
// args: offset is the offset in the list
//
//	limit is the maximum number of records to return
func (svc *DigitwinDirectoryService) ReadTDs(senderID string,
	args digitwin.DirectoryReadTDsArgs) (resp []string, err error) {

	tdList := make([]string, 0, args.Limit)
	svc.cachemux.RLock()
	defer svc.cachemux.RUnlock()
	// Use the thingKeys index to ensure consistent iteration and to quickly
	// skip offset items (maps are not consistent between iterations)
	if args.Offset >= len(svc.thingKeys) {
		// empty result
		return tdList, nil
	}
	if args.Offset+args.Limit > len(svc.thingKeys) {
		args.Limit = len(svc.thingKeys) - args.Offset
	}
	tdKeys := svc.thingKeys[args.Offset:args.Limit]
	// add the TD documents
	for _, k := range tdKeys {
		tddjson := svc.tdCache[k]
		tdList = append(tdList, string(tddjson))
	}
	return tdList, nil
}

// RemoveTD deletes the TD document from the given agent with the ThingID
func (svc *DigitwinDirectoryService) RemoveTD(senderID string, thingID string) error {

	slog.Info("RemoveTD",
		slog.String("thingID", thingID),
		slog.String("senderID", senderID))
	// remove from both cache and bucket
	err := svc.tdBucket.Delete(thingID)
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	delete(svc.tdCache, thingID)
	// delete from the index array. A bit primitive but it works
	for i, key := range svc.thingKeys {
		if key == thingID {
			svc.thingKeys[i] = svc.thingKeys[len(svc.thingKeys)-1]
			svc.thingKeys = svc.thingKeys[:len(svc.thingKeys)-1]
			break
		}
	}
	return err
}

// Start the directory service and open the directory stored TD bucket
func (svc *DigitwinDirectoryService) Start() (err error) {
	slog.Info("Starting DigitwinDirectoryService")
	// fill the in-memory cache
	err = svc.LoadCacheFromStore()
	if err != nil {
		return err
	}
	return err
}

// Stop the service
func (svc *DigitwinDirectoryService) Stop() {
	slog.Info("Stopping DigitwinDirectoryService")
	if svc.tdBucket != nil {
		_ = svc.tdBucket.Close()
		svc.tdBucket = nil
	}
}

// UpdateThing adds or updates the Thing Description document
// intended for internal use, so not a TDD action.
// Added things are written to the store.
func (svc *DigitwinDirectoryService) updateThing(
	senderID string, dtThingID string, tddjson string) error {

	slog.Info("UpdateThing",
		slog.String("senderID", senderID),
		slog.String("dtThingID", dtThingID))

	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	_, exists := svc.tdCache[dtThingID]
	// append the key if it doesn't yet exist
	if !exists {
		svc.thingKeys = append(svc.thingKeys, dtThingID)
	}
	// track the publisher
	svc.tdCache[dtThingID] = tddjson

	// serialize to persist
	err := svc.tdBucket.Set(dtThingID, []byte(tddjson))
	return err
}

// UpdateTD handles the action to update a TD in the directory. This upgrades the TD
// to the digital twin version including Forms for protocol bindings and
// publishes an event with the updated TD.
//
// This also replaces spaces in thingID and keys with dashes
//
//	senderID is the agent and owner of the TD
//	tddjson is the json encoded TD
func (svc *DigitwinDirectoryService) UpdateTD(senderID string, tddjson string) error {
	var err error

	// 1: parse the TD json
	td := tdd.TD{}
	// we know the argument is a string with TD document text. It can be immediately converted to TD object
	err = td.LoadFromJSON(tddjson)

	if err != nil {
		err = errors.New("UpdateTD: Message does not contain the TD in JSON format")
	} else {
		td.ID = strings.ReplaceAll(td.ID, " ", "-")
	}

	if err != nil {
		return err
	}

	// 1: create the digitwin ThingID for this TD
	// events use 'agent' thingIDs, only known to agents.
	// Digitwin adds the "dtw:{agentID}:" prefix, as the event now belongs to the virtual digital twin.
	dtThingID := tdd.MakeDigiTwinThingID(senderID, td.ID)
	td.ID = dtThingID

	// 2: modify the TD to escape all keys as they are used in paths
	td.EscapeKeys()

	// 3: Upgrade the forms with the transports
	if svc.tb != nil {
		svc.tb.AddTDForms(&td)
	}

	// 4: store the digitwin TD in the directory
	dtwTddjson, _ := json.Marshal(&td)
	err = svc.updateThing(senderID, dtThingID, string(dtwTddjson))

	// 5: publish event with updated digitwin TD
	if err == nil && svc.tb != nil {

		ev := hubclient.NewThingMessage(
			vocab.MessageTypeEvent, dtThingID, vocab.EventTypeTD, string(dtwTddjson), senderID)

		svc.tb.SendEvent(ev)
	}
	return err
}

// NewDigitwinDirectory creates a new service instance for the directory of Thing TD documents.
//
//	store is an instance of the bucket store to store the directory data. This is opened by 'Start' and closed by 'Stop'
func NewDigitwinDirectory(store buckets.IBucketStore, tb api.ITransportBinding) *DigitwinDirectoryService {
	tdBucket := store.GetBucket(TDBucketName)
	svc := &DigitwinDirectoryService{
		store:    store,
		tdBucket: tdBucket,
		tb:       tb,
	}
	return svc
}
