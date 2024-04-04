package service

import (
	"context"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/runtime/directory"
	"log/slog"
	"time"
)

const TDBucketName = "td"

// DirectoryService stores, reads and queries TD documents
// TODO: replace TD forms with hub protocols.
//
// TD documents are stored in the bucket store under the name {TDBucketName}
type DirectoryService struct {
	cfg       *directory.DirectoryConfig
	cursorMgr *DirectoryCursorMgr

	store        buckets.IBucketStore
	tdBucketName string
	tdBucket     buckets.IBucket
}

// CursorMgr returns the directory cursor manager
func (svc *DirectoryService) CursorMgr() *DirectoryCursorMgr {
	return svc.cursorMgr
}

// GetTD returns the TD document in json format for the given Thing ID
func (svc *DirectoryService) GetTD(thingID string) (string, error) {

	raw, err := svc.tdBucket.Get(thingID)
	return string(raw), err
}

// GetTDs returns a list of json encoded TD documents
//
//	offset is the offset in the list
//	limit is the maximum number of records to return
func (svc *DirectoryService) GetTDs(offset, limit int) (tddList []string, err error) {
	tddList = make([]string, 0, limit)

	cursor, err := svc.tdBucket.Cursor(context.Background())
	if err != nil {
		return tddList, err
	}
	if offset > 0 {
		cursor.NextN(uint(offset))
	}
	docs, itemsRemaining := cursor.NextN(uint(limit))
	_ = itemsRemaining
	for _, tdd := range docs {
		tddList = append(tddList, string(tdd))
	}
	return tddList, nil
}

// QueryTDs the collection of TD documents
//func (svc *DirectoryService) QueryTDs(query string) (tddList []string, err error) {
//	// TBD: query based on what?
//	return nil, fmt.Errorf("not yet implemented")
//}

// RemoveTD deletes the TD from the given agent with the ThingID
func (svc *DirectoryService) RemoveTD(senderID string, thingID string) error {
	slog.Info("RemoveTD", slog.String("thingID", thingID),
		slog.String("senderID", senderID))

	err := svc.tdBucket.Delete(thingID)
	return err
}

// Start the directory service and open the directory stored TD bucket
func (svc *DirectoryService) Start() (err error) {
	slog.Warn("Starting DirectoryService")
	// listen for requests

	return err
}

// Stop the service
func (svc *DirectoryService) Stop() {
	slog.Warn("Stopping DirectoryService")
	if svc.tdBucket != nil {
		_ = svc.tdBucket.Close()
	}
}

// UpdateTD adds or updates the TDD
func (svc *DirectoryService) UpdateTD(senderID string, thingID string, tdd string) error {
	slog.Info("UpdateTD",
		slog.String("senderID", senderID),
		slog.String("thingID", thingID))

	// TODO: validate the thingID matches the tdd
	// TODO: update the forms to point to the Hub instead of the device

	err := svc.tdBucket.Set(thingID, []byte(tdd))
	return err
}

// NewDirectoryService creates a new service instance for the directory of Thing TD documents.
//
//	store is an instance of the bucket store to store the directory data. This is opened by 'Start' and closed by 'Stop'
func NewDirectoryService(cfg *directory.DirectoryConfig, store buckets.IBucketStore) *DirectoryService {
	tdBucket := store.GetBucket(TDBucketName)
	cursorMgr := NewDirectoryCursor(tdBucket, time.Second*cfg.CursorLifespan)
	svc := &DirectoryService{
		cfg:          cfg,
		store:        store,
		tdBucketName: TDBucketName,
		tdBucket:     tdBucket,
		cursorMgr:    cursorMgr,
	}
	return svc
}
