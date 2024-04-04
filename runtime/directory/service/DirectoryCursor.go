package service

import (
	"context"
	"github.com/hiveot/hub/lib/buckets"
	"time"
)

// DirectoryCursorMgr manages iteration cursors
type DirectoryCursorMgr struct {
	// cache of cursors for remote clients
	cursorCache *buckets.CursorCache
	// maximum lifespan of a cursor after last use
	cursorLifespan time.Duration

	// directory bucket
	tdBucket buckets.IBucket
}

// CloseCursor release the resources associated with the cursor.
// This invalidates all values obtained from the cursor
func (svc *DirectoryCursorMgr) CloseCursor(cursorKey string, clientID string) {
	_ = svc.cursorCache.Release(cursorKey, clientID)
}

// First returns the first entry in the directory
func (svc *DirectoryCursorMgr) First(cursorKey string, senderID string) (
	thingID string, tdd string, valid bool, err error) {

	cursor, err := svc.cursorCache.Get(cursorKey, senderID, true)
	if err != nil {
		return "", "", false, err
	}
	k, v, valid := cursor.First()
	if !valid {
		// store is empty
		return "", "", false, nil
	}
	return k, string(v), valid, nil
}

// Next moves the cursor to the next key from the current cursor
// First() or Seek must have been called first.
// This returns the next item, a flag if the value is valid
func (svc *DirectoryCursorMgr) Next(cursorKey string, clientID string) (
	id string, tdd string, valid bool, err error) {

	cursor, err := svc.cursorCache.Get(cursorKey, clientID, true)
	if err != nil {
		return "", "", false, err
	}

	k, v, valid := cursor.Next()
	return k, string(v), valid, nil
}

// NextN moves the cursor to the next N places from the current cursor
// and return a list with N values in incremental time order.
// itemsRemaining is false if the iterator has reached the end.
// Intended to speed up with batch iterations over rpc.
func (svc *DirectoryCursorMgr) NextN(cursorKey string, clientID string, limit uint) (
	tddList []string, itemsRemaining bool, err error) {

	cursor, err := svc.cursorCache.Get(cursorKey, clientID, true)
	if err != nil {
		return tddList, false, err
	}
	tddList = make([]string, 0, limit)
	// obtain a map of [addr]TDJson
	docMap, itemsRemaining := cursor.NextN(limit)
	for _, doc := range docMap {
		tddList = append(tddList, string(doc))
	}
	return tddList, itemsRemaining, err
}

// NewCursor creates a new iteration cursor for TD documents.
//
// Iteration cursors are intended for remote clients to efficiently read a large set of data.
// They are stateful and need to be closed after use with CloseCursor. They can close
// automatically if there has been no activity for a period of time.
//
// This returns the cursor key. Both clientID and key are needed for further iteration.
// Use CloseCursor to release the cursor
func (svc *DirectoryCursorMgr) NewCursor(clientID string) (cursorKey string, err error) {
	cursor, err := svc.tdBucket.Cursor(context.Background())
	cursorKey = svc.cursorCache.Add(cursor, svc.tdBucket, clientID, svc.cursorLifespan)
	return cursorKey, err
}

// NewDirectoryCursor create a new instance of the directory iteration cursor management
//
//	tdBucket is the bucket containing TD documents
//	lifespan is the maximum lifespan of a cursor before it is released
func NewDirectoryCursor(tdBucket buckets.IBucket, lifespan time.Duration) *DirectoryCursorMgr {
	svc := DirectoryCursorMgr{
		tdBucket:       tdBucket,
		cursorLifespan: lifespan,
		cursorCache:    buckets.NewCursorCache(),
	}
	return &svc
}
