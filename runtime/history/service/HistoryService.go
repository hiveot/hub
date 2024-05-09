package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"path"
)

type HistoryService struct {
	storesDir string
	store     buckets.IBucketStore
}

func (svc *HistoryService) Start() (err error) {
	storeDir := path.Join(svc.storesDir, "values")
	storeName := "history"
	svc.store, err = bucketstore.NewBucketStore(
		storeDir, storeName, buckets.BackendPebble)
	if err == nil {
		err = svc.store.Open()
	}
	return err
}
func (svc *HistoryService) Stop() {
	_ = svc.store.Close()
}

// NewHistoryService returns a new history service instance
func NewHistoryService(storesDir string) *HistoryService {
	svc := HistoryService{
		storesDir: storesDir,
	}
	return &svc
}
