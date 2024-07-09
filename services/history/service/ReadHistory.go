package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/services/history/historyapi"
	"log/slog"
	"time"
)

// ReadHistory provides read access to the history of things values.
type ReadHistory struct {
	// routing address of the things to read history of
	bucketStore buckets.IBucketStore
	// cache of remote cursors
	cursorCache *buckets.CursorCache

	isRunning bool
}

// GetCursor returns an iterator for ThingValues containing a TD document
// The inactivity lifespan is currently fixed to 1 minute.
func (svc *ReadHistory) GetCursor(
	senderID string, args historyapi.GetCursorArgs) (*historyapi.GetCursorResp, error) {

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
	key := svc.cursorCache.Add(cursor, bucket, senderID, args.FilterOnKey, time.Minute)
	resp := &historyapi.GetCursorResp{CursorKey: key}
	return resp, nil
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
