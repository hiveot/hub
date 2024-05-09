package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/protocols"
)

const OutboxBucketName = "outbox"

// DigiTwinOutbox is the digital twin outbox for sending events to subscribers.
//
// The typical message outflow is:
//
//	[digital twin outbox] -> protocol binding(s) => subscriber consumer
//
// These respond with a delivery status update
type DigiTwinOutbox struct {
	pm     *protocols.ProtocolsManager
	bucket buckets.IBucket
	latest *DigiTwinLatestStore
}

// HandleEvent adds an event to the inbox
func (svc *DigiTwinOutbox) HandleEvent(msg *things.ThingMessage) (stat api.DeliveryStatus) {
	// change the thingID to that of the digitwin
	dtThingID := MakeDigiTwinThingID(msg.SenderID, msg.ThingID)
	msg.ThingID = dtThingID

	// store for reading the last received events
	svc.latest.StoreMessage(msg)

	// keep the history
	//svc.history.AddMessage(msg)

	// send the event to subscribers
	stat = svc.pm.SendEvent(msg)
	return stat
}

// ReadLatest returns the latest values of a thing
// Read the latest value(s) of a Thing
func (svc *DigiTwinOutbox) ReadLatest(
	args outbox.ReadLatestArgs) (outbox.ReadLatestResp, error) {

	recs, err := svc.latest.ReadLatest(
		vocab.MessageTypeEvent, args.ThingID, args.Keys, args.Since)
	resp := outbox.ReadLatestResp{Values: recs}
	return resp, err
}

// RemoveValue Remove Thing event value
// Intended to remove outliers
func (svc *DigiTwinOutbox) RemoveValue(args outbox.RemoveValueArgs) error {
	return fmt.Errorf("not yet implemented")
}

func (svc *DigiTwinOutbox) Start() error {
	return nil
}
func (svc *DigiTwinOutbox) Stop() {
}

// NewDigiTwinOutbox returns a new instance of the outnbox using the given storage bucket
func NewDigiTwinOutbox(bucketStore buckets.IBucketStore) *DigiTwinOutbox {
	bucket := bucketStore.GetBucket(OutboxBucketName)
	outbox := &DigiTwinOutbox{
		bucket: bucket,
	}
	return outbox
}
