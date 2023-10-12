package service

import (
	"encoding/json"
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/thing"
	"log/slog"
	"time"
)

// ReadDirectoryService is a provides the capability to read and iterate the directory
type ReadDirectoryService struct {
	// the client that is reading the directory
	clientID string
	// read bucket that holds the TD documents
	bucket buckets.IBucket
	// capTable maps function names to methods
	capabilityMap map[string]interface{}
	//
	cursorCache *buckets.CursorCache
	//
	isRunning bool
}

// GetCursor returns an iterator for ThingValues containing a TD document
// The lifespan is currently fixed to 1 minute.
// FIXME: marshalling a cursor isn't possible right now
func (svc *ReadDirectoryService) GetCursor(
	clientID string) (directory.GetCursorResp, error) {

	dirCursor := svc.bucket.Cursor()
	// TODO: what lifespan is reasonable?
	key := svc.cursorCache.AddCursor(dirCursor, clientID, time.Minute)
	resp := directory.GetCursorResp{CursorKey: key}
	return resp, nil
}

// GetTD returns the TD document for the given Thing ID in JSON format
func (svc *ReadDirectoryService) GetTD(
	clientID string, args *directory.GetTDArgs) (resp *directory.GetTDResp, err error) {

	//logrus.Infof("clientID=%s, thingID=%s", svc.clientID, thingID)
	// bucket keys are made of the agentID / thingID
	thingAddr := args.AgentID + "/" + args.ThingID
	raw, err := svc.bucket.Get(thingAddr)
	if raw != nil {
		tv := thing.ThingValue{}
		err = json.Unmarshal(raw, &tv)
		resp = &directory.GetTDResp{
			Value: tv,
		}
	}
	return resp, err
}

// GetTDsRaw returns a collection of ThingValue documents
// Intended for transferring documents without unnecessary marshalling
func (svc *ReadDirectoryService) GetTDsRaw(
	clientID string, args *directory.GetTDsArgs) (map[string][]byte, error) {

	cursor := svc.bucket.Cursor()
	if args.Offset > 0 {
		// TODO: add support for cursor.Skip
		cursor.NextN(uint(args.Offset))
	}

	docs, itemsRemaining := cursor.NextN(uint(args.Limit))
	_ = itemsRemaining
	return docs, nil
}

// GetTDs returns a collection of TD documents
// this is rather inefficient. Should the client do the unmarshalling of the docs array?
// that would break the matching API. Maybe an internal method that returns a raw batch?
func (svc *ReadDirectoryService) GetTDs(
	clientID string, args *directory.GetTDsArgs) (res *directory.GetTDsResp, err error) {

	batch := make([]thing.ThingValue, 0, args.Limit)
	cursor := svc.bucket.Cursor()
	if args.Offset > 0 {
		// FIXME: add support for cursor.Skip
		cursor.NextN(uint(args.Offset))
	}
	docs, itemsRemaining := cursor.NextN(uint(args.Limit))
	// FIXME: the unmarshalled ThingValue will be remarshalled when sending it as a reply.
	_ = itemsRemaining
	for key, val := range docs {
		tv := thing.ThingValue{}
		err = json.Unmarshal(val, &tv)
		if err == nil {
			batch = append(batch, tv)
		} else {
			slog.Warn("unable to unmarshal TV", "err", err, "key", key)
		}
	}
	res = &directory.GetTDsResp{Values: batch}
	return res, err
}

//// ListTDs returns an array of TD documents in JSON text
//func (srv *DirectoryKVStoreServer) ListTDs(_ context.Context, limit int, offset int) ([]string, error) {
//	res := make([]string, 0)
//	docs, err := srv.store.List(srv.defaultBucket, limit, offset, nil)
//	if err == nil {
//		for _, doc := range docs {
//			res = append(res, doc)
//		}
//	}
//	return res, err
//}

// ListTDcb provides a callback with an array of TD documents in JSON text
//func (srv *DirectoryKVStoreServer) ListTDcb(
//	ctx context.Context, handler func(td string, isLast bool) error) error {
//	_ = ctx
//	batch := make([]string, 0)
//	docs, err := srv.store.List(srv.defaultBucket, 0, 0, nil)
//	if err == nil {
//		// convert map to array
//		for _, doc := range docs {
//			batch = append(batch, doc)
//		}
//		// for testing, callback one at a time
//		//err = handler(batch, true)
//		for i, tddoc := range batch {
//			docList := []string{tddoc}
//			isLast := i == len(batch)-1
//			err = handler(docList, isLast)
//		}
//	}
//	return err
//}

// QueryTDs returns an array of TD documents that match the jsonPath query
//  thingIDs optionally restricts the result to the given IDs
//func (srv *DirectoryKVStoreServer) QueryTDs(_ context.Context, jsonPathQuery string, limit int, offset int) ([]string, error) {
//
//	resp, err := srv.store.Query(jsonPathQuery, limit, offset, nil)
//	return resp, err
//	//res := make([]string, 0)
//	//if err == nil {
//	//	for _, docText := range resp {
//	//		var td thing.ThingDescription
//	//		err = json.Unmarshal([]byte(docText), &td)
//	//		res.Things = append(res.Things, &td)
//	//	}
//	//}
//	//return res, err
//}

// QueryTDs returns the TD's filtered using JSONpath on the TD content
// See 'docs/query-tds.md' for examples
// disabled as this is not used
//QueryTDs(ctx context.Context, jsonPath string, limit int, offset int) (tds []string, err error)

// Start managing director cursors
func (svc *ReadDirectoryService) Start() {
	svc.isRunning = true
	// start the main loop to cleanup expired cursors
	go func() {
		for svc.isRunning {
			ciList := svc.cursorCache.GetExpiredCursors()
			for _, ci := range ciList {
				slog.Info("Releasing expired cursor",
					slog.String("clientID", ci.OwnerID), slog.String("key", ci.Key))
				// release the expired cursor and remove it from the cache
				cursor := ci.Cursor.(buckets.IBucketCursor)
				cursor.Release()
				svc.cursorCache.RemoveCursor(ci.Key)
			}
			time.Sleep(time.Minute)
		}
	}()
}

// Stop releases this capability and allocated resources after its use
func (svc *ReadDirectoryService) Stop() {
	svc.isRunning = false
	// logrus.Infof("Released")
	err := svc.bucket.Close()
	_ = err
}

// NewReadDirectoryService returns the capability to read the directory
// bucket with the TD documents. Will be closed when done.
func NewReadDirectoryService(clientID string, bucket buckets.IBucket) (
	*ReadDirectoryService, map[string]interface{}) {

	// logrus.Infof("NewReadDirectoryService for bucket: ", bucket.ID())
	svc := &ReadDirectoryService{
		clientID:    clientID,
		bucket:      bucket,
		cursorCache: buckets.NewCursorCache(),
	}
	svc.capabilityMap = map[string]interface{}{
		directory.CursorFirstMethod: svc.First,
		directory.CursorNextMethod:  svc.Next,
		directory.CursorNextNMethod: svc.NextN,
		//directory.CursorPrevMethod: svc.Prev,
		directory.GetCursorMethod: svc.GetCursor,
		directory.GetTDMethod:     svc.GetTD,
		directory.GetTDsMethod:    svc.GetTDs,
		//directory.GetTDsMethod:    svc.GetTDsRaw,
	}

	return svc, svc.capabilityMap
}
