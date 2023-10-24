package service

import (
	"github.com/hiveot/hub/core/history/historyapi"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/hiveot/hub/lib/thing"
)

// key of filter by event/action name, stored in context
const filterContextKey = "name"

// The HistoryCursor contains the bucket instance created for a cursor.
// It is created when a cursor is requested, stored in the cursorCache and
// released when the cursor is released or expires.
//type HistoryCursor struct {
//	// agentID
//	agentID    string //
//	thingID    string
//	filterName string                // optional event name to filter on
//	bucket     buckets.IBucket       // bucket being iterator
//	bc         buckets.IBucketCursor // the iteration
//}

// convert the storage key and raw data to a thing value object
// this must match the encoding done in AddHistory
//
//	bucketID is the ID of the bucket, which this service defines as agentID/thingID
//	key is the value's key, which is defined as timestamp/valueName
//
// This returns the value, or nil if the key is invalid
func decodeValue(bucketID string, key string, data []byte) (thingValue *thing.ThingValue, valid bool) {
	// key is constructed as  timestamp/agentID/thingID/valueName/{a|e}
	parts := strings.Split(key, "/")
	if len(parts) < 2 {
		return thingValue, false
	}
	millisec, _ := strconv.ParseInt(parts[0], 10, 64)
	// the bucketID consists of the agentID/thingID
	addrParts := strings.Split(bucketID, "/")
	if len(addrParts) < 2 {
		return nil, false
	}
	agentID := addrParts[0]
	thingID := addrParts[1]

	thingValue = &thing.ThingValue{
		ThingID:     thingID,
		AgentID:     agentID,
		Name:        parts[1],
		Data:        data,
		CreatedMSec: millisec,
	}
	return thingValue, true
}

// findNextName iterates the cursor until the next value containing 'name' is found and the
// timestamp doesn't exceed untilTime.
// A successive call with an increased timestamp should return the next batch of results. Intended
// to iterated an hours/day/week at a time.
// This returns the next value, or nil if the value was not found.
//
//	cursor to iterate
//	name is the event name to match
//	until is the time not to exceed in the result. Intended to avoid unnecessary iteration in range queries
func (svc *ReadHistoryService) findNextName(
	cursor buckets.IBucketCursor, name string, until time.Time) (thingValue *thing.ThingValue, found bool) {
	found = false
	for {
		k, v, valid := cursor.Next()
		if !valid {
			// key is invalid. This means we reached the end of cursor
			return nil, false
		}
		// key is constructed as  {timestamp}/{valueName}/{a|e}
		parts := strings.Split(k, "/")
		if len(parts) != 3 {
			// key exists but is invalid. skip this entry
			slog.Warn("findNextName: invalid key", "key", k)
		} else {
			// check timestamp and name must match
			timestampmsec, _ := strconv.ParseInt(parts[0], 10, 64)
			if timestampmsec > until.UnixMilli() {
				// we passed the given time limit
				// undo the last step so that followup requests with a new time limit can include this result
				cursor.Prev()
				return thingValue, false
			}
			if name == parts[1] {
				// found a match. Decode and return it
				thingValue, found = decodeValue(cursor.BucketID(), k, v)
				return
			}
			// name doesn't match. Skip this entry
		}
	}
}

// findPrevName iterates the cursor until the previous value containing 'name' is found and the
// timestamp is not before 'until' time.
// A successive call with an increased timestamp should return the next batch of results. Intended
// to iterate an hours/day/week at a time.
// This returns the previous value, or nil if the value was not found.
//
//	name is the event name to match
//	until is the time not to exceed in the result. Intended to avoid unnecesary iteration in range queries
func (svc *ReadHistoryService) findPrevName(
	cursor buckets.IBucketCursor, name string, until time.Time) (thingValue *thing.ThingValue, found bool) {
	found = false
	for {
		k, v, valid := cursor.Prev()
		if !valid {
			// key is invalid. This means we reached the beginning of cursor
			return thingValue, false
		}
		// key is constructed as  {timestamp}/{valueName}/{a|e}
		parts := strings.Split(k, "/")
		if len(parts) != 3 {
			// key exists but is invalid. skip this entry
		} else {
			// check timestamp and name must match
			timestampmsec, _ := strconv.ParseInt(parts[0], 10, 64)
			if timestampmsec < until.UnixMilli() {
				// we passed the given time limit
				// undo the last step so that followup requests with a new time limit can include this result
				cursor.Next()
				return thingValue, false
			}
			if name == parts[1] {
				// found a match. Decode and return it
				thingValue, found = decodeValue(cursor.BucketID(), k, v)
				return
			}
			// name doesn't match. Skip this entry
		}
	}
}

// First returns the oldest value in the history
func (svc *ReadHistoryService) First(
	ctx hubclient.ServiceContext, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {
	until := time.Now()

	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}
	k, v, valid := cursor.First()
	if !valid {
		// bucket is empty
		return nil, nil
	}

	thingValue, valid := decodeValue(cursor.BucketID(), k, v)
	filterName := cursor.Context().Value(filterContextKey).(string)
	if valid && filterName != "" && thingValue.Name != filterName {
		thingValue, valid = svc.findNextName(cursor, filterName, until)
	}
	resp := historyapi.CursorSingleResp{
		Value: thingValue,
		Valid: valid,
	}
	return &resp, err
}

