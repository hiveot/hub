package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
)

const LatestEventsBucketName = "latestEvents"

// DigiTwinOutboxService is the digital twin outbox for sending events to subscribers.
//
// The typical message outflow is:
//
//	[digital twin outbox] -> protocol binding(s) => subscriber consumer
//
// These respond with a delivery status update
type DigiTwinOutboxService struct {
	pm     api.ITransportBinding
	latest *DigiTwinLatestStore
}

// HandleEvent adds an event to the outbox
func (svc *DigiTwinOutboxService) HandleEvent(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	// events use 'raw' thingIDs, only known to agents.
	// Digitwin adds the "ht:{agentID}:" prefix, as the event now belongs to the virtual digital twin.
	// Same procedure at the DigiTwinDirectory
	dtThingID := things.MakeDigiTwinThingID(msg.SenderID, msg.ThingID)
	msg.ThingID = dtThingID

	// store for reading the last received events
	svc.latest.StoreMessage(msg)

	// send the event to subscribers
	_ = svc.pm.SendEvent(msg)
	return stat
}

// ReadLatest returns the latest values of a thing
// Read the latest value(s) of a Thing
func (svc *DigiTwinOutboxService) ReadLatest(senderID string,
	args digitwin.OutboxReadLatestArgs) (values map[string]any, err error) {

	recs, err := svc.latest.ReadLatest(
		vocab.MessageTypeEvent, args.ThingID, args.Keys, args.Since)
	if err == nil {
		// this mapping is ugly. It can't be described with a TD dataschema :'(
		values = make(map[string]any)
		for key, val := range recs {
			values[key] = val
		}
	}
	return values, err
}

// RemoveValue Remove Thing event value
// Intended to remove outliers
func (svc *DigiTwinOutboxService) RemoveValue(senderID string, messageID string) error {
	return fmt.Errorf("not yet implemented")
}

func (svc *DigiTwinOutboxService) Start() error {
	// the 'latestStore' loads on demand
	return nil
}
func (svc *DigiTwinOutboxService) Stop() {
	svc.latest.Stop()
}

// NewDigiTwinOutbox returns a new instance of the outbox using the given storage bucket
func NewDigiTwinOutbox(bucketStore buckets.IBucketStore, pm api.ITransportBinding) *DigiTwinOutboxService {
	latestBucket := bucketStore.GetBucket(LatestEventsBucketName)
	latestStore := NewDigiTwinLatestStore(latestBucket)
	outbox := &DigiTwinOutboxService{
		latest: latestStore,
		pm:     pm,
	}
	return outbox
}
