package service

import (
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"log/slog"
	"path"
)

const StateStoreName = "state.kvbtree"

// StateService handles storage of client data records
type StateService struct {
	// backend storage
	storeDir string
	store    buckets.IBucketStore
}

func (svc *StateService) Delete(clientID string, key string) (err error) {
	bucket := svc.store.GetBucket(clientID)
	err = bucket.Delete(key)
	_ = bucket.Close()
	return err
}

func (svc *StateService) Get(clientID string, key string) (value string, err error) {
	bucket := svc.store.GetBucket(clientID)
	data, err := bucket.Get(key)
	// bucket returns an error if key is not found.
	if err == nil {
		err = bucket.Close()
	}
	return string(data), err
}

func (svc *StateService) GetMultiple(clientID string, keys []string) (map[string]string, error) {

	bucket := svc.store.GetBucket(clientID)
	kvRaw, err := bucket.GetMultiple(keys)
	err = bucket.Close()
	// convert values back to string
	kvStrings := make(map[string]string)
	for k, v := range kvRaw {
		kvStrings[k] = string(v)
	}
	return kvStrings, err
}

func (svc *StateService) Set(clientID string, key string, value string) (err error) {

	slog.Info("Set", slog.String("key", key))
	bucket := svc.store.GetBucket(clientID)
	// bucket returns an error if key is invalid
	err = bucket.Set(key, []byte(value))
	if err != nil {
		slog.Warn("Set; Invalid key", slog.String("key", key))
	}
	_ = bucket.Close()
	return err
}

func (svc *StateService) SetMultiple(clientID string, kv map[string]string) (err error) {
	slog.Info("SetMultiple", slog.Int("count", len(kv)))
	// convert to string :(
	storage := make(map[string][]byte)
	for k, v := range kv {
		storage[k] = []byte(v)
	}

	bucket := svc.store.GetBucket(clientID)
	err = bucket.SetMultiple(storage)
	_ = bucket.Close()
	return err
}

// Start the service
// This sets the permission for roles (any) that can use the state store and opens the store
func (svc *StateService) Start() (err error) {
	slog.Info("Starting the state service")
	storePath := path.Join(svc.storeDir, StateStoreName)
	svc.store = kvbtree.NewKVStore(storePath)

	err = svc.store.Open()
	return err
}

// Stop the service
func (svc *StateService) Stop() {
	slog.Warn("Stopping the state service")
	_ = svc.store.Close()
}

// NewStateService creates a new service instance using the kvstore
func NewStateService(storeDir string) *StateService {

	svc := &StateService{
		storeDir: storeDir,
	}

	return svc
}
