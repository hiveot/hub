package service

import (
	"encoding/json"
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/ser"
	"strings"

	"github.com/hiveot/hub/lib/thing"
)

// convert the storage key and raw data to a thing value object
// This returns the value, or nil if the key is invalid
func (svc *ReadDirectoryService) _decodeValue(key string, data []byte) (thingValue thing.ThingValue, valid bool) {
	// key is constructed as  {timestamp}/{valueName}/{a|e}
	parts := strings.Split(key, "/")
	if len(parts) < 2 {
		return thingValue, false
	}
	thingValue = thing.ThingValue{}
	_ = json.Unmarshal(data, &thingValue)
	return thingValue, true
}

// First returns the first entry in the directory
func (svc *ReadDirectoryService) First(
	ctx hubclient.ServiceContext, args directoryapi.CursorFirstArgs) (*directoryapi.CursorFirstResp, error) {

	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}
	k, v, valid := cursor.First()
	if !valid {
		// store is empty
		return nil, nil
	}
	thingValue, valid := svc._decodeValue(k, v)
	resp := directoryapi.CursorFirstResp{
		Value:     thingValue,
		Valid:     valid,
		CursorKey: args.CursorKey,
	}
	return &resp, nil
}

// Next moves the cursor to the next key from the current cursor
// First() or Seek must have been called first.
// Shouldn't next have a parameter?
func (svc *ReadDirectoryService) Next(
	ctx hubclient.ServiceContext, args directoryapi.CursorNextArgs) (*directoryapi.CursorNextResp, error) {

	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}

	k, v, valid := cursor.Next()
	thingValue, valid := svc._decodeValue(k, v)
	resp := directoryapi.CursorNextResp{
		Value:     thingValue,
		Valid:     valid,
		CursorKey: args.CursorKey,
	}
	return &resp, nil
}

// NextN moves the cursor to the next N places from the current cursor
// and return a list with N values in incremental time order.
// itemsRemaining is false if the iterator has reached the end.
// Intended to speed up with batch iterations over rpc.
func (svc *ReadDirectoryService) NextN(
	ctx hubclient.ServiceContext, args directoryapi.CursorNextNArgs) (*directoryapi.CursorNextNResp, error) {

	cursor, err := svc.cursorCache.Get(args.CursorKey, ctx.ClientID, true)
	if err != nil {
		return nil, err
	}
	values := make([]thing.ThingValue, 0, args.Limit)
	// obtain a map of [addr]TDJson
	docMap, itemsRemaining := cursor.NextN(args.Limit)
	for _, doc := range docMap {
		tv := thing.ThingValue{}
		err2 := ser.Unmarshal(doc, &tv)
		if err2 == nil {
			values = append(values, tv)
		} else {
			err = err2 // return the last error
		}
	}
	resp := directoryapi.CursorNextNResp{
		Values:         values,
		ItemsRemaining: itemsRemaining,
		CursorKey:      args.CursorKey,
	}
	return &resp, err
}

// Release close the cursor and release its resources.
// This invalidates all values obtained from the cursor
func (svc *ReadDirectoryService) Release(
	ctx hubclient.ServiceContext, args directoryapi.CursorReleaseArgs) error {

	return svc.cursorCache.Release(args.CursorKey, ctx.ClientID)
}