// Last positions the cursor at the last key in the ordered list
func (svc *ReadHistoryService) Last(
	ctx hubclient.ServiceContext, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {

	// the beginning of time?
	until := time.Time{}
	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}
	k, v, valid := cursor.Last()

	resp := &historyapi.CursorSingleResp{
		Value: nil,
		Valid: valid,
	}

	if !valid {
		// bucket is empty
		return resp, nil
	}
	thingValue, valid := decodeValue(cursor.BucketID(), k, v)
	filterName := cursor.Context().Value(filterContextKey).(string)
	if valid && filterName != "" && thingValue.Name != filterName {
		thingValue, valid = svc.findPrevName(cursor, filterName, until)
	}
	resp.Value = thingValue
	resp.Valid = valid
	return resp, nil
}

// Next moves the cursor to the next key from the current cursor
// First() or Seek must have been called first.
func (svc *ReadHistoryService) Next(
	ctx hubclient.ServiceContext, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {

	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}
	k, v, valid := cursor.Next()
	resp := &historyapi.CursorSingleResp{
		Value: nil,
		Valid: valid,
	}
	if !valid {
		return resp, nil
	}
	thingValue, valid := decodeValue(cursor.BucketID(), k, v)
	filterName := cursor.Context().Value(filterContextKey).(string)
	if valid && filterName != "" && filterName != thingValue.Name {
		until := time.Now()
		thingValue, valid = svc.findNextName(cursor, filterName, until)
	}

	resp.Value = thingValue
	resp.Valid = valid

	return resp, nil
}

// NextN moves the cursor to the next N places from the current cursor
// and return a list with N values in incremental time order.
// itemsRemaining is false if the iterator has reached the end.
// Intended to speed up with batch iterations over rpc.
func (svc *ReadHistoryService) NextN(
	ctx hubclient.ServiceContext, args historyapi.CursorNArgs) (*historyapi.CursorNResp, error) {

	values := make([]*thing.ThingValue, 0, args.Limit)
	nextArgs := historyapi.CursorArgs{CursorKey: args.CursorKey}
	itemsRemaining := true

	// tbd is it faster to use NextN and sort the keys?
	for i := 0; i < args.Limit; i++ {
		nextResp, err := svc.Next(ctx, nextArgs)
		if !nextResp.Valid || err != nil {
			itemsRemaining = false
			break
		}
		values = append(values, nextResp.Value)
	}
	resp := &historyapi.CursorNResp{}
	resp.Values = values
	resp.ItemsRemaining = itemsRemaining
	return resp, nil
}

// Prev moves the cursor to the previous key from the current cursor
// Last() or Seek must have been called first.
func (svc *ReadHistoryService) Prev(
	ctx hubclient.ServiceContext, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {

	until := time.Time{}
	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}
	k, v, valid := cursor.Prev()
	resp := &historyapi.CursorSingleResp{
		Value: nil,
		Valid: valid,
	}
	if !valid {
		return resp, nil
	}
	thingValue, valid := decodeValue(cursor.BucketID(), k, v)
	filterName := cursor.Context().Value(filterContextKey).(string)
	if valid && filterName != "" && filterName != thingValue.Name {
		thingValue, valid = svc.findPrevName(cursor, filterName, until)
	}
	resp.Value = thingValue
	resp.Valid = valid
	return resp, nil
}

// PrevN moves the cursor back N places from the current cursor
// and return a list with N values in decremental time order.
// itemsRemaining is true if the iterator has reached the beginning
// Intended to speed up with batch iterations over rpc.
func (svc *ReadHistoryService) PrevN(
	ctx hubclient.ServiceContext, args historyapi.CursorNArgs) (*historyapi.CursorNResp, error) {

	values := make([]*thing.ThingValue, 0, args.Limit)
	prevArgs := historyapi.CursorArgs{CursorKey: args.CursorKey}
	itemsRemaining := true

	// tbd is it faster to use NextN and sort the keys? - for a remote store yes
	for i := 0; i < args.Limit; i++ {
		prevResp, err := svc.Prev(ctx, prevArgs)
		if !prevResp.Valid || err != nil {
			itemsRemaining = false
			break
		}
		values = append(values, prevResp.Value)
	}
	resp := &historyapi.CursorNResp{}
	resp.Values = values
	resp.ItemsRemaining = itemsRemaining
	return resp, nil
}

// Release closes the bucket and cursor
// This invalidates all values obtained from the cursor
func (svc *ReadHistoryService) Release(
	ctx hubclient.ServiceContext, args historyapi.CursorReleaseArgs) error {

	return svc.cursorCache.Release(ctx.ClientID, args.CursorKey)
}

// Seek positions the cursor at the given searchKey and corresponding value.
// If the key is not found, the next key is returned.
func (svc *ReadHistoryService) Seek(
	ctx hubclient.ServiceContext, args historyapi.CursorSeekArgs) (*historyapi.CursorSingleResp, error) {

	until := time.Now()
	//ts, err := dateparse.ParseAny(timestampMsec)
	//if err != nil {
	slog.Info("Seek using timestamp",
		slog.Int64("timestampMsec", args.TimeStampMSec),
		slog.String("clientID", ctx.ClientID),
	)

	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}

	// search the first occurrence at or after the given timestamp
	// the buck index uses the stringified timestamp
	searchKey := strconv.FormatInt(args.TimeStampMSec, 10) //+ "/" + thingValue.ID

	k, v, valid := cursor.Seek(searchKey)
	resp := &historyapi.CursorSingleResp{
		Value: nil,
		Valid: valid,
	}
	if !valid {
		// bucket is empty
		return resp, nil
	}
	thingValue, valid := decodeValue(cursor.BucketID(), k, v)
	filterName := cursor.Context().Value(filterContextKey).(string)
	if valid && filterName != "" && thingValue.Name != filterName {
		thingValue, valid = svc.findNextName(cursor, filterName, until)
	}
	resp.Value = thingValue
	resp.Valid = valid

	return resp, nil
}
