package buckets

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

type ClientCursors []IBucketCursor

type CursorInfo struct {
	Key    string
	Filter string
	// optional bucket instance this cursor operates on
	// if provided it will be released with the cursor
	Bucket IBucket
	// the stored cursor
	Cursor IBucketCursor
	// clientID of the cursor owner
	OwnerID string
	// last use of the cursor
	LastUsed time.Time
	// lifespan of cursor after last use
	Lifespan time.Duration
}

// CursorCache manages a set of cursors that can be addressed remotely by key.
// Intended for servers that let remote clients iterate a cursor in the bucket store.
//
// Added cursors are stored in a map by generated key along with some metadata
// such as its expiry and optionally the bucket that was allocated to use the iterator.
// The key is passed back to the client for use during iterations.
// The client must release the cursor it when done.
//
// To prevent memory leaks due to not releasing a cursor, cursors are given a limited
// non-used lifespan, after which they are removed. The default is 1 minute.
//
// Cursors are linked to their owner to prevent 'accidental' use by others. Only
// if the client's ID matches that of the cursor owner, it can be used.
type CursorCache struct {
	// lookup a cursor by key
	cursorsByKey map[string]*CursorInfo

	// at 1000 cursors per sec this lasts 500M years between reboots ;)
	cursorCounter uint64
	mux           sync.RWMutex
	// stop the background loop
	stopCh chan bool
}

// Add adds a cursor to the tracker and returns its key
//
//	cursor is the object holding the cursor
//	bucket instance created specifically for this cursor. optional.
//	clientID of the owner
//	lifespan of the cursor after last use
func (cc *CursorCache) Add(
	cursor IBucketCursor, bucket IBucket, clientID string, filter string, lifespan time.Duration) string {
	cc.mux.Lock()
	defer cc.mux.Unlock()

	cc.cursorCounter++
	// the key is not a secret, only the owner can use it
	key := strconv.FormatUint(cc.cursorCounter, 16)
	ci := &CursorInfo{
		Key:      key,
		Filter:   filter,
		Bucket:   bucket,
		Cursor:   cursor,
		OwnerID:  clientID,
		Lifespan: lifespan,
	}
	cc.cursorsByKey[key] = ci
	return key
}

// Get returns the cursor with the given key.
// An error is returned if the cursor is not found, has expired, or belongs to a different owner.
//
//	cursorKey obtained with Add()
//	clientID requesting the cursor
//	updateLastUsed resets the lifespan of the cursor to start now
func (cc *CursorCache) Get(cursorKey string, clientID string, updateLastUsed bool) (
	cursor IBucketCursor, ci *CursorInfo, err error) {

	cc.mux.Lock()
	defer cc.mux.Unlock()

	ci, found := cc.cursorsByKey[cursorKey]
	if !found {
		slog.Warn("Cursor not found or expired",
			slog.String("cursorKey", cursorKey),
			slog.String("clientID", clientID))
		return nil, ci, fmt.Errorf("Cursor not found or expired")
	} else if ci.OwnerID != clientID {
		slog.Warn("Cursor belongs to different client",
			slog.String("cursorKey", cursorKey),
			slog.String("ownerID", ci.OwnerID),
			slog.String("clientID", clientID))
		return nil, ci, fmt.Errorf("cursor doesn't belong to client '%s'", clientID)
	}
	if found && updateLastUsed {
		ci.LastUsed = time.Now().UTC()
	}
	return ci.Cursor, ci, nil
}

// GetExpiredCursors returns a list of cursors that have expired
// It is up to the user to remove and release the cursor
func (cc *CursorCache) GetExpiredCursors() []*CursorInfo {
	expiredCursors := make([]*CursorInfo, 0)
	cc.mux.RLock()
	defer cc.mux.RUnlock()

	now := time.Now().UTC()
	// rather brute force, might need a sorted list if heavily used
	// however, it is not expected to have a lot of active cursors.
	for _, ci := range cc.cursorsByKey {
		// if cursor hasn't been used it is considered expired
		if now.Sub(ci.LastUsed) > ci.Lifespan {
			expiredCursors = append(expiredCursors, ci)
		}
	}
	return expiredCursors
}

// GetCursorsByOwner returns a list of cursors that are owned by a client.
// Intended to remove cursors whose owner has disconnected.
// It is up to the user to remove and release the cursor
func (cc *CursorCache) GetCursorsByOwner(ownerID string) []*CursorInfo {
	ownedCursors := make([]*CursorInfo, 0)
	cc.mux.RLock()
	defer cc.mux.RUnlock()
	// rather brute force, might need to switch this to a map if heavily used
	for _, ci := range cc.cursorsByKey {
		if ci.OwnerID == ownerID {
			ownedCursors = append(ownedCursors, ci)
		}
	}
	return ownedCursors
}

// Release releases the cursor and removes the cursor from the tracker
// If a bucket was included it will be closed as well.
func (cc *CursorCache) Release(clientID string, cursorKey string) error {
	slog.Info("Release", "cursorKey", cursorKey)
	cc.mux.Lock()
	defer cc.mux.Unlock()
	ci, found := cc.cursorsByKey[cursorKey]
	if !found {
		slog.Error("Release. Cursor not found", "clientID", clientID, "cursorKey", cursorKey)
		return nil
	}
	if ci.OwnerID != clientID {
		slog.Warn("Release: Cursor belongs to different client",
			slog.String("cursorKey", cursorKey),
			slog.String("ownerID", ci.OwnerID),
			slog.String("clientID", clientID))
		return fmt.Errorf("cursor doesn't belong to client '%s'", clientID)
	}

	delete(cc.cursorsByKey, cursorKey)
	ci.Cursor.Release()
	if ci.Bucket != nil {
		err := ci.Bucket.Close()
		if err != nil {
			slog.Error("failed closing bucket after releasing cursor", "err", err)
		}
	}
	return nil
}

// Start starts a background loop to remove expired cursors
func (cc *CursorCache) Start() {
	go func() {
		for {
			ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
			select {
			case <-ctx.Done():
			case <-cc.stopCh:
				cancelFn()
				break
			}
			cancelFn()
			ciList := cc.GetExpiredCursors()
			for _, ci := range ciList {
				slog.Warn("Releasing expired cursor",
					slog.String("agentID", ci.OwnerID), slog.String("key", ci.Key))
				// release the expired cursor and remove it from the cache
				_ = cc.Release(ci.OwnerID, ci.Key)
			}
		}
	}()
}

// Stop the background auto-expiry loop if running
func (cc *CursorCache) Stop() {
	cc.stopCh <- true
}

func NewCursorCache() *CursorCache {
	cc := CursorCache{
		cursorsByKey:  make(map[string]*CursorInfo),
		cursorCounter: 1,
		mux:           sync.RWMutex{},
		stopCh:        make(chan bool),
	}
	return &cc
}
