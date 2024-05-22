package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/directory"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"sync"
)

const TDBucketName = "td"

// DigitwinDirectory the Digitwin Directory stores, reads and queries TD documents
// This service keeps an in-memory Thing Description instance backed by a persistent bucket store.
//
// When a TDD is received, its ID is replaced by that of the digitwin thingID and its form entries
// are replaced to point to the Hub protocols, after which the new TDD is persisted in the bucket
// store.
//
// Read and query methods using the in-memory cache for fast performance.
type DigitwinDirectory struct {
	store    buckets.IBucketStore
	tdBucket buckets.IBucket

	// tdCache holds an in-memory cache version of stored TDs
	tdCache map[string]*things.TD
	// list of DTW thingIDs used as a consistent iterator for reading batches
	thingKeys []string
	// mutex for accessing the cache
	cachemux sync.RWMutex
}

// HandleTDEvent updates a TD when receiving a TD event, sent by agents.
// Note that the TD is that as provided by the agent. The directory converts it to the
// digital twin format.
// TODO: Update the forms to match current protocols.
func (svc *DigitwinDirectory) HandleTDEvent(
	msg *things.ThingMessage) (stat api.DeliveryStatus) {

	// events use 'agent' thingIDs, only known to agents.
	// Digitwin adds the "dtw:{agentID}:" prefix, as the event now belongs to the virtual digital twin.
	dtThingID := things.MakeDigiTwinThingID(msg.SenderID, msg.ThingID)

	td := things.TD{}
	err := json.Unmarshal(msg.Data, &td)
	if err == nil {
		td.ID = dtThingID
		err = svc.UpdateThing(msg.SenderID, dtThingID, &td)

	}
	if err != nil {
		stat.Error = fmt.Sprintf(
			"StoreEvent. Failed updating TD of Agent/Thing '%s/%s': %s",
			msg.SenderID, msg.ThingID, err.Error())
		slog.Error(stat.Error)
	}
	stat.Completed(msg, err)
	return stat
}

// LoadCacheFromStore loads the cache from store
func (svc *DigitwinDirectory) LoadCacheFromStore() error {
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	cursor, err := svc.tdBucket.Cursor()
	if err != nil {
		return err
	}
	svc.thingKeys = make([]string, 0, 1000)
	svc.tdCache = make(map[string]*things.TD)
	//k, v, valid := cursor.First()
	//_ = k
	//_ = v
	//_ = valid
	for {
		// read in batches of 1000 TD documents
		tdmap, itemsRemaining := cursor.NextN(1000)
		for thingID, tddjson := range tdmap {
			svc.thingKeys = append(svc.thingKeys, thingID)
			tdd := &things.TD{}
			err = json.Unmarshal(tddjson, tdd)
			if err != nil {
				slog.Error("LoadCacheFromStore: Unmarshalling TDDoc failed. TD ignored",
					slog.String("thingID", thingID),
					slog.String("err", err.Error()))
			} else {
				svc.tdCache[thingID] = tdd
			}
		}
		if !itemsRemaining {
			break
		}
	}
	return nil
}

// QueryTDs query the collection of TD documents
func (svc *DigitwinDirectory) QueryTDs(args directory.QueryTDsArgs) (resp directory.QueryTDsResp, err error) {
	// TBD: query based on what?
	return resp, fmt.Errorf("not yet implemented")
}

// ReadTD returns the TD document in json format for the given Thing ID
func (svc *DigitwinDirectory) ReadTD(args directory.ReadTDArgs) (resp directory.ReadTDResp, err error) {
	svc.cachemux.RLock()
	defer svc.cachemux.RUnlock()
	td, found := svc.tdCache[args.ThingID]
	if !found {
		err = fmt.Errorf("Thing with ID '%s' not found", args.ThingID)
		return resp, err
	}
	// TODO: re-marshalling is inefficient. Do this on startup
	tdJSON, _ := json.Marshal(td)
	return directory.ReadTDResp{Output: string(tdJSON)}, err
}

// ReadTDs returns a list of TD documents
//
//	offset is the offset in the list
//	limit is the maximum number of records to return
func (svc *DigitwinDirectory) ReadTDs(args directory.ReadTDsArgs) (resp directory.ReadTDsResp, err error) {
	tdList := make([]string, 0, args.Limit)
	svc.cachemux.RLock()
	defer svc.cachemux.RUnlock()
	// Use the thingKeys index to ensure consistent iteration and to quickly
	// skip offset items (maps are not consistent between iterations)
	if args.Offset >= len(svc.thingKeys) {
		// empty result
		resp.Output = tdList
		return resp, nil
	}
	if args.Offset+args.Limit > len(svc.thingKeys) {
		args.Limit = len(svc.thingKeys) - args.Offset
	}
	tdKeys := svc.thingKeys[args.Offset:args.Limit]
	// add the TD documents
	for _, k := range tdKeys {
		v := svc.tdCache[k]
		// TODO: re-marshalling is inefficient. Do this on startup
		tdjson, _ := json.Marshal(v)
		tdList = append(tdList, string(tdjson))
	}
	resp.Output = tdList
	return resp, nil
}

// RemoveTD deletes the TD document from the given agent with the ThingID
func (svc *DigitwinDirectory) RemoveTD(args directory.RemoveTDArgs) error {
	slog.Info("RemoveThing",
		slog.String("thingID", args.ThingID))
	// remove from both cache and bucket
	err := svc.tdBucket.Delete(args.ThingID)
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	delete(svc.tdCache, args.ThingID)
	// delete from the index array. A bit primitive but it works
	for i, key := range svc.thingKeys {
		if key == args.ThingID {
			svc.thingKeys[i] = svc.thingKeys[len(svc.thingKeys)-1]
			svc.thingKeys = svc.thingKeys[:len(svc.thingKeys)-1]
			break
		}
	}
	return err
}

// Start the directory service and open the directory stored TD bucket
func (svc *DigitwinDirectory) Start() (err error) {
	slog.Info("Starting DigitwinDirectory")
	// fill the in-memory cache
	err = svc.LoadCacheFromStore()
	if err != nil {
		return err
	}
	return err
}

// Stop the service
func (svc *DigitwinDirectory) Stop() {
	slog.Info("Stopping DigitwinDirectory")
	if svc.tdBucket != nil {
		_ = svc.tdBucket.Close()
	}
}

// UpdateThing adds or updates the Thing Description document
// Added things are written to the store.
func (svc *DigitwinDirectory) UpdateThing(senderID string, dtThingID string, tdd *things.TD) error {
	slog.Info("UpdateThing",
		slog.String("senderID", senderID),
		slog.String("dtThingID", dtThingID))

	// TODO: update the forms to point to the Hub instead of the device
	svc.cachemux.Lock()
	defer svc.cachemux.Unlock()
	_, exists := svc.tdCache[dtThingID]
	// append the key if it doesn't yet exist
	if !exists {
		svc.thingKeys = append(svc.thingKeys, dtThingID)
	}
	// track the publisher
	svc.tdCache[dtThingID] = tdd

	// serialize to persist
	updatedTDD, _ := json.Marshal(tdd)
	err := svc.tdBucket.Set(dtThingID, updatedTDD)
	return err
}

// NewDigitwinDirectory creates a new service instance for the directory of Thing TD documents.
//
//	store is an instance of the bucket store to store the directory data. This is opened by 'Start' and closed by 'Stop'
func NewDigitwinDirectory(store buckets.IBucketStore) *DigitwinDirectory {
	tdBucket := store.GetBucket(TDBucketName)
	svc := &DigitwinDirectory{
		store:    store,
		tdBucket: tdBucket,
	}
	return svc
}
