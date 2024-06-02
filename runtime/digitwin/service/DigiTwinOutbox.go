package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

const OutboxBucketName = "eventHistory"
const LatestEventsBucketName = "latestEvents"

// DigiTwinOutbox is the digital twin outbox for sending events to subscribers.
//
// The typical message outflow is:
//
//	[digital twin outbox] -> protocol binding(s) => subscriber consumer
//
// These respond with a delivery status update
type DigiTwinOutbox struct {
	pm     api.ITransportBinding
	bucket buckets.IBucket
	latest *DigiTwinLatestStore
}

// HandleEvent adds an event to the outbox
func (svc *DigiTwinOutbox) HandleEvent(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	// events use 'raw' thingIDs, only known to agents.
	// Digitwin adds the "ht:{agentID}:" prefix, as the event now belongs to the virtual digital twin.
	// Same procedure at the DigiTwinDirectory
	dtThingID := things.MakeDigiTwinThingID(msg.SenderID, msg.ThingID)
	msg.ThingID = dtThingID

	// store for reading the last received events
	svc.latest.StoreMessage(msg)

	// TODO: prevent a double marshal?
	msgJSON, _ := json.Marshal(msg)
	err := svc.bucket.Set(msg.MessageID, msgJSON)
	stat.Completed(msg, err)

	// keep the history
	//svc.history.AddMessage(msg)

	// send the event to subscribers
	// Ignore the delivery result as the event is stored successfully
	_ = svc.pm.SendEvent(msg)
	return stat
}

// ReadLatest returns the latest values of a thing
// Read the latest value(s) of a Thing
func (svc *DigiTwinOutbox) ReadLatest(
	args outbox.ReadLatestArgs) (outbox.ReadLatestResp, error) {

	recs, err := svc.latest.ReadLatest(
		vocab.MessageTypeEvent, args.ThingID, args.Keys, args.Since)
	recsJSON, _ := json.Marshal(recs)
	resp := outbox.ReadLatestResp{Values: string(recsJSON)}
	return resp, err
}

// RemoveValue Remove Thing event value
// Intended to remove outliers
func (svc *DigiTwinOutbox) RemoveValue(args outbox.RemoveValueArgs) error {
	return fmt.Errorf("not yet implemented")
}

func (svc *DigiTwinOutbox) Start() error {
	// the 'latestStore' loads on demand
	return nil
}
func (svc *DigiTwinOutbox) Stop() {
	svc.latest.Stop()
	_ = svc.bucket.Close()
}

// NewDigiTwinOutbox returns a new instance of the outbox using the given storage bucket
func NewDigiTwinOutbox(bucketStore buckets.IBucketStore, pm api.ITransportBinding) *DigiTwinOutbox {
	eventsBucket := bucketStore.GetBucket(OutboxBucketName)
	latestBucket := bucketStore.GetBucket(LatestEventsBucketName)
	latestStore := NewDigiTwinLatestStore(latestBucket)
	outbox := &DigiTwinOutbox{
		latest: latestStore,
		bucket: eventsBucket,
		pm:     pm,
	}
	return outbox
}
