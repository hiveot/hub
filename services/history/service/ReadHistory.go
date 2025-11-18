package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivehub/lib/buckets"
	"github.com/hiveot/hivehub/services/history/historyapi"
	"github.com/hiveot/hivekitgo/messaging"
)

// ReadHistory provides read access to the history of things values.
type ReadHistory struct {
	// routing address of the things to read history of
	bucketStore buckets.IBucketStore
	// cache of remote cursors
	cursorCache *buckets.CursorCache

	isRunning bool
}

// GetCursor returns an iterator for ThingMessage objects.
func (svc *ReadHistory) GetCursor(
	senderID string, args historyapi.GetCursorArgs) (*historyapi.GetCursorResp, error) {

	lifespan := time.Minute
	if args.LifespanSec != 0 {
		lifespan = time.Duration(args.LifespanSec) * time.Second
	}
	if args.ThingID == "" {
		return nil, fmt.Errorf("missing thingID")
	}
	thingAddr := args.ThingID
	slog.Info("GetCursor for bucket: ", "addr", thingAddr)
	bucket := svc.bucketStore.GetBucket(thingAddr)
	cursor, err := bucket.Cursor()
	//
	if err != nil {
		return nil, err
	}
	key := svc.cursorCache.Add(cursor, bucket, senderID, args.FilterOnName, lifespan)
	resp := &historyapi.GetCursorResp{CursorKey: key}
	return resp, nil
}

// internal read history function
func (svc *ReadHistory) readHistory(
	thingID string, filterOnKey string, timestamp string, durationSec int, limit int) (
	values []*messaging.ThingValue, itemsRemaining bool, err error) {

	values = make([]*messaging.ThingValue, 0)

	if limit <= 0 {
		limit = historyapi.DefaultLimit
	}
	if thingID == "" {
		return nil, false, fmt.Errorf("missing thingID")
	}
	bucket := svc.bucketStore.GetBucket(thingID)
	cursor, err := bucket.Cursor()
	if err != nil {
		return nil, false, err
	}
	defer cursor.Release()

	ts, _ := dateparse.ParseAny(timestamp)
	item0, valid := svc.seek(cursor, ts, filterOnKey)
	if valid {
		// item0 is nil when seek after the last available item
		values = append(values, item0)
	}
	var batch []*messaging.ThingValue
	until := ts.Add(time.Duration(durationSec) * time.Second)
	if durationSec > 0 {
		// read forward in time
		batch, itemsRemaining = svc.nextN(cursor, filterOnKey, until, limit)
	} else {
		// read backwards in time
		batch, itemsRemaining = svc.prevN(cursor, filterOnKey, until, limit)
	}
	values = append(values, batch...)
	return values, itemsRemaining, err
}

// ReadHistory the history for the given time, duration and limit
// For more extensive result use the cursor
// To go back in time use the negative duration.
func (svc *ReadHistory) ReadHistory(
	_ string, args historyapi.ReadHistoryArgs) (resp historyapi.ReadHistoryResp, err error) {

	items, remaining, err := svc.readHistory(
		args.ThingID, args.FilterOnName, args.Timestamp, args.Duration, args.Limit)

	resp.Values = items
	resp.ItemsRemaining = remaining
	return resp, err
}

// Start the read history handler.
// This starts the cursor cache
func (svc *ReadHistory) Start() (err error) {
	svc.cursorCache.Start()
	return nil
}

// Stop the read history capability
// this unsubscribes from requests and stops the cursor cleanup task.
func (svc *ReadHistory) Stop() {
	svc.isRunning = false
	svc.cursorCache.Stop()
}

// NewReadHistory starts the capability to read from a things's history
//
//	hc with the message bus connection. Its ID will be used as the agentID that provides the capability.
//	thingBucket is the open bucket used to store history data
func NewReadHistory(bucketStore buckets.IBucketStore) (svc *ReadHistory) {

	svc = &ReadHistory{
		bucketStore: bucketStore,
		cursorCache: buckets.NewCursorCache(),
	}
	return svc
}
