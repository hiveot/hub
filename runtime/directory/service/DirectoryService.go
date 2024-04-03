package service

import (
	"context"
	"encoding/json"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/directory"
	"log/slog"
)

const TDBucketName = "td"

// DirectoryService is a wrapper around the internal store store
// This implements the IDirectory interface
type DirectoryService struct {
	cfg *directory.DirectoryConfig

	store        buckets.IBucketStore
	agentID      string // thingID of the service instance
	tdBucketName string
	tdBucket     buckets.IBucket
}

// GetTD returns the TD document in json format for the given Thing ID
func (svc *DirectoryService) GetTD(agentID, thingID string) (string, error) {

	//logrus.Infof("agentID=%s, thingID=%s", svc.agentID, thingID)
	// store keys are made of the agentID / thingID
	thingAddr := agentID + "/" + thingID
	raw, err := svc.tdBucket.Get(thingAddr)
	return string(raw), err
}

// GetTDs returns a collection of all TD documents
func (svc *DirectoryService) GetTDs(offset, limit int) ([]things.ThingValue, error) {
	batch := make([]things.ThingValue, 0, limit)

	cursor, err := svc.tdBucket.Cursor(context.Background())
	if err != nil {
		return nil, err
	}
	if offset > 0 {
		cursor.NextN(uint(offset))
	}
	docs, itemsRemaining := cursor.NextN(uint(limit))
	// FIXME: the unmarshalled ThingValue will be remarshalled when sending it as a reply.
	_ = itemsRemaining
	for key, val := range docs {
		tv := things.ThingValue{}
		err = json.Unmarshal(val, &tv)
		if err == nil {
			batch = append(batch, tv)
		} else {
			slog.Warn("unable to unmarshal TD", "err", err, "key", key)
		}
	}
	return batch, nil
}

// RemoveTD deletes the TD from the given agent with the ThingID
func (svc *DirectoryService) RemoveTD(agentID, thingID string) error {
	slog.Info("RemoveTD",
		slog.String("agentID", agentID),
		slog.String("thingID", thingID))

	thingAddr := agentID + "/" + thingID
	err := svc.tdBucket.Delete(thingAddr)
	return err
}

// Start the directory service and open the directory stored TD bucket
func (svc *DirectoryService) Start() (err error) {
	slog.Warn("Starting DirectoryService")
	// listen for requests
	tdBucket := svc.store.GetBucket(svc.tdBucketName)
	svc.tdBucket = tdBucket

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
// TODO: update the forms to point to the Hub instead of the device
func (svc *DirectoryService) UpdateTD(tv things.ThingValue) error {
	slog.Info("UpdateTD",
		slog.String("senderID", tv.SenderID),
		slog.String("agentID", tv.AgentID),
		slog.String("thingID", tv.ThingID))

	bucketData, _ := json.Marshal(tv)
	thingAddr := tv.AgentID + "/" + tv.ThingID
	err := svc.tdBucket.Set(thingAddr, bucketData)
	return err
}

// NewDirectoryService creates a new service instance for the directory of Thing TD documents.
//
//	store is an instance of the bucket store to store the directory data. This is opened by 'Start' and closed by 'Stop'
func NewDirectoryService(cfg *directory.DirectoryConfig, store buckets.IBucketStore) *DirectoryService {
	svc := &DirectoryService{
		cfg:          cfg,
		store:        store,
		tdBucketName: TDBucketName,
	}
	return svc
}
