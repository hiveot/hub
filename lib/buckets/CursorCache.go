package buckets

import (
	"strconv"
	"sync"
	"time"
)

type ClientCursors []IBucketCursor

type CursorInfo struct {
	Key string
	// the stored cursor
	Cursor interface{}
	// clientID of the cursor owner
	OwnerID string
	// expires
	Expires time.Time
}

// CursorCache manages a cache of cursors that can be addressed remotely by key.
// Intended for servers that let remote clients to iterate a cursor.
//
// The approach taken is store each cursor in a cache and generate a key for easy lookup.
// The key is passed back to the client for use during iterations.
// The client must release the cursor it when done.
//
// To prevent memory leaks due to not releasing a cursor the following constraints are used:
//   - cursors are linked to clientIDs. When a client disconnects, all cursors from this
//     client can be removed in one call.
//   - the number of cursors per client is limited. In most cases a few should suffice.
//     if the number of iterators exceeds the limit, the policy can be either to delete
//     the oldest or refuse a new iterator.
//   - the lifespan of unreleased cursors is limited as provided during creation.
type CursorCache struct {
	// lookup a cursor by key
	cursorsByKey map[string]CursorInfo

	// at 1000 cursors per sec this lasts 500M years between reboots ;)
	cursorCounter uint64
	mux           sync.RWMutex
}

// AddCursor adds a cursor to the tracker and returns its key
//
//	cursor is the object holding the cursor
//	clientID of the owner
//	lifespan of the cursor
func (sc *CursorCache) AddCursor(
	cursor interface{}, clientID string, lifespan time.Duration) string {
	sc.mux.Lock()
	defer sc.mux.Unlock()

	sc.cursorCounter++
	// the key is not a secret, only the owner can use it
	key := strconv.FormatUint(sc.cursorCounter, 16)
	ci := CursorInfo{
		Key:     key,
		Cursor:  cursor,
		OwnerID: clientID,
		Expires: time.Now().Add(lifespan),
	}
	sc.cursorsByKey[key] = ci
	return key
}

// GetCursor returns the cursor with the given key or nil if not found
func (sc *CursorCache) GetCursor(cursorKey string) (ci CursorInfo, found bool) {
	sc.mux.Lock()
	defer sc.mux.Unlock()

	ci, found = sc.cursorsByKey[cursorKey]
	return ci, found
}

// RemoveCursor removes the cursor from the tracker
// The ownser has to release the cursor after removal.
func (sc *CursorCache) RemoveCursor(cursorKey string) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	delete(sc.cursorsByKey, cursorKey)
}

// GetExpiredCursors returns a list of cursors that have expired
// It is up to the user to remove and release the cursor
func (sc *CursorCache) GetExpiredCursors() []CursorInfo {
	expiredCursors := make([]CursorInfo, 0)
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	now := time.Now()
	// rather brute force, might need a sorted list if heavily used
	for _, ci := range sc.cursorsByKey {
		if ci.Expires.Sub(now) < 0 {
			expiredCursors = append(expiredCursors, ci)
		}
	}
	return expiredCursors
}

// GetCursorsByOwner returns a list of cursors that are ownsed by a client.
// Intended to remove cursors whose owner has disconnected.
// It is up to the user to remove and release the cursor
func (sc *CursorCache) GetCursorsByOwner(ownerID string) []CursorInfo {
	ownedCursors := make([]CursorInfo, 0)
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	// rather brute force, might need to switch this to a map if heavily used
	for _, ci := range sc.cursorsByKey {
		if ci.OwnerID == ownerID {
			ownedCursors = append(ownedCursors, ci)
		}
	}
	return ownedCursors
}

func NewCursorCache() *CursorCache {
	cc := CursorCache{
		cursorsByKey:  make(map[string]CursorInfo),
		cursorCounter: 1,
		mux:           sync.RWMutex{},
	}
	return &cc
}
