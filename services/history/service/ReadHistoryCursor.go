package service

import (
	"github.com/araddon/dateparse"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/history/historyapi"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/hiveot/hub/lib/things"
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

// convert the storage key and raw data to a things value object
// this must match the encoding done in AddHistory
//
//	bucketID is the ID of the bucket, which is the digital twin thingID
//	storageKey is the value's key, which is defined as timestamp/valueKey
//	data is the stored message data
//
// This returns the value, or nil if the key is invalid
func decodeValue(bucketID string, storageKey string, data string) (thingValue *things.ThingMessage, valid bool) {

	// key is constructed as  timestamp/name/{a|e|c}/sender, where sender can be omitted
	parts := strings.Split(storageKey, "/")
	if len(parts) < 2 {
		return thingValue, false
	}
	createdMsec, _ := strconv.ParseInt(parts[0], 10, 64)
	createdTime := time.UnixMilli(createdMsec)
	key := parts[1]
	senderID := ""
	messageType := vocab.MessageTypeEvent
	if len(parts) >= 2 {
		if parts[2] == "a" {
			messageType = vocab.MessageTypeAction
		} else if parts[2] == "p" {
			messageType = vocab.MessageTypeProperty
		}
	}
	if len(parts) > 3 {
		senderID = parts[3]
	}
	// FIXME: keep the messageID? serialize the ThingMessage
	messageID := ""

	thingValue = &things.ThingMessage{
		ThingID:     bucketID, // digital twin thingID that includes the agent prefix
		MessageID:   messageID,
		Key:         key,
		Data:        data,
		Created:     createdTime.Format(utils.RFC3339Milli),
		MessageType: messageType,
		SenderID:    senderID,
	}
	return thingValue, true
}

// First returns the oldest value in the history
func (svc *ReadHistory) First(senderID string, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {
	until := time.Now()

	cursor, ci, err := svc.cursorCache.Get(args.CursorKey, senderID, true)
	if err != nil {
		return nil, err
	}
	k, v, valid := cursor.First()
	if !valid {
		// bucket is empty
		return nil, nil
	}

	thingValue, valid := decodeValue(cursor.BucketID(), k, string(v))
	filterName := ci.Filter
	if valid && filterName != "" && thingValue.Key != filterName {
		thingValue, valid = svc.next(cursor, filterName, until)
	}
	resp := historyapi.CursorSingleResp{
		Value: thingValue,
		Valid: valid,
	}
	return &resp, err
}

// Last positions the cursor at the last key in the ordered list
func (svc *ReadHistory) Last(senderID string, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {

	// the beginning of time?
	until := time.Time{}
	cursor, ci, err := svc.cursorCache.Get(args.CursorKey, senderID, true)
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
	thingValue, valid := decodeValue(cursor.BucketID(), k, string(v))
	filterName := ci.Filter
	if valid && filterName != "" && thingValue.Key != filterName {
		// search back to the last valid value
		thingValue, valid = svc.prev(cursor, filterName, until)
	}
	resp.Value = thingValue
	resp.Valid = valid
	return resp, nil
}

// next iterates the cursor until the next value containing 'name' is found and the
// timestamp doesn't exceed untilTime.
// A successive call with an increased timestamp should return the next batch of results. Intended
// to iterated an hours/day/week at a time.
// This returns the next value, or nil if the value was not found.
//
//	cursor to iterate
//	name is the event name to match
//	until is the time not to exceed in the result. Intended to avoid unnecessary iteration in range queries
func (svc *ReadHistory) next(
	cursor buckets.IBucketCursor, key string, until time.Time) (
	thingValue *things.ThingMessage, found bool) {

	untilMilli := until.UnixMilli()
	found = false
	for {
		k, v, valid := cursor.Next()
		if !valid {
			// key is invalid. This means we reached the end of cursor
			return nil, false
		}
		// key is constructed as  {timestamp}/{valueName}/{a|e|c}/{sender}
		parts := strings.Split(k, "/")
		if len(parts) != 4 {
			// key exists but is invalid. skip this entry
			slog.Warn("findNextName: invalid key", "key", k)
		} else {
			// check timestamp and name must match
			timestampmsec, _ := strconv.ParseInt(parts[0], 10, 64)
			if untilMilli > 0 && timestampmsec > untilMilli {
				// we passed the given time limit
				// undo the last step so that followup requests with a new time limit can include this result
				cursor.Prev()
				return thingValue, false
			}
			if key == "" || key == parts[1] {
				// found a match. Decode and return it
				thingValue, found = decodeValue(cursor.BucketID(), k, string(v))
				return thingValue, found
			}
			// name doesn't match. Skip this entry
		}
	}
}

// Next moves the cursor to the next key from the current cursor
// First() or Seek must have been called first.
// This returns an error if the cursor is not found.
func (svc *ReadHistory) Next(senderID string, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {

	cursor, ci, err := svc.cursorCache.Get(args.CursorKey, senderID, true)
	if err != nil {
		return nil, err
	}
	until := time.Now()
	value, valid := svc.next(cursor, ci.Filter, until)
	resp := historyapi.CursorSingleResp{
		Value: value,
		Valid: valid,
	}

	return &resp, nil
}

// Read the next number of items until time or count limit is reached
func (svc *ReadHistory) nextN(
	cursor buckets.IBucketCursor, filterKey string, endTime time.Time, limit int) (
	items []*things.ThingMessage, itemsRemaining bool) {

	items = make([]*things.ThingMessage, 0, limit)
	itemsRemaining = true

	for i := 0; i < limit; i++ {
		value, valid := svc.next(cursor, filterKey, endTime)
		if !valid {
			itemsRemaining = false
			break
		}
		items = append(items, value)
	}
	return items, itemsRemaining
}

// NextN moves the cursor to the next N places from the current cursor
// and return a list with N values in incremental time order.
// itemsRemaining is false if the iterator has reached the end.
// Intended to speed up with batch iterations over rpc.
func (svc *ReadHistory) NextN(senderID string, args historyapi.CursorNArgs) (*historyapi.CursorNResp, error) {

	until := time.Now()
	if args.Until != "" {
		until, _ = dateparse.ParseAny(args.Until)
	}
	limit := args.Limit
	if limit <= 0 {
		limit = historyapi.DefaultLimit
	}
	cursor, ci, err := svc.cursorCache.Get(args.CursorKey, senderID, true)
	if err != nil {
		return nil, err
	}
	values, itemsRemaining := svc.nextN(cursor, ci.Filter, until, limit)

	resp := &historyapi.CursorNResp{}
	resp.Values = values
	resp.ItemsRemaining = itemsRemaining
	return resp, nil
}

// prev iterates the cursor until the previous value passes the filters and the
// timestamp is not before 'until' time.
//
// This supports 2 filters, a key of the value and a timestamp.
// Since key and timestamp are part of the bucket key these checks are fast.
//
// This returns the previous value, or nil if the value was not found.
//
//	cursor is a valid bucket cursor
//	key is the event key to match or "" for all keys
//	until is the limit of the time to read. Intended for time-range queries and
//	to avoid unnecessary iteration in range queries
func (svc *ReadHistory) prev(
	cursor buckets.IBucketCursor, key string, until time.Time) (
	thingValue *things.ThingMessage, found bool) {

	untilMilli := until.UnixMilli()
	found = false
	for {
		k, v, valid := cursor.Prev()
		if !valid {
			// key is invalid. This means we reached the beginning of cursor
			return thingValue, false
		}
		// key is constructed as  {timestamp}/{valueName}/{a|e|c}/sender
		parts := strings.Split(k, "/")
		if len(parts) != 4 {
			// key exists but is invalid. skip this entry
		} else {
			// check timestamp and name must match
			timestampmsec, _ := strconv.ParseInt(parts[0], 10, 64)
			if timestampmsec < untilMilli {
				// we passed the given time limit
				// undo the last step so that followup requests with a new time limit can include this result
				cursor.Next()
				return nil, false
			}

			if key == "" || key == parts[1] {
				// found a match. Decode and return it
				thingValue, found = decodeValue(cursor.BucketID(), k, string(v))
				return thingValue, found
			}
			// filter doesn't match. Skip this entry
		}
	}
}

// Read the previous number of items until time or count limit is reached
func (svc *ReadHistory) prevN(
	cursor buckets.IBucketCursor, filterKey string, endTime time.Time, limit int) (
	items []*things.ThingMessage, itemsRemaining bool) {

	items = make([]*things.ThingMessage, 0, limit)
	itemsRemaining = true

	for i := 0; i < limit; i++ {
		value, valid := svc.prev(cursor, filterKey, endTime)
		if !valid {
			itemsRemaining = false
			break
		}
		items = append(items, value)
	}
	return items, itemsRemaining
}

// Prev moves the cursor to the previous key from the current cursor
// Last() or Seek must have been called first.
func (svc *ReadHistory) Prev(
	senderID string, args historyapi.CursorArgs) (*historyapi.CursorSingleResp, error) {

	until := time.Time{}
	cursor, ci, err := svc.cursorCache.Get(args.CursorKey, senderID, true)
	if err != nil {
		return nil, err
	}
	value, valid := svc.prev(cursor, ci.Filter, until)
	resp := historyapi.CursorSingleResp{
		Value: value,
		Valid: valid,
	}
	return &resp, nil
}

// PrevN returns up to N results or until the time limit is reached.
// itemsRemaining is true if the iterator has reached the count limit, indicating
// that more items can be read before the time limit is reached.
func (svc *ReadHistory) PrevN(senderID string, args historyapi.CursorNArgs) (*historyapi.CursorNResp, error) {

	until := time.Time{} // zero time
	if args.Until != "" {
		until, _ = dateparse.ParseAny(args.Until)
	}
	limit := args.Limit
	if limit <= 0 {
		limit = historyapi.DefaultLimit
	}
	cursor, ci, err := svc.cursorCache.Get(args.CursorKey, senderID, true)
	if err != nil {
		return nil, err
	}

	values, itemsRemaining := svc.prevN(cursor, ci.Filter, until, limit)

	resp := &historyapi.CursorNResp{}
	resp.Values = values
	resp.ItemsRemaining = itemsRemaining
	return resp, nil
}

// Release closes the bucket and cursor
// This invalidates all values obtained from the cursor
func (svc *ReadHistory) Release(senderID string, args historyapi.CursorReleaseArgs) error {

	return svc.cursorCache.Release(senderID, args.CursorKey)
}

// seek internal function for seeking with a cursor
func (svc *ReadHistory) seek(cursor buckets.IBucketCursor, ts time.Time, key string) (
	tm *things.ThingMessage, valid bool) {
	until := time.Now()

	// search the first occurrence at or after the given timestamp
	// the bucket index uses the stringified timestamp
	msec := ts.UnixMilli()
	searchKey := strconv.FormatInt(msec, 10)

	k, v, valid := cursor.Seek(searchKey)
	if !valid {
		// bucket is empty, no error
		return nil, valid
	}
	thingValue, valid := decodeValue(cursor.BucketID(), k, string(v))
	if valid && key != "" && thingValue.Key != key {
		thingValue, valid = svc.next(cursor, key, until)
	}
	return thingValue, valid
}

// Seek positions the cursor at the given searchKey and corresponding value.
// If the key is not found, the next key is returned.
// This returns an error if the cursor is not found.
func (svc *ReadHistory) Seek(senderID string, args historyapi.CursorSeekArgs) (
	*historyapi.CursorSingleResp, error) {

	slog.Info("Seek using timestamp",
		slog.String("timestamp", args.TimeStamp),
	)

	cursor, ci, err := svc.cursorCache.Get(args.CursorKey, senderID, true)
	if err != nil {
		return nil, err
	}

	// search the first occurrence at or after the given timestamp
	// the buck index uses the stringified timestamp
	ts, _ := dateparse.ParseAny(args.TimeStamp)
	thingValue, valid := svc.seek(cursor, ts, ci.Filter)

	resp := &historyapi.CursorSingleResp{
		Value: thingValue,
		Valid: valid,
	}
	return resp, nil
}
