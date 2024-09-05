package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
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

// GetTD returns the JSON encoded TD of this service
func (svc *DigiTwinOutboxService) GetTD() string {
	tdDoc := digitwin.OutboxTD
	return tdDoc
}

// HandleEvent adds an event to the outbox
func (svc *DigiTwinOutboxService) HandleEvent(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	// events use 'raw' thingIDs, only known to agents.
	// Digitwin adds the "ht:{agentID}:" prefix, as the event now belongs to the virtual digital twin.
	// Same procedure at the DigiTwinDirectory
	dtThingID := tdd.MakeDigiTwinThingID(msg.SenderID, msg.ThingID)
	msg.ThingID = dtThingID

	// store for reading the last received events
	svc.latest.StoreMessage(msg)

	// broadcast the event to subscribers
	stat = svc.pm.SendEvent(msg)
	return stat
}

// ReadAllProperties returns the current property values of a thing
func (svc *DigiTwinOutboxService) ReadAllProperties(senderID string,
	args digitwin.OutboxReadLatestArgs) (values map[string]any, err error) {

	recs, err := svc.latest.ReadLatest(
		vocab.MessageTypeProperty, args.ThingID, args.Keys, args.Since)
	if err == nil {
		// this mapping is ugly. It can't be described with a TD dataschema :'(
		values = make(map[string]any)
		for key, val := range recs {
			values[key] = val
		}
	}
	return values, err
}

// ReadLatest returns the latest event values of a thing
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
