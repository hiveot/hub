package service

import (
	"github.com/hiveot/hub/core/history"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"time"

	"github.com/hiveot/hub/lib/thing"
)

// GetPropertiesFunc is a callback function to retrieve latest properties of a Thing
// latest properties are stored separate from the history.
type GetPropertiesFunc func(thingAddr string, names []string) []thing.ThingValue

// ReadHistoryService provides read access to the history of thing values.
type ReadHistoryService struct {
	// routing address of the thing to read history of
	bucketStore buckets.IBucketStore
	// cache of remote cursors
	cursorCache *buckets.CursorCache
	// The service implements the getPropertyValues function as it does the caching and
	// provides concurrency control.
	getPropertiesFunc GetPropertiesFunc

	readSub   hubclient.ISubscription
	isRunning bool
}

// GetCursor returns an iterator for ThingValues containing a TD document
// The inactivity lifespan is currently fixed to 1 minute.
func (svc *ReadHistoryService) GetCursor(
	clientID string, args history.GetCursorArgs) (history.GetCursorResp, error) {

	thingAddr := args.AgentID + "/" + args.ThingID
	bucket := svc.bucketStore.GetBucket(thingAddr)
	cursor := bucket.Cursor()
	key := svc.cursorCache.Add(cursor, bucket, clientID, time.Minute)
	resp := history.GetCursorResp{CursorKey: key}
	return resp, nil
}

// GetLatest returns the most recent property and event values of the Thing.
// Latest Properties are tracked in a 'latest' record which holds a map of propertyName:ThingValue records
//
//	providing 'names' can speed up read access significantly
func (svc *ReadHistoryService) GetLatest(
	agentID string, thingID string, names []string) (values []thing.ThingValue) {
	thingAddr := agentID + "/" + thingID
	values = svc.getPropertiesFunc(thingAddr, names)
	return values
}

// Stop the read history capability
// this unsubscribes from requests and stops the cursor cleanup task.
func (svc *ReadHistoryService) Stop() {
	svc.isRunning = false
	svc.cursorCache.Stop()
	svc.readSub.Unsubscribe()
}

// StartReadHistoryService starts the capability to read from a thing's history
//
//	hc with the message bus connection. Its ID will be used as the agentID that provides the capability.
//	thingBucket is the open bucket used to store history data
//	getPropertiesFunc implements the aggregation of the Thing's most recent property values
func StartReadHistoryService(
	hc hubclient.IHubClient, bucketStore buckets.IBucketStore, getPropertiesFunc GetPropertiesFunc,
) (svc *ReadHistoryService, err error) {

	svc = &ReadHistoryService{
		bucketStore:       bucketStore,
		getPropertiesFunc: getPropertiesFunc,
	}
	capMethods := map[string]interface{}{
		history.CursorFirstMethod:   svc.First,
		history.CursorLastMethod:    svc.Last,
		history.CursorNextMethod:    svc.Next,
		history.CursorNextNMethod:   svc.NextN,
		history.CursorPrevMethod:    svc.Prev,
		history.CursorPrevNMethod:   svc.PrevN,
		history.CursorReleaseMethod: svc.Release,
		history.CursorSeekMethod:    svc.Seek,
		history.GetCursorMethod:     svc.GetCursor,
		history.GetLatestMethod:     svc.GetLatest,
	}
	svc.readSub, err = hubclient.SubRPCCapability(
		hc, history.ReadHistoryCap, capMethods)
	if err != nil {
		svc.cursorCache.Stop()
	}
	return svc, err
}
