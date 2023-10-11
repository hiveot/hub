package service

import (
	"encoding/json"
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/thing"
	"log/slog"
)

// ReadDirectoryService is a provides the capability to read and iterate the directory
type ReadDirectoryService struct {
	// the client that is reading the directory
	clientID string
	// read bucket that holds the TD documents
	bucket buckets.IBucket
	// capTable maps function names to methods
	capabilityMap map[string]interface{}
}

// GetCursor returns an iterator for ThingValues containing a TD document
// FIXME: marshalling a cursor isn't possible right now
func (svc *ReadDirectoryService) GetCursor() (cursor directory.IDirectoryCursor, err error) {
	//logrus.Infof("clientID=%s", svc.clientID)

	// FIXME: how to transfer the cursor?

	dirCursor := NewDirectoryCursor(svc.bucket.Cursor())
	return dirCursor, nil
}

// GetTD returns the TD document for the given Thing ID in JSON format
func (svc *ReadDirectoryService) GetTD(args *directory.GetTDArgs) (resp *directory.GetTDResp, err error) {
	//logrus.Infof("clientID=%s, thingID=%s", svc.clientID, thingID)
	// bucket keys are made of the agentID / thingID
	thingAddr := args.AgentID + "/" + args.ThingID
	raw, err := svc.bucket.Get(thingAddr)
	if raw != nil {
		resp = &directory.GetTDResp{}
		err = json.Unmarshal(raw, resp)
	}
	return resp, err
}

// GetTDsRaw returns a collection of TD documents
// Intended for transferring documents without unnecesary marshalling
func (svc *ReadDirectoryService) GetTDsRaw(args *directory.GetTDsArgs) (map[string][]byte, error) {
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
func (svc *ReadDirectoryService) GetTDs(args *directory.GetTDsArgs) (res *directory.GetTDsResp, err error) {
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

// Stop releases this capability and allocated resources after its use
func (svc *ReadDirectoryService) Stop() {
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
		clientID: clientID,
		bucket:   bucket,
	}
	svc.capabilityMap = map[string]interface{}{
		directory.GetCursorMethod: svc.GetCursor,
		directory.GetTDMethod:     svc.GetTD,
		directory.GetTDsMethod:    svc.GetTDs,
		//directory.GetTDsMethod:    svc.GetTDsRaw,
	}

	return svc, svc.capabilityMap
}
